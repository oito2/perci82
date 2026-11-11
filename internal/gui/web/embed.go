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

// Package web embeds the frontend files both Perci82 applications share:
// the vendored libraries, the sidebar logo images, the oito2 stylesheet
// and the common scripts.
// Each application serves them under its own frontend (see gui.Run).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:vendor all:assets all:css all:js
var files embed.FS

// FS returns the shared frontend files, rooted so that "js/core.js" is a
// valid path.
func FS() fs.FS {
	return files
}
