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

// Package androidstudio manages the Android Studio IDE. Install takes a
// local .tar.gz path chosen by the user; there is no update action, and
// reinstalling with a newer tarball replaces the previous install.
//
// Installs into /opt/android-studio owned by the current user: root only
// prepares the empty directory, and the extraction itself runs
// unprivileged (see Install).
package androidstudio

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

const installDir = "/opt/android-studio"

// prepareScript is the privileged part of Install ($1 install dir, $2 backup
// dir, $3 user). A previous install replaces the backup only when it is
// complete (it has bin/studio.sh) or there is no backup yet; a partial
// directory left by a failed extraction is discarded and the backup of the
// last good install is kept.
const prepareScript = `set -e
if [ -e "$1" ]; then
  if [ -x "$1/bin/studio.sh" ] || [ ! -e "$2" ]; then rm -rf -- "$2"; mv -- "$1" "$2"; else rm -rf -- "$1"; fi
fi
mkdir -m 0755 -- "$1"
chown -- "$3:" "$1"`

// desktopFile is the .desktop file Android Studio's own "Create Desktop
// Entry" step writes. It is not created here; Uninstall removes it.
func desktopFile(home string) string {
	return filepath.Join(home, ".local", "share", "applications", "jetbrains-studio.desktop")
}

// Installed reports whether a complete Android Studio is installed: the
// install directory holds bin/studio.sh (a partial extraction doesn't).
func Installed() bool {
	info, err := os.Stat(filepath.Join(installDir, "bin", "studio.sh"))
	return err == nil && info.Mode().IsRegular()
}

// Install extracts tarballPath into /opt/android-studio, owned by the
// current user, then launches the first-run setup wizard (studio.sh)
// detached: it is not awaited, so the step finishes right away. Safe to
// re-run with a newer tarball.
//
// Root never touches the tarball's contents: one privileged batch moves any
// previous install aside to installDir+".old", creates an empty installDir
// and hands it to the user; the extraction then runs as the user. The
// archive's symlinks, setuid bits and owners therefore take effect only
// with the user's own privileges.
func Install(ctx context.Context, exe *executor.Executor, stdout io.Writer, tarballPath string) error {
	if _, err := os.Stat(tarballPath); err != nil {
		return errors.New(i18n.Tr("dev.fileNotFound", i18n.Args{"path": tarballPath}))
	}

	user := executor.CurrentUser()
	if user == "" {
		return errors.New("could not determine the current user")
	}

	// Entries must all live under android-studio/, which
	// --strip-components=1 below strips; any other tarball is rejected.
	if err := validateTarballEntries(tarballPath); err != nil {
		return err
	}

	oldDir := installDir + ".old"
	steps := []executor.PrivilegedStep{{
		Announce: i18n.Tr("studio.run.prepare", i18n.Args{"dir": installDir, "user": user}),
		Name:     "sh",
		Args:     []string{"-c", prepareScript, "sh", installDir, oldDir, user},
	}}
	if err := exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, steps); err != nil {
		return fmt.Errorf("prepare %s: %w", installDir, err)
	}

	ui.Info(stdout, i18n.Tr("studio.run.extract", i18n.Args{"file": tarballPath, "dir": installDir}))
	if err := exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout},
		"tar", "-xzf", tarballPath, "-C", installDir, "--strip-components=1", "--no-same-owner",
	); err != nil {
		if _, statErr := os.Stat(oldDir); statErr == nil {
			ui.Warning(stdout, i18n.Tr("studio.run.oldKept", i18n.Args{"dir": oldDir}))
		}
		return fmt.Errorf("extract: %w", err)
	}

	// The previous install is owned by the user, so removing it needs no
	// second password prompt. Skipped in DryRun.
	if exe.DryRun {
		ui.Info(stdout, "[dry-run] remove "+oldDir)
	} else if err := os.RemoveAll(oldDir); err != nil {
		ui.Warning(stdout, i18n.Tr("studio.run.oldRemoveFailed", i18n.Args{"dir": oldDir, "error": err.Error()}))
	}

	studioSh := filepath.Join(installDir, "bin", "studio.sh")
	ui.Info(stdout, i18n.Tr("studio.run.wizard", nil))
	// setsid + & detaches studio.sh from perci's own process (its own
	// session, stdin/stdout/stderr closed) so exe.Run returns as soon as
	// the process launches.
	launchScript := "setsid " + executor.ShellQuote(studioSh) + " >/dev/null 2>&1 < /dev/null &"
	if err := exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, "bash", "-c", launchScript); err != nil {
		ui.Warning(stdout, i18n.Tr("studio.run.wizardFailed", i18n.Args{"error": err.Error()}))
		ui.Warning(stdout, i18n.Tr("studio.run.openByHand", i18n.Args{"path": studioSh}))
	}

	ui.Success(stdout, i18n.Tr("studio.run.installed", i18n.Args{"dir": installDir}))
	return nil
}

// validateTarballEntries lists tarballPath's contents and rejects it unless
// every entry is a relative path under android-studio/ with no ".."
// segment. The archive is read in-process (archive/tar), read-only, so it
// also runs under DryRun.
func validateTarballEntries(tarballPath string) error {
	f, err := os.Open(tarballPath)
	if err != nil {
		return fmt.Errorf("open tarball: %w", err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.Tr("dev.notTarGz", nil), err)
	}
	defer func() { _ = gz.Close() }()
	var names []string
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("list the tarball: %w", err)
		}
		names = append(names, hdr.Name)
	}
	return checkTarballEntries(strings.Join(names, "\n"))
}

// checkTarballEntries is validateTarballEntries' pure part (`tar -t` output in).
func checkTarballEntries(listing string) error {
	found := false
	for _, entry := range strings.Split(listing, "\n") {
		if entry == "" {
			continue
		}
		found = true
		if strings.HasPrefix(entry, "/") || !strings.HasPrefix(entry, "android-studio/") || slices.Contains(strings.Split(entry, "/"), "..") {
			return errors.New(i18n.Tr("studio.notStudio", i18n.Args{"entry": entry}))
		}
	}
	if !found {
		return i18n.NewError("dev.emptyTarball")
	}
	return nil
}

// Uninstall removes the installation directory and the desktop entry (when
// present). User config/cache (~/.config/Google/AndroidStudio*, etc.) and
// the Android SDK/emulators (~/Android/Sdk, ~/.android) are left in
// place.
func Uninstall(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	sudo := executor.Options{RequiresSudo: true, Stdout: stdout, Stderr: stdout}

	ui.Info(stdout, i18n.Tr("dev.removingDir", i18n.Args{"dir": installDir}))
	if err := exe.Run(ctx, sudo, "rm", "-rf", "--", installDir, installDir+".old"); err != nil {
		return fmt.Errorf("remove %s: %w", installDir, err)
	}

	if home, err := os.UserHomeDir(); err == nil {
		_ = os.Remove(desktopFile(home))
	}

	ui.Success(stdout, i18n.Tr("studio.run.uninstalled", nil))
	return nil
}
