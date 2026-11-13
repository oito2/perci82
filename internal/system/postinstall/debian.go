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

package postinstall

import (
	"context"
	"io"

	"github.com/oito2/perci82/internal/distro"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// Shared building blocks of the Debian-family profiles (Mint, Ubuntu,
// Zorin, Pop!_OS): every action they have in common is defined once here,
// and each profile file only composes its own list. IDs, labels and
// descriptions are what the GUI's checklist shows and what DependsOn
// refers to.

// aptStep runs one privileged, non-interactive apt-get command.
func aptStep(ctx context.Context, exe *executor.Executor, stdout io.Writer, msg string, args ...string) error {
	ui.Info(stdout, msg)
	return exe.Run(ctx, executor.Options{
		RequiresSudo: true,
		Stdout:       stdout,
		Stderr:       stdout,
		Env:          distro.AptEnv,
	}, "apt-get", distro.AptArgs(args...)...)
}

// enableUbuntuComponents enables universe and multiverse (whichever is
// missing) in one privileged batch — one password prompt, not one per
// component.
func enableUbuntuComponents(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	var steps []executor.PrivilegedStep
	enabled := aptComponents()
	for _, component := range []string{"universe", "multiverse"} {
		if enabled[component] {
			continue
		}
		steps = append(steps, executor.PrivilegedStep{
			Announce: i18n.Tr("postinstall.run.enablingComponent", i18n.Args{"component": component}),
			Name:     "add-apt-repository", Args: []string{"-y", component},
		})
	}
	if len(steps) == 0 {
		ui.Info(stdout, i18n.Tr("pi.run.reposEnabled", nil))
		return nil
	}
	return exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, steps)
}

func actEnableUbuntuComponents() Action {
	return Action{ID: "enable-repos", Label: "pi.enableRepos",
		Description: "pi.enableRepos.desc",
		Applied:     (*Probe).ubuntuComponentsEnabled,
		Run:         enableUbuntuComponents}
}

func actUpdateIndex(dependsOn string) Action {
	return Action{ID: "update-index", Label: "pi.updateIndex",
		Description: "pi.updateIndex.desc",
		DependsOn:   dependsOn,
		Run: func(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
			return aptStep(ctx, exe, stdout, i18n.Tr("pi.run.updateIndex", nil), "update")
		}}
}

func actAptUpgrade(dependsOn string) Action {
	return Action{ID: "upgrade-system", Label: "pi.upgrade",
		Description: "pi.upgrade.desc",
		DependsOn:   dependsOn,
		Run: func(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
			return aptStep(ctx, exe, stdout, i18n.Tr("pi.run.upgrade", nil), "full-upgrade", "-y")
		}}
}

const (
	// backportsBudgiePPA holds the latest stable Budgie components built by
	// the Ubuntu Budgie team for the supported LTS; backportsBudgieURI is
	// the path its apt source points at.
	backportsBudgiePPA = "ppa:ubuntubudgie/backports-budgie"
	backportsBudgieURI = "/ubuntubudgie/backports-budgie/"
)

// actBackportsBudgie adds the backports-budgie PPA, refreshes the package
// index and applies the upgrades, in one privileged batch.
func actBackportsBudgie(dependsOn string) Action {
	return Action{ID: "backports-budgie", Label: "pi.backportsBudgie",
		Description: "pi.backportsBudgie.desc", Args: i18n.Args{"ppa": backportsBudgiePPA},
		DependsOn: dependsOn,
		Applied:   func(*Probe) bool { return aptSourceEnabled(backportsBudgieURI) },
		Run:       applyBackportsBudgie}
}

// applyBackportsBudgie runs add-apt-repository, apt-get update and apt-get
// full-upgrade in one privileged batch (one password prompt). The upgrade
// keeps the current configuration files.
func applyBackportsBudgie(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	steps := []executor.PrivilegedStep{
		{Announce: i18n.Tr("pi.run.addPPA", nil), Name: "add-apt-repository", Args: []string{"-y", "-n", backportsBudgiePPA}},
		{Announce: i18n.Tr("pi.run.updateIndex", nil), Name: "apt-get", Env: distro.AptEnv, Args: distro.AptArgs("update")},
		{Announce: i18n.Tr("pi.run.budgieUpgrade", nil), Name: "apt-get", Env: distro.AptEnv, Args: distro.AptArgs("full-upgrade", "-y")},
	}
	return exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, steps)
}

