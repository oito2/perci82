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

package repo

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// libsecret binary locations across distros.
var libsecretPaths = []string{
	"/usr/share/doc/git/contrib/credential/libsecret/git-credential-libsecret",
	"/usr/lib/git-core/git-credential-libsecret",
	"/usr/libexec/git-core/git-credential-libsecret",
	"/usr/lib/git/git-credential-libsecret",
}

func resolveCredHelper(ctx context.Context, exe *executor.Executor) string {
	for _, p := range libsecretPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if exe.CommandAvailable(ctx, "git-credential-libsecret") {
		return "libsecret"
	}
	return "cache"
}

// setCredentialHelper sets git's credential.helper in the given scope
// ("--global" or "--local", the latter in dir), unless a helper is already
// configured there — one set up by the user (gh, Git Credential Manager)
// is never replaced. With no libsecret helper available it falls back to
// "cache" and says so.
func setCredentialHelper(ctx context.Context, exe *executor.Executor, stdout io.Writer, scope, dir string) error {
	probe := []string{"config", "--get-all", "credential.helper"}
	if scope == "--global" {
		probe = []string{"config", "--global", "--get-all", "credential.helper"}
	}
	if out, err := exe.Output(ctx, executor.Options{}, "git", gitDirArgs(dir, probe...)...); err == nil {
		if current := strings.Join(strings.Fields(out), ", "); current != "" {
			ui.Info(stdout, i18n.Tr("repo.run.helperKept", i18n.Args{"helper": current}))
			return nil
		}
	}

	cred := resolveCredHelper(ctx, exe)
	if cred == "cache" {
		ui.Warning(stdout, i18n.Tr("repo.run.helperCache", nil))
	}
	args := []string{"config", scope, "--", "credential.helper", cred}
	if err := exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, "git", gitDirArgs(dir, args...)...); err != nil {
		return fmt.Errorf("git %v: %w", args, err)
	}
	return nil
}
