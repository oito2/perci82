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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/dev/localbin"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
)

// serviceBase is embedded by value in every domain service struct,
// sharing the plumbing common to all of them (wailsApp/exe, pickFolder,
// runAction, eventWriter). pickFolder/runAction stay unexported so
// they're never bindable by `wails3 generate bindings` even though
// they're promoted onto every embedding struct.
type serviceBase struct {
	wailsApp *application.App
	exe      *executor.Executor
	// emitFn replaces wailsApp.Event.Emit when set — tests only.
	emitFn func(name string, data any)
}

// emit sends a Wails event to every window.
func (b *serviceBase) emit(name string, data any) {
	if b.emitFn != nil {
		b.emitFn(name, data)
		return
	}
	b.wailsApp.Event.Emit(name, data)
}

// approvedPaths holds every path the user picked in a native dialog during
// this session (all Pick* methods go through approvePath). Methods that
// write or read files at a path coming from the frontend accept only these
// (requireApprovedPath): the frontend is a webview, and a bound method that
// writes arbitrary content to any path it's handed would turn anything
// running there into a write-anywhere primitive (~/.bashrc, autostart
// entries...). Package-level, not per service: each service embeds its own
// serviceBase by value.
var approvedPaths = struct {
	sync.Mutex
	set map[string]struct{}
}{set: map[string]struct{}{}}

// approvePath records a dialog's result (empty = cancelled) and passes it
// through unchanged.
func approvePath(path string, err error) (string, error) {
	if err == nil && path != "" {
		approvedPaths.Lock()
		approvedPaths.set[filepath.Clean(path)] = struct{}{}
		approvedPaths.Unlock()
	}
	return path, err
}

// requireApprovedPath rejects any path that didn't come from a native
// dialog in this session.
func requireApprovedPath(path string) error {
	if path == "" {
		return errors.New(i18n.Tr("path.noneSelected", nil))
	}
	approvedPaths.Lock()
	_, ok := approvedPaths.set[filepath.Clean(path)]
	approvedPaths.Unlock()
	if !ok {
		return errors.New(i18n.Tr("path.notPicked", i18n.Args{"path": path}))
	}
	return nil
}

// requireFolder validates a project folder coming from the frontend before
// anything acts on it: an absolute path to an existing directory. The
// domain functions treat "" as "the current directory" — in a GUI that's
// wherever Perci was launched from ($HOME, or / from autostart), so an
// empty value would write project files or a git repo there.
func requireFolder(p string) error {
	if !filepath.IsAbs(p) {
		return errors.New(i18n.Tr("path.invalidFolder", i18n.Args{"path": p}))
	}
	info, err := os.Stat(p)
	if err != nil || !info.IsDir() {
		return errors.New(i18n.Tr("path.notAFolder", i18n.Args{"path": p}))
	}
	return nil
}

// pickFolder opens the native "choose a folder" dialog with the options
// every Pick*Folder screen-specific method wants — CanCreateDirectories so
// the user can create a brand-new empty folder from inside the dialog (the
// common case for "clone/init/install something new here"). Only the title
// differs between callers.
func (b *serviceBase) pickFolder(title string) (string, error) {
	return approvePath(b.wailsApp.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:                title,
		CanChooseDirectories: true,
		CanChooseFiles:       false,
		CanCreateDirectories: true,
	}).PromptForSingleSelection())
}

// Output batching limits: a "log-line" event is emitted at most once per
// logBatchInterval, or at once when the pending bytes reach logBatchMax.
const (
	logBatchInterval = 30 * time.Millisecond
	logBatchMax      = 16 << 10
)

// eventWriter connects the io.Writer convention every domain function
// already uses (ui.Info/Warning/Success/Step, real command output via
// internal/executor) to the Execução tab's terminal panel: the output
// becomes "log-line" events. Writes are batched — apt or docker build emit
// thousands of small writes — so an event goes out at most every interval
// (the first write after a quiet period goes out immediately; a timer
// emits whatever accumulated otherwise) or as soon as logBatchMax bytes are
// pending. Safe for concurrent use (a command's stdout and stderr are
// copied from different goroutines), and an incomplete UTF-8 sequence at
// the end of the pending bytes is held back until the rest arrives —
// emitted on its own it would show as "�" in the terminal. Flush emits
// whatever is left once the action ends.
type eventWriter struct {
	emit func(string)
	// interval overrides logBatchInterval when non-zero.
	interval time.Duration
	mu       sync.Mutex
	buf      []byte
	last     time.Time
	timer    *time.Timer
}

func (b *serviceBase) newEventWriter() *eventWriter {
	return &eventWriter{emit: func(s string) { b.emit("log-line", s) }}
}

func (w *eventWriter) batchInterval() time.Duration {
	if w.interval > 0 {
		return w.interval
	}
	return logBatchInterval
}

func (w *eventWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	if len(w.buf) >= logBatchMax || time.Since(w.last) >= w.batchInterval() {
		w.emitComplete()
	} else if w.timer == nil {
		w.timer = time.AfterFunc(w.batchInterval(), w.onTimer)
	}
	return len(p), nil
}

// onTimer emits the output accumulated since the last emission.
func (w *eventWriter) onTimer() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.timer = nil
	w.emitComplete()
}

