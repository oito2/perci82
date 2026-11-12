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

package update

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/oito2/perci82/internal/distro"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// Options are the optional cleanups the "Atualizar Sistema" screen offers,
// both off by default: a plain update never deletes logs or removes
// packages on its own.
type Options struct {
	// CleanJournal vacuums journal entries older than 7 days.
	CleanJournal bool
	// Autoremove removes orphaned packages (apt/dnf autoremove) and unused
	// Flatpak runtimes.
	Autoremove bool
}

// Run updates the system: packages, snap and flatpak (when available),
// plus the cleanups opts enables.
//
// Progress reports through ui.Step, one call per top-level phase:
// package/snap/system-Flatpak/journal (all privileged) count as a single
// phase, and the user's Flatpak installation (never privileged) is its
// own. A phase is counted only when present.
//
// Every privileged command (package manager + snap + system Flatpak +
// journalctl) runs as ONE authentication via executor.RunSudoSequence.
// Each PrivilegedStep's Announce field is echoed from inside the single
// privileged shell script, as plain text.
func Run(ctx context.Context, exe *executor.Executor, stdout io.Writer, opts Options) error {
	ui.PrintHeader(stdout, i18n.Tr("update.header", nil), distro.DisplayName())

	family := distro.Detect()
	if family == distro.Unknown {
		ui.Err(stdout, i18n.Tr("update.unsupported", nil))
		return errors.New("no supported package manager found (apt/dnf)")
	}
	ui.Info(stdout, i18n.Tr("update.detected", i18n.Args{"family": family}))

	steps, err := packageSteps(family, opts.Autoremove)
	if err != nil {
		ui.Err(stdout, err.Error())
		return err
	}

	hasSnap := exe.CommandAvailable(ctx, "snap")
	hasFlatpak := exe.CommandAvailable(ctx, "flatpak")
	hasJournal := exe.CommandAvailable(ctx, "journalctl")

	if hasSnap {
		steps = append(steps, executor.PrivilegedStep{
			Announce: i18n.Tr("update.snap", nil),
			Soft:     true, SoftReport: true, WarnMessage: i18n.Tr("update.snapFailed", nil),
			Name: "snap", Args: []string{"refresh"},
		})
	} else {
		ui.Info(stdout, i18n.Tr("update.noSnap", nil))
	}
	if hasFlatpak {
		steps = append(steps, systemFlatpakSteps(opts.Autoremove)...)
	}
	if opts.CleanJournal && hasJournal {
		steps = append(steps, executor.PrivilegedStep{
			Announce: i18n.Tr("update.journal", nil),
			Soft:     true, SoftReport: true, WarnMessage: i18n.Tr("update.journalFailed", nil),
			Name: "journalctl", Args: []string{"--vacuum-time=7d"},
		})
	}

	total := 1 // the privileged batch above (packages + snap + system Flatpaks + journal, whichever apply) always runs
	if hasFlatpak {
		total++
	}
	stepIdx := 0
	nextStep := func(label string) {
		stepIdx++
		ui.Step(stdout, stepIdx, total, label)
	}

	nextStep(i18n.Tr("update.packagesStep", nil))
	ui.Info(stdout, i18n.Tr("update.auth", nil))
	warned := false
	if err := exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, steps); err != nil {
		if !errors.Is(err, executor.ErrCompletedWithWarnings) {
			ui.Err(stdout, i18n.Tr("update.packagesFailed", i18n.Args{"error": err}))
			return err
		}
		warned = true
	}

	if hasFlatpak {
		nextStep(i18n.Tr("update.userFlatpakStep", nil))
		if err := updateUserFlatpak(ctx, exe, stdout, opts.Autoremove); err != nil {
			ui.Warning(stdout, i18n.Tr("update.userFlatpakFailed", i18n.Args{"error": err}))
			warned = true
		}
	} else {
		ui.Info(stdout, i18n.Tr("update.noFlatpak", nil))
	}

	return finish(stdout, warned)
}

