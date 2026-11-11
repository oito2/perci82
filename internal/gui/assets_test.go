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
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestOverlayFS_TopWinsThenBottom(t *testing.T) {
	o := overlayFS{
		top:    fstest.MapFS{"index.html": {Data: []byte("app")}, "js/app.js": {Data: []byte("app js")}},
		bottom: fstest.MapFS{"index.html": {Data: []byte("common")}, "js/core.js": {Data: []byte("core")}},
	}
	for name, want := range map[string]string{"index.html": "app", "js/app.js": "app js", "js/core.js": "core"} {
		got, err := fs.ReadFile(o, name)
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", name, got, err, want)
		}
	}
	if _, err := o.Open("missing.js"); err == nil {
		t.Error("a file in neither layer must not open")
	}
}
