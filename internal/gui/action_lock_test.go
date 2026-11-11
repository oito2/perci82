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
	"errors"
	"testing"

	"github.com/oito2/perci82/internal/config"
)

// actionLockHeld reports whether action.lock is held, by trying it as the other
// application and releasing it again when it was free.
func actionLockHeld(t *testing.T) bool {
	t.Helper()
	release, err := config.LockActions(config.AppSystem)
	if err == nil {
		release()
		return false
	}
	var busy *config.ActionBusyError
	if !errors.As(err, &busy) {
		t.Fatalf("LockActions: %v", err)
	}
	return true
}

// Inside an action, the first privileged command takes action.lock and
// the action keeps it until it ends; outside one, the lock lasts one
// command; while the other application holds it, the command is refused
// naming that application.
func TestPrivilegedGate_ActionLock(t *testing.T) {
	isolateActions(t)
	setActionLockApp(config.AppDev)
	t.Cleanup(func() { setActionLockApp("") })

	beginAction()
	if actionLockHeld(t) {
		t.Fatal("action.lock taken before any privileged command")
	}
	for range 2 {
		leave, err := privilegedGate()
		if err != nil {
			t.Fatal(err)
		}
		leave()
		if !actionLockHeld(t) {
			t.Fatal("action.lock released before the action ended")
		}
	}
	endAction()
	if actionLockHeld(t) {
		t.Fatal("action.lock still held after the action ended")
	}

	leave, err := privilegedGate()
	if err != nil {
		t.Fatal(err)
	}
	if !actionLockHeld(t) {
		t.Fatal("privileged command outside an action ran without action.lock")
	}
	leave()
	if actionLockHeld(t) {
		t.Fatal("action.lock still held after the command outside an action")
	}

	release, err := config.LockActions(config.AppSystem)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	beginAction()
	defer endAction()
	_, err = privilegedGate()
	var busy *config.ActionBusyError
	if !errors.As(err, &busy) || busy.Holder != config.AppSystem {
		t.Fatalf("gate while Perci82 System holds the lock: err = %v", err)
	}
}

// Without an application set (tests, tools), the gate never touches
// action.lock.
func TestPrivilegedGate_NoAppNoLock(t *testing.T) {
	isolateActions(t)
	release, err := config.LockActions(config.AppSystem)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	leave, err := privilegedGate()
	if err != nil {
		t.Fatalf("gate without an application: %v", err)
	}
	leave()
}
