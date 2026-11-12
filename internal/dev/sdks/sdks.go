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

// Package sdks installs, updates and removes the SDKs managed by
// perci.gnl — Go, Rust and Flutter — one at a time, wrapping the golang,
// rust and flutter packages.
package sdks

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/oito2/perci82/internal/dev/flutter"
	"github.com/oito2/perci82/internal/dev/golang"
	"github.com/oito2/perci82/internal/dev/nettimeout"
	"github.com/oito2/perci82/internal/dev/prereqs"
	"github.com/oito2/perci82/internal/dev/rust"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// SDK describes a language/SDK toolchain managed by perci.gnl.
type SDK struct {
	Name string
	ID   string // "go", "rust" or "flutter"
	// Description is the message key of the SDK's description;
	// DescriptionText returns it in the current language.
	Description string
	// Requires lists the Names of the prereqs.Catalogue items its installer
	// needs.
	Requires []string
	// RemoveWarning is the message key of the text shown in the uninstall
	// confirmation; RemoveWarningText returns it in the current language.
	RemoveWarning string
}

// DescriptionText returns the SDK's description in the current language.
func (s SDK) DescriptionText() string {
	return i18n.Tr(s.Description, i18n.Args{"url": flutter.AndroidTermsURL})
}

// RemoveWarningText returns the SDK's uninstall warning in the current
// language.
func (s SDK) RemoveWarningText() string {
	return i18n.Tr(s.RemoveWarning, nil)
}

// Catalogue lists all SDKs managed by perci.gnl.
var Catalogue = []SDK{
	{Name: "Go SDK", ID: "go",
		Description:   "sdk.go",
		Requires:      []string{"Curl"},
		RemoveWarning: "sdk.go.removeWarning"},
	{Name: "Rust (rustup)", ID: "rust",
		Description:   "sdk.rust",
		Requires:      []string{"Curl", "Build Essential"},
		RemoveWarning: "sdk.rust.removeWarning"},
	{Name: "Flutter + Dart SDK", ID: "flutter",
		Description:   "sdk.flutter",
		Requires:      []string{"Git"},
		RemoveWarning: "sdk.flutter.removeWarning"},
}

// paths resolves the fixed locations used by InstalledMap and the
// install, update and remove operations.
func paths() (home, flutterDir, flutterBin string, err error) {
	home, err = os.UserHomeDir()
	if err != nil {
		return "", "", "", fmt.Errorf("get home folder: %w", err)
	}
	flutterDir = filepath.Join(home, "development", "flutter")
	flutterBin = filepath.Join(flutterDir, "bin", "flutter")
	return home, flutterDir, flutterBin, nil
}

// InstalledMap returns which SDKs are currently installed (by Name).
func InstalledMap(ctx context.Context, exe *executor.Executor) map[string]bool {
	result := make(map[string]bool, len(Catalogue))

	goInstalled, _ := golang.InstalledVersion(ctx, exe)
	result["Go SDK"] = goInstalled
	result["Rust (rustup)"] = rust.IsInstalled()

	if _, _, flutterBin, err := paths(); err == nil {
		result["Flutter + Dart SDK"] = flutter.IsInstalled(flutterBin)
	}
	return result
}

// MissingRequirements returns the labels of s's requirements that aren't
// installed.
func MissingRequirements(ctx context.Context, exe *executor.Executor, s SDK) []string {
	var missing []string
	for _, req := range s.Requires {
		if !prereqs.Installed(ctx, exe, req) {
			missing = append(missing, i18n.Tr("dev.requirement", i18n.Args{"name": req}))
		}
	}
	return missing
}

// InstallOne installs s.
func InstallOne(ctx context.Context, exe *executor.Executor, stdout io.Writer, s SDK) error {
	home, flutterDir, flutterBin, err := paths()
	if err != nil {
		return err
	}
	return install(ctx, exe, stdout, s.ID, home, flutterDir, flutterBin)
}

