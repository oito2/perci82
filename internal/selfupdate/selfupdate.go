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

// Package selfupdate finds the newest published version of each Perci82
// application, tells whether the running one was installed by a .deb or
// .rpm package, gives the package manager commands that upgrade and remove
// it, and removes its settings on uninstall. The package manager installs,
// upgrades and removes the applications themselves.
package selfupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/oito2/perci82/internal/distro"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/version"
)

const githubRepo = "oito2/perci82"

// Apps lists the application IDs this package handles (packaging.AppSystem
// and packaging.AppDev).
var Apps = []string{"system", "dev"}

// appName returns the name of the application app shown to the user.
func appName(app string) string {
	switch app {
	case "system":
		return "Perci82 System"
	case "dev":
		return "Perci82 Dev"
	}
	return "Perci82"
}

// BinaryName returns the installed binary name of app ("perci82-system"),
// which is also its package name.
func BinaryName(app string) string {
	return "perci82-" + app
}

// tagPrefix returns the prefix of app's release tags ("system-v").
func tagPrefix(app string) string {
	return app + "-v"
}

// checkApp returns an error for an app not in Apps.
func checkApp(app string) error {
	if !slices.Contains(Apps, app) {
		return fmt.Errorf("unknown application: %q", app)
	}
	return nil
}

// httpClient makes the version lookups; its timeout keeps a stalled
// connection from hanging the caller. Tests replace it with an
// httptest.Server's client.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// latestBaseURL is the folder of the package repository site holding
// latest/<app>.json, and apiBaseURL the GitHub API root. Tests point both
// at a local server.
var (
	latestBaseURL = "https://oito2.github.io/perci82/latest/"
	apiBaseURL    = "https://api.github.com"
)

// Release is the newest published version of an application.
type Release struct {
	// App is the application ("system" or "dev").
	App string
	// Version is the version without the application prefix ("v1.2.3").
	Version string
	// Tag is the release tag ("system-v1.2.3").
	Tag string
}

// maxLatestSize caps latest/<app>.json and maxReleasesSize the release
// list read from the GitHub API.
const (
	maxLatestSize   = 64 << 10
	maxReleasesSize = 8 << 20
)

// releaseVersion matches the version part of a release tag ("v1.2.3").
var releaseVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// LatestRelease returns app's newest published version: from
// latest/<app>.json on the package repository site, or, when that fails,
// from the GitHub API's release list (authenticated with the GitHub CLI's
// token when there is one, which raises the API's request limit).
func LatestRelease(ctx context.Context, app string) (*Release, error) {
	if err := checkApp(app); err != nil {
		return nil, err
	}
	rel, siteErr := latestFromSite(ctx, app)
	if siteErr == nil {
		return rel, nil
	}
	rel, apiErr := latestFromAPI(ctx, app)
	if apiErr == nil {
		return rel, nil
	}
	return nil, fmt.Errorf("%w (GitHub API: %w)", siteErr, apiErr)
}

// get requests url with the given headers and returns the open response
// body when the status is 200.
func get(ctx context.Context, url string, header map[string]string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%s returned status %d", req.URL.Host, resp.StatusCode)
	}
	return resp.Body, nil
}

