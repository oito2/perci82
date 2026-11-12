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
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/oito2/perci82/internal/dev/nettimeout"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// Clone clones a repository and, when name/email are given, applies them
// as its local identity (optional: both empty keeps the global one). A
// non-empty dir must be empty (or not exist yet): git clone refuses anything
// else, and checking first gives a clear message instead of git's own
// error. Everything that can be refused is checked before cloning, so a
// failure never leaves a clone behind with the action reported as failed.
func Clone(ctx context.Context, exe *executor.Executor, stdout io.Writer, url, dir, name, email string) error {
	if err := checkIdentityPair(name, email); err != nil {
		return err
	}
	if dir != "" {
		empty, err := IsEmptyDir(dir)
		if err != nil {
			return fmt.Errorf("check folder: %w", err)
		}
		if !empty {
			return errors.New(i18n.Tr("repo.notEmpty", i18n.Args{"path": dir}))
		}
	}

	opts := executor.Options{Stdout: stdout, Stderr: stdout}
	args := []string{"clone", "--", url}
	if dir != "" {
		args = append(args, dir)
	}

	ui.Info(stdout, i18n.Tr("repo.run.cloning", nil))
	cloneCtx, cancel := nettimeout.Context(ctx)
	defer cancel()
	if err := exe.Run(cloneCtx, opts, "git", args...); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	if name == "" && email == "" {
		return nil
	}
	target := dir
	if target == "" {
		target = cloneTargetName(url)
	}
	ui.Info(stdout, i18n.Tr("repo.run.identity", i18n.Args{"path": target}))
	return ApplyLocalIdentityAt(ctx, exe, stdout, name, email, target)
}

// cloneTargetName is the directory `git clone <url>` creates on its own:
// the last path segment without ".git", also for scp-style URLs
// (git@host:org/repo.git).
func cloneTargetName(url string) string {
	u := strings.TrimRight(url, "/")
	if i := strings.LastIndexAny(u, "/:"); i >= 0 {
		u = u[i+1:]
	}
	return strings.TrimSuffix(u, ".git")
}

// checkIdentityPair accepts both name and email, or neither — just one of
// them can't be applied (ApplyLocalIdentityAt needs both).
func checkIdentityPair(name, email string) error {
	if (name == "") != (email == "") {
		return i18n.NewError("repo.identityPair")
	}
	return nil
}

// IsEmptyDir reports whether dir has no entries at all (hidden ones
// included). A dir that doesn't exist yet counts as empty — git clone
// creates it.
func IsEmptyDir(dir string) (bool, error) {
	f, err := os.Open(dir)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	_, err = f.Readdirnames(1)
	if errors.Is(err, io.EOF) {
		return true, nil
	}
	return false, err
}
