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
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oito2/perci82/internal/config"
)

// notesSetup gives the test its own HOME with project_notes_path set to a
// notes base (empty base clears it) and returns a project folder "loja"
// and the notes folder it gets.
func notesSetup(t *testing.T, withBase bool) (project, notes string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	base := filepath.Join(home, "anotacoes")
	if err := config.Update(func(c *config.Config) error {
		if withBase {
			c.ProjectNotesPath = base
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	project = filepath.Join(home, "ws", "loja")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	return project, filepath.Join(base, "loja")
}

func jsonOf(t *testing.T, path string) map[string]any {
	t.Helper()
	doc := map[string]any{}
	if err := json.Unmarshal([]byte(readFile(t, path)), &doc); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return doc
}

func TestNotes_GenerateWiresEveryAgent(t *testing.T) {
	project, notes := notesSetup(t, true)
	var out bytes.Buffer
	if err := GenerateSharedFiles(project, ModelsNamed([]string{"Go"}), AgentIDs(), true, &out); err != nil {
		t.Fatal(err)
	}
	if !isNotesFolder(notes) {
		t.Fatalf("the notes folder was not set up: %s", notes)
	}
	if _, err := os.Stat(filepath.Join(project, ".instructions", "PROJECT.md")); !os.IsNotExist(err) {
		t.Error(".instructions/PROJECT.md was created although the project has notes")
	}
	rules := filepath.Join(notes, "PROJECT.md")
	agents := readFile(t, filepath.Join(project, "AGENTS.md"))
	if !strings.Contains(agents, "`"+rules+"`") || !strings.Contains(agents, "## Project Notes") || strings.Contains(agents, "`.instructions/PROJECT.md`**") {
		t.Errorf("AGENTS.md doesn't point at the notes:\n%s", agents)
	}
	if !strings.Contains(readFile(t, filepath.Join(project, "CLAUDE.md")), "\n@"+rules+"\n") {
		t.Error("CLAUDE.md doesn't import the notes PROJECT.md")
	}
	if !strings.Contains(readFile(t, filepath.Join(project, "GEMINI.md")), "\n@"+rules+"\n") {
		t.Error("GEMINI.md doesn't import the notes PROJECT.md")
	}

	claude := jsonOf(t, filepath.Join(project, ".claude", "settings.json"))["permissions"].(map[string]any)
	if !slices.Contains(claude["additionalDirectories"].([]any), any(notes)) || !slices.Contains(claude["allow"].([]any), any("Edit(/"+notes+"/**)")) {
		t.Errorf("Claude Code settings = %v", claude)
	}
	gemini := jsonOf(t, filepath.Join(project, ".gemini", "settings.json"))
	if !slices.Contains(gemini["context"].(map[string]any)["includeDirectories"].([]any), any(notes)) {
		t.Errorf("Gemini settings = %v", gemini)
	}
	oc := jsonOf(t, filepath.Join(project, "opencode.json"))
	if !slices.Contains(oc["instructions"].([]any), any(rules)) || len(oc["permissions"].([]any)) != 2 {
		t.Errorf("opencode.json = %v", oc)
	}
	if !strings.Contains(readFile(t, filepath.Join(project, ".codex", "config.toml")), `writable_roots = ["`+notes+`"]`) {
		t.Error("Codex config lacks the notes folder")
	}

	rec := config.AIContext{Name: "Loja", Path: project, Models: []string{"Go"}, Agents: AgentIDs()}
	if st, err := ContextState(rec); err != nil || st.State != StatusApplied {
		t.Errorf("state after generating = %+v, %v", st, err)
	}

	// Without notes the context is outdated, and generating again takes
	// every grant out while the user's own entries stay.
	if err := config.Update(func(c *config.Config) error { c.ProjectNotesPath = ""; return nil }); err != nil {
		t.Fatal(err)
	}
	if st, _ := ContextState(rec); st.State != StatusOutdated {
		t.Errorf("state without the notes setting = %q, want outdated", st.State)
	}
	settings := filepath.Join(project, ".claude", "settings.json")
	doc := jsonOf(t, settings)
	perms := doc["permissions"].(map[string]any)
	perms["allow"] = append(perms["allow"].([]any), "Bash(make *)")
	data, _ := json.Marshal(doc)
	write(t, settings, string(data))
	if err := GenerateSharedFiles(project, ModelsNamed([]string{"Go"}), AgentIDs(), true, &out); err != nil {
		t.Fatal(err)
	}
	perms = jsonOf(t, settings)["permissions"].(map[string]any)
	if perms["additionalDirectories"] != nil || !slices.Equal(perms["allow"].([]any), []any{"Bash(make *)"}) {
		t.Errorf("Claude Code grants left behind: %v", perms)
	}
	for _, gone := range []string{filepath.Join(".gemini", "settings.json"), filepath.Join(".codex", "config.toml")} {
		if _, err := os.Stat(filepath.Join(project, gone)); !os.IsNotExist(err) {
			t.Errorf("%s survived", gone)
		}
	}
	if oc := jsonOf(t, filepath.Join(project, "opencode.json")); oc["permissions"] != nil || !slices.Equal(oc["instructions"].([]any), []any{".instructions/PROJECT.md"}) {
		t.Errorf("opencode.json without notes = %v", oc)
	}
	if !isNotesFolder(notes) {
		t.Error("the notes folder must never be removed")
	}
}

func TestNotes_MoveTheProjectRules(t *testing.T) {
	project, notes := notesSetup(t, true)
	legacy := filepath.Join(project, ".instructions", "PROJECT.md")
	tpl, err := readTpl("templates/PROJECT.md")
	if err != nil {
		t.Fatal(err)
	}
	own := tpl + "\n- Use tabs.\n"
	write(t, legacy, own)

	changes, err := Preview(project, nil, []string{AgentCodex})
	if err != nil {
		t.Fatal(err)
	}
	want := []Change{{Path: filepath.Join(notes, "PROJECT.md"), Action: "create"}, {Path: ".instructions/PROJECT.md", Action: "remove"}}
	for _, w := range want {
		if !slices.Contains(changes, w) {
			t.Errorf("preview lacks %+v: %+v", w, changes)
		}
	}
	if _, err := os.Stat(notes); !os.IsNotExist(err) {
		t.Fatal("the preview created the notes folder")
	}

	var out bytes.Buffer
	if err := GenerateSharedFiles(project, nil, []string{AgentCodex}, true, &out); err != nil {
		t.Fatal(err)
	}
	if readFile(t, filepath.Join(notes, "PROJECT.md")) != own {
		t.Errorf("the project's own rules were not moved as they are")
	}
	if readFile(t, legacy+".perci-bak") != own {
		t.Error("the old PROJECT.md is not kept as .perci-bak")
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error(".instructions/PROJECT.md is still there")
	}

	// A second project rules file joins an existing notes PROJECT.md under
	// a heading; an untouched template is just removed.
	write(t, legacy, tpl+"\n- Short functions.\n")
	if err := GenerateSharedFiles(project, nil, []string{AgentCodex}, true, &out); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(notes, "PROJECT.md"))
	if !strings.Contains(got, "- Use tabs.") || !strings.Contains(got, "## Moved from .instructions/PROJECT.md (") || !strings.Contains(got, "- Short functions.") {
		t.Errorf("notes PROJECT.md after a second move:\n%s", got)
	}
	write(t, legacy, tpl)
	if err := GenerateSharedFiles(project, nil, []string{AgentCodex}, true, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error("an untouched template was kept")
	}
	if strings.Count(readFile(t, filepath.Join(notes, "PROJECT.md")), "# Project Rules") != 1 {
		t.Error("the template was appended to the notes PROJECT.md")
	}
}

func TestNotes_NoSettingKeepsTheProjectRules(t *testing.T) {
	project, notes := notesSetup(t, false)
	var out bytes.Buffer
	if err := GenerateSharedFiles(project, nil, []string{AgentClaude}, true, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, ".instructions", "PROJECT.md")); err != nil {
		t.Error(".instructions/PROJECT.md was not created")
	}
	if _, err := os.Stat(notes); !os.IsNotExist(err) {
		t.Error("a notes folder was created without the setting")
	}
	if strings.Contains(readFile(t, filepath.Join(project, "AGENTS.md")), "Project Notes") {
		t.Error("AGENTS.md has a notes section without the setting")
	}
}

func TestApplyListEdit_KeepsOtherKeysAndDropsEmptyObjects(t *testing.T) {
	doc := map[string]any{"model": "x", "permissions": map[string]any{"allow": []any{"Bash(ls *)", "Bash(old *)"}}}
	managed := func(s string) bool { return s == "Bash(old *)" }
	if changed, err := applyListEdit(doc, listEdit{keys: []string{"permissions", "allow"}, managed: managed}); err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if !slices.Equal(doc["permissions"].(map[string]any)["allow"].([]any), []any{"Bash(ls *)"}) {
		t.Errorf("doc = %v", doc)
	}
	doc = map[string]any{"context": map[string]any{"includeDirectories": []any{"/n"}}}
	if _, err := applyListEdit(doc, listEdit{keys: []string{"context", "includeDirectories"}, managed: func(string) bool { return true }}); err != nil {
		t.Fatal(err)
	}
	if len(doc) != 0 {
		t.Errorf("empty objects left: %v", doc)
	}
	if _, err := applyListEdit(map[string]any{"context": "x"}, listEdit{keys: []string{"context", "a"}, add: []string{"y"}}); err == nil {
		t.Error("a non-object value was replaced")
	}
}

func TestWriteJSONGrants_LeavesNonObjectFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	for _, body := range []string{"null", "[1]", "{bad"} {
		write(t, path, body)
		var out bytes.Buffer
		if err := writeJSONGrants(path, []listEdit{{keys: []string{"a", "b"}, add: []string{"x"}}}, &out); err != nil {
			t.Fatalf("%s: %v", body, err)
		}
		if readFile(t, path) != body || out.Len() == 0 {
			t.Errorf("%s: file changed or no warning (%q)", body, out.String())
		}
	}
}
