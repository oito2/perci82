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

package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/statusnotifier"
)

// TrayService binds the system tray / compact window domain. mainWindow
// is captured on the main window's creation; tray is non-nil only while
// TrayEnabled is on; compactWindow is the single compact window the
// "Contêineres" tray menu item opens/focuses (one reused window).
// windowIcon, trayIconName and the tray pixmaps are set by Run before the
// app starts. windowMu protects tray, trayDark and compactWindow — each
// bound method the frontend calls runs on its own goroutine, so concurrent
// SetTrayEnabled calls (e.g. a quick double-click on the tray toggle) can't
// leave `tray` inconsistent.
type TrayService struct {
	serviceBase

	mainWindow    *application.WebviewWindow
	windowMu      sync.Mutex
	tray          *statusnotifier.Item
	compactWindow *application.WebviewWindow
	windowIcon    []byte
	// trayIconName is the themed icon the panel paints in its own color.
	trayIconName string
	// trayPixmapDark and trayPixmapLight are the PNG fallbacks for dark and
	// light desktops, shown when the panel doesn't find trayIconName.
	trayPixmapDark  []byte
	trayPixmapLight []byte
	// trayDark reports whether the desktop prefers a dark color scheme.
	trayDark bool
}

// compactTabs is the fixed set of tabs the single compact window's tab
// bar offers — each one matches the `?compact=` query value /
// "compact-switch-tab" event payload, which the frontend resolves back to
// a MenuItem by actionId — the key here is literally the actionId of the
// item ("docker-manage").
var compactTabs = map[string]bool{
	"docker-manage": true,
}

// GetTrayEnabled exposes the tray toggle's current state (Home ::
// Configurações), so it starts checked correctly.
func (t *TrayService) GetTrayEnabled() bool {
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	return cfg.TrayEnabled
}

// SetTrayEnabled turns the system tray icon on/off (Home ::
// Configurações) — applies instantly, no restart needed: the icon is
// created and removed at runtime. Always reloads the config
// before writing — don't overwrite fields another process changed while
// the GUI was open.
func (t *TrayService) SetTrayEnabled(enabled bool) error {
	if err := config.Update(func(cfg *config.Config) error {
		cfg.TrayEnabled = enabled
		return nil
	}); err != nil {
		return err
	}

	if enabled {
		if err := t.ensureTray(); err != nil {
			return err
		}
		return writeAutostartDesktopFile()
	}

	t.destroyTray()
	return removeAutostartDesktopFile()
}

// ensureTray creates the tray icon if it doesn't exist yet — idempotent
// (called both at boot, when TrayEnabled is already on, and from
// SetTrayEnabled(true)). Panels open the menu themselves (right click; a
// single left click on GNOME's); activating the icon (left click; a double
// click on GNOME's) shows the main window.
func (t *TrayService) ensureTray() error {
	t.windowMu.Lock()
	defer t.windowMu.Unlock()

	if t.tray != nil {
		return nil
	}
	opts := statusnotifier.Options{
		ID:         "perci82-dev",
		Title:      "Perci82 Dev",
		IconName:   t.trayIconName,
		OnActivate: func() { application.InvokeAsync(t.FocusMainWindow) },
		Menu:       t.trayMenu(trayLocale()),
	}
	if p, ok := t.fallbackPixmap(); ok {
		opts.Pixmap = &p
	}
	item, err := statusnotifier.Start(opts)
	if err != nil {
		return fmt.Errorf("show the tray icon: %w", err)
	}
	t.tray = item
	return nil
}

// fallbackPixmap returns the tray pixmap matching the desktop's color
// scheme: the white pictogram on dark desktops, the graphite one on light
// ones. The caller holds windowMu.
func (t *TrayService) fallbackPixmap() (statusnotifier.Pixmap, bool) {
	data := t.trayPixmapLight
	if t.trayDark {
		data = t.trayPixmapDark
	}
	if len(data) == 0 {
		return statusnotifier.Pixmap{}, false
	}
	p, err := statusnotifier.PixmapFromPNG(data)
	if err != nil {
		return statusnotifier.Pixmap{}, false
	}
	return p, true
}

