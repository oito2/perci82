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

package distro

import "testing"

func TestReleaseTooOld(t *testing.T) {
	tests := []struct {
		id, idLike, version string
		want                bool
		min                 string
	}{
		{"linuxmint", "ubuntu debian", "21.3", true, "22"},
		{"linuxmint", "ubuntu debian", "22.1", false, "22"},
		{"linuxmint", "debian", "6", true, "7"}, // LMDE 6
		{"linuxmint", "debian", "7", false, "7"},
		{"debian", "", "12", true, "13"},
		{"debian", "", "13", false, "13"},
		{"ubuntu", "debian", "22.04", true, "24.04"},
		{"ubuntu", "debian", "24.04", false, "24.04"},
		{"zorin", "ubuntu debian", "17", true, "18"},
		{"pop", "ubuntu debian", "24.04", false, "24.04"},
		{"fedora", "", "40", false, ""},
		{"elementary", "ubuntu debian", "7", false, ""},
		{"debian", "", "", false, "13"}, // sid/testing have no VERSION_ID
	}
	for _, tt := range tests {
		min, got := releaseTooOld(tt.id, tt.idLike, tt.version)
		if got != tt.want || min.Minimum != tt.min {
			t.Errorf("releaseTooOld(%q, %q, %q) = %v, %v; want %q, %v", tt.id, tt.idLike, tt.version, min, got, tt.min, tt.want)
		}
	}
}

func TestVersionLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"21.3", "22", true},
		{"22", "22", false},
		{"24.04", "24.10", true},
		{"24.10", "24.04", false},
		{"13", "13.1", true},
		{"x", "1", false},
	}
	for _, tt := range tests {
		if got := VersionLess(tt.a, tt.b); got != tt.want {
			t.Errorf("VersionLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestOSReleaseField(t *testing.T) {
	if got := osReleaseField("ID=linuxmint\nID_LIKE=\"ubuntu debian\"\n", "ID_LIKE"); got != "ubuntu debian" {
		t.Errorf("ID_LIKE = %q", got)
	}
}
