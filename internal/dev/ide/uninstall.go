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

package ide

import (
	"context"
	"fmt"
	"io"

	"github.com/oito2/perci82/internal/dev/prereqs"
	"github.com/oito2/perci82/internal/distro"
	"github.com/oito2/perci82/internal/executor"
)

// UninstallOne uninstalls a single IDE.
func UninstallOne(ctx context.Context, exe *executor.Executor, stdout io.Writer, e IDE, family string) error {
	switch e.Cmd {
	case "zed":
		script := `rm -rf "$HOME/.local/zed.app" "$HOME/.local/bin/zed" "$HOME/.local/share/applications/dev.zed.Zed.desktop" 2>/dev/null; true`
		return exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, "bash", "-c", script)
	case "code":
		return removeWithRepo(ctx, exe, stdout, family, vscodeRepo)
	case "codium":
		return removeWithRepo(ctx, exe, stdout, family, vscodiumRepo)
	}
	return fmt.Errorf("no uninstaller for %s", e.Name)
}

// removeWithRepo removes repo's package and then the repository files
// installing it added (the apt list and keyring, or the dnf .repo file),
// in one privileged batch (one password prompt).
func removeWithRepo(ctx context.Context, exe *executor.Executor, stdout io.Writer, family string, repo distro.SignedRepo) error {
	if err := prereqs.CheckRemoval(ctx, exe, family, repo.PkgName); err != nil {
		return err
	}
	var steps []executor.PrivilegedStep
	switch family {
	case distro.Debian:
		steps = []executor.PrivilegedStep{
			{Name: "apt-get", Env: distro.AptEnv, Args: distro.AptArgs("purge", "-y", "--", repo.PkgName)},
			{Name: "rm", Args: []string{"-f", "--", repo.AptListPath, repo.KeyringPath}},
		}
	case distro.Fedora:
		steps = []executor.PrivilegedStep{
			{Name: "dnf", Args: []string{"remove", "-y", "--no-autoremove", "--", repo.PkgName}},
			{Name: "rm", Args: []string{"-f", "--", repo.DnfRepoPath}},
		}
	default:
		return fmt.Errorf("unsupported distro family for package removal: %s", family)
	}
	return exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, steps)
}
