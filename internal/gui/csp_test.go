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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	scriptTag   = regexp.MustCompile(`(?s)<script([^>]*)>(.*?)</script>`)
	cspMeta     = regexp.MustCompile(`<meta http-equiv="Content-Security-Policy" content="([^"]*)"`)
	scriptSrc   = regexp.MustCompile(`<script[^>]+src="\./([^"]+)"`)
	remoteRef   = regexp.MustCompile(`(?i)<(?:script|link)[^>]+(?:src|href)="(?:https?:)?//`)
)

// appDists lists each application's frontend folder, relative to this
// package.
var appDists = []string{"../../cmd/perci82-system/frontend/dist", "../../cmd/perci82-dev/frontend/dist"}

func readIndexHTML(t *testing.T, dist string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dist, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	// Comments mention "<script>" in prose; only real markup counts.
	return htmlComment.ReplaceAllString(string(data), "")
}

// The CSP allows scripts only from the app's own origin — no inline code,
// no 'unsafe-inline'.
func TestCSP_NoInlineScripts(t *testing.T) {
	for _, dist := range appDists {
		t.Run(dist, func(t *testing.T) { checkCSP(t, readIndexHTML(t, dist)) })
	}
}

func checkCSP(t *testing.T, html string) {
	m := cspMeta.FindStringSubmatch(html)
	if m == nil {
		t.Fatal("index.html has no Content-Security-Policy <meta>")
	}
	scriptSrcDirective := ""
	for _, d := range strings.Split(m[1], ";") {
		if d = strings.TrimSpace(d); strings.HasPrefix(d, "script-src ") {
			scriptSrcDirective = d
		}
	}
	if scriptSrcDirective != "script-src 'self'" {
		t.Errorf("script-src must be exactly 'self', got %q", scriptSrcDirective)
	}
	for _, s := range scriptTag.FindAllStringSubmatch(html, -1) {
		if !strings.Contains(s[1], "src=") && strings.TrimSpace(s[2]) != "" {
			t.Errorf("inline <script> found (%d bytes) — move it to a file under js/", len(s[2]))
		}
	}
}

// Every local <script src> must exist (a typo would silently drop a screen).
// Every script a page loads exists in the application's own frontend or in
// the shared one (web/), which the application serves underneath it.
func TestIndexHTML_ScriptsExist(t *testing.T) {
	for _, dist := range appDists {
		for _, m := range scriptSrc.FindAllStringSubmatch(readIndexHTML(t, dist), -1) {
			_, ownErr := os.Stat(filepath.Join(dist, m[1]))
			_, sharedErr := os.Stat(filepath.Join("web", m[1]))
			if ownErr != nil && sharedErr != nil {
				t.Errorf("%s: <script src=%q> is in neither %s nor web/", dist, m[1], dist)
			}
		}
	}
}

// Nothing is loaded from another origin (CDN).
func TestIndexHTML_NoRemoteScriptsOrStyles(t *testing.T) {
	for _, dist := range appDists {
		html := readIndexHTML(t, dist)
		if loc := remoteRef.FindStringIndex(html); loc != nil {
			t.Errorf("%s: remote <script>/<link> reference found: %q", dist, html[loc[0]:loc[1]])
		}
	}
}