// latestFromSite reads latest/<app>.json ({"app","version","tag"}) and
// checks that it describes a release of app.
func latestFromSite(ctx context.Context, app string) (*Release, error) {
	body, err := get(ctx, latestBaseURL+app+".json", nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()
	var latest struct {
		App     string `json:"app"`
		Version string `json:"version"`
		Tag     string `json:"tag"`
	}
	if err := json.NewDecoder(io.LimitReader(body, maxLatestSize)).Decode(&latest); err != nil {
		return nil, fmt.Errorf("decode latest/%s.json: %w", app, err)
	}
	if latest.App != app || !releaseVersion.MatchString(latest.Version) || latest.Tag != app+"-"+latest.Version {
		return nil, fmt.Errorf("latest/%s.json describes no release of %s", app, appName(app))
	}
	return &Release{App: app, Version: latest.Version, Tag: latest.Tag}, nil
}

// latestFromAPI lists the repository's releases on the GitHub API and
// returns app's newest one: drafts, prereleases and other applications'
// tags are skipped, and among the tags "<app>-vMAJOR.MINOR.PATCH" the
// highest version wins.
func latestFromAPI(ctx context.Context, app string) (*Release, error) {
	header := map[string]string{
		"Accept":     "application/vnd.github+json",
		"User-Agent": BinaryName(app) + "/" + version.Version,
	}
	if token := githubToken(ctx); token != "" {
		header["Authorization"] = "Bearer " + token
	}
	body, err := get(ctx, fmt.Sprintf("%s/repos/%s/releases?per_page=100", apiBaseURL, githubRepo), header)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()

	var releases []struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(io.LimitReader(body, maxReleasesSize)).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var best *Release
	var bestVersion semver
	for _, r := range releases {
		if r.Draft || r.Prerelease || !strings.HasPrefix(r.TagName, tagPrefix(app)) {
			continue
		}
		ver := strings.TrimPrefix(r.TagName, app+"-")
		if !releaseVersion.MatchString(ver) {
			continue
		}
		v, ok := parseSemver(ver)
		if !ok {
			continue
		}
		if best == nil || compareSemver(v, bestVersion) > 0 {
			best, bestVersion = &Release{App: app, Version: ver, Tag: r.TagName}, v
		}
	}
	if best == nil {
		return nil, errors.New(i18n.Tr("selfupdate.noRelease", i18n.Args{"app": appName(app)}))
	}
	return best, nil
}

// githubToken returns a GitHub token for the API request: GH_TOKEN or
// GITHUB_TOKEN, else the GitHub CLI's ("gh auth token"), else "". A
// variable so tests never run gh.
var githubToken = func(ctx context.Context) string {
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	gh, err := exec.LookPath("gh")
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, gh, "auth", "token").Output() //nolint:gosec // fixed arguments, gh resolved by LookPath
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// packagedBinDir is where the .deb and .rpm packages install the
// binaries.
const packagedBinDir = "/usr/bin"

// executablePath locates the running binary — os.Executable, replaced by
// tests.
var executablePath = os.Executable

// ownedByPackage reports whether path belongs to an installed dpkg or rpm
// package — a variable so tests never query the real package database.
var ownedByPackage = func(ctx context.Context, path string) bool {
	for _, query := range [][]string{{"dpkg-query", "-S", path}, {"rpm", "-qf", path}} {
		bin, err := exec.LookPath(query[0])
		if err != nil {
			continue
		}
		cmd := exec.CommandContext(ctx, bin, query[1:]...) //nolint:gosec // fixed query, bin resolved by LookPath
		if cmd.Run() == nil {
			return true
		}
	}
	return false
}

// Packaged reports whether the running binary is app's packaged one:
// /usr/bin/perci82-<app>, owned by an installed .deb or .rpm package.
func Packaged(ctx context.Context, app string) bool {
	if checkApp(app) != nil {
		return false
	}
	exe, err := executablePath()
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	want := filepath.Join(packagedBinDir, BinaryName(app))
	return exe == want && ownedByPackage(ctx, exe)
}

// detectFamily returns the distribution family — distro.Detect, replaced
// by tests.
var detectFamily = distro.Detect

// UpgradeCommand returns the command that upgrades app's package with this
// distribution's package manager, or "" on a family without apt or dnf.
func UpgradeCommand(app string) string {
	switch detectFamily() {
	case distro.Debian:
		return "sudo apt update && sudo apt install --only-upgrade " + BinaryName(app)
	case distro.Fedora:
		return "sudo dnf upgrade --refresh " + BinaryName(app)
	}
	return ""
}

// RemoveCommand returns the command that removes app's package with this
// distribution's package manager, or "" on a family without apt or dnf.
func RemoveCommand(app string) string {
	switch detectFamily() {
	case distro.Debian:
		return "sudo apt remove " + BinaryName(app)
	case distro.Fedora:
		return "sudo dnf remove " + BinaryName(app)
	}
	return ""
}
