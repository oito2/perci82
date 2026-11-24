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

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/oito2/perci82/internal/appstack"
)

func TestNewestInLine(t *testing.T) {
	rels := []release{
		{Version: "v3.10.0", Time: "2026-01-01T00:00:00+00:00"},
		{Version: "3.9.2"},
		{Version: "4.0.1"},
		{Version: "3.11.0-RC1"},
		{Version: "30.1.0"},
	}
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	got, ok := newestInLine(rels, "3", now)
	if !ok || got.Version != "3.10.0" {
		t.Errorf("newest in 3 = %+v, %v", got, ok)
	}
	if got, _ := newestInLine(rels, "3", time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)); got.Version != "3.9.2" {
		t.Errorf("a release after the cut-off was taken: %+v", got)
	}
	if _, ok := newestInLine(rels, "5", now); ok {
		t.Error("found a release in a missing line")
	}
	if compareVersions("10.5.66", "10.5.9") <= 0 || compareVersions("1.2.3", "1.2.3") != 0 {
		t.Error("compareVersions doesn't compare numerically")
	}
}

// run reports current tools, newer releases (exit 1) and failed lookups
// (exit 2).
func TestRun(t *testing.T) {
	pinned := map[string]string{}
	for _, p := range appstack.PinnedTools() {
		pinned[p.Package+" "+p.Line] = p.Version
	}
	current := func(_ context.Context, pkg string) ([]release, error) {
		var out []release
		for k, v := range pinned {
			if strings.HasPrefix(k, pkg+" ") {
				out = append(out, release{Version: v})
			}
		}
		return out, nil
	}
	var out, errOut bytes.Buffer
	now := time.Date(2030, 3, 1, 0, 0, 0, 0, time.UTC)
	if code := run(t.Context(), &out, &errOut, current, now); code != 0 {
		t.Fatalf("all current: code %d\n%s%s", code, out.String(), errOut.String())
	}

	newer := func(ctx context.Context, pkg string) ([]release, error) {
		rels, _ := current(ctx, pkg)
		if pkg == "moodlehq/moodle-cs" {
			rels = append(rels, release{Version: "v3.99.0", Time: "2030-01-02T03:04:05+00:00"})
		}
		return rels, nil
	}
	out.Reset()
	if code := run(t.Context(), &out, &errOut, newer, now); code != 1 || !strings.Contains(out.String(), "newer 3.99.0 (2030-01-02)") {
		t.Errorf("newer release: code %d\n%s", code, out.String())
	}
	out.Reset()
	if code := run(t.Context(), &out, &errOut, newer, time.Date(2030, 1, 5, 0, 0, 0, 0, time.UTC)); code != 0 || !strings.Contains(out.String(), "3.99.0 waits until it is 14 days old") {
		t.Errorf("a too recent release: code %d\n%s", code, out.String())
	}

	failing := func(context.Context, string) ([]release, error) { return nil, errors.New("offline") }
	if code := run(t.Context(), &out, &errOut, failing, now); code != 2 {
		t.Errorf("failed lookup: code %d", code)
	}
}

func TestFetchPackagist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/acme/tool.json" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"packages":{"acme/tool":[{"version":"1.2.3","time":"2026-01-01T00:00:00+00:00","require":{}},{"version":"1.2.2","time":"2025-01-01T00:00:00+00:00"}]}}`)
	}))
	defer srv.Close()
	orig := packagistURL
	packagistURL = srv.URL + "/"
	t.Cleanup(func() { packagistURL = orig })
	rels, err := fetchPackagist(t.Context(), "acme/tool")
	if err != nil || len(rels) != 2 || rels[0].Version != "1.2.3" {
		t.Errorf("fetchPackagist = %+v, %v", rels, err)
	}
	if _, err := fetchPackagist(t.Context(), "acme/missing"); err == nil {
		t.Error("a missing package was accepted")
	}
}
