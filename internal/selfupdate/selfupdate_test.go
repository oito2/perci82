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

package selfupdate

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oito2/perci82/internal/distro"
)

// withServer points latestBaseURL, apiBaseURL and httpClient at a local
// server running handler, and makes githubToken return token, for the
// duration of the test.
func withServer(t *testing.T, token string, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	origLatest, origAPI, origClient, origToken := latestBaseURL, apiBaseURL, httpClient, githubToken
	latestBaseURL, apiBaseURL, httpClient = srv.URL+"/latest/", srv.URL, srv.Client()
	githubToken = func(context.Context) string { return token }
	t.Cleanup(func() {
		latestBaseURL, apiBaseURL, httpClient, githubToken = origLatest, origAPI, origClient, origToken
	})
	return srv
}

const releasesPath = "/repos/oito2/perci82/releases"

// The site's latest/<app>.json answers first; the GitHub API isn't asked.
func TestLatestRelease_FromSite(t *testing.T) {
	withServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/system.json":
			fmt.Fprint(w, `{"app":"system","version":"v1.2.0","tag":"system-v1.2.0"}`)
		case "/latest/dev.json":
			fmt.Fprint(w, `{"app":"dev","version":"v1.3.1","tag":"dev-v1.3.1"}`)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	})
	for app, want := range map[string]Release{
		"system": {App: "system", Version: "v1.2.0", Tag: "system-v1.2.0"},
		"dev":    {App: "dev", Version: "v1.3.1", Tag: "dev-v1.3.1"},
	} {
		rel, err := LatestRelease(t.Context(), app)
		if err != nil || *rel != want {
			t.Errorf("LatestRelease(%s) = %+v, %v; want %+v", app, rel, err, want)
		}
	}
}

// A missing or inconsistent latest/<app>.json falls back to the GitHub
// API, which picks the app's highest published version and sends the
// token when there is one.
func TestLatestRelease_FallsBackToAPI(t *testing.T) {
	for name, site := range map[string]string{
		"missing":      "",
		"other app":    `{"app":"dev","version":"v9.0.0","tag":"dev-v9.0.0"}`,
		"bad version":  `{"app":"system","version":"1.2","tag":"system-1.2"}`,
		"tag mismatch": `{"app":"system","version":"v9.0.0","tag":"system-v1.0.0"}`,
		"not json":     `<html>`,
	} {
		t.Run(name, func(t *testing.T) {
			withServer(t, "tok", func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/latest/system.json":
					if site == "" {
						http.NotFound(w, r)
						return
					}
					fmt.Fprint(w, site)
				case releasesPath:
					if got := r.Header.Get("Authorization"); got != "Bearer tok" {
						t.Errorf("Authorization = %q", got)
					}
					fmt.Fprint(w, `[
						{"tag_name":"dev-v9.0.0"},
						{"tag_name":"system-v1.2.3"},
						{"tag_name":"system-v1.2.10"},
						{"tag_name":"system-v1.9.0","draft":true},
						{"tag_name":"system-v2.0.0","prerelease":true},
						{"tag_name":"system-v3.0.0-rc.1"},
						{"tag_name":"v5.0.0"}]`)
				default:
					http.NotFound(w, r)
				}
			})
			rel, err := LatestRelease(t.Context(), "system")
			if err != nil || rel.Version != "v1.2.10" || rel.Tag != "system-v1.2.10" {
				t.Errorf("got %+v, %v; want system v1.2.10", rel, err)
			}
		})
	}
}

// Without a token the API request carries no Authorization header; when
// both sources fail, the error names both.
func TestLatestRelease_BothFail(t *testing.T) {
	withServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("Authorization sent without a token")
		}
		w.WriteHeader(http.StatusForbidden)
	})
	_, err := LatestRelease(t.Context(), "dev")
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "GitHub API") {
		t.Errorf("err = %v, want both failures reported", err)
	}
}

func TestLatestRelease_NoReleaseOfTheApp(t *testing.T) {
	withServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == releasesPath {
			fmt.Fprint(w, `[{"tag_name":"system-v1.0.0"}]`)
			return
		}
		http.NotFound(w, r)
	})
	if _, err := LatestRelease(t.Context(), "dev"); err == nil {
		t.Error("a release of another app was accepted")
	}
}

func TestLatestRelease_UnknownApp(t *testing.T) {
	if _, err := LatestRelease(context.Background(), "other"); err == nil {
		t.Error("an unknown app was accepted")
	}
}

// Packaged requires the running binary to be /usr/bin/perci82-<app> and
// owned by a package.
func TestPackaged(t *testing.T) {
	origExe, origOwned := executablePath, ownedByPackage
	t.Cleanup(func() { executablePath, ownedByPackage = origExe, origOwned })
	for _, c := range []struct {
		name  string
		exe   string
		owned bool
		want  bool
	}{
		{"packaged", "/usr/bin/perci82-dev", true, true},
		{"not owned", "/usr/bin/perci82-dev", false, false},
		{"other folder", "/usr/local/bin/perci82-dev", true, false},
		{"other app's binary", "/usr/bin/perci82-system", true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			executablePath = func() (string, error) { return c.exe, nil }
			var asked string
			ownedByPackage = func(_ context.Context, path string) bool { asked = path; return c.owned }
			if got := Packaged(t.Context(), "dev"); got != c.want {
				t.Errorf("Packaged = %v, want %v", got, c.want)
			}
			if c.want && asked != c.exe {
				t.Errorf("asked the package database about %q", asked)
			}
		})
	}
	if Packaged(t.Context(), "other") {
		t.Error("an unknown app was reported as packaged")
	}
}

