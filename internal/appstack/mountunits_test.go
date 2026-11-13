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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
)

// The mount unit refuses a link on every folder on the way to the
// development folder and to the destination, each folder once.
func TestMountUnitContent_RefusesLinks(t *testing.T) {
	got := mountUnitContent(MoodlePluginPlan{App: "mdle", Component: "local_x", DevPath: "/home/u/dev/x", Dest: "/home/u/ws/localhost/html/mdle/local/x"})
	for _, p := range []string{"/home", "/home/u", "/home/u/dev", "/home/u/dev/x", "/home/u/ws", "/home/u/ws/localhost/html/mdle", "/home/u/ws/localhost/html/mdle/local/x"} {
		if n := strings.Count(got, "\n"+noLinkAssert+p+"\n"); n != 1 {
			t.Errorf("%s is checked %d times, want 1:\n%s", p, n, got)
		}
	}
	unit, _, _ := strings.Cut(got, "[Mount]")
	if !strings.Contains(unit, noLinkAssert) {
		t.Errorf("the checks must be in [Unit]:\n%s", got)
	}
}

// A destination reached through a link inside the app is refused before
// anything is installed.
func TestAddMoodlePlugin_RefusesLinkOnTheWay(t *testing.T) {
	workspace, dev := pluginStack(t, map[string]string{"local": "public/local"})
	html := AppHTMLDir(workspace, "mdle")
	if err := os.Rename(filepath.Join(html, "public", "local"), filepath.Join(html, "public", "real-local")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real-local", filepath.Join(html, "public", "local")); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	exe := &executor.Executor{DryRun: true, UsePolicyKit: true, Stdout: &buf, Stderr: &buf}
	if err := AddMoodlePlugin(context.Background(), exe, &buf, dev, "mdle", "local", "example"); err == nil {
		t.Fatal("a link on the way to the destination should be refused")
	}
	if strings.Contains(buf.String(), "pkexec") {
		t.Errorf("nothing privileged should run:\n%s", buf.String())
	}
	if n := len(loadCfg(t).MoodlePlugins); n != 0 {
		t.Errorf("%d mounts recorded, want 0", n)
	}
}

// A unit written before the link checks is found and rewritten in one
// privileged batch; a current unit is left alone.
func TestUpdateMountUnits_RewritesOldUnits(t *testing.T) {
	workspace, dev := pluginStack(t, map[string]string{"local": "public/local"})
	unitDir := t.TempDir()
	old := systemdUnitDir
	systemdUnitDir = unitDir
	t.Cleanup(func() { systemdUnitDir = old })

	html, err := filepath.EvalSymlinks(AppHTMLDir(workspace, "mdle"))
	if err != nil {
		t.Fatal(err)
	}
	rec := []config.MoodlePluginMount{
		{App: "mdle", Component: "local_a", DevPath: dev, RelPath: "public/local/a"},
		{App: "mdle", Component: "local_b", DevPath: dev, RelPath: "public/local/b"},
	}
	for _, m := range rec {
		if err := os.MkdirAll(filepath.Join(html, m.RelPath), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.Update(func(c *config.Config) error { c.MoodlePlugins = rec; return nil }); err != nil {
		t.Fatal(err)
	}
	planOf := func(m config.MoodlePluginMount) MoodlePluginPlan {
		return MoodlePluginPlan{App: m.App, Component: m.Component, DevPath: m.DevPath, Dest: filepath.Join(html, m.RelPath), RelPath: m.RelPath, HTML: html}
	}
	current := mountUnitContent(planOf(rec[1]))
	stale := strings.Join(withoutLinkAsserts(strings.Split(mountUnitContent(planOf(rec[0])), "\n")), "\n")
	writeFile(t, filepath.Join(unitDir, mountUnitName(filepath.Join(html, rec[0].RelPath))), stale, 0o644)
	writeFile(t, filepath.Join(unitDir, mountUnitName(filepath.Join(html, rec[1].RelPath))), current, 0o644)

	cfg := loadCfg(t)
	if n := OutdatedMountUnits(cfg); n != 1 {
		t.Fatalf("OutdatedMountUnits = %d, want 1", n)
	}
	var buf bytes.Buffer
	exe := &executor.Executor{DryRun: true, UsePolicyKit: true, Stdout: &buf, Stderr: &buf}
	if err := UpdateMountUnits(context.Background(), exe, &buf); err != nil {
		t.Fatalf("UpdateMountUnits: %v", err)
	}
	out := buf.String()
	if n := strings.Count(out, "pkexec"); n != 1 {
		t.Errorf("%d escalations, want 1\n%s", n, out)
	}
	staleUnit := mountUnitName(filepath.Join(html, rec[0].RelPath))
	currentUnit := mountUnitName(filepath.Join(html, rec[1].RelPath))
	if !strings.Contains(out, filepath.Join(unitDir, staleUnit)) || strings.Contains(out, filepath.Join(unitDir, currentUnit)) {
		t.Errorf("only the old unit should be rewritten:\n%s", out)
	}
	if !strings.Contains(out, "'daemon-reload'") || strings.Contains(out, "enable") {
		t.Errorf("the units are reloaded, not restarted:\n%s", out)
	}
}

// withoutLinkAsserts drops the link checks from a unit's lines.
func withoutLinkAsserts(lines []string) []string {
	var out []string
	for _, l := range lines {
		if !strings.HasPrefix(l, noLinkAssert) {
			out = append(out, l)
		}
	}
	return out
}
