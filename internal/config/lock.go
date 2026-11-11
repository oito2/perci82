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

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	configLockFile = "config.lock"
	actionLockFile = "action.lock"
)

// ensureDir creates ~/.perci with mode 0700 (tightening an existing,
// looser directory) and returns its path.
func ensureDir() (string, error) {
	path, err := configPath()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", fmt.Errorf("chmod config dir: %w", err)
	}
	return dir, nil
}

// openLockFile opens (creating it with mode 0600) the lock file name in
// ~/.perci.
func openLockFile(name string) (*os.File, error) {
	dir, err := ensureDir()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	return f, nil
}

// lockConfig takes the exclusive config.lock, waiting while another
// process holds it, and returns the function that releases it.
func lockConfig() (func(), error) {
	f, err := openLockFile(configLockFile)
	if err != nil {
		return nil, err
	}
	if err := flock(f, syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock %s: %w", configLockFile, err)
	}
	return func() {
		_ = flock(f, syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

// flock calls flock(2) on f, retrying when a signal interrupts it.
func flock(f *os.File, how int) error {
	for {
		err := syscall.Flock(int(f.Fd()), how)
		if !errors.Is(err, syscall.EINTR) {
			return err
		}
	}
}

// ActionBusyError is returned by LockActions while another process holds
// action.lock. Holder is the application that holds it, empty when it
// could not be read.
type ActionBusyError struct {
	Holder App
}

func (e *ActionBusyError) Error() string {
	if e.Holder == "" {
		return "outra ação de um aplicativo Perci82 está em andamento — aguarde ela terminar"
	}
	return "outra ação do " + e.Holder.DisplayName() + " está em andamento — aguarde ela terminar"
}

// LockActions takes the exclusive action.lock for app without waiting,
// writing app into the file so another process can name the holder, and
// returns the function that releases it. While another process holds the
// lock it returns an *ActionBusyError.
func LockActions(app App) (func(), error) {
	f, err := openLockFile(actionLockFile)
	if err != nil {
		return nil, err
	}
	if err := flock(f, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, &ActionBusyError{Holder: readHolder()}
		}
		return nil, fmt.Errorf("lock %s: %w", actionLockFile, err)
	}
	if err := f.Truncate(0); err == nil {
		_, _ = f.WriteAt([]byte(app), 0)
	}
	return func() {
		_ = f.Truncate(0)
		_ = flock(f, syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

// readHolder returns the application written into action.lock, empty
// when the file can't be read or holds an unknown name.
func readHolder() App {
	path, err := configPath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(path), actionLockFile))
	if err != nil {
		return ""
	}
	switch app := App(strings.TrimSpace(string(data))); app {
	case AppSystem, AppDev:
		return app
	}
	return ""
}