// UpdateOne updates an installed s: Go to the latest stable release (no-op
// when it already is), Rust through "rustup update", Flutter through
// "flutter upgrade".
func UpdateOne(ctx context.Context, exe *executor.Executor, stdout io.Writer, s SDK) error {
	_, _, flutterBin, err := paths()
	if err != nil {
		return err
	}
	return update(ctx, exe, stdout, s.ID, flutterBin)
}

// UninstallOne removes s.
func UninstallOne(ctx context.Context, exe *executor.Executor, stdout io.Writer, s SDK) error {
	home, flutterDir, _, err := paths()
	if err != nil {
		return err
	}
	return remove(ctx, exe, stdout, s.ID, home, flutterDir)
}

// install installs an SDK that isn't installed yet.
func install(ctx context.Context, exe *executor.Executor, stdout io.Writer, id, home, flutterDir, flutterBin string) error {
	switch id {
	case "go":
		// Needs go.dev reachable for the download and its checksum.
		latest, checksum, err := golang.LatestRelease(ctx)
		if err != nil {
			return fmt.Errorf("get the newest Go version: %w", err)
		}
		if err := golang.Install(ctx, exe, stdout, latest, checksum); err != nil {
			return err
		}
		// Puts Go's bin directory on PATH in the shell rc file.
		golang.EnsurePathInBashrc(stdout)
		return nil
	case "rust":
		return rust.Install(ctx, exe, stdout)
	case "flutter":
		if err := flutter.CloneFlutter(ctx, exe, stdout, home, flutterDir); err != nil {
			return err
		}
		flutter.EnsurePathInBashrc(stdout, home)
		if err := flutter.InstallPrereqs(ctx, exe, stdout); err != nil {
			ui.Warning(stdout, i18n.Tr("sdk.run.flutterPrereqsFailed", i18n.Args{"error": err.Error()}))
		}
		if err := flutter.EnsureAndroidCmdlineTools(ctx, exe, stdout, home, flutterBin); err != nil {
			ui.Warning(stdout, i18n.Tr("sdk.run.cmdlineToolsFailed", i18n.Args{"error": err.Error()}))
		}
		if err := flutter.EnsureChromeWrapper(ctx, exe, stdout, home); err != nil {
			ui.Warning(stdout, i18n.Tr("sdk.run.chromeWrapperFailed", i18n.Args{"error": err.Error()}))
		}
		ui.Info(stdout, i18n.Tr("sdk.run.flutterDoctor", nil))
		_ = exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, flutterBin, "doctor")
		return nil
	}
	return fmt.Errorf("no installer for %s", id)
}

// latestGoRelease and installedGoVersion look up the Go versions update
// compares. Replaced in tests.
var (
	latestGoRelease    = golang.LatestRelease
	installedGoVersion = golang.InstalledVersion
)

// update updates an installed SDK.
func update(ctx context.Context, exe *executor.Executor, stdout io.Writer, id, flutterBin string) error {
	switch id {
	case "go":
		latest, checksum, err := latestGoRelease(ctx)
		if err != nil {
			return fmt.Errorf("get the newest Go version: %w", err)
		}
		if _, current := installedGoVersion(ctx, exe); current == latest {
			ui.Info(stdout, i18n.Tr("sdk.run.goUpToDate", i18n.Args{"version": current}))
			return nil
		}
		return golang.Install(ctx, exe, stdout, latest, checksum)
	case "rust":
		return rust.Update(ctx, exe, stdout)
	case "flutter":
		ui.Info(stdout, i18n.Tr("sdk.run.flutterUpgrade", nil))
		ctx, cancel := nettimeout.Context(ctx)
		defer cancel()
		return exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, flutterBin, "upgrade")
	}
	return fmt.Errorf("no updater for %s", id)
}

func remove(ctx context.Context, exe *executor.Executor, stdout io.Writer, id, home, flutterDir string) error {
	switch id {
	case "go":
		return golang.Uninstall(ctx, exe, stdout)
	case "rust":
		return rust.Uninstall(ctx, exe, stdout)
	case "flutter":
		return flutter.Uninstall(ctx, exe, stdout, home, flutterDir)
	}
	return fmt.Errorf("no uninstaller for %s", id)
}
