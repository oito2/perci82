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
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oito2/perci82/internal/config"
)

// saveCustom saves c, failing the test on an error.
func saveCustom(t *testing.T, c CustomContext) CustomContext {
	t.Helper()
	saved, err := SaveCustomContext(c)
	if err != nil {
		t.Fatalf("SaveCustomContext(%+v): %v", c, err)
	}
	return saved
}

func TestCustomContexts_SaveListAndFind(t *testing.T) {
	isolateHome(t)
	if list, err := CustomContexts(); err != nil || len(list) != 0 {
		t.Fatalf("empty store = %+v, %v", list, err)
	}

	py := saveCustom(t, CustomContext{Name: "  Python Scripts ", AppliesTo: "Writing\n  Python", Content: "Use ruff.\r\nType hints."})
	want := CustomContext{ID: "python-scripts", Name: "Python Scripts", AppliesTo: "Writing Python", Content: "Use ruff.\nType hints.\n"}
	if py != want {
		t.Fatalf("saved = %+v, want %+v", py, want)
	}
	other := saveCustom(t, CustomContext{Name: "python scripts!", Content: "x"})
	if other.ID != "" && other.ID == py.ID {
		t.Fatalf("second context reused the ID %q", other.ID)
	}
	if _, err := SaveCustomContext(CustomContext{Name: "PYTHON SCRIPTS", Content: "x"}); err == nil {
		t.Error("a name used by another context was accepted")
	}

	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".perci", "user-contexts", "python-scripts.md")
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("file %s: %v, %v", path, info, err)
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("folder mode: %v, %v", info, err)
	}

	// Renaming keeps the ID; the list is ordered by name.
	py.Name = "A Python"
	if got := saveCustom(t, py); got.ID != "python-scripts" {
		t.Errorf("renamed ID = %q", got.ID)
	}
	list, err := CustomContexts()
	if err != nil || len(list) != 2 || list[0].Name != "A Python" {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if got, err := FindCustomContext("python-scripts"); err != nil || got.Name != "A Python" {
		t.Errorf("FindCustomContext = %+v, %v", got, err)
	}
	for _, id := range []string{"../config", "Python", ""} {
		if _, err := FindCustomContext(id); !errors.Is(err, ErrCustomNotFound) {
			t.Errorf("FindCustomContext(%q) = %v", id, err)
		}
	}
	if _, err := SaveCustomContext(CustomContext{ID: "nope", Name: "N", Content: "x"}); !errors.Is(err, ErrCustomNotFound) {
		t.Errorf("save over a missing ID: %v", err)
	}

	// Files that aren't a valid context are skipped.
	write(t, filepath.Join(filepath.Dir(path), "broken.md"), "no header")
	write(t, filepath.Join(filepath.Dir(path), "Bad Name.md"), "---\nname: X\n---\ny\n")
	if list, _ := CustomContexts(); len(list) != 2 {
		t.Errorf("list with junk files = %+v", list)
	}
}

func TestCustomContexts_Validation(t *testing.T) {
	isolateHome(t)
	for _, c := range []CustomContext{
		{Name: "", Content: "x"},
		{Name: "a\nb", Content: "x"},
		{Name: strings.Repeat("n", 81), Content: "x"},
		{Name: "n", Content: "  \n "},
		{Name: "n", AppliesTo: strings.Repeat("a", 301), Content: "x"},
		{Name: "n", Content: strings.Repeat("x", maxCustomContentLen+1)},
		{Name: "n", Content: "\xff"},
	} {
		if _, err := SaveCustomContext(c); err == nil {
			t.Errorf("SaveCustomContext(%.40q...) accepted", c.Name+"|"+c.AppliesTo+"|"+c.Content)
		}
	}
}

