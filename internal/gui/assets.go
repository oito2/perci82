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
	"io/fs"
)

// overlayFS serves each path from top when top has it, and from bottom
// otherwise: an application's own frontend files over the shared ones.
type overlayFS struct {
	top, bottom fs.FS
}

// Open opens name from top, falling back to bottom when top has no such
// file.
func (o overlayFS) Open(name string) (fs.File, error) {
	f, err := o.top.Open(name)
	if err == nil {
		return f, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return o.bottom.Open(name)
}