// emitComplete emits the pending bytes up to the last complete UTF-8 rune
// and keeps the rest. The caller holds w.mu.
func (w *eventWriter) emitComplete() {
	n := completeUTF8Prefix(w.buf)
	if n == 0 {
		return
	}
	w.emit(string(w.buf[:n]))
	w.buf = append(w.buf[:0], w.buf[n:]...)
	w.last = time.Now()
}

// Flush emits any pending bytes, even an invalid trailing sequence, and
// stops the batching timer.
func (w *eventWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.emitComplete()
	if len(w.buf) > 0 {
		w.emit(string(w.buf))
		w.buf = w.buf[:0]
	}
}

// completeUTF8Prefix returns the length of b without a trailing rune that
// is still missing bytes. Anything else — including invalid bytes — is
// passed through, so a non-UTF-8 stream is never held back.
func completeUTF8Prefix(b []byte) int {
	n := len(b)
	for i := 1; i < utf8.UTFMax && i <= n; i++ {
		if utf8.RuneStart(b[n-i]) {
			if !utf8.FullRune(b[n-i:]) {
				return n - i
			}
			break
		}
	}
	return n
}

// runMu serializes long-running actions across every service and window:
// package-level because each service embeds its own serviceBase, and the
// tray's compact window runs actions through the same services.
// Two actions at once would interleave their output in the terminal, race
// on the package manager's lock and on the Nginx config, and prompt for the
// password twice.
var runMu sync.Mutex

// actionLock holds ~/.perci/action.lock, shared with the other Perci82
// application, from the first privileged command of an action until the
// action ends, so the two applications never run package managers or
// pkexec at the same time. app is the running application (empty turns
// the lock off); mu guards every field.
var actionLock struct {
	mu       sync.Mutex
	app      config.App
	inAction bool
	release  func()
}

// setActionLockApp makes privilegedGate take action.lock for app.
func setActionLockApp(app config.App) {
	actionLock.mu.Lock()
	defer actionLock.mu.Unlock()
	actionLock.app = app
}

// beginAction marks the start of an action holding runMu.
func beginAction() {
	actionLock.mu.Lock()
	defer actionLock.mu.Unlock()
	actionLock.inAction = true
}

// endAction marks the end of the action started by beginAction, releasing
// action.lock when the action took it.
func endAction() {
	actionLock.mu.Lock()
	defer actionLock.mu.Unlock()
	actionLock.inAction = false
	if actionLock.release != nil {
		actionLock.release()
		actionLock.release = nil
	}
}

// privilegedGate is installed as the executor's privileged gate: inside an
// action it takes action.lock on the first privileged command and keeps it
// until endAction; outside one it holds the lock for that command only.
// While the other application holds the lock, the command is refused with
// a *config.ActionBusyError.
func privilegedGate() (func(), error) {
	actionLock.mu.Lock()
	defer actionLock.mu.Unlock()
	if actionLock.app == "" {
		return func() {}, nil
	}
	if !actionLock.inAction {
		return config.LockActions(actionLock.app)
	}
	if actionLock.release == nil {
		release, err := config.LockActions(actionLock.app)
		if err != nil {
			return nil, err
		}
		actionLock.release = release
	}
	return func() {}, nil
}

// errActionRunning is returned (synchronously, to the one caller) when
// another action is still in progress — never announced through
// "action-done", which every window receives.
var errActionRunning = i18n.NewError("action.running")

// runAction runs fn on its own goroutine (the "Executar" button click must
// not block the UI), with its output going to the terminal via eventWriter,
// and notifies the frontend when it finishes via the "action-done" event.
// Only one action runs at a time (runMu); a panic inside fn is turned into
// a failed action instead of taking the whole process down — possibly in
// the middle of a privileged package-manager run — and "action-done" is
// always emitted, after runMu is released, so an action started as soon as
// it arrives is not refused.
func (b *serviceBase) runAction(fn func(stdout io.Writer) error) error {
	if !runMu.TryLock() {
		return errActionRunning
	}
	go func() {
		err := func() error {
			defer runMu.Unlock()
			beginAction()
			defer endAction()
			w := b.newEventWriter()
			err := runRecovered(func() error { return fn(w) })
			w.Flush()
			// The action may have installed a tool into ~/.local/bin or
			// through nvm: refreshed before action-done, whose handlers
			// re-check what's installed.
			localbin.RefreshPath(context.Background(), b.exe)
			return err
		}()
		b.emitDone(err)
	}()
	return nil
}

// runExclusive runs fn synchronously while holding the action lock (runMu)
// — for quick bound methods that change state without the Execução tab. It
// returns errActionRunning when another action holds the lock.
func runExclusive(fn func() error) error {
	if !runMu.TryLock() {
		return errActionRunning
	}
	defer runMu.Unlock()
	beginAction()
	defer endAction()
	return runRecovered(fn)
}

// runRecovered calls fn, converting a panic into an error.
func runRecovered(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v", r)
		}
	}()
	return fn()
}

// emitDone sends the "action-done" event the frontend's run lifecycle
// waits for.
func (b *serviceBase) emitDone(err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	b.emit("action-done", map[string]interface{}{
		"ok":    err == nil,
		"error": msg,
	})
}
