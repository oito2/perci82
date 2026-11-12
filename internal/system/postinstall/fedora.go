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
	"fmt"
	"io"

	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// fedoraPackages is Fedora's own repositories only: 7zip replaces
// p7zip/p7zip-plugins, and unrar (RPM Fusion nonfree) is its own action
// (fedoraInstallUnrar), since dnf refuses a whole transaction when one
// package is missing.
var fedoraPackages = []string{
	"git",
	"curl",
	"wget",
	"htop",
	"make",
	"gcc",
	"gcc-c++",
	"tree",
	"jq",
	"7zip",
	"unzip",
	"net-tools",
	"plocate",
	"python3-pip",
}

// --- Granular actions for the GUI's "Linux :: Pós-instalação" screen ------
// They share the same helpers and the same fedoraPackages list.

func fedoraUpgradeSystem(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	return step(ctx, exe, stdout, i18n.Tr("pi.run.fedoraUpgrade", nil), "dnf", "upgrade", "--refresh", "-y")
}

func fedoraEnableRPMFusion(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	return enableRPMFusion(ctx, exe, stdout)
}

// fedoraInstallCodecs swaps Fedora's patent-free ffmpeg-free for the full
// ffmpeg, then installs the multimedia group — one privileged batch.
func fedoraInstallCodecs(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	return exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, []executor.PrivilegedStep{
		{Announce: i18n.Tr("pi.run.ffmpegSwap", nil), Name: "dnf", Args: []string{"swap", "-y", "ffmpeg-free", "ffmpeg", "--allowerasing"}},
		{Announce: i18n.Tr("pi.run.multimedia", nil), Name: "dnf", Args: []string{"group", "install", "-y", "multimedia"}},
	})
}

func fedoraInstallUnrar(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	ui.Info(stdout, i18n.Tr("pi.run.unrar", nil))
	return dnfInstall(ctx, exe, stdout, "unrar")
}

func fedoraInstallEssentials(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	return dnfInstall(ctx, exe, stdout, fedoraPackages...)
}

func fedoraInstallNTFS(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	if err := dnfInstall(ctx, exe, stdout, "ntfs-3g"); err != nil {
		ui.Warning(stdout, i18n.Tr("pi.run.noNTFS", nil))
	}
	return nil
}

func fedoraApplySysctl(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	return configureSysctl(ctx, exe, stdout)
}

func fedoraEnableTRIM(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	return step(ctx, exe, stdout, i18n.Tr("pi.run.trim", nil), "systemctl", "enable", "--now", "fstrim.timer")
}

// fedoraVAAPIIntel installs intel-media-driver (RPM Fusion nonfree, every
// codec).
func fedoraVAAPIIntel(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	ui.Info(stdout, i18n.Tr("pi.run.vaapiIntel", nil))
	return dnfInstall(ctx, exe, stdout, "intel-media-driver")
}

// fedoraVAAPIAMD installs mesa-va-drivers-freeworld (RPM Fusion free, with
// H.264/HEVC).
func fedoraVAAPIAMD(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	ui.Info(stdout, i18n.Tr("pi.run.vaapiAMD", nil))
	return dnfInstall(ctx, exe, stdout, "mesa-va-drivers-freeworld")
}

// fedoraCleanup runs both dnf commands as ONE sudo/pkexec authentication.
// autoremove is Soft, so its failure is warned but never aborts the batch.
func fedoraCleanup(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	return exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, []executor.PrivilegedStep{
		{Announce: i18n.Tr("pi.run.orphans", nil), Soft: true, WarnMessage: i18n.Tr("pi.run.orphansFailed", nil), Name: "dnf", Args: []string{"autoremove", "-y"}},
		{Announce: i18n.Tr("pi.run.dnfClean", nil), Name: "dnf", Args: []string{"clean", "all"}},
	})
}

