// Copyright (C) 2026  oito2
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package gui holds the Wails v3 services of the two Perci82 desktop
// applications (GTK4 + WebKitGTK 6.0 on Linux) and Run, which wires the
// services a cmd/ entry point registers to the main window.
package gui

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/dev/localbin"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/gui/web"
	"github.com/oito2/perci82/internal/manager/gitaccount"
	"github.com/oito2/perci82/internal/theme"
	"github.com/oito2/perci82/internal/ui"
)

// Profile describes one Perci82 application.
type Profile struct {
	// Name is the window title and the Wails application name.
	Name string
	// ID is the reverse-DNS application ID: the GTK application ID, the
	// Wayland app_id matched against the .desktop file, and the
	// single-instance lock name.
	ID string
	// WindowIcon is the PNG window icon.
	WindowIcon []byte
	// TrayIcon is the PNG tray icon, used only when a TrayService is bound.
	TrayIcon []byte
	// App is the application config.Update writes keys for and the holder
	// written into the shared action lock.
	App config.App
}

// Service is a bound service Run can wire: every service type of this
// package.
type Service interface {
	base() *serviceBase
}

func (b *serviceBase) base() *serviceBase { return b }

// Bound pairs a service with its Wails registration.
type Bound struct {
	svc        Service
	registered application.Service
}

// Bind pairs svc with registered, which must be application.NewService(svc)
// called in the entry point's main package: the Wails binding generator
// only finds services registered there.
func Bind(svc Service, registered application.Service) Bound {
	return Bound{svc: svc, registered: registered}
}

// handleSecondInstance reacts to a later launch of the application while
// this instance is running: args are the second process's os.Args. A
// launch with hiddenFlag (the login autostart) changes nothing; any other
// launch calls focus to bring the main window to the front.
func handleSecondInstance(args []string, focus func()) {
	if len(args) > 1 && slices.Contains(args[1:], hiddenFlag) {
		return
	}
	focus()
}

// blockQuit reports whether quitting must be refused because an action is
// running; when it is, it shows the main window and emits "quit-blocked".
func blockQuit(b *serviceBase, mainWindow *application.WebviewWindow) bool {
	if quitAllowed.Load() || !actionRunning() {
		return false
	}
	mainWindow.Show().Focus()
	b.emit("quit-blocked", nil)
	return true
}

// actionRunning reports whether a runAction/runExclusive action holds
// runMu.
func actionRunning() bool {
	if runMu.TryLock() {
		runMu.Unlock()
		return false
	}
	return true
}

// watchSystemSettings reports the desktop's appearance changes — a
// variable so tests never watch the machine's session bus.
var watchSystemSettings = theme.WatchSystem

