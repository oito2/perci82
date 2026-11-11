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
	"testing"
)

// An AI contexts export written by Perci 1.9.0 loads without conversion,
// plans its folders against the workspace and writes back with the same
// bytes.
func TestLoadTransfer_Perci19File(t *testing.T) {
	const path = "testdata/perci-1.9/ai-contexts-export.yaml"
	f, err := LoadTransfer(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Kind != TransferKind || f.Version != 1 || len(f.Contexts) != 2 {
		t.Fatalf("export loaded incompletely: %+v", f)
	}

	plan := PlanImport(f, nil, "/home/pessoa/workspace", nil)
	if len(plan) != 2 || plan[0].Path != "/home/pessoa/workspace/github/labs/site" || plan[1].Path != "/srv/projetos/fora" {
		t.Errorf("planned folders = %+v", plan)
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
