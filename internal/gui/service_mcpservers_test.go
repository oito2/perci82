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
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
)

// A server binary is deleted, with its record, only when no other scope
// still registers the server; its row shows the installed release.
func TestRemoveUnusedMCPBinary(t *testing.T) {
	isolateActions(t)
	home := os.Getenv("HOME")
	bin := filepath.Join(home, ".local", "bin", "build82")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("bin"))
	project := filepath.Join(home, "project")
	if err := config.Update(func(c *config.Config) error {
		c.MCPServers = []config.MCPServerInstall{
			{Slug: "build82", Global: true, Agents: []string{"claude"}},
			{Slug: "build82", Folder: project, Agents: []string{"claude"}},
		}
		c.MCPBinaries = []config.MCPBinary{{Slug: "build82", Version: "v1.3.0", SHA256: hex.EncodeToString(sum[:])}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	d := &DevToolsService{serviceBase{exe: executor.New(nil, nil)}}

	rows, err := d.GetMCPServerRows(true, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Slug == "build82" && (!r.Installed || r.Version != "v1.3.0") {
			t.Errorf("row = %+v", r)
		}
	}

	if err := d.removeUnusedMCPBinary(io.Discard, "build82", true, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("binary removed while the project still registers it: %v", err)
	}

	if err := recordMCPServerInstall("build82", false, project, "", nil, false); err != nil {
		t.Fatal(err)
	}
	if err := d.removeUnusedMCPBinary(io.Discard, "build82", true, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); !os.IsNotExist(err) {
		t.Errorf("binary still there: %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.MCPBinaries) != 0 {
		t.Errorf("binary record kept: %+v", cfg.MCPBinaries)
	}
}