// actAptInstall installs pkgs; acceptMSFontsEULA pre-accepts the Microsoft
// fonts license first, for lists that include (or pull in) it.
func actAptInstall(id, label, description, dependsOn string, acceptMSFontsEULA bool, pkgs ...string) Action {
	return Action{ID: id, Label: label, Description: description, DependsOn: dependsOn,
		Applied: func(p *Probe) bool { return p.debInstalled(pkgs...) },
		Run: func(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
			if acceptMSFontsEULA {
				return acceptEulaAndAptInstall(ctx, exe, stdout, pkgs...)
			}
			return aptInstall(ctx, exe, stdout, pkgs...)
		}}
}

func actSysctl() Action {
	return Action{ID: "apply-sysctl", Label: "pi.sysctl",
		Description: "pi.sysctl.desc",
		Applied:     (*Probe).sysctlConfigured,
		Run:         configureSysctl}
}

func actSwap() Action {
	return Action{ID: "setup-swap", Label: "pi.swap",
		Description: "pi.swap.desc",
		Applied:     (*Probe).swapActive,
		Run: func(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
			return setupSwapfile(ctx, exe, stdout)
		}}
}

func actTRIM() Action {
	return Action{ID: "enable-trim", Label: "pi.trim",
		Description: "pi.trim.desc",
		Applied:     (*Probe).trimEnabled,
		Run: func(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
			return step(ctx, exe, stdout, i18n.Tr("pi.run.trim", nil), "systemctl", "enable", "--now", "fstrim.timer")
		}}
}

func actAptVAAPIIntel(dependsOn string) Action {
	return Action{ID: "vaapi-intel", Label: "pi.vaapiIntel",
		Description: "pi.vaapiIntel.desc",
		DependsOn:   dependsOn,
		Applied:     func(p *Probe) bool { return p.debInstalled("intel-media-va-driver") },
		Run: func(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
			ui.Info(stdout, i18n.Tr("pi.run.vaapiIntel", nil))
			return aptInstall(ctx, exe, stdout, "intel-media-va-driver")
		}}
}

// actAptVAAPIAMD installs mesa-va-drivers, which is a real package up to
// Ubuntu 24.04 and a virtual one provided by mesa-libgallium on 26.04; apt
// resolves either.
func actAptVAAPIAMD(dependsOn string) Action {
	return Action{ID: "vaapi-amd", Label: "pi.vaapiAMD",
		Description: "pi.vaapiAMD.desc",
		DependsOn:   dependsOn,
		Applied:     func(p *Probe) bool { return p.debInstalled("mesa-va-drivers") },
		Run: func(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
			ui.Info(stdout, i18n.Tr("pi.run.vaapiAMD", nil))
			return aptInstall(ctx, exe, stdout, "mesa-va-drivers")
		}}
}

// flatpakEssentialApps are the Flathub apps actFlatpakEssentials installs.
var flatpakEssentialApps = []string{"org.videolan.VLC", "net.codelogistics.webapps"}

// flatpakWebAppsOnly is the Flathub app list for systems that already ship
// VLC as a distro package.
var flatpakWebAppsOnly = []string{"net.codelogistics.webapps"}

// actFlatpakEssentials ensures Flatpak + Flathub and installs VLC and a
// web-app shortcut manager.
func actFlatpakEssentials(label, description string) Action {
	return actFlatpakApps(label, description, flatpakEssentialApps...)
}

// actFlatpakApps ensures Flatpak + Flathub and installs appIDs.
func actFlatpakApps(label, description string, appIDs ...string) Action {
	return Action{ID: "setup-flatpak", Label: label, Description: description,
		Applied: func(p *Probe) bool { return p.flatpakAppsInstalled(appIDs...) },
		Run: func(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
			return flatpakSetupAndInstall(ctx, exe, stdout, appIDs...)
		}}
}

func actUbuntuDrivers(dependsOn string) Action {
	return Action{ID: "detect-drivers", Label: "pi.drivers",
		Description: "pi.drivers.desc",
		DependsOn:   dependsOn,
		Run: func(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
			return step(ctx, exe, stdout, i18n.Tr("pi.run.drivers", nil), "ubuntu-drivers", "autoinstall")
		}}
}

const flatpakSetupDescription = "pi.flatpakSetup.desc"

// flatpakWebAppsDescription describes actFlatpakApps with flatpakWebAppsOnly.
const flatpakWebAppsDescription = "pi.flatpakWebApps.desc"