// setColorScheme records the desktop's color scheme ("dark", "light" or
// "" for no preference) and swaps the tray's fallback pixmap to match.
func (t *TrayService) setColorScheme(scheme string) {
	t.windowMu.Lock()
	defer t.windowMu.Unlock()

	dark := scheme == "dark"
	if dark == t.trayDark {
		return
	}
	t.trayDark = dark
	if t.tray == nil {
		return
	}
	if p, ok := t.fallbackPixmap(); ok {
		t.tray.SetPixmap(p)
	}
}

// trayLocale returns the locale of Perci82 Dev's language choice (field
// language); an unreadable config reads as "system".
func trayLocale() string {
	cfg, err := config.Load()
	if err != nil {
		return i18n.Resolve(i18n.SystemChoice)
	}
	return i18n.Resolve(cfg.Language)
}

// setLanguage rebuilds the tray menu in locale while the tray icon
// exists; without one it does nothing.
func (t *TrayService) setLanguage(locale string) {
	t.windowMu.Lock()
	defer t.windowMu.Unlock()

	if t.tray == nil {
		return
	}
	t.tray.SetMenu(t.trayMenu(locale))
}

// destroyTray undoes ensureTray when the toggle is turned off.
func (t *TrayService) destroyTray() {
	t.windowMu.Lock()
	defer t.windowMu.Unlock()

	if t.tray == nil {
		return
	}
	_ = t.tray.Close()
	t.tray = nil
}

// trayMenu returns the tray menu, labeled in locale — Abrir Aplicativo
// (shows/focuses the main window, same as the compact window's header
// button) + Contêineres (opens/focuses the single compact window, already
// on its tab) + Sair (the only way to quit Perci82 Dev while the tray is
// enabled, since closing the main window only hides it). The clicks arrive
// on the D-Bus goroutine and run on the UI thread.
func (t *TrayService) trayMenu(locale string) []statusnotifier.MenuItem {
	return []statusnotifier.MenuItem{
		{Label: i18n.T(locale, "tray.open", nil), OnClick: func() {
			application.InvokeAsync(t.FocusMainWindow)
		}},
		{Separator: true},
		{Label: i18n.T(locale, "tray.containers", nil), OnClick: func() {
			application.InvokeAsync(func() { t.openCompactWindow("docker-manage") })
		}},
		{Separator: true},
		{Label: i18n.T(locale, "tray.quit", nil), OnClick: func() {
			application.InvokeAsync(func() {
				if t.quitBlocked() {
					return
				}
				t.wailsApp.Quit()
			})
		}},
	}
}

// quitAllowed is set once Perci is quitting on purpose (a finished
// uninstall), so quitBlocked no longer refuses.
var quitAllowed atomic.Bool

// quitBlocked reports whether quitting must be refused because an action
// holds the action lock. When it does, it shows the main window and emits
// "quit-blocked" so the frontend opens the Execução tab with a notice.
func (t *TrayService) quitBlocked() bool {
	if quitAllowed.Load() || !actionRunning() {
		return false
	}
	t.FocusMainWindow()
	// The action may be running in the compact window: it is brought to
	// the front too, above the main window.
	t.windowMu.Lock()
	if t.compactWindow != nil {
		t.compactWindow.Show().Focus()
	}
	t.windowMu.Unlock()
	t.emit("quit-blocked", nil)
	return true
}

// mainCloseDecision decides what closing the main window does: with the
// tray enabled it cancels the close and hides the window; otherwise it
// cancels the close only while an action is running (quitBlocked), and lets
// the application quit when idle.
func (t *TrayService) mainCloseDecision() (cancel, hide bool) {
	if cfg, err := config.Load(); err == nil && cfg.TrayEnabled {
		return true, true
	}
	if t.quitBlocked() {
		return true, false
	}
	return false, false
}

