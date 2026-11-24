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
	"regexp"
	"strings"
)

// PinnedTool is one tool version the PHP images pin: its Packagist package,
// the release line kept (major, or major.minor for PHPUnit) and the
// pinned version.
type PinnedTool struct {
	Package string
	Line    string
	Version string
}

var (
	pinComposer = regexp.MustCompile(`^(\d+\.\d+\.\d+)@`)
	pinPHPCS    = regexp.MustCompile(`PHPCS_VERSION=(\d+\.\d+\.\d+);`)
	pinPHPUnit  = regexp.MustCompile(`VER=(\d+\.\d+\.\d+);`)
	pinArchive  = regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/archive/refs/tags/v?(\d+\.\d+\.\d+)\.tar\.gz`)
)

// archivePackages maps the GitHub repositories of the phpcs standards the
// PHP images download to their Packagist packages.
var archivePackages = map[string]string{
	"moodlehq/moodle-cs":        "moodlehq/moodle-cs",
	"PHPCSStandards/PHPCSExtra": "phpcsstandards/phpcsextra",
	"PHPCSStandards/PHPCSUtils": "phpcsstandards/phpcsutils",
}

// major returns the first component of version.
func major(version string) string {
	return strings.SplitN(version, ".", 2)[0]
}

// PinnedTools returns the tool versions the PHP images pin, read from
// their Dockerfile: Composer, PHP_CodeSniffer, each PHPUnit release line
// and the phpcs standards.
func PinnedTools() []PinnedTool {
	var out []PinnedTool
	if m := pinComposer.FindStringSubmatch(composerImage); m != nil {
		out = append(out, PinnedTool{Package: "composer/composer", Line: major(m[1]), Version: m[1]})
	}
	if m := pinPHPCS.FindStringSubmatch(phpDockerfileBody); m != nil {
		out = append(out, PinnedTool{Package: "squizlabs/php_codesniffer", Line: major(m[1]), Version: m[1]})
	}
	for _, m := range pinPHPUnit.FindAllStringSubmatch(phpunitSelect, -1) {
		parts := strings.SplitN(m[1], ".", 3)
		out = append(out, PinnedTool{Package: "phpunit/phpunit", Line: parts[0] + "." + parts[1], Version: m[1]})
	}
	for _, m := range pinArchive.FindAllStringSubmatch(phpDockerfileBody, -1) {
		if pkg, ok := archivePackages[m[1]+"/"+m[2]]; ok {
			out = append(out, PinnedTool{Package: pkg, Line: major(m[3]), Version: m[3]})
		}
	}
	return out
}
