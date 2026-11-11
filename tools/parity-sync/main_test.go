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

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeOito2UI writes an oito2_ui checkout (not a git repository) with
// every exported fixture plus extra ones, and returns its path.
func fakeOito2UI(t *testing.T, extra ...string) string {
	t.Helper()
	dir := t.TempDir()
	fixtures := filepath.Join(dir, sourceFolder)
	if err := os.MkdirAll(fixtures, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range append(append(append([]string{}, usedFixtures...), ignoredFixtures...), extra...) {
		content := "{\n\t\"name\": \"" + name + "\"\n}\n"
		if name == "manifest" {
			content = "{\n\t\"flutter\": \"3.47.8\"\n}\n"
		}
		if err := os.WriteFile(filepath.Join(fixtures, name+".json"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestSync_CopiesUsedFixturesAndRecordsSource(t *testing.T) {
	from, root := fakeOito2UI(t), t.TempDir()
	var out, errOut bytes.Buffer
	if err := syncFixtures(from, root, false, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	for _, name := range usedFixtures {
		want, _ := os.ReadFile(filepath.Join(from, sourceFolder, name+".json"))
		got, err := os.ReadFile(filepath.Join(root, targetFolder, name+".json"))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s not copied byte for byte (err %v)", name, err)
		}
	}
	for _, name := range ignoredFixtures {
		if _, err := os.Stat(filepath.Join(root, targetFolder, name+".json")); err == nil {
			t.Errorf("%s copied, but Perci82 doesn't use it", name)
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, targetFolder, sourceRecord))
	if err != nil {
		t.Fatal(err)
	}
	var src map[string]any
	if err := json.Unmarshal(raw, &src); err != nil {
		t.Fatal(err)
	}
	if src["repository"] != repository || src["manifest"].(map[string]any)["flutter"] != "3.47.8" {
		t.Errorf("source.json = %s", raw)
	}
	if errOut.Len() != 0 {
		t.Errorf("unexpected warnings: %s", errOut.String())
	}
}

func TestSync_CheckReportsOnlyUsedFixtures(t *testing.T) {
	from, root := fakeOito2UI(t), t.TempDir()
	var out, errOut bytes.Buffer
	if err := syncFixtures(from, root, false, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if err := syncFixtures(from, root, true, &out, &errOut); err != nil {
		t.Fatalf("check right after a sync: %v", err)
	}

	// A change in a fixture Perci82 doesn't use passes the check.
	if err := os.WriteFile(filepath.Join(from, sourceFolder, "syntax.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := syncFixtures(from, root, true, &out, &errOut); err != nil {
		t.Errorf("check after an unused fixture changed: %v", err)
	}

	// A change in a used one fails it, and the copy is left as it was.
	if err := os.WriteFile(filepath.Join(from, sourceFolder, "schemes.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	errOut.Reset()
	if err := syncFixtures(from, root, true, &out, &errOut); err == nil {
		t.Fatal("check passed with a stale schemes.json")
	}
	if !strings.Contains(errOut.String(), "internal/theme/parity/schemes.json differs") {
		t.Errorf("stderr = %q", errOut.String())
	}
	if got, _ := os.ReadFile(filepath.Join(root, targetFolder, "schemes.json")); string(got) == "{}\n" {
		t.Error("--check overwrote the copy")
	}
}

func TestSync_NewFixtureIsAWarning(t *testing.T) {
	from, root := fakeOito2UI(t, "icons"), t.TempDir()
	var out, errOut bytes.Buffer
	if err := syncFixtures(from, root, false, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if err := syncFixtures(from, root, true, &out, &errOut); err != nil {
		t.Fatalf("a new fixture failed the check: %v", err)
	}
	if !strings.Contains(errOut.String(), "warning: oito2_ui exports icons.json") {
		t.Errorf("stderr = %q, want a warning about icons.json", errOut.String())
	}
}

func TestRun_MissingCheckoutFails(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"--from", filepath.Join(t.TempDir(), "none"), "--check"}, t.TempDir(), &out, &errOut); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}
