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

package theme

import (
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// A burst of signals is waited out; the read happens once it ends.
func TestSettle(t *testing.T) {
	orig := settleDelay
	settleDelay = 50 * time.Millisecond
	t.Cleanup(func() { settleDelay = orig })
	signals := make(chan *dbus.Signal, 8)
	for range 5 {
		signals <- &dbus.Signal{}
	}
	if !settle(t.Context(), signals) {
		t.Fatal("settle ended early")
	}
	if len(signals) != 0 {
		t.Errorf("%d signals left after the burst", len(signals))
	}
	close(signals)
	if settle(t.Context(), signals) {
		t.Error("a closed channel should end settle")
	}
}