func TestCommands(t *testing.T) {
	orig := detectFamily
	t.Cleanup(func() { detectFamily = orig })
	for family, want := range map[string][2]string{
		distro.Debian:  {"sudo apt update && sudo apt install --only-upgrade perci82-system", "sudo apt remove perci82-system"},
		distro.Fedora:  {"sudo dnf upgrade --refresh perci82-system", "sudo dnf remove perci82-system"},
		distro.Unknown: {"", ""},
	} {
		detectFamily = func() string { return family }
		if got := UpgradeCommand("system"); got != want[0] {
			t.Errorf("%s: UpgradeCommand = %q, want %q", family, got, want[0])
		}
		if got := RemoveCommand("system"); got != want[1] {
			t.Errorf("%s: RemoveCommand = %q, want %q", family, got, want[1])
		}
	}
}

// writeConfig writes data as ~/.perci/config.yaml under a fresh HOME and
// returns ~/.perci.
func writeConfig(t *testing.T, data string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".perci")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// withInstallDirs makes installDirs return dirs for the duration of the
// test.
func withInstallDirs(t *testing.T, dirs ...string) {
	t.Helper()
	orig := installDirs
	installDirs = func(string) []string { return dirs }
	t.Cleanup(func() { installDirs = orig })
}

// Without RemoveConfig nothing is removed; the output ends with the
// command that removes the package.
func TestUninstall_KeepsSettings(t *testing.T) {
	cfgDir := writeConfig(t, "language: en\n")
	orig := detectFamily
	detectFamily = func() string { return distro.Debian }
	t.Cleanup(func() { detectFamily = orig })
	var buf bytes.Buffer
	if err := Uninstall(&buf, "system", UninstallOptions{}); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(cfgDir, "config.yaml")); string(data) != "language: en\n" {
		t.Errorf("config changed: %q", data)
	}
	if !strings.Contains(buf.String(), "sudo apt remove perci82-system") {
		t.Errorf("no remove command in:\n%s", buf.String())
	}
	if err := Uninstall(&buf, "other", UninstallOptions{}); err == nil {
		t.Error("an unknown app was accepted")
	}
}

// "Também remover configurações" removes only the application's own keys
// and data folders; ~/.perci goes only when nothing else needs it.
func TestUninstall_RemoveConfigKeepsWhatOthersNeed(t *testing.T) {
	cases := []struct {
		name      string
		config    string
		installed string // another binary next to the running one
		wantDir   bool
	}{
		{"last application, nothing else left", "language: en\ntray_enabled: true\n", "", false},
		{"Perci82 System still installed", "language: en\n", "perci82-system", true},
		{"old Perci still installed", "language: en\n", "prci", true},
		{"other settings left", "language: en\nflatpak_scope: user\n", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgDir := writeConfig(t, tc.config)
			for _, d := range []string{"appstack/nginx", "git"} {
				if err := os.MkdirAll(filepath.Join(cfgDir, d), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			outside := filepath.Join(filepath.Dir(cfgDir), "escape")
			if err := os.Mkdir(outside, 0o700); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			exe := filepath.Join(dir, "perci82-dev")
			if err := os.WriteFile(exe, nil, 0o755); err != nil {
				t.Fatal(err)
			}
			orig := executablePath
			executablePath = func() (string, error) { return exe, nil }
			t.Cleanup(func() { executablePath = orig })
			if tc.installed != "" {
				if err := os.WriteFile(filepath.Join(dir, tc.installed), nil, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			withInstallDirs(t, dir)

			var buf bytes.Buffer
			opts := UninstallOptions{RemoveConfig: true, DataDirs: []string{"appstack", "git", "../escape", ""}}
			if err := Uninstall(&buf, "dev", opts); err != nil {
				t.Fatalf("err = %v", err)
			}
			if _, err := os.Stat(exe); err != nil {
				t.Errorf("the binary was touched: %v", err)
			}
			for _, d := range []string{"appstack", "git"} {
				if _, err := os.Stat(filepath.Join(cfgDir, d)); !os.IsNotExist(err) {
					t.Errorf("~/.perci/%s left behind", d)
				}
			}
			if _, err := os.Stat(outside); err != nil {
				t.Errorf("a data folder outside ~/.perci was removed: %v", err)
			}
			_, err := os.Stat(cfgDir)
			if gotDir := err == nil; gotDir != tc.wantDir {
				t.Fatalf("~/.perci kept = %v, want %v\n%s", gotDir, tc.wantDir, buf.String())
			}
			if tc.wantDir {
				data, _ := os.ReadFile(filepath.Join(cfgDir, "config.yaml"))
				if strings.Contains(string(data), "language") || strings.Contains(string(data), "tray_enabled") {
					t.Errorf("Dev keys left in config.yaml:\n%s", data)
				}
			}
		})
	}
}
