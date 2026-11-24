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
)

func TestCategories_AddRenameDelete(t *testing.T) {
	isolateHome(t)
	a, b := t.TempDir(), t.TempDir()
	for _, p := range []string{a, b} {
		if err := RecordContext(filepath.Base(p), p, nil, []string{AgentCodex}); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{" Trabalho ", "pessoal"} {
		if _, err := AddCategory(name); err != nil {
			t.Fatalf("AddCategory(%q) = %v", name, err)
		}
	}
	for _, bad := range []string{"", "   ", "TRABALHO", strings.Repeat("x", 41), "a\nb"} {
		if _, err := AddCategory(bad); err == nil {
			t.Errorf("AddCategory(%q) accepted", bad)
		}
	}
	if got, _ := Categories(); !slices.Equal(got, []string{"pessoal", "Trabalho"}) {
		t.Errorf("Categories = %v, want sorted without regard to case", got)
	}

	if err := SetContextCategory(a, "Trabalho"); err != nil {
		t.Fatal(err)
	}
	if err := SetContextCategory(b, "Nope"); err == nil {
		t.Error("an unknown category was accepted")
	}
	if err := SetContextCategory(t.TempDir(), "Trabalho"); err == nil {
		t.Error("an unregistered folder was accepted")
	}
	// Updating the record keeps its category.
	if err := RecordContext("A", a, []string{"Linux Bash"}, []string{AgentClaude}); err != nil {
		t.Fatal(err)
	}
	if rec, _ := FindContext(a); rec.Category != "Trabalho" {
		t.Errorf("category after RecordContext = %q", rec.Category)
	}
	if n, _ := CategoryUses("Trabalho"); n != 1 {
		t.Errorf("CategoryUses = %d, want 1", n)
	}

	if err := RenameCategory("Trabalho", "pessoal"); err == nil {
		t.Error("renaming onto another category was accepted")
	}
	if err := RenameCategory("Trabalho", "Clientes"); err != nil {
		t.Fatal(err)
	}
	if rec, _ := FindContext(a); rec.Category != "Clientes" {
		t.Errorf("category after rename = %q", rec.Category)
	}
	if err := RenameCategory("Clientes", "clientes"); err != nil {
		t.Errorf("a change of case was refused: %v", err)
	}

	if err := DeleteCategory("clientes"); err != nil {
		t.Fatal(err)
	}
	if rec, _ := FindContext(a); rec.Category != "" {
		t.Errorf("category after delete = %q", rec.Category)
	}
	if got, _ := Categories(); !slices.Equal(got, []string{"pessoal"}) {
		t.Errorf("Categories after delete = %v", got)
	}
	if err := DeleteCategory("clientes"); err == nil {
		t.Error("deleting a missing category was accepted")
	}
	if err := CheckCategory(""); err != nil {
		t.Errorf("CheckCategory(\"\") = %v", err)
	}
	if err := CheckCategory("Nope"); err == nil {
		t.Error("CheckCategory accepted an unknown category")
	}
}

// An exported folder carries its category; importing it creates the
// category when this machine lacks it, reusing one that differs only in
// case.
func TestTransfer_Category(t *testing.T) {
	isolateHome(t)
	fx := newImportFixture(t)
	if _, err := AddCategory("clientes"); err != nil {
		t.Fatal(err)
	}
	fx.file.Contexts[0].Category = "Clientes"
	fx.file.Contexts[1].Category = "Novos"
	picks := []ImportPick{{Source: fx.file.Contexts[0].Name, Name: "Um"}, {Source: fx.file.Contexts[1].Name, Name: "Dois"}}
	if err := Import(&bytes.Buffer{}, fx.file, picks, fx.ws, true); err != nil {
		t.Fatal(err)
	}
	all, _ := Contexts()
	got := map[string]string{}
	for _, c := range all {
		got[c.Name] = c.Category
	}
	if got["Um"] != "clientes" || got["Dois"] != "Novos" {
		t.Errorf("imported categories = %v", got)
	}
	if cats, _ := Categories(); !slices.Equal(cats, []string{"clientes", "Novos"}) {
		t.Errorf("Categories = %v", cats)
	}
	f := Exportable(all, fx.ws)
	for _, c := range f.Contexts {
		if c.Category != got[c.Name] {
			t.Errorf("exported %s with category %q", c.Name, c.Category)
		}
	}
}

// RegenerateSharedFiles replaces an edited file and keeps its earlier
// version as .perci-bak only when asked to.
func TestRegenerateSharedFiles_BackupOnlyWhenAsked(t *testing.T) {
	isolateHome(t)
	for _, backup := range []bool{false, true} {
		dir := t.TempDir()
		if err := RegenerateSharedFiles(dir, ModelsNamed([]string{"Linux Bash"}), []string{AgentCodex}, backup, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		agents := filepath.Join(dir, "AGENTS.md")
		if err := os.WriteFile(agents, []byte("edited by hand\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := RegenerateSharedFiles(dir, ModelsNamed([]string{"Linux Bash"}), []string{AgentCodex}, backup, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		if data, _ := os.ReadFile(agents); string(data) == "edited by hand\n" {
			t.Errorf("backup=%v: AGENTS.md was not replaced", backup)
		}
		bak, err := os.ReadFile(agents + ".perci-bak")
		switch {
		case backup && (err != nil || string(bak) != "edited by hand\n"):
			t.Errorf("backup=true: .perci-bak = %q, %v", bak, err)
		case !backup && err == nil:
			t.Error("backup=false: a .perci-bak was written")
		}
	}
}