// finish reports the end of the update: success, or — when a Soft step of
// the batch or the user's Flatpak update failed — a completion with warnings.
func finish(stdout io.Writer, warned bool) error {
	if warned {
		ui.Warning(stdout, i18n.Tr("update.warnings", nil))
		return executor.ErrCompletedWithWarnings
	}
	ui.Success(stdout, i18n.Tr("update.done", nil))
	return nil
}

// packageSteps returns family's package-manager commands as
// executor.RunSudoSequence steps. Announce carries each step's narration;
// full-upgrade alone covers the upgrade.
func packageSteps(family string, autoremove bool) ([]executor.PrivilegedStep, error) {
	switch family {
	case distro.Debian:
		aptStep := func(announce string, args ...string) executor.PrivilegedStep {
			return executor.PrivilegedStep{Announce: announce, Env: distro.AptEnv, Name: "apt-get", Args: distro.AptArgs(args...)}
		}
		steps := []executor.PrivilegedStep{
			aptStep(i18n.Tr("update.aptUpdate", nil), "update"),
			aptStep(i18n.Tr("update.aptUpgrade", nil), "full-upgrade", "-y"),
		}
		if autoremove {
			steps = append(steps, aptStep(i18n.Tr("update.autoremove", nil), "autoremove", "-y"))
		}
		return append(steps, aptStep(i18n.Tr("update.aptClean", nil), "autoclean", "-y")), nil
	case distro.Fedora:
		steps := []executor.PrivilegedStep{
			{Announce: i18n.Tr("update.dnfUpgrade", nil), Name: "dnf", Args: []string{"upgrade", "--refresh", "-y"}},
		}
		if autoremove {
			steps = append(steps, executor.PrivilegedStep{Announce: i18n.Tr("update.autoremove", nil), Soft: true, SoftReport: true, WarnMessage: i18n.Tr("update.dnfAutoremoveFailed", nil), Name: "dnf", Args: []string{"autoremove", "-y"}})
		}
		return steps, nil
	default:
		return nil, fmt.Errorf("unsupported distribution family: %s", family)
	}
}

// flatpakEnv keeps Flatpak's progress output plain text.
var flatpakEnv = []string{"TERM=dumb"}

// systemFlatpakSteps updates the system Flatpak installation and, with
// autoremove, removes its unused runtimes. The steps run as root inside the
// privileged batch, so Flatpak never asks for authorization on its own.
// Both are Soft: a Flatpak failure does not abort the system update, but is
// reported as a completion with warnings.
func systemFlatpakSteps(autoremove bool) []executor.PrivilegedStep {
	steps := []executor.PrivilegedStep{{
		Announce: i18n.Tr("update.systemFlatpak", nil),
		Soft:     true, SoftReport: true, WarnMessage: i18n.Tr("update.systemFlatpakFailed", nil),
		Env: flatpakEnv, Name: "flatpak", Args: []string{"update", "--system", "--noninteractive", "-y"},
	}}
	if autoremove {
		steps = append(steps, executor.PrivilegedStep{
			Announce: i18n.Tr("update.systemFlatpakUnused", nil),
			Soft:     true, SoftReport: true, WarnMessage: i18n.Tr("update.systemFlatpakUnusedFailed", nil),
			Env: flatpakEnv, Name: "flatpak", Args: []string{"uninstall", "--system", "--unused", "--noninteractive", "-y"},
		})
	}
	return steps
}

// updateUserFlatpak updates the user's Flatpak installation and, with
// autoremove, removes its unused runtimes, running as the current user.
func updateUserFlatpak(ctx context.Context, exe *executor.Executor, stdout io.Writer, autoremove bool) error {
	opts := executor.Options{Stdout: stdout, Stderr: stdout, Env: flatpakEnv}
	if err := exe.Run(ctx, opts, "flatpak", "update", "--user", "--noninteractive", "-y"); err != nil {
		return fmt.Errorf("flatpak update: %w", err)
	}
	if !autoremove {
		return nil
	}
	if err := exe.Run(ctx, opts, "flatpak", "uninstall", "--user", "--unused", "--noninteractive", "-y"); err != nil {
		return fmt.Errorf("flatpak uninstall --unused: %w", err)
	}
	return nil
}
