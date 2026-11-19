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

package packaging

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

// nfpmConfig holds the fields of an nfpm configuration the packaging tests
// check.
type nfpmConfig struct {
	Name        string `yaml:"name"`
	Arch        string `yaml:"arch"`
	Version     string `yaml:"version"`
	Release     string `yaml:"release"`
	Maintainer  string `yaml:"maintainer"`
	License     string `yaml:"license"`
	Homepage    string `yaml:"homepage"`
	Description string `yaml:"description"`
	Overrides   map[string]struct {
		Depends []string `yaml:"depends"`
	} `yaml:"overrides"`
	Contents []struct {
		Src      string `yaml:"src"`
		Expand   bool   `yaml:"expand"`
		Dst      string `yaml:"dst"`
		Type     string `yaml:"type"`
		Packager string `yaml:"packager"`
		FileInfo struct {
			Mode int `yaml:"mode"`
		} `yaml:"file_info"`
	} `yaml:"contents"`
}

func readNFPMConfig(t *testing.T, app string) nfpmConfig {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("nfpm", "perci82-"+app+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var c nfpmConfig
	if err := yaml.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNFPMConfigs(t *testing.T) {
	for _, app := range []string{AppSystem, AppDev} {
		t.Run(app, func(t *testing.T) {
			c := readNFPMConfig(t, app)
			pkg := "perci82-" + app
			if c.Name != pkg || c.Arch != "amd64" || c.Version != "${VERSION}" || c.Release != "1" {
				t.Errorf("name/arch/version/release = %q/%q/%q/%q", c.Name, c.Arch, c.Version, c.Release)
			}
			if c.License != "GPL-3.0-or-later" || c.Maintainer == "" || c.Homepage != "https://github.com/oito2/perci82" || c.Description == "" {
				t.Errorf("metadata incomplete: %+v", c)
			}
			if got, want := c.Overrides["deb"].Depends, []string{"libgtk-4-1", "libwebkitgtk-6.0-4"}; !slices.Equal(got, want) {
				t.Errorf("deb depends = %v, want %v", got, want)
			}
			if got, want := c.Overrides["rpm"].Depends, []string{"gtk4", "webkitgtk6.0"}; !slices.Equal(got, want) {
				t.Errorf("rpm depends = %v, want %v", got, want)
			}

			dsts := map[string]string{}
			for _, f := range c.Contents {
				dsts[f.Dst] = f.Src
				if f.Src == "${PKG_BINARY}" {
					if !f.Expand || f.FileInfo.Mode != 0o755 {
						t.Errorf("binary expand/mode = %v/%o, want true/755", f.Expand, f.FileInfo.Mode)
					}
					continue
				}
				if f.FileInfo.Mode != 0o644 {
					t.Errorf("%s mode = %o, want 644", f.Dst, f.FileInfo.Mode)
				}
				matches, err := filepath.Glob(filepath.Join("..", f.Src))
				if err != nil || len(matches) == 0 {
					t.Errorf("source %s matches no file (%v)", f.Src, err)
				}
			}
			for dst, src := range map[string]string{
				"/usr/bin/" + pkg: "${PKG_BINARY}",
				"/usr/share/applications/" + IconName(app) + ".desktop": "packaging/" + IconName(app) + ".desktop",
				"/usr/share/icons/hicolor":                              "packaging/hicolor/*/apps/" + IconName(app) + "*",
				"/usr/share/doc/" + pkg + "/copyright":                  "packaging/nfpm/copyright",
				"/usr/share/licenses/" + pkg + "/LICENSE":               "LICENSE",
			} {
				if dsts[dst] != src {
					t.Errorf("content %s comes from %q, want %q", dst, dsts[dst], src)
				}
			}
		})
	}
}

// TestNFPMIconsGlob checks that each package's icon pattern takes every
// hicolor icon of its own app (all PNG sizes, the scalable and the symbolic
// SVG) and none of the other app's.
func TestNFPMIconsGlob(t *testing.T) {
	for _, app := range []string{AppSystem, AppDev} {
		matches, err := filepath.Glob(filepath.Join("hicolor", "*", "apps", IconName(app)+"*"))
		if err != nil {
			t.Fatal(err)
		}
		want := []string{
			filepath.Join("hicolor", "scalable", "apps", IconName(app)+".svg"),
			filepath.Join("hicolor", "symbolic", "apps", IconName(app)+"-symbolic.svg"),
		}
		for _, size := range IconSizes {
			want = append(want, filepath.Join("hicolor", filepath.Dir(filepath.Dir(IconRelPath(app, size))), "apps", IconName(app)+".png"))
		}
		slices.Sort(matches)
		slices.Sort(want)
		if !slices.Equal(matches, want) {
			t.Errorf("%s icons = %v, want %v", app, matches, want)
		}
	}
}
