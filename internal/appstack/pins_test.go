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
	"slices"
	"testing"
)

// Every pinned tool of the PHP images is found, with its release line.
func TestPinnedTools(t *testing.T) {
	got := PinnedTools()
	var pkgs []string
	for _, p := range got {
		pkgs = append(pkgs, p.Package+" "+p.Line)
		if p.Version == "" {
			t.Errorf("%s has no version", p.Package)
		}
	}
	want := []string{
		"composer/composer 2",
		"squizlabs/php_codesniffer 3",
		"phpunit/phpunit 9.6", "phpunit/phpunit 10.5", "phpunit/phpunit 11.5", "phpunit/phpunit 12.5",
		"moodlehq/moodle-cs 3",
		"phpcsstandards/phpcsextra 1",
		"phpcsstandards/phpcsutils 1",
	}
	if !slices.Equal(pkgs, want) {
		t.Errorf("pinned tools = %v, want %v", pkgs, want)
	}
}
