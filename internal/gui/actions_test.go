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

package gui

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/dev/agentskills"
	"github.com/oito2/perci82/internal/dev/ide"
	"github.com/oito2/perci82/internal/dev/llm"
	"github.com/oito2/perci82/internal/dev/mcpservers"
	"github.com/oito2/perci82/internal/dev/prereqs"
	"github.com/oito2/perci82/internal/dev/sdks"
	"github.com/oito2/perci82/internal/dev/terminal"
	"github.com/oito2/perci82/internal/system/apps"
	"github.com/oito2/perci82/internal/system/templates"
)

// isolateActions gives every action a throwaway HOME and a PATH with
// nothing on it, so whatever the machine running the tests has installed
// reads as missing and no real command can run (the executor is DryRun:
// commands are printed, not executed).
func isolateActions(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	t.Setenv("SHELL", "/bin/bash")
}

// Each button's bound method reaches the domain code it's meant to: run in
// DryRun, the action finishes (action-done ok) and its output carries the
// command that identifies that domain function.
func TestActions_ReachTheirDomainCode(t *testing.T) {
	cases := []struct {
		name string
		call func(b *serviceBase) error
		want []string
	}{
		{"RunTerminals", func(b *serviceBase) error {
			return (&DevSetupService{*b}).RunTerminals([]string{"Alacritty"})
		}, []string{"alacritty"}},
		{"ApplyStarship", func(b *serviceBase) error { return (&DevSetupService{*b}).ApplyStarship() },
			[]string{"starship.rs/install.sh"}},
		{"RemoveStarship", func(b *serviceBase) error { return (&DevSetupService{*b}).RemoveStarship() },
			[]string{"Starship removidas"}},
		// DryRun answers every shell-based "is it installed?" with yes, so
		// an empty selection removes them all.
		{"RunAIApps", func(b *serviceBase) error {
			return (&DevSetupService{*b}).RunAIApps(nil)
		}, []string{"uninstall", "@openai/codex"}},
		{"RunPrereqAction", func(b *serviceBase) error {
			return (&DevSetupService{*b}).RunPrereqAction("Git", "install")
		}, []string{"git"}},
		{"RunIDEAction", func(b *serviceBase) error { return (&DevSetupService{*b}).RunIDEAction("Zed Editor", "update") },
			[]string{"zed.dev"}},
		{"UninstallAndroidStudio", func(b *serviceBase) error { return (&DevSetupService{*b}).UninstallAndroidStudio() },
			[]string{"/opt/android-studio"}},
		{"RunFonts", func(b *serviceBase) error {
			return (&LinuxService{*b}).RunFonts([]string{"Carlito"})
		}, []string{"carlito"}},
		{"RunFlatpakApps", func(b *serviceBase) error {
			return (&LinuxService{*b}).RunFlatpakApps([]string{"org.gnome.Loupe"})
		}, []string{"flatpak", "org.gnome.Loupe"}},
		{"RunSystemUpdate", func(b *serviceBase) error { return (&LinuxService{*b}).RunSystemUpdate(false, false) },
			[]string{"[dry-run] /usr/bin/sudo", "update"}},
		{"ApplyGlobalGitIdentity", func(b *serviceBase) error {
			return (&RepositoriesService{serviceBase: *b}).ApplyGlobalGitIdentity("Dev", "dev@example.com")
		}, []string{"config --global -- user.name Dev"}},
		{"InstallAgentSkill", func(b *serviceBase) error {
			return (&DevToolsService{*b}).InstallAgentSkill("code-review", true, "")
		}, []string{"skills@1.7.0", "add", "code-review", "--global"}},
		{"InstallMCPServer", func(b *serviceBase) error {
			return (&DevToolsService{*b}).InstallMCPServer("dart-flutter", true, "", "")
		}, []string{"claude [plugin install dart-flutter@dart-flutter"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			isolateActions(t)
			b, rec := newTestBase(t)
			if err := c.call(b); err != nil {
				t.Fatalf("refused: %v", err)
			}
			logs, done := rec.wait(t)
			if done["ok"] != true {
				t.Errorf("action failed: %v\n%s", done["error"], logs)
			}
			for _, w := range c.want {
				if !strings.Contains(logs, w) {
					t.Errorf("output missing %q:\n%s", w, logs)
				}
			}
		})
	}
}