func TestNewCustomID(t *testing.T) {
	taken := []CustomContext{{ID: "go"}, {ID: "go-2"}}
	for name, want := range map[string]string{
		"Go":                       "go-3",
		"Regras -- do Time":        "regras-do-time",
		"Ação Rápida":              "a-o-r-pida",
		"!!!":                      "context",
		strings.Repeat("abc ", 30): strings.TrimRight(strings.Repeat("abc-", 9), "-"),
	} {
		if got := newCustomID(name, taken); got != want || !validCustomID(got) {
			t.Errorf("newCustomID(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestCustomContext_GeneratedLikeAStandard(t *testing.T) {
	isolateHome(t)
	c := saveCustom(t, CustomContext{Name: "Team Rules", AppliesTo: "Any task in this repository", Content: "Rule one.\n"})
	dir := t.TempDir()
	active := ModelsNamed([]string{"Go", CustomModelName(c.ID)})
	if len(active) != 2 {
		t.Fatalf("ModelsNamed = %+v", active)
	}
	if err := GenerateSharedFiles(dir, active, []string{AgentClaude, AgentCodex}, true, io.Discard); err != nil {
		t.Fatal(err)
	}

	instr := filepath.Join(dir, ".instructions", "CUSTOM-TEAM-RULES.md")
	if got := readFile(t, instr); got != signCustom("Rule one.\n") {
		t.Errorf("instruction file = %q", got)
	}
	// Always loaded by Claude Code: no paths frontmatter.
	if got := readFile(t, filepath.Join(dir, ".claude", "rules", "perci", "CUSTOM-TEAM-RULES.md")); got != signCustom("Rule one.\n") {
		t.Errorf("Claude rule = %q", got)
	}
	agents := readFile(t, filepath.Join(dir, "AGENTS.md"))
	if !strings.Contains(agents, "\n- Any task in this repository: `.instructions/CUSTOM-TEAM-RULES.md`") {
		t.Errorf("AGENTS.md lacks the context:\n%s", agents)
	}
	if own, err := projectContent(agents, ""); err != nil || own != "" {
		t.Errorf("the custom line counts as hand-written: %q, %v", own, err)
	}
	if ms, _ := Detected(dir); !slices.Equal(ms, []string{"Go", CustomModelName(c.ID)}) {
		t.Errorf("Detected = %v", ms)
	}
	m := active[1]
	if st := GuideStatus(dir, m); st != StatusApplied {
		t.Errorf("status after apply = %q", st)
	}

	// Editing the context makes the project outdated; Atualizar brings it
	// back.
	c.Content = "Rule two.\n"
	saveCustom(t, c)
	m = ModelsNamed([]string{CustomModelName(c.ID)})[0]
	if st := GuideStatus(dir, m); st != StatusOutdated {
		t.Errorf("status after editing the context = %q", st)
	}
	if err := GenerateSharedFiles(dir, []Model{m}, []string{AgentClaude}, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	if st := GuideStatus(dir, m); st != StatusApplied {
		t.Errorf("status after updating = %q", st)
	}

	// A hand edit in the project is "edited" and survives unchecking.
	write(t, instr, readFile(t, instr)+"mine\n")
	if st := GuideStatus(dir, m); st != StatusEdited {
		t.Errorf("status after a hand edit = %q", st)
	}
	if err := GenerateSharedFiles(dir, nil, []string{AgentClaude}, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !exists(t, instr) {
		t.Error("an edited custom context was removed")
	}
	if exists(t, filepath.Join(dir, ".claude", "rules", "perci", "CUSTOM-TEAM-RULES.md")) {
		t.Error("the unedited Claude rule was kept")
	}
}

func TestCustomContext_RemovedWhenUnchecked(t *testing.T) {
	isolateHome(t)
	c := saveCustom(t, CustomContext{Name: "Old", Content: "v1\n"})
	dir := t.TempDir()
	if err := GenerateSharedFiles(dir, ModelsNamed([]string{CustomModelName(c.ID)}), []string{AgentClaude}, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	// An earlier version, still signed, is removed too.
	c.Content = "v2\n"
	saveCustom(t, c)
	if err := GenerateSharedFiles(dir, nil, []string{AgentClaude}, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{".instructions/CUSTOM-OLD.md", ".claude/rules/perci/CUSTOM-OLD.md"} {
		if exists(t, filepath.Join(dir, p)) {
			t.Errorf("%s kept", p)
		}
	}
}

func TestDeleteCustomContext_RefusedWhileInUse(t *testing.T) {
	isolateHome(t)
	c := saveCustom(t, CustomContext{Name: "Used", Content: "x\n"})
	dir := t.TempDir()
	if err := RecordContext("Proj", dir, []string{CustomModelName(c.ID)}, []string{AgentCodex}); err != nil {
		t.Fatal(err)
	}
	err := DeleteCustomContext(c.ID)
	if err == nil || !strings.Contains(err.Error(), "Proj") {
		t.Fatalf("delete in use: %v", err)
	}
	if err := ForgetContext(dir); err != nil {
		t.Fatal(err)
	}
	if err := DeleteCustomContext(c.ID); err != nil {
		t.Fatalf("delete unused: %v", err)
	}
	if _, err := FindCustomContext(c.ID); !errors.Is(err, ErrCustomNotFound) {
		t.Errorf("still found: %v", err)
	}
	if err := DeleteCustomContext(c.ID); !errors.Is(err, ErrCustomNotFound) {
		t.Errorf("delete twice: %v", err)
	}
}

func TestTransfer_CarriesCustomContexts(t *testing.T) {
	isolateHome(t)
	used := saveCustom(t, CustomContext{Name: "Used", AppliesTo: "Always", Content: "u\n"})
	saveCustom(t, CustomContext{Name: "Unused", Content: "n\n"})
	ws := t.TempDir()
	f := Exportable([]config.AIContext{{Name: "P", Path: filepath.Join(ws, "p"), Models: []string{"Go", CustomModelName(used.ID)}, Agents: []string{AgentCodex}}}, ws)
	if len(f.Custom) != 1 || f.Custom[0] != used {
		t.Fatalf("exported custom = %+v", f.Custom)
	}
	path := filepath.Join(t.TempDir(), "x.yaml")
	if err := WriteTransfer(path, f); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadTransfer(path)
	if err != nil || len(loaded.Custom) != 1 || loaded.Custom[0] != used {
		t.Fatalf("LoadTransfer = %+v, %v", loaded, err)
	}

	// On another machine: the context is known through the file and is
	// created by the import.
	isolateHome(t)
	if err := os.Mkdir(filepath.Join(ws, "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	plans := PlanImport(loaded, nil, ws, nil)
	if len(plans) != 1 || plans[0].State != ImportApply || len(plans[0].Unknown) != 0 || !slices.Contains(plans[0].Models, CustomModelName(used.ID)) {
		t.Fatalf("plan = %+v", plans)
	}
	picks := []ImportPick{{Source: "P", Name: "P"}}
	if _, changes, err := PreviewImport(loaded, picks, nil, ws); err != nil || !slices.ContainsFunc(changes[0], func(c Change) bool { return c.Path == ".instructions/CUSTOM-USED.md" }) {
		t.Fatalf("preview = %+v, %v", changes, err)
	}
	var out bytes.Buffer
	if err := Import(&out, loaded, picks, ws, true); err != nil {
		t.Fatalf("import: %v\n%s", err, out.String())
	}
	if got, err := FindCustomContext(used.ID); err != nil || got != used {
		t.Errorf("created = %+v, %v", got, err)
	}
	if !exists(t, filepath.Join(ws, "p", ".instructions", "CUSTOM-USED.md")) {
		t.Error("the project did not get the custom context")
	}
}

func TestImport_KeepsALocalCustomContext(t *testing.T) {
	isolateHome(t)
	local := saveCustom(t, CustomContext{Name: "Rules", Content: "local\n"})
	ws := t.TempDir()
	if err := os.Mkdir(filepath.Join(ws, "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	f := TransferFile{Kind: TransferKind, Version: 1,
		Contexts: []TransferContext{{Name: "P", Path: "p", Relative: true, Models: []string{CustomModelName(local.ID)}, Agents: []string{AgentCodex}}},
		Custom:   []CustomContext{{ID: local.ID, Name: "Rules", Content: "from file\n"}},
	}
	var out bytes.Buffer
	if err := Import(&out, f, []ImportPick{{Source: "P", Name: "P"}}, ws, true); err != nil {
		t.Fatal(err)
	}
	if got, _ := FindCustomContext(local.ID); got != local {
		t.Errorf("local context replaced: %+v", got)
	}
	if !strings.Contains(out.String(), "Rules") {
		t.Errorf("no warning: %q", out.String())
	}
}

func TestTransferFile_RefusesBadCustomContexts(t *testing.T) {
	for _, c := range []CustomContext{
		{ID: "../x", Name: "X", Content: "x\n"},
		{ID: "x", Name: "", Content: "x\n"},
		{ID: "x", Name: "X", Content: "no final newline"},
	} {
		f := TransferFile{Kind: TransferKind, Version: 1, Custom: []CustomContext{c}}
		if err := f.Validate(); err == nil {
			t.Errorf("Validate accepted %+v", c)
		}
	}
	twice := TransferFile{Kind: TransferKind, Version: 1, Custom: []CustomContext{{ID: "x", Name: "X", Content: "x\n"}, {ID: "x", Name: "Y", Content: "y\n"}}}
	if err := twice.Validate(); err == nil {
		t.Error("Validate accepted a repeated ID")
	}
}