// FocusMainWindow brings the main window to the front — used by the tray
// menu's "Abrir Aplicativo" item and the button of the same name in the
// compact window's header (#compact-header), to scale up from the tray to
// the full app (also when it starts hidden).
func (t *TrayService) FocusMainWindow() {
	if t.mainWindow == nil {
		return
	}
	t.mainWindow.Show().Focus()
}

// openCompactWindow opens the tray menu's single compact window (if it
// doesn't exist yet) already on the requested tab, or — if it's already
// open — just focuses it and tells the frontend to switch tabs via an
// event (one window, sized 735×825). The header (image+"Perci82 Dev"+"Abrir
// Aplicativo") and the tabs themselves live entirely in the frontend
// (?compact=<tab> skips the sidebar).
func (t *TrayService) openCompactWindow(tab string) {
	if !compactTabs[tab] {
		return
	}

	t.windowMu.Lock()
	defer t.windowMu.Unlock()

	if t.compactWindow != nil {
		t.compactWindow.Show().Focus()
		t.wailsApp.Event.Emit("compact-switch-tab", tab)
		return
	}

	win := t.wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Perci82 Dev",
		Width:     735,
		Height:    825,
		MinWidth:  480,
		MinHeight: 560,
		URL:       "/?compact=" + tab,
		Linux: application.LinuxWindow{
			Icon:             t.windowIcon,
			WebviewGpuPolicy: application.WebviewGpuPolicyNever,
		},
	})
	t.compactWindow = win

	win.RegisterHook(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		t.windowMu.Lock()
		t.compactWindow = nil
		t.windowMu.Unlock()
	})
}

// autostartDesktopPath is where writeAutostartDesktopFile/
// removeAutostartDesktopFile read/write — XDG autostart, read by the
// graphical session at login (not by the application menu:
// NoDisplay=true below).
// autostartFileName is the login autostart entry's file name under
// ~/.config/autostart.
const autostartFileName = "io.github.oito2.perci82.dev.desktop"

func autostartDesktopPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "autostart", autostartFileName), nil
}

// writeAutostartDesktopFile creates/updates the hidden tray autostart
// entry — called from inside SetTrayEnabled(true). Exec points at the
// real binary resolved via os.Executable() (the same function
// internal/selfupdate uses to find the binary itself), not the `perci`
// symlink — which may not exist, depending on how it was installed.
func writeAutostartDesktopFile() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	path, err := autostartDesktopPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	execLine, err := desktopExecQuote(exe)
	if err != nil {
		return err
	}
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Perci82 Dev
Comment=Perci82 Dev iniciado oculto na bandeja do sistema
Exec=%s %s
X-GNOME-Autostart-enabled=true
NoDisplay=true
Terminal=false
`, execLine, hiddenFlag)
	return os.WriteFile(path, []byte(content), 0o644)
}

// hiddenFlag starts Perci with its main window hidden — passed by the
// tray's autostart entry, so logging in doesn't pop the full window up.
const hiddenFlag = "--hidden"

// desktopExecQuote renders one argument for a Desktop Entry's Exec key: in
// double quotes, with double quote, backtick, dollar sign and backslash
// backslash-escaped (the spec's quoting rule), every backslash then escaped
// again (its string escaping rule, applied first when reading), and `%`
// doubled (field codes).
func desktopExecQuote(arg string) (string, error) {
	if strings.ContainsAny(arg, "\n\r") {
		return "", fmt.Errorf("invalid path for a .desktop file: %q", arg)
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range arg {
		switch r {
		case '"', '`', '$':
			b.WriteString(`\\`)
			b.WriteRune(r)
		case '\\':
			b.WriteString(`\\\\`)
		case '%':
			b.WriteString("%%")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String(), nil
}

// removeAutostartDesktopFile undoes writeAutostartDesktopFile when the
// toggle is turned off — a missing file (never enabled) is not an error.
func removeAutostartDesktopFile() error {
	path, err := autostartDesktopPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