// Settings saved straight from the Home screen: a known value round-trips
// through config.yaml, anything else falls back to the default instead of
// being stored as-is, and each application saves its theme under its own
// key, writing only keys it owns.
func TestHomeSettings_RoundTripAndFallback(t *testing.T) {
	isolateActions(t)
	b, _ := newTestBase(t)
	dev := &DevHomeService{homeCore{*b}}
	sys := &SystemHomeService{homeCore{*b}}

	themes := dev.GetGUIThemes()
	if len(themes) < 2 {
		t.Fatal("theme list too short")
	}
	first, last := themes[0], themes[len(themes)-1]

	useApp(t, config.AppDev)
	if err := dev.SetTheme(last); err != nil || dev.GetTheme() != last {
		t.Errorf("Dev theme = %q (err %v), want %q", dev.GetTheme(), err, last)
	}
	useApp(t, config.AppSystem)
	if err := sys.SetTheme(first); err != nil || sys.GetTheme() != first {
		t.Errorf("System theme = %q (err %v), want %q", sys.GetTheme(), err, first)
	}
	if dev.GetTheme() != last {
		t.Errorf("System's SetTheme changed the Dev theme to %q", dev.GetTheme())
	}

	if err := sys.SetTheme("../../etc/passwd"); err != nil {
		t.Fatal(err)
	}
	if sys.GetTheme() == "../../etc/passwd" {
		t.Error("an unknown value was stored as-is")
	}

	if err := sys.SetFlatpakScope("user"); err != nil || sys.GetFlatpakScope() != "user" {
		t.Errorf("scope = %q (err %v), want user", sys.GetFlatpakScope(), err)
	}
	if err := sys.SetFlatpakScope("anything"); err != nil || sys.GetFlatpakScope() != "system" {
		t.Errorf("unknown scope should fall back to system, got %q (err %v)", sys.GetFlatpakScope(), err)
	}

	// Perci82 System refuses to write a key Perci82 Dev owns.
	var foreign *config.ForeignKeyError
	if err := dev.SetTheme(first); !errors.As(err, &foreign) {
		t.Errorf("Dev's SetTheme under Perci82 System: err = %v, want *config.ForeignKeyError", err)
	}
	if got := dev.GetSidebarLogo(); got != config.DefaultSidebarLogo {
		t.Errorf("sidebar logo = %q, want the default %q", got, config.DefaultSidebarLogo)
	}
}

// useApp makes config.Update write only app's keys until the test ends.
func useApp(t *testing.T, app config.App) {
	t.Helper()
	config.SetApp(app)
	t.Cleanup(func() { config.SetApp("") })
}

func TestDockerHelpers(t *testing.T) {
	isolateActions(t)
	b, _ := newTestBase(t)
	s := &DockerService{*b}

	cat := s.GetDockerCreateCatalog()
	if len(cat.PHPVersions) == 0 || len(cat.NodeVersions) == 0 || len(cat.MoodleVersions) == 0 {
		t.Errorf("catalog missing versions: %+v", cat)
	}
	if cat.MariaDBReady {
		t.Error("MariaDB can't be ready without its credentials in config.yaml")
	}
	if p1, p2 := s.GenMariaDBPassword(), s.GenMariaDBPassword(); len(p1) < 12 || p1 == p2 {
		t.Errorf("passwords %q / %q: want long and different", p1, p2)
	}
	if v := s.GetPHPVersionsForMoodleVersion(cat.MoodleVersions[0]); len(v) == 0 {
		t.Errorf("no PHP versions for Moodle %s", cat.MoodleVersions[0])
	}
	if err := s.RemoveDockerContainer("unknown", ""); err == nil {
		t.Error("RemoveDockerContainer: unknown kind must fail")
	}
}

func TestRecreateDockerContainer_UnknownKindFailsTheAction(t *testing.T) {
	isolateActions(t)
	b, rec := newTestBase(t)
	if err := (&DockerService{*b}).RecreateDockerContainer("unknown", ""); err != nil {
		t.Fatal(err)
	}
	if _, done := rec.wait(t); done["ok"] != false {
		t.Errorf("action-done = %v, want a failure", done)
	}
}

// countActive counts the catalogue entries that are not retired.
func countActive[T any](catalogue []T, retired func(T) bool) int {
	n := 0
	for _, e := range catalogue {
		if !retired(e) {
			n++
		}
	}
	return n
}

