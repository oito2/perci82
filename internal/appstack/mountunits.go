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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// noLinkAssert is the directive a mount unit carries for every folder on
// the way to its What= and Where=: systemd refuses to start the mount when
// one of them has become a link since the unit was written.
const noLinkAssert = "AssertPathIsSymbolicLink=!"

// noLinkAsserts returns one noLinkAssert line per folder from the root
// down to each of paths (the root itself excluded), each folder once.
func noLinkAsserts(paths ...string) string {
	seen := map[string]bool{}
	var b strings.Builder
	for _, path := range paths {
		var chain []string
		for p := filepath.Clean(path); p != "/" && p != "."; p = filepath.Dir(p) {
			chain = append(chain, p)
		}
		for i := len(chain) - 1; i >= 0; i-- {
			if !seen[chain[i]] {
				seen[chain[i]] = true
				b.WriteString(noLinkAssert + unitPath(chain[i]) + "\n")
			}
		}
	}
	return b.String()
}

// outdatedUnit is a recorded mount whose installed unit predates the link
// checks, with the plan its unit is rewritten from.
type outdatedUnit struct {
	plan MoodlePluginPlan
	unit string
}

// outdatedUnits returns the recorded mounts whose unit, named after the
// mount's current destination, is a perci unit without the link checks.
func outdatedUnits(cfg *config.Config) []outdatedUnit {
	workspace, err := resolveWorkspace(cfg)
	if err != nil {
		return nil
	}
	var out []outdatedUnit
	for _, m := range cfg.MoodlePlugins {
		if !filepath.IsLocal(m.RelPath) || !filepath.IsAbs(m.DevPath) {
			continue
		}
		html := AppHTMLDir(workspace, m.App)
		if resolved, err := filepath.EvalSymlinks(html); err == nil {
			html = resolved
		}
		dest := filepath.Join(html, m.RelPath)
		unit := mountUnitName(dest)
		data, err := os.ReadFile(filepath.Join(systemdUnitDir, unit))
		if err != nil || !strings.HasPrefix(string(data), mountUnitHeader) || strings.Contains(string(data), noLinkAssert) {
			continue
		}
		out = append(out, outdatedUnit{
			plan: MoodlePluginPlan{App: m.App, Component: m.Component, DevPath: m.DevPath, Dest: dest, RelPath: m.RelPath, HTML: html},
			unit: unit,
		})
	}
	return out
}

// OutdatedMountUnits counts the recorded mounts whose unit predates the
// link checks.
func OutdatedMountUnits(cfg *config.Config) int {
	return len(outdatedUnits(cfg))
}

// UpdateMountUnits rewrites, in one privileged batch (one password
// prompt), every mount unit that predates the link checks. The mounts stay
// as they are: systemd checks the new conditions the next time it starts
// them. A mount whose folders already hold a link is left out, with a
// warning.
func UpdateMountUnits(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	stackMu.Lock()
	defer stackMu.Unlock()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	units := outdatedUnits(cfg)
	if len(units) == 0 {
		ui.Info(stdout, i18n.Tr("devfolder.run.unitsCurrent", nil))
		return nil
	}
	var steps []executor.PrivilegedStep
	for _, u := range units {
		if !noLinksOnTheWay(u.plan.DevPath) || !noLinksOnTheWay(u.plan.Dest) {
			ui.Warning(stdout, i18n.Tr("devfolder.run.unitLinkSkipped", i18n.Args{"unit": u.unit}))
			continue
		}
		content := mountUnitContent(u.plan)
		tmp, err := stageUnit(content)
		if tmp != "" {
			defer func() { _ = os.Remove(tmp) }()
		}
		if err != nil {
			return fmt.Errorf("prepare the mount unit: %w", err)
		}
		sum := sha256.Sum256([]byte(content))
		install, err := executor.PrivilegedInstallSteps(tmp, filepath.Join(systemdUnitDir, u.unit), hex.EncodeToString(sum[:]), "0644")
		if err != nil {
			return err
		}
		install[0].Announce = i18n.Tr("devfolder.run.installUnit", i18n.Args{"unit": u.unit})
		steps = append(steps, install...)
	}
	if len(steps) == 0 {
		return errors.New(i18n.Tr("devfolder.run.noUnitUpdated", nil))
	}
	steps = append(steps, executor.PrivilegedStep{Name: "systemctl", Args: []string{"daemon-reload"}})
	ui.Info(stdout, i18n.Tr("devfolder.run.authUnits", nil))
	if err := exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, steps); err != nil {
		return fmt.Errorf("update the mount units: %w", err)
	}
	ui.Success(stdout, i18n.Tr("devfolder.run.unitsUpdated", nil))
	return nil
}

// noLinksOnTheWay reports whether path exists and no folder on the way to
// it is a link.
func noLinksOnTheWay(path string) bool {
	real, err := filepath.EvalSymlinks(path)
	return err == nil && real == filepath.Clean(path)
}

// stageUnit writes content to a temporary file and returns its path (also
// on error, when the file was created, for the caller to remove).
func stageUnit(content string) (string, error) {
	f, err := os.CreateTemp("", "perci-mount-*.mount")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		return f.Name(), err
	}
	return f.Name(), f.Close()
}
