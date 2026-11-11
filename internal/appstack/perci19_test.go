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

package appstack

import (
	"bytes"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// A Docker export written by Perci 1.9.0 loads without conversion and
// writes back with the same bytes: the file format is unchanged.
func TestLoadExportedConfig_Perci19File(t *testing.T) {
	const path = "testdata/perci-1.9/docker-export.yaml"
	exported, err := LoadExportedConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if exported.WorkspacePath != "/home/pessoa/workspace" || !exported.Docker.NginxCreated ||
		exported.Docker.MariaDB.DBPass != "p@ss: 'x' #1" || len(exported.Docker.Apps) != 3 ||
		len(exported.MoodlePlugins) != 1 || exported.MoodlePlugins[0].MegaIgnore != "" {
		t.Fatalf("export loaded incompletely: %+v", exported)
	}
	if app := exported.Docker.Apps[1]; app.Type != "php_node" || app.DevPort != 5173 || app.WorkerCommand == "" {
		t.Errorf("php_node app = %+v", app)
	}
	if _, err := exported.Filter(SelectAll(exported)); err != nil {
		t.Errorf("Filter(SelectAll): %v", err)
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := yaml.Marshal(exported)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("re-exported file differs:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
