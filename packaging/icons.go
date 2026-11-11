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

// Package packaging embeds the icon sets of the two Perci82 applications:
// one freedesktop hicolor tree holding both apps' icons, named after their
// application IDs, and the Perci82 Dev tray pictograms (bandeja/). The
// 512 px icon doubles as the window icon.
package packaging

import (
	"embed"
	"fmt"
)

//go:embed hicolor bandeja
var icons embed.FS

// The application names Icon, IconName and IconRelPath accept.
const (
	AppSystem = "system"
	AppDev    = "dev"
)

// IconSizes lists every hicolor PNG size shipped for each application,
// largest first.
var IconSizes = []int{512, 256, 128, 64, 48, 32, 24, 22, 16}

// TraySizes lists every PNG size of the tray pictograms, largest first.
var TraySizes = []int{64, 48, 32, 24, 22, 16}

// IconName returns an application's freedesktop icon name: its application
// ID, used as the desktop entry's Icon= key and the installed file's
// basename (e.g. "io.github.oito2.perci82.dev").
func IconName(app string) string {
	return "io.github.oito2.perci82." + app
}

// IconRelPath returns an application's PNG icon path relative to a hicolor
// theme root (e.g. "48x48/apps/io.github.oito2.perci82.dev.png").
func IconRelPath(app string, size int) string {
	return fmt.Sprintf("%dx%d/apps/%s.png", size, size, IconName(app))
}

// Icon returns an application's PNG icon at one of IconSizes.
func Icon(app string, size int) ([]byte, error) {
	return icons.ReadFile("hicolor/" + IconRelPath(app, size))
}

// MustIcon is Icon for icons known to be embedded; it panics otherwise.
func MustIcon(app string, size int) []byte {
	data, err := Icon(app, size)
	if err != nil {
		panic(err)
	}
	return data
}

// TrayIcon returns the Perci82 Dev tray pictogram at one of TraySizes:
// variant "claro" (white, for dark panels) or "escuro" (graphite, for
// light panels).
func TrayIcon(variant string, size int) ([]byte, error) {
	return icons.ReadFile(fmt.Sprintf("bandeja/perci82-dev-tray-%s-%d.png", variant, size))
}

// MustTrayIcon is TrayIcon for icons known to be embedded; it panics
// otherwise.
func MustTrayIcon(variant string, size int) []byte {
	data, err := TrayIcon(variant, size)
	if err != nil {
		panic(err)
	}
	return data
}
