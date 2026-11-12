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

const fixture = `{
	"en": {"material": {"ok": "OK", "cancel": "Cancel"}, "getters": {"copied": "Copied & done"}},
	"pt-BR": {"material": {"ok": "OK", "cancel": "Cancelar"}, "getters": {"copied": "Copiado"}}
}`

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "l10n.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRun_WritesThenChecks(t *testing.T) {
	l10n := writeFixture(t, fixture)
	out := t.TempDir()
	var stderr bytes.Buffer

	if code := run([]string{"-l10n", l10n, "-out", out, "-check"}, &stderr); code != 1 {
		t.Fatalf("check before writing got exit %d, want 1", code)
	}
	if code := run([]string{"-l10n", l10n, "-out", out}, &stderr); code != 0 {
		t.Fatalf("write got exit %d: %s", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(out, "en.json"))
	if err != nil {
		t.Fatal(err)
	}
	var en map[string]string
	if err := json.Unmarshal(data, &en); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"oito2.ok": "OK", "oito2.cancel": "Cancel", "oito2.copied": "Copied & done"}
	if len(en) != len(want) {
		t.Errorf("en got %v, want %v", en, want)
	}
	for k, v := range want {
		if en[k] != v {
			t.Errorf("en[%s] got %q, want %q", k, en[k], v)
		}
	}
	if !strings.Contains(string(data), "&") {
		t.Errorf("HTML characters escaped: %s", data)
	}

	stderr.Reset()
	if code := run([]string{"-l10n", l10n, "-out", out, "-check"}, &stderr); code != 0 {
		t.Fatalf("check after writing got exit %d: %s", code, stderr.String())
	}
	if err := os.WriteFile(filepath.Join(out, "pt-BR.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"-l10n", l10n, "-out", out, "-check"}, &stderr); code != 1 {
		t.Fatalf("check of a stale catalog got exit %d, want 1", code)
	}
}

func TestGenerate_Errors(t *testing.T) {
	cases := map[string]string{
		"missing locale": `{"en": {"material": {}, "getters": {}}}`,
		"duplicate name": `{"en": {"material": {"ok": "OK"}, "getters": {"ok": "OK"}}, "pt-BR": {"material": {}, "getters": {}}}`,
		"invalid JSON":   `{`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := generate([]byte(content)); err == nil {
				t.Error("got nil error, want one")
			}
		})
	}
}

func TestRun_Usage(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"-out", t.TempDir()}, &stderr); code != 2 {
		t.Errorf("got exit %d, want 2", code)
	}
}

// The committed catalogs match the committed fixture.
func TestCommittedCatalogsUpToDate(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"-l10n", "../../theme/parity/l10n.json", "-out", "../../gui/web/i18n/oito2", "-check"}
	if code := run(args, &stderr); code != 0 {
		t.Errorf("got exit %d: %s", code, stderr.String())
	}
}
