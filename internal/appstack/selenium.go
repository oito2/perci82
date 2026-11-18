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

package appstack

import (
	"context"
	"fmt"
	"io"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// SeleniumContainerName is the fixed name of the stack's shared Selenium
// container; Moodle's Behat reaches its WebDriver endpoint at
// http://selenium:4444/wd/hub through NetworkName.
const SeleniumContainerName = "selenium"

// SeleniumImage is the Selenium standalone Chrome image the Selenium
// container is created from. The tag names one Grid release and build
// date, so a newer Chrome never arrives without a Perci82 Dev update.
const SeleniumImage = "selenium/standalone-chrome:4.52.0-20261101"

// SeleniumVNCPort is the noVNC port of the Selenium container, published on
// the host's loopback address only.
const SeleniumVNCPort = 7900

// SeleniumWebDriverURL is the WebDriver endpoint of the Selenium container
// as seen from the stack's other containers.
const SeleniumWebDriverURL = "http://" + SeleniumContainerName + ":4444/wd/hub"

// SeleniumExists reports whether a container named SeleniumContainerName
// exists in any state.
func SeleniumExists(ctx context.Context, exe *executor.Executor) bool {
	return ContainerStatus(ctx, exe, SeleniumContainerName) != ""
}

// RemoveSelenium force-removes the Selenium container, running or stopped.
// The caller is responsible for confirming with the user first.
func RemoveSelenium(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	return removeContainer(ctx, exe, stdout, SeleniumContainerName, "selenium container")
}

// seleniumRunArgs builds the "docker run" argument list for the Selenium
// container. It never restarts on its own ("--restart no"), so it is off
// after the host reboots until started again; Chrome needs a /dev/shm far
// larger than Docker's 64 MB default.
func seleniumRunArgs() []string {
	return []string{
		"run", "-d",
		"--name", SeleniumContainerName,
		"--network", NetworkName,
		"--restart", "no",
		"--shm-size", "2g",
		"-e", "SE_NODE_MAX_SESSIONS=2",
		"-e", "SE_NODE_OVERRIDE_MAX_SESSIONS=true",
		"-p", fmt.Sprintf("127.0.0.1:%d:%d", SeleniumVNCPort, SeleniumVNCPort),
		"--", SeleniumImage,
	}
}

// CreateSelenium creates the shared Selenium container and records it in
// config.yaml. Assumes no container named SeleniumContainerName exists
// yet; the caller checks SeleniumExists and calls RemoveSelenium first when
// recreating.
func CreateSelenium(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	if !isDockerInstalled(ctx, exe) {
		return i18n.NewError("docker.notInstalled")
	}
	if err := EnsureNetwork(ctx, exe, stdout); err != nil {
		return err
	}
	ui.Info(stdout, i18n.Tr("docker.run.creating", i18n.Args{"name": SeleniumContainerName}))
	if err := exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, "docker", seleniumRunArgs()...); err != nil {
		return fmt.Errorf("create the selenium container: %w", err)
	}
	if err := config.Update(func(cfg *config.Config) error {
		cfg.Docker.SeleniumCreated = true
		return nil
	}); err != nil {
		return fmt.Errorf("%s: %w", i18n.Tr("docker.createdNotSaved", i18n.Args{"name": SeleniumContainerName}), err)
	}
	return nil
}

// RecreateSelenium removes the existing Selenium container, if any, and
// creates it again, with no confirmation of its own.
func RecreateSelenium(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	if SeleniumExists(ctx, exe) {
		if err := RemoveSelenium(ctx, exe, stdout); err != nil {
			return err
		}
	}
	return CreateSelenium(ctx, exe, stdout)
}

// DeleteSelenium removes the Selenium container (if any) and clears
// cfg.Docker.SeleniumCreated. The caller is responsible for confirming with
// the user first.
func DeleteSelenium(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	if SeleniumExists(ctx, exe) {
		if err := RemoveSelenium(ctx, exe, stdout); err != nil {
			return err
		}
	}
	if err := config.Update(func(cfg *config.Config) error {
		cfg.Docker.SeleniumCreated = false
		return nil
	}); err != nil {
		return fmt.Errorf("%s: %w", i18n.Tr("docker.removedNotSaved", i18n.Args{"name": SeleniumContainerName}), err)
	}
	return nil
}
