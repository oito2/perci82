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
	"os"
	"path/filepath"
	"testing"
)

// A repositories export (format version 2) written by Perci 1.9.0 loads
// without conversion and writes back with the same bytes.
func TestLoadTransfer_Perci19File(t *testing.T) {
	const path = "testdata/perci-1.9/git-accounts-export.yaml"
	f, err := LoadTransfer(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Version != 2 || len(f.Accounts) != 2 || len(f.Accounts[0].Repos) != 2 ||
		f.Accounts[0].ContactEmail != "contato@example.com" || f.Accounts[0].Repos[0].ContactEmail != "site@example.com" ||
		f.Accounts[1].SSHPort != 2222 {
		t.Fatalf("export loaded incompletely: %+v", f)
	}
	if acc := f.Accounts[1].account(); acc.Provider != "gitlab" || acc.SSHPort != 2222 {
		t.Errorf("account = %+v", acc)
	}

	out := filepath.Join(t.TempDir(), "again.yaml")
	if err := WriteTransfer(out, f); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("re-exported file differs:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