// A retired catalogue entry is listed only in the scope where it is still
// installed, flagged so the screen offers only its removal.
func TestRetiredEntries_ListedOnlyWhileInstalled(t *testing.T) {
	isolateActions(t)
	b, _ := newTestBase(t)
	devTools := &DevToolsService{*b}

	if rows := rowsOrNil(devTools.GetAgentSkillsRows(true, "")); slices.ContainsFunc(rows, func(r AgentSkillRow) bool { return r.Slug == "php-specialist" }) {
		t.Error("a retired skill that is not installed must not be listed")
	}
	if err := recordAgentSkillInstall("php-specialist", true, "", []string{"claude-code"}, true); err != nil {
		t.Fatal(err)
	}
	if err := recordMCPServerInstall("sqlite", true, "", "/db.sqlite", []string{"claude"}, true); err != nil {
		t.Fatal(err)
	}
	skills := rowsOrNil(devTools.GetAgentSkillsRows(true, ""))
	if i := slices.IndexFunc(skills, func(r AgentSkillRow) bool { return r.Slug == "php-specialist" }); i < 0 || !skills[i].Retired || !skills[i].Installed {
		t.Errorf("installed retired skill row missing or unflagged: %+v", skills)
	}
	servers := rowsOrNil(devTools.GetMCPServerRows(true, ""))
	if i := slices.IndexFunc(servers, func(r MCPServerRow) bool { return r.Slug == "sqlite" }); i < 0 || !servers[i].Retired || !servers[i].Installed {
		t.Errorf("registered retired MCP row missing or unflagged: %+v", servers)
	}
	if local := rowsOrNil(devTools.GetMCPServerRows(false, t.TempDir())); slices.ContainsFunc(local, func(r MCPServerRow) bool { return r.Slug == "sqlite" }) {
		t.Error("a retired MCP registered in another scope must not be listed")
	}
}

// rowsOrNil returns rows, or nil on err so the assertions on them fail.
func rowsOrNil[T any](rows []T, err error) []T {
	if err != nil {
		return nil
	}
	return rows
}

// The screens' read-only getters answer without panicking and list one row
// per catalogue entry. GetSelfUpdateInfo is left out: it asks GitHub.
func TestGetters_ListTheirCatalogues(t *testing.T) {
	isolateActions(t)
	b, _ := newTestBase(t)
	home, linux := &DevHomeService{homeCore{*b}}, &LinuxService{*b}
	devSetup, devTools, docker := &DevSetupService{*b}, &DevToolsService{*b}, &DockerService{*b}

	lists := map[string]struct {
		got  int
		want int
	}{
		"terminals": {len(devSetup.GetTerminalsInfo()), len(terminal.Catalogue)},
		"ai apps":   {len(devSetup.GetAIAppsInfo()), len(llm.Catalogue)},
		"prereqs":   {len(devSetup.GetPrereqsInfo()), len(prereqs.Catalogue)},
		"sdks":      {len(devSetup.GetSDKsInfo()), len(sdks.Catalogue)},
		"ides":      {len(devSetup.GetIDEsInfo()), len(ide.Catalogue)},
		"flatpak":   {len(linux.GetFlatpakAppsInfo()), len(apps.Catalogue)},
		"templates": {len(linux.GetTemplatesInfo()), len(templates.Catalogue)},
		"skills":    {mustLen(devTools.GetAgentSkillsRows(true, "")), countActive(agentskills.Catalogue, func(s agentskills.Skill) bool { return s.Retired })},
		"mcp":       {mustLen(devTools.GetMCPServerRows(true, "")), countActive(mcpservers.Catalogue, func(s mcpservers.Server) bool { return s.Retired })},
	}
	for name, l := range lists {
		if l.got != l.want || l.got == 0 {
			t.Errorf("%s: %d rows, want %d", name, l.got, l.want)
		}
	}
	if len(linux.GetFontsInfo()) == 0 || len(devTools.GetAIContextItems(t.TempDir())) == 0 {
		t.Error("fonts / AI contexts: empty lists")
	}
	if home.GetAppVersion() == "" || len(home.GetCategories()) == 0 || len((&SystemHomeService{}).GetCategories()) == 0 {
		t.Error("empty app version or sidebar categories")
	}
	// Single-app screens and the rest: just answer.
	_ = devSetup.GetAndroidStudioInfo()
	_ = devSetup.GetAntigravityIDEInfo()
	_ = linux.GetPostinstallProfile()
	_ = docker.GetContainerRows()
	_ = home.GetWorkspacePath()
	_ = (&SystemHomeService{}).GetFlatpakScope()
}

func mustLen[T any](rows []T, err error) int {
	if err != nil {
		return -1
	}
	return len(rows)
}
