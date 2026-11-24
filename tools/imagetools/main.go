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

// Command imagetools compares the tool versions the PHP images pin
// (appstack.PinnedTools) with the newest stable release of each tool's line
// on Packagist, and lists the ones with a newer release. A release younger
// than minAge is listed as waiting and doesn't count. It changes nothing.
//
//	go run ./tools/imagetools    exit 1 when a tool has a newer release at least minAge old
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/oito2/perci82/internal/appstack"
)

// minAge is how long a release is out before the images adopt it.
const minAge = 14 * 24 * time.Hour

func main() {
	os.Exit(run(context.Background(), os.Stdout, os.Stderr, fetchPackagist, time.Now()))
}

// release is one version of a Packagist package.
type release struct {
	Version string `json:"version"`
	Time    string `json:"time"`
}

// fetchFunc returns the releases of a Packagist package.
type fetchFunc func(ctx context.Context, pkg string) ([]release, error)

// run checks every pinned tool at time now and returns the exit code: 0
// when all are current, 1 when one has a newer release at least minAge
// old, 2 when a lookup failed.
func run(ctx context.Context, stdout, stderr io.Writer, fetch fetchFunc, now time.Time) int {
	code := 0
	for _, p := range appstack.PinnedTools() {
		rels, err := fetch(ctx, p.Package)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", p.Package, err)
			code = 2
			continue
		}
		newest, ok := newestInLine(rels, p.Line, now.Add(-minAge))
		waiting, _ := newestInLine(rels, p.Line, now)
		if ok && compareVersions(waiting.Version, newest.Version) > 0 && compareVersions(waiting.Version, p.Version) > 0 {
			fmt.Fprintf(stdout, "%s %s: %s waits until it is %d days old (%s)\n", p.Package, p.Line, waiting.Version, int(minAge.Hours()/24), dateOf(waiting.Time))
		}
		switch {
		case !ok:
			fmt.Fprintf(stderr, "%s: no release in line %s\n", p.Package, p.Line)
			code = 2
		case compareVersions(newest.Version, p.Version) > 0:
			fmt.Fprintf(stdout, "%s %s: pinned %s, newer %s (%s)\n", p.Package, p.Line, p.Version, newest.Version, dateOf(newest.Time))
			if code == 0 {
				code = 1
			}
		default:
			fmt.Fprintf(stdout, "%s %s: %s is current\n", p.Package, p.Line, p.Version)
		}
	}
	return code
}

// stableVersion matches a stable release, with an optional "v".
var stableVersion = regexp.MustCompile(`^v?(\d+\.\d+\.\d+)$`)

// newestInLine returns the newest stable release of rels in line (a major,
// or major.minor) published by before, with the "v" removed. A release
// without a readable time counts as published.
func newestInLine(rels []release, line string, before time.Time) (release, bool) {
	var best release
	found := false
	for _, r := range rels {
		m := stableVersion.FindStringSubmatch(r.Version)
		if m == nil || !strings.HasPrefix(m[1], line+".") {
			continue
		}
		if at, err := time.Parse(time.RFC3339, r.Time); err == nil && at.After(before) {
			continue
		}
		if !found || compareVersions(m[1], best.Version) > 0 {
			best, found = release{Version: m[1], Time: r.Time}, true
		}
	}
	return best, found
}

// compareVersions compares two MAJOR.MINOR.PATCH versions numerically.
func compareVersions(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return len(pa) - len(pb)
}

// dateOf returns the date part of a Packagist release time.
func dateOf(t string) string {
	if len(t) >= 10 {
		return t[:10]
	}
	return t
}

// packagistURL is the Packagist metadata endpoint, replaced by tests.
var packagistURL = "https://repo.packagist.org/p2/"

// maxMetadataSize caps a package's metadata.
const maxMetadataSize = 32 << 20

// fetchPackagist reads the releases of pkg from Packagist's metadata API.
// Its minified format only lists the fields that change from the previous
// release, which the version and the time always do.
func fetchPackagist(ctx context.Context, pkg string) ([]release, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, packagistURL+pkg+".json", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("packagist returned status %d", resp.StatusCode)
	}
	var doc struct {
		Packages map[string][]release `json:"packages"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxMetadataSize)).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	rels, ok := doc.Packages[pkg]
	if !ok {
		return nil, errors.New("package not in the metadata")
	}
	return rels, nil
}
