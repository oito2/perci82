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

// Command perci82-dev is Perci82 Dev, the development environment desktop
// application: SDKs, IDEs, Docker containers, Git repositories, AI
// contexts, skills and MCP servers.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

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
	home := &gui.DevHomeService{}
	devSetup := &gui.DevSetupService{}
	docker := &gui.DockerService{}
	devTools := &gui.DevToolsService{}
	repos := &gui.RepositoriesService{}
	tray := &gui.TrayService{}
	return gui.Run(gui.Profile{
		Name:       "Perci82 Dev",
		ID:         packaging.IconName(packaging.AppDev),
		WindowIcon: packaging.MustIcon(packaging.AppDev, 512),
		// The white pictogram, for the dark panels most desktops use.
		TrayIcon: packaging.MustTrayIcon("claro", 64),
	}, dist, os.Args, os.Stdout, os.Stderr,
		gui.Bind(home, application.NewService(home)),
		gui.Bind(devSetup, application.NewService(devSetup)),
		gui.Bind(docker, application.NewService(docker)),
		gui.Bind(devTools, application.NewService(devTools)),
		gui.Bind(repos, application.NewService(repos)),
		gui.Bind(tray, application.NewService(tray)),
	)
}
