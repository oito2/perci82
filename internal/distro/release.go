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

import (
	"errors"
	"strconv"
	"strings"

	"github.com/oito2/perci82/internal/i18n"
)

// Release is a distribution and the oldest version of it the package
// names Perci82 Dev installs exist on.
type Release struct {
	Name    string // shown to the user, e.g. "Linux Mint"
	Minimum string // VERSION_ID, e.g. "22"
}

// minimumReleases maps os-release IDs to their oldest supported release.
// Older releases lack packages such as libfuse2t64 and openjdk-21-jdk.
var minimumReleases = map[string]Release{
	"ubuntu":    {"Ubuntu", "24.04"},
	"debian":    {"Debian", "13"},
	"linuxmint": {"Linux Mint", "22"},
	"zorin":     {"Zorin OS", "18"},
	"pop":       {"Pop!_OS", "24.04"},
}

// lmde is LMDE, Linux Mint's Debian-based edition: ID=linuxmint without
// ubuntu in ID_LIKE, numbered after Debian (LMDE 7 is Debian 13).
var lmde = Release{"LMDE", "7"}

// ReleaseTooOld reports whether the running distribution is older than
// its oldest supported release, and returns that release with the
// running version.
func ReleaseTooOld() (min Release, version string, tooOld bool) {
	loadOSRelease()
	min, tooOld = releaseTooOld(cachedID, cachedIDLike, cachedVer)
	return min, cachedVer, tooOld
}

// releaseTooOld is ReleaseTooOld for explicit os-release values. A
// distribution without a known minimum, or without a version, is never
// too old.
func releaseTooOld(id, idLike, version string) (Release, bool) {
	min, ok := minimumReleases[id]
	if id == "linuxmint" && !strings.Contains(" "+idLike+" ", " ubuntu ") {
		min, ok = lmde, true
	}
	if !ok || version == "" {
		return min, false
	}
	return min, VersionLess(version, min.Minimum)
}

// VersionLess reports whether the dotted numeric version a is older than
// b ("21.3" < "22", "24.04" < "24.10"). Missing parts count as 0 and a
// part that is not a number stops the comparison.
func VersionLess(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < max(len(as), len(bs)); i++ {
		x, y := 0, 0
		var errX, errY error
		if i < len(as) {
			x, errX = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, errY = strconv.Atoi(bs[i])
		}
		if errX != nil || errY != nil {
			return false
		}
		if x != y {
			return x < y
		}
	}
	return false
}

// osReleaseField returns the value of key= in os-release content, cleaned
// like the other fields.
func osReleaseField(content, key string) string {
	for _, line := range strings.Split(content, "\n") {
		if v, ok := strings.CutPrefix(line, key+"="); ok {
			return clean(v)
		}
	}
	return ""
}

// RequireSupportedRelease returns an error naming the oldest supported
// release when the running distribution is older than it, and nil
// otherwise.
func RequireSupportedRelease() error {
	min, version, tooOld := ReleaseTooOld()
	if !tooOld {
		return nil
	}
	return errors.New(i18n.Tr("distro.releaseTooOld", i18n.Args{"name": min.Name, "version": version, "minimum": min.Minimum}))
}