// Run starts the application described by p, serving assets (the
// application's own frontend, over the shared web.FS files) and binding
// services, and returns the process exit code. args are the process
// arguments (os.Args); stderr receives startup warnings and errors.
func Run(p Profile, assets fs.FS, args []string, stdout, stderr io.Writer, services ...Bound) int {
	// Started by ssh-keygen/ssh-add as their askpass program: answer the
	// passphrase prompt and exit without starting the GUI.
	if gitaccount.IsAskpassCall() {
		return gitaccount.RespondAskpass(stdout)
	}

	// Writes only this application's config keys, and privileged commands
	// wait for no action of the other application.
	config.SetApp(p.App)
	setActionLockApp(p.App)
	executor.SetPrivilegedGate(privilegedGate)

	var tray *TrayService
	registered := make([]application.Service, 0, len(services))
	for _, s := range services {
		registered = append(registered, s.registered)
		if t, ok := s.svc.(*TrayService); ok {
			tray = t
		}
	}
	// A language switch relabels the tray menu.
	if tray != nil {
		for _, s := range services {
			if h, ok := s.svc.(*DevHomeService); ok {
				h.onLanguageChanged = tray.setLanguage
			}
		}
	}

	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		fmt.Fprintln(stderr, "aviso: não foi possível ler as configurações, usando os padrões:", cfgErr)
	}
	trayOn := tray != nil && cfgErr == nil && cfg.TrayEnabled
	// The tray's autostart entry launches the binary with hiddenFlag: on
	// login the application starts in the tray only. Honored only while the
	// tray is enabled — without it, a hidden window would leave no way to
	// reach the app.
	startHidden := trayOn && slices.Contains(args[1:], hiddenFlag)

	var mainWindow *application.WebviewWindow
	focusMain := func() {
		if mainWindow != nil {
			mainWindow.Show().Focus()
		}
	}

	wailsApp := application.New(application.Options{
		Name:     p.Name,
		Services: registered,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(overlayFS{top: assets, bottom: web.FS()}),
		},
		Linux: application.LinuxOptions{ApplicationID: p.ID},
		// Only one process of each application runs per session: a second
		// launch hands its arguments to the running instance over D-Bus and
		// exits, so it never creates a second window or tray icon.
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: p.ID,
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				handleSecondInstance(data.Args, func() { application.InvokeAsync(focusMain) })
			},
		},
	})

	// internal/executor's UsePolicyKit makes RequiresSudo escalate via
	// pkexec instead of sudo — the GUI has no interactive terminal for sudo
	// to prompt on. Each service gets its own Executor instance.
	var first *serviceBase
	for _, s := range services {
		b := s.svc.base()
		b.wailsApp = wailsApp
		b.exe = executor.New(nil, nil)
		b.exe.UsePolicyKit = true
		if r, ok := s.svc.(*RepositoriesService); ok {
			// ssh-keygen and ssh-add read passphrases from this executable
			// (see the askpass check above).
			if self, err := os.Executable(); err == nil {
				r.askpass = gitaccount.Askpass{Program: self}
			}
		}
		if first == nil {
			first = b
		}
	}
	if first == nil {
		fmt.Fprintln(stderr, "erro ao iniciar o "+p.Name+": nenhum serviço registrado")
		return 1
	}

	// A GUI started from the desktop doesn't get the PATH a login shell
	// builds (~/.local/bin, nvm's Node): tools installed there would read
	// as missing. Blocking, but only a shell sourcing nvm.sh — before any
	// binding can check what's installed.
	localbin.RefreshPath(context.Background(), first.exe)

	// ui.Step normally just writes one line; the hook also emits a "step"
	// event to the frontend (title + progress bar in the Execução tab).
	ui.SetStepHook(func(index, total int, label string) {
		wailsApp.Event.Emit("step", map[string]interface{}{
			"index": index,
			"total": total,
			"label": label,
		})
	})

	mainWindow = wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     p.Name,
		Width:     1100,
		Height:    720,
		MinWidth:  800,
		MinHeight: 560,
		Hidden:    startHidden,
		Linux: application.LinuxWindow{
			Icon: p.WindowIcon,
			// Explicit: options.Linux == nil would default to
			// WebviewGpuPolicyNever, but passing Icon makes it non-nil, and
			// the zero value would then be WebviewGpuPolicyOnDemand.
			WebviewGpuPolicy: application.WebviewGpuPolicyNever,
		},
	})

	if tray != nil {
		tray.mainWindow = mainWindow
		tray.windowIcon = p.WindowIcon
		tray.trayIcon = p.TrayIcon
	}

	// Closing (X): with a TrayService bound, it hides into the tray while
	// the tray is enabled (read from config on every close, so toggling it
	// at runtime takes effect immediately). Otherwise X quits, unless an
	// action is running: then the window stays open.
	mainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if tray != nil {
			cancel, hide := tray.mainCloseDecision()
			if cancel {
				e.Cancel()
			}
			if hide {
				mainWindow.Hide()
			}
			return
		}
		if blockQuit(first, mainWindow) {
			e.Cancel()
		}
	})

	if trayOn {
		tray.ensureTray()
		// Rewritten on every start while the tray is on, so an entry with an
		// outdated format or pointing at a moved binary is brought up to date.
		if err := writeAutostartDesktopFile(); err != nil {
			fmt.Fprintln(stderr, "aviso: não foi possível atualizar o autostart da bandeja:", err)
		}
	}

	// "Definido pelo sistema" follows the desktop's appearance settings
	// while the application runs.
	watchCtx, stopWatch := context.WithCancel(context.Background())
	var watchers sync.WaitGroup
	defer watchers.Wait()
	defer stopWatch()
	watchers.Go(func() {
		_ = watchSystemSettings(watchCtx, func(s theme.SystemSettings) {
			wailsApp.Event.Emit("system-scheme-changed", s.Resolve())
		})
	})

	if err := wailsApp.Run(); err != nil {
		fmt.Fprintln(stderr, "erro ao iniciar o "+p.Name+":", err)
		return 1
	}
	return 0
}
