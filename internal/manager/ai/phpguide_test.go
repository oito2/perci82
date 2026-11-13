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

package ai

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oito2/perci82/internal/appstack"
	"github.com/oito2/perci82/internal/config"
)

// phpGuideSetup gives the test its own HOME with a workspace holding the
// Moodle app "mdle" (database on, a development folder mounted) and
// returns the app's project root and the mounted development folder.
func phpGuideSetup(t *testing.T) (project, dev string) {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	ws := filepath.Join(home, "ws")
	project = appstack.AppHTMLDir(ws, "mdle")
	dev = filepath.Join(home, "dev", "local_x")
	for _, d := range []string{project, dev} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.Update(func(c *config.Config) error {
		c.WorkspacePath = ws
		c.Docker.MariaDB = config.MariaDBConfig{DBUser: "dev_user", DBPass: "s3cret-pass"}
		c.Docker.Apps = []config.AppContainer{{Name: "Moodle", Folder: "mdle", Type: config.AppTypeMoodle,
			URL: "mdle.localhost", PHPVersion: "8.3", MoodleVersion: "5.1+", DBAccess: true,
			DisabledTools: []string{appstack.ToolComposer}}}
		c.MoodlePlugins = []config.MoodlePluginMount{{App: "mdle", Component: "local_x", DevPath: dev, RelPath: "public/local/x"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	isAppCommand = func(name string) bool { return name == "mdle" || name == "velho" }
	t.Cleanup(func() { isAppCommand = defaultIsAppCommand })
	return project, dev
}

// defaultIsAppCommand is isAppCommand as the package defines it.
var defaultIsAppCommand = isAppCommand

func TestPHPGuide_WrittenFromTheApp(t *testing.T) {
	project, dev := phpGuideSetup(t)
	var out bytes.Buffer
	if err := GenerateSharedFiles(project, ModelsNamed([]string{"Moodle", phpPerciName}), AgentIDs(), true, &out); err != nil {
		t.Fatal(err)
	}
	guide := readFile(t, filepath.Join(project, ".instructions", "PHP-PERCI82.md"))
	for _, want := range []string{
		"| App | `mdle` (Moodle) |", "| URL | https://mdle.localhost |", "| PHP | 8.3 |", "| Moodle | 5.1+ |",
		"`mdle phpcs --standard=moodle <path>`", "Turned off in this app: composer.",
		"public/admin/cli/upgrade.php", "user `dev_user`", "`" + dev + "` → `/var/www/html/public/local/x` (local_x)",
	} {
		if !strings.Contains(guide, want) {
			t.Errorf("the guide lacks %q", want)
		}
	}
	if strings.Contains(guide, "s3cret-pass") {
		t.Error("the guide holds the database password")
	}
	if !signedUnedited(guide) || GuideStatus(project, Model{Name: phpPerciName, Instruction: "templates/instructions/PHP-PERCI82.md"}) != StatusApplied {
		t.Error("the written guide is not applied")
	}

	allow := jsonOf(t, filepath.Join(project, ".claude", "settings.json"))["permissions"].(map[string]any)["allow"].([]any)
	if !slices.Contains(allow, any("Bash(mdle *)")) {
		t.Errorf("Claude Code allow = %v", allow)
	}
	tools := jsonOf(t, filepath.Join(project, ".gemini", "settings.json"))["tools"].(map[string]any)["allowed"].([]any)
	if !slices.Equal(tools, []any{"run_shell_command(mdle)"}) {
		t.Errorf("Gemini tools = %v", tools)
	}
	if !strings.Contains(readFile(t, filepath.Join(project, ".codex", "rules", "perci.rules")), `prefix_rule(pattern=["mdle"], decision="allow")`) {
		t.Error("the Codex rule is missing")
	}
	perms := jsonOf(t, filepath.Join(project, "opencode.json"))["permissions"].([]any)
	if len(perms) != 1 || perms[0].(map[string]any)["resource"] != "mdle *" {
		t.Errorf("OpenCode permissions = %v", perms)
	}

	// The app changes: the guide is outdated, not edited.
	rec := config.AIContext{Name: "M", Path: project, Models: []string{"Moodle", phpPerciName}, Agents: AgentIDs()}
	if st, _ := ContextState(rec); st.State != StatusApplied {
		t.Fatalf("state = %q, want applied", st.State)
	}
	if err := config.Update(func(c *config.Config) error { c.Docker.Apps[0].PHPVersion = "8.4"; return nil }); err != nil {
		t.Fatal(err)
	}
	if st, _ := ContextState(rec); st.State != StatusOutdated {
		t.Errorf("state after the app changed = %q, want outdated", st.State)
	}
	write(t, filepath.Join(project, ".instructions", "PHP-PERCI82.md"), guide+"\nmine\n")
	if st, _ := ContextState(rec); st.State != StatusEdited {
		t.Errorf("state after an edit = %q, want edited", st.State)
	}

	// Without the standard the grants go, the user's own stay.
	if err := GenerateSharedFiles(project, ModelsNamed([]string{"Moodle"}), AgentIDs(), true, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, ".codex", "rules", "perci.rules")); !os.IsNotExist(err) {
		t.Error("the Codex rule survived")
	}
	if _, err := os.Stat(filepath.Join(project, ".gemini", "settings.json")); !os.IsNotExist(err) {
		t.Error("the Gemini settings survived")
	}
}

func TestPHPGuide_MountedDevelopmentFolder(t *testing.T) {
	_, dev := phpGuideSetup(t)
	p, ok := phpAppFor(dev)
	if !ok || p.app.Folder != "mdle" || p.mount == nil {
		t.Fatalf("the development folder is not matched to its app: %+v %v", p, ok)
	}
	if !strings.Contains(p.vars()["ROOTS"], "this project is the development folder `"+dev+"`, mounted at `/var/www/html/public/local/x`") {
		t.Errorf("ROOTS = %q", p.vars()["ROOTS"])
	}
}

func TestPHPGuide_NoAppNoGuide(t *testing.T) {
	phpGuideSetup(t)
	other := filepath.Join(os.Getenv("HOME"), "elsewhere")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	changes, err := Preview(other, ModelsNamed([]string{phpPerciName}), []string{AgentClaude})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(changes, Change{Path: ".instructions/PHP-PERCI82.md", Action: "skip"}) {
		t.Errorf("the preview doesn't say the guide is skipped: %+v", changes)
	}
	var out bytes.Buffer
	if err := GenerateSharedFiles(other, ModelsNamed([]string{phpPerciName}), []string{AgentClaude}, true, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(other, ".instructions", "PHP-PERCI82.md")); !os.IsNotExist(err) {
		t.Error("a guide was written without an app")
	}
	if !strings.Contains(out.String(), "nenhum app do Docker usa esta pasta") || strings.Contains(readFile(t, filepath.Join(other, "AGENTS.md")), "PHP-PERCI82") {
		t.Errorf("no warning, or AGENTS.md lists the guide:\n%s", out.String())
	}
	rec := config.AIContext{Name: "E", Path: other, Models: []string{phpPerciName}, Agents: []string{AgentClaude}}
	if st, _ := ContextState(rec); st.State != StatusApplied {
		t.Errorf("a context whose guide has no app = %q, want applied", st.State)
	}
}

// The PHP Perci82 rule of Claude Code never needs its Docker app to be
// recognised as Perci's: previews (which run on a copy no app holds) and
// updates or removal after the app is gone still work, and the rule goes.
func TestPHPGuide_RuleRemovedWithoutTheApp(t *testing.T) {
	project, _ := phpGuideSetup(t)
	var out bytes.Buffer
	if err := GenerateSharedFiles(project, ModelsNamed([]string{"Moodle", phpPerciName}), AgentIDs(), true, &out); err != nil {
		t.Fatal(err)
	}
	rule := filepath.Join(project, ".claude", "rules", "perci", "PHP-PERCI82.md")
	if _, err := os.Stat(rule); err != nil {
		t.Fatalf("the Claude Code rule is missing: %v", err)
	}
	if _, err := PreviewRemove(project); err != nil {
		t.Errorf("PreviewRemove: %v", err)
	}
	if _, err := Preview(project, ModelsNamed([]string{"Moodle"}), AgentIDs()); err != nil {
		t.Errorf("Preview without the standard: %v", err)
	}

	if err := config.Update(func(c *config.Config) error { c.Docker.Apps = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := GenerateSharedFiles(project, ModelsNamed([]string{"Moodle"}), AgentIDs(), true, &out); err != nil {
		t.Fatalf("update after the app was removed: %v", err)
	}
	if _, err := os.Stat(rule); !os.IsNotExist(err) {
		t.Error("the PHP Perci82 rule survived the update")
	}
	if err := RemoveContext(project, &out); err != nil {
		t.Errorf("remove after the app was removed: %v", err)
	}
}
