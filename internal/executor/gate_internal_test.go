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

package executor

import (
	"errors"
	"testing"
)

// Every privileged command passes the privileged gate before it runs: a
// gate error refuses it, an unprivileged command and a dry run never call
// the gate.
func TestPrivilegedGate(t *testing.T) {
	refused := errors.New("refused")
	calls := 0
	SetPrivilegedGate(func() (func(), error) {
		calls++
		return nil, refused
	})
	t.Cleanup(func() { SetPrivilegedGate(nil) })

	e := New(nil, nil)
	ctx := t.Context()
	if err := e.Run(ctx, Options{RequiresSudo: true}, "true"); !errors.Is(err, refused) {
		t.Errorf("Run: err = %v, want the gate's error", err)
	}
	if _, err := e.Output(ctx, Options{RequiresSudo: true}, "true"); !errors.Is(err, refused) {
		t.Errorf("Output: err = %v, want the gate's error", err)
	}
	if err := e.RunSudoSequence(ctx, Options{}, []PrivilegedStep{{Name: "true"}}); !errors.Is(err, refused) {
		t.Errorf("RunSudoSequence: err = %v, want the gate's error", err)
	}
	if calls != 3 {
		t.Errorf("gate called %d times, want 3", calls)
	}

	if err := e.Run(ctx, Options{}, "true"); err != nil {
		t.Errorf("unprivileged Run: %v", err)
	}
	dry := &Executor{DryRun: true}
	if err := dry.Run(ctx, Options{RequiresSudo: true}, "true"); err != nil {
		t.Errorf("dry-run Run: %v", err)
	}
	if calls != 3 {
		t.Errorf("gate called by an unprivileged command or a dry run (%d calls)", calls)
	}
}
