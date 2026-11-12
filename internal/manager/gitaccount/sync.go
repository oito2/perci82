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

package gitaccount

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// ErrNotAccountRepo means a folder is not one of the repositories listed
// for an account.
var ErrNotAccountRepo = i18n.NewError("git.notListed")

// SelectRepos returns the repositories listed for acc (ListRepos), or only
// the one at dir when dir is not empty.
func SelectRepos(ctx context.Context, exe *executor.Executor, acc config.GitAccount, dir string) ([]RepoState, error) {
	res, err := ListRepos(ctx, exe, acc)
	if err != nil {
		return nil, err
	}
	if dir == "" {
		return res.Repos, nil
	}
	for _, st := range res.Repos {
		if st.Dir == filepath.Clean(dir) {
			return []RepoState{st}, nil
		}
	}
	return nil, fmt.Errorf("%s: %w", dir, ErrNotAccountRepo)
}

// netTimeout bounds each network command of one repository.
const netTimeout = 90 * time.Second

// fetchConcurrency bounds the repositories fetched at the same time.
const fetchConcurrency = 4

// netEnv makes git never wait for input: no terminal prompt, no askpass
// program, and ssh in batch mode with acc's key and known_hosts file —
// set even when the account's includeIf rule is missing.
func netEnv(acc config.GitAccount) ([]string, error) {
	sshCmd, err := SSHCommand(acc)
	if err != nil {
		return nil, err
	}
	return []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
		"GIT_SSH_COMMAND=" + sshCmd + " -o BatchMode=yes -o ConnectTimeout=15",
	}, nil
}

// gitNet runs a network git command in dir with netEnv and netTimeout,
// returning git's trimmed error output on failure.
func gitNet(ctx context.Context, exe *executor.Executor, env []string, dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, netTimeout)
	defer cancel()
	var out bytes.Buffer
	opts := executor.Options{Env: env, Stdout: &out, Stderr: &out}
	if err := exe.Run(ctx, opts, "git", append([]string{"-C", dir}, args...)...); err != nil {
		if ctx.Err() != nil {
			return i18n.NewError("git.timeout")
		}
		if msg := lastLines(out.String(), 3); msg != "" {
			return errors.New(msg)
		}
		return err
	}
	return nil
}

// lastLines returns the last n non-empty lines of s, joined by " / ".
func lastLines(s string, n int) string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " / ")
}

// Fetch runs `git fetch --prune origin` in every repository of repos that
// has an origin remote, a few at a time, writing one line per repository
// to stdout. It fails when any fetch failed.
func Fetch(ctx context.Context, exe *executor.Executor, stdout io.Writer, acc config.GitAccount, repos []RepoState) error {
	env, err := netEnv(acc)
	if err != nil {
		return err
	}
	w := executor.NewSyncWriter(stdout)
	var failed atomic.Int32
	err = executor.RunConcurrent(ctx, repos, fetchConcurrency, func(st RepoState) {
		if st.RemoteClass == RemoteNone {
			ui.Warning(w, "– "+i18n.Tr("git.run.noRemoteSkipped", i18n.Args{"path": st.Path}))
			return
		}
		if err := gitNet(ctx, exe, env, st.Dir, "fetch", "--prune", "origin"); err != nil {
			failed.Add(1)
			ui.Err(w, "✗ "+st.Path+": "+err.Error())
			return
		}
		ui.Success(w, "✓ "+st.Path)
	})
	if err != nil {
		return err
	}
	if n := failed.Load(); n > 0 {
		return errors.New(i18n.Tr("git.checkFailed", i18n.Args{"count": n}))
	}
	return nil
}

// Pull brings every repository of repos up to date with its upstream,
// fast-forward only: fetches, then merges with --ff-only. A repository
// without remote or upstream, on a detached HEAD, with uncommitted changes
// to tracked files or diverged from its upstream is skipped, with the
// reason on stdout. It fails when any fetch or merge failed; skips are not
// failures.
func Pull(ctx context.Context, exe *executor.Executor, stdout io.Writer, acc config.GitAccount, repos []RepoState) error {
	env, err := netEnv(acc)
	if err != nil {
		return err
	}
	failed := 0
	for i, st := range repos {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ui.Step(stdout, i+1, len(repos), st.Path)
		if reason := pullBlocker(st); reason != "" {
			ui.Warning(stdout, "– "+i18n.Tr("git.run.skipped", i18n.Args{"path": st.Path, "reason": reason}))
			continue
		}
		if err := gitNet(ctx, exe, env, st.Dir, "fetch", "--prune", "origin"); err != nil {
			failed++
			ui.Err(stdout, "✗ "+st.Path+": "+err.Error())
			continue
		}
		now := ReadRepo(ctx, exe, acc, st.Dir)
		switch {
		case now.Error != "":
			failed++
			ui.Err(stdout, "✗ "+st.Path+": "+now.Error)
			continue
		case now.Behind == 0:
			ui.Info(stdout, "= "+i18n.Tr("git.run.upToDate", i18n.Args{"path": st.Path}))
			continue
		case now.Ahead > 0:
			ui.Warning(stdout, "– "+i18n.Tr("git.run.diverged", i18n.Args{"path": st.Path, "ahead": now.Ahead, "behind": now.Behind}))
			continue
		}
		if err := gitNet(ctx, exe, env, st.Dir, "merge", "--ff-only", "@{upstream}"); err != nil {
			failed++
			ui.Err(stdout, "✗ "+st.Path+": "+err.Error())
			continue
		}
		ui.Success(stdout, "✓ "+i18n.Tr("git.run.pulled", i18n.Args{"path": st.Path, "count": now.Behind}))
	}
	if failed > 0 {
		return errors.New(i18n.Tr("git.pullFailed", i18n.Args{"count": failed}))
	}
	return nil
}

// pullBlocker says why Pull skips st, or "" when it doesn't.
func pullBlocker(st RepoState) string {
	switch {
	case st.Error != "":
		return i18n.Tr("git.block.unreadable", nil)
	case st.RemoteClass == RemoteNone:
		return i18n.Tr("git.block.noRemote", nil)
	case st.Detached:
		return i18n.Tr("git.block.detached", nil)
	case st.Upstream == "":
		return i18n.Tr("git.block.noUpstream", i18n.Args{"branch": st.Branch})
	case st.Modified > 0:
		return i18n.Tr("git.block.modified", nil)
	}
	return ""
}
