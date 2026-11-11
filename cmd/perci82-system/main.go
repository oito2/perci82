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

// Command perci82-system is Perci82 System, the Linux administration
// desktop application: post-install setup, system updates, fonts, file
// templates and Flatpak apps.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/gui"
	"github.com/oito2/perci82/packaging"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	os.Exit(run())
}

func run() int {
	dist, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro ao carregar a interface:", err)
		return 1
	}
	home := &gui.SystemHomeService{}
	linux := &gui.LinuxService{}
	return gui.Run(gui.Profile{
		App:        config.AppSystem,
		Name:       "Perci82 System",
		ID:         packaging.IconName(packaging.AppSystem),
		WindowIcon: packaging.MustIcon(packaging.AppSystem, 512),
	}, dist, os.Args, os.Stdout, os.Stderr,
		gui.Bind(home, application.NewService(home)),
		gui.Bind(linux, application.NewService(linux)),
	)
}