var fedoraProfile = Profile{
	ID:    "fedora",
	Label: "Fedora 44",
	Actions: []Action{
		{ID: "upgrade-system", Label: "pi.fedoraUpgrade",
			Description: "pi.fedoraUpgrade.desc",
			Run:         fedoraUpgradeSystem},
		{ID: "enable-rpmfusion", Label: "pi.rpmFusion",
			Description: "pi.rpmFusion.desc",
			Applied:     func(p *Probe) bool { return p.rpmInstalled("rpmfusion-free-release", "rpmfusion-nonfree-release") },
			Run:         fedoraEnableRPMFusion},
		{ID: "install-codecs", Label: "pi.codecs",
			Description: "pi.fedoraCodecs.desc",
			DependsOn:   "enable-rpmfusion", Run: fedoraInstallCodecs,
			Applied: func(p *Probe) bool {
				return p.rpmInstalled("ffmpeg", "gstreamer1-plugin-libav", "gstreamer1-plugins-good")
			}},
		{ID: "install-essentials", Label: "pi.essentials",
			Description: "pi.fedoraEssentials.desc",
			Applied:     func(p *Probe) bool { return p.rpmInstalled(fedoraPackages...) },
			Run:         fedoraInstallEssentials},
		{ID: "install-unrar", Label: "pi.unrar",
			Description: "pi.unrar.desc",
			DependsOn:   "enable-rpmfusion", Run: fedoraInstallUnrar,
			Applied: func(p *Probe) bool { return p.rpmInstalled("unrar") }},
		{ID: "install-ntfs", Label: "pi.ntfs",
			Description: "pi.ntfs.desc",
			Applied:     func(p *Probe) bool { return p.rpmInstalled("ntfs-3g") },
			Run:         fedoraInstallNTFS},
		{ID: "apply-sysctl", Label: "pi.sysctl",
			Description: "pi.sysctl.desc",
			Applied:     (*Probe).sysctlConfigured,
			Run:         fedoraApplySysctl},
		{ID: "enable-trim", Label: "pi.trim",
			Description: "pi.trim.desc",
			Applied:     (*Probe).trimEnabled,
			Run:         fedoraEnableTRIM},
		{ID: "vaapi-intel", Label: "pi.vaapiIntel",
			Description: "pi.fedoraVaapiIntel.desc",
			DependsOn:   "enable-rpmfusion", Run: fedoraVAAPIIntel,
			Applied: func(p *Probe) bool { return p.rpmInstalled("intel-media-driver") }},
		{ID: "vaapi-amd", Label: "pi.vaapiAMD",
			Description: "pi.fedoraVaapiAMD.desc",
			DependsOn:   "enable-rpmfusion", Run: fedoraVAAPIAMD,
			Applied: func(p *Probe) bool { return p.rpmInstalled("mesa-va-drivers-freeworld") }},
		{ID: "cleanup", Label: "pi.cleanup",
			Description: "pi.cleanup.desc",
			Run:         fedoraCleanup},
	},
}

func enableRPMFusion(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	ui.Info(stdout, i18n.Tr("pi.run.rpmFusion", nil))

	args, err := rpmFusionInstallArgs(ctx, exe)
	if err != nil {
		return err
	}
	if args == nil {
		ui.Info(stdout, i18n.Tr("pi.run.rpmFusionEnabled", nil))
		return nil
	}
	return exe.Run(ctx, executor.Options{RequiresSudo: true, Stdout: stdout, Stderr: stdout}, "dnf", args...)
}

// rpmFusionInstallArgs returns the dnf arguments that install only the RPM
// Fusion release packages (free, nonfree) still missing, or nil when both
// are installed.
func rpmFusionInstallArgs(ctx context.Context, exe *executor.Executor) ([]string, error) {
	missing := rpmFusionMissing(ctx, exe)
	if len(missing) == 0 {
		return nil, nil
	}
	ver, err := exe.Output(ctx, executor.Options{}, "rpm", "-E", "%fedora")
	if err != nil {
		return nil, fmt.Errorf("detect the Fedora version: %w", err)
	}
	args := []string{"install", "-y", "--"}
	for _, flavor := range missing {
		args = append(args, fmt.Sprintf(
			"https://download1.rpmfusion.org/%[1]s/fedora/rpmfusion-%[1]s-release-%[2]s.noarch.rpm", flavor, stripNewline(ver)))
	}
	return args, nil
}

// fedoraBudgieProfile is the Fedora Budgie spin: the same actions as
// fedoraProfile, which don't depend on the desktop.
var fedoraBudgieProfile = Profile{ID: "fedora-budgie", Label: "Fedora 44 Budgie", Actions: fedoraProfile.Actions}
