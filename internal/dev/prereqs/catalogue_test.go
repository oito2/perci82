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

package prereqs

import (
	"bytes"
	"errors"
	"testing"

	"github.com/oito2/perci82/internal/distro"
	"github.com/oito2/perci82/internal/executor"
)

func TestLibfuse2Pkg(t *testing.T) {
	tests := []struct {
		name   string
		family string
		want   string
	}{
		{"fedora", distro.Fedora, "fuse-libs"},
		{"debian", distro.Debian, "libfuse2t64"},
		{"unknown family falls back to debian package", "arch", "libfuse2t64"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := libfuse2Pkg(tt.family); got != tt.want {
				t.Errorf("libfuse2Pkg(%q) = %q, want %q", tt.family, got, tt.want)
			}
		})
	}
}

func TestLibrsvgPkg(t *testing.T) {
	tests := []struct {
		name   string
		family string
		want   string
	}{
		{"fedora", distro.Fedora, "librsvg2-tools"},
		{"debian", distro.Debian, "librsvg2-bin"},
		{"unknown family falls back to debian package", "arch", "librsvg2-bin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := librsvgPkg(tt.family); got != tt.want {
				t.Errorf("librsvgPkg(%q) = %q, want %q", tt.family, got, tt.want)
			}
		})
	}
}

func TestMkcertPkgs(t *testing.T) {
	tests := []struct {
		name   string
		family string
		want   []string
	}{
		{"fedora", distro.Fedora, []string{"mkcert", "nss-tools"}},
		{"debian", distro.Debian, []string{"mkcert", "libnss3-tools"}},
		{"unknown family falls back to debian packages", "arch", []string{"mkcert", "libnss3-tools"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mkcertPkgs(tt.family)
			if len(got) != len(tt.want) {
				t.Fatalf("mkcertPkgs(%q) = %v, want %v", tt.family, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("mkcertPkgs(%q)[%d] = %q, want %q", tt.family, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDockerPkgs(t *testing.T) {
	tests := []struct {
		name   string
		family string
		hasV2  bool
		want   []string
	}{
		{"fedora", distro.Fedora, false, []string{"moby-engine", "docker-compose", "docker-buildx"}},
		{"ubuntu", distro.Debian, true, []string{"docker.io", "docker-compose-v2", "docker-buildx"}},
		{"debian, without docker-compose-v2", distro.Debian, false, []string{"docker.io", "docker-compose", "docker-buildx"}},
		{"unknown family falls back to debian packages", "arch", true, []string{"docker.io", "docker-compose-v2", "docker-buildx"}},
	}
	defer func(orig func(string) bool) { aptHasPackage = orig }(aptHasPackage)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aptHasPackage = func(name string) bool { return tt.hasV2 && name == "docker-compose-v2" }
			got := dockerPkgs(tt.family)
			if len(got) != len(tt.want) {
				t.Fatalf("dockerPkgs(%q) = %v, want %v", tt.family, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("dockerPkgs(%q)[%d] = %q, want %q", tt.family, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// The package's tests never ask the real apt which packages exist.
func init() {
	aptHasPackage = func(name string) bool { return name == "docker-compose-v2" }
}

// Installs and updates are refused on a release older than the package
// names exist on, before anything runs; removal still works.
func TestInstallRefusedOnOldRelease(t *testing.T) {
	defer func(orig func() error) { requireRelease = orig }(requireRelease)
	requireRelease = func() error { return errors.New("too old") }
	exe := &executor.Executor{DryRun: true}
	var out bytes.Buffer
	p := Prereq{ID: "libfuse2", Name: "Libfuse2"}
	if err := installFor(t.Context(), exe, &out, p, distro.Debian); err == nil || err.Error() != "too old" {
		t.Errorf("install = %v, want the release error", err)
	}
	if err := updateFor(t.Context(), exe, &out, p, distro.Debian); err == nil || err.Error() != "too old" {
		t.Errorf("update = %v, want the release error", err)
	}
	if out.Len() != 0 {
		t.Errorf("something ran: %q", out.String())
	}
}

// The package's tests never depend on the release of the machine running them.
func init() {
	requireRelease = func() error { return nil }
}
