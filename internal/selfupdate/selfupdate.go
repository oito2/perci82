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
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/oito2/perci82/internal/distro"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
	"github.com/oito2/perci82/internal/version"
)

const githubRepo = "oito2/perci82"
const checksumsAssetName = "checksums.txt"

// Apps lists the application IDs this package updates and uninstalls
// (packaging.AppSystem and packaging.AppDev).
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

// BinaryName returns the installed binary name of app ("perci82-system").
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

// httpClient is used for the GitHub API request. Without an explicit
// timeout, a stalled connection (slow server, dead network) would hang Run()
// forever — the ctx it receives only cancels on SIGINT/SIGTERM. Tests replace
// it with an httptest.Server's own client to trust that server's self-signed
// TLS certificate; downloadClient reuses its Transport for the same reason.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// maxBinarySize caps the downloaded binary; anything larger is refused
// explicitly instead of being truncated into a confusing checksum mismatch.
const maxBinarySize = 256 << 20

// downloadClient returns the client for release assets (checksums.txt and
// the binary). Every redirect hop is re-validated with validateDownloadURL —
// GitHub's browser_download_url redirects to its asset CDN, and without this
// only the first URL would ever be checked. timeout 0 means no overall
// deadline: the binary download is bounded by Run's context instead, since
// http.Client.Timeout also covers reading the body and a ~25 MB binary on a
// slow link easily takes longer than any fixed request timeout.
func downloadClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: httpClient.Transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects downloading %s", via[0].URL)
			}
			return validateDownloadURL(req.URL.String())
		},
	}
}

// apiBaseURL is the GitHub API root. Tests override it to point at a local
// httptest.Server instead of the real GitHub API.
var apiBaseURL = "https://api.github.com"

// trustedDownloadHost/trustedDownloadHostSuffix are the hosts
// validateDownloadURL accepts for binary/checksum downloads. Tests override
// them to accept the local httptest.Server's host:port.
var (
	trustedDownloadHost       = "github.com"
	trustedDownloadHostSuffix = ".githubusercontent.com"
)

// executablePath locates the running binary — os.Executable, replaced by
// tests with a file in a temp dir.
var executablePath = os.Executable

// Release holds information about a GitHub release.
//
// ChecksumsURL is the release's checksums.txt (empty when the release has
// none). ChecksumSHA256 is empty until it is resolved from ChecksumsURL —
// LatestRelease does not download the file, so a plain version check stays a
// single API request; Apply resolves it and refuses to go on without it.
type Release struct {
	// App is the application the release belongs to ("system" or "dev").
	App            string
	Version        string
	DownloadURL    string
	ChecksumsURL   string
	ChecksumSHA256 string
}

// assetName is app's release binary for the running architecture.
func assetName(app string) string {
	return fmt.Sprintf("%s-linux-%s", BinaryName(app), runtime.GOARCH)
}

// Run checks for a newer version of app and applies the update if one is
// available. There is no overall deadline: the API requests have their
// own timeout, the binary download stops when no data arrives for
// stallTimeout (a slow link may take long, never forever), and the
// privileged swap is never cut while the password dialog is open.
func Run(ctx context.Context, exe *executor.Executor, stdout io.Writer, app string) error {
	if err := checkApp(app); err != nil {
		return err
	}

	ui.PrintHeader(stdout, i18n.Tr("selfupdate.header", i18n.Args{"app": appName(app)}), distro.DisplayName())
	ui.Info(stdout, i18n.Tr("selfupdate.current", i18n.Args{"version": version.Version}))

	if version.Version == "dev" {
		return i18n.NewError("selfupdate.devBuild")
	}

	rel, err := LatestRelease(ctx, app)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}

	ui.Info(stdout, i18n.Tr("selfupdate.latest", i18n.Args{"version": rel.Version}))

	if !IsNewer(rel.Version, version.Version) {
		ui.Success(stdout, i18n.Tr("selfupdate.upToDate", nil))
		return nil
	}

	ui.Info(stdout, i18n.Tr("selfupdate.downloading", i18n.Args{"version": rel.Version, "platform": runtime.GOOS + "/" + runtime.GOARCH}))
	if err := Apply(ctx, exe, stdout, rel); err != nil {
		return fmt.Errorf("update binary: %w", err)
	}

	ui.Success(stdout, i18n.Tr("selfupdate.done", i18n.Args{"app": appName(app), "version": rel.Version}))
	return nil
}

// maxReleasesSize caps the release list read from the GitHub API.
const maxReleasesSize = 8 << 20

// releaseVersion matches the version part of a release tag ("v1.2.3").
var releaseVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// LatestRelease lists the repository's releases on the GitHub API and
// returns app's newest one: drafts, prereleases and other applications'
// tags are skipped, and among the tags "<app>-vMAJOR.MINOR.PATCH" the
// highest version wins. Release.Version has no application prefix
// ("v1.2.3").
func LatestRelease(ctx context.Context, app string) (*Release, error) {
	if err := checkApp(app); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/repos/%s/releases?per_page=100", apiBaseURL, githubRepo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", BinaryName(app)+"/"+version.Version)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	type asset struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}
	var releases []struct {
		TagName    string  `json:"tag_name"`
		Draft      bool    `json:"draft"`
		Prerelease bool    `json:"prerelease"`
		Assets     []asset `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxReleasesSize)).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	best := -1
	var bestVersion semver
	for i, r := range releases {
		if r.Draft || r.Prerelease || !strings.HasPrefix(r.TagName, tagPrefix(app)) {
			continue
		}
		tag := strings.TrimPrefix(r.TagName, app+"-")
		if !releaseVersion.MatchString(tag) {
			continue
		}
		v, ok := parseSemver(tag)
		if !ok {
			continue
		}
		if best < 0 || compareSemver(v, bestVersion) > 0 {
			best, bestVersion = i, v
		}
	}
	if best < 0 {
		return nil, errors.New(i18n.Tr("selfupdate.noRelease", i18n.Args{"app": appName(app)}))
	}
	rel := releases[best]

	name := assetName(app)
	var downloadURL, checksumsURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case name:
			downloadURL = a.BrowserDownloadURL
		case checksumsAssetName:
			checksumsURL = a.BrowserDownloadURL
		}
	}
	if downloadURL == "" {
		return nil, fmt.Errorf("asset '%s' not found in release %s", name, rel.TagName)
	}
	if err := validateDownloadURL(downloadURL); err != nil {
		return nil, err
	}
	if checksumsURL != "" {
		if err := validateDownloadURL(checksumsURL); err != nil {
			return nil, err
		}
	}

	return &Release{
		App:          app,
		Version:      strings.TrimPrefix(rel.TagName, app+"-"),
		DownloadURL:  downloadURL,
		ChecksumsURL: checksumsURL,
	}, nil
}

// resolveChecksum fills rel.ChecksumSHA256 from the release's checksums.txt
// when it is not set yet. A release without checksums.txt is refused: there
// would be nothing to verify the binary against.
func resolveChecksum(ctx context.Context, rel *Release) error {
	if rel.ChecksumSHA256 != "" {
		return nil
	}
	if rel.ChecksumsURL == "" {
		return fmt.Errorf("'%s' not found in release %s — update refused (no checksum to verify the binary)", checksumsAssetName, rel.Version)
	}
	if err := validateDownloadURL(rel.ChecksumsURL); err != nil {
		return err
	}
	name := assetName(rel.App)
	checksum, err := fetchChecksum(ctx, rel.ChecksumsURL, name)
	if err != nil {
		return fmt.Errorf("get checksum of %s: %w", name, err)
	}
	rel.ChecksumSHA256 = checksum
	return nil
}

// validateDownloadURL rejects anything that isn't an https URL on github.com
// or a *.githubusercontent.com asset CDN host. downloadURL/checksumsURL come
// straight from the GitHub API response — this is a cheap extra layer of
// defense in case that response is ever tampered with or the parsing logic
// changes, on top of the checksum verification in Apply.
func validateDownloadURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("download URL must use https, got %q", u.Scheme)
	}
	if u.Host != trustedDownloadHost && !strings.HasSuffix(u.Host, trustedDownloadHostSuffix) {
		return fmt.Errorf("untrusted download host: %q", u.Host)
	}
	return nil
}

// fetchChecksum downloads a "sha256sum"-style checksums file from checksumsURL
// and returns the hash for the line matching assetName.
func fetchChecksum(ctx context.Context, checksumsURL, assetName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumsURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := downloadClient(30 * time.Second).Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 1<<20))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == assetName {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no entry for '%s' in %s", assetName, checksumsAssetName)
}

// Apply downloads the new binary and replaces the current one. It resolves
// and verifies the release checksum first and refuses to continue without it.
func Apply(ctx context.Context, exe *executor.Executor, stdout io.Writer, rel *Release) (err error) {
	if err := resolveChecksum(ctx, rel); err != nil {
		return err
	}

	currentExe, err := executablePath()
	if err != nil {
		return fmt.Errorf("locate current binary: %w", err)
	}

	// Download next to the current binary when possible so the final swap is
	// a same-filesystem rename. The system temp dir is often a separate
	// mount (e.g. tmpfs), where os.Rename fails with EXDEV, a distinct error
	// from permission denied. sameFS tracks which directory tmp actually
	// landed in so the swap logic below knows whether a plain rename can
	// even work.
	sameFS := true
	tmp, err := os.CreateTemp(filepath.Dir(currentExe), "perci82-*.new")
	if err != nil {
		sameFS = false
		tmp, err = os.CreateTemp("", "perci82-*.new")
		if err != nil {
			return fmt.Errorf("create temporary file: %w", err)
		}
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	// tmp is kept open and passed straight to downloadFile instead of being
	// closed and reopened by tmpPath — reopening by path would leave a window
	// where tmpPath could be deleted and recreated as a symlink to an
	// arbitrary target (TOCTOU), which downloadFile would then follow and
	// write into, and the later os.Rename would move that symlink
	// itself into place as currentExe.
	ui.Info(stdout, i18n.Tr("selfupdate.downloadingTo", i18n.Args{"path": tmpPath}))
	downloadErr := downloadFile(ctx, rel.DownloadURL, tmp)
	closeErr := tmp.Close()
	if downloadErr != nil {
		return fmt.Errorf("download: %w", downloadErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close temporary file: %w", closeErr)
	}

	ui.Info(stdout, i18n.Tr("selfupdate.verifying", nil))
	if err := verifyChecksum(tmpPath, rel.ChecksumSHA256); err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}

	// currentExe's directory is writable by the user (tmp was created right
	// next to it): the swap needs no privilege, a same-filesystem rename is
	// atomic, and the file never passes through root.
	if sameFS {
		if err := os.Chmod(tmpPath, 0o755); err != nil {
			return fmt.Errorf("chmod: %w", err)
		}
		if err := os.Rename(tmpPath, currentExe); err == nil {
			return nil
		} else if !os.IsPermission(err) {
			return fmt.Errorf("replace binary: %w", err)
		}
		ui.Warning(stdout, i18n.Tr("selfupdate.permissionDenied", nil))
	}

	// Privileged swap: root copies tmpPath into a root-owned staged file,
	// re-verifies the checksum on that copy and renames it over currentExe
	// (executor.PrivilegedInstallSteps) — never a plain `mv` of a file the
	// user can still rewrite while the password dialog is open.
	// Once the password dialog is shown, the batch runs to its end: killing
	// pkexec half-way would leave a root-owned staged file behind.
	if err := exe.PrivilegedInstall(context.WithoutCancel(ctx),
		executor.Options{Stdout: stdout, Stderr: stdout},
		tmpPath, currentExe, strings.ToLower(strings.TrimSpace(rel.ChecksumSHA256)), "0755",
	); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

// downloadFile writes url's body into dest, which the caller already has open
// (see the TOCTOU note in Apply — this must never reopen a file by path).
func downloadFile(ctx context.Context, url string, dest *os.File) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stall := time.AfterFunc(stallTimeout, cancel)
	defer stall.Stop()
	resp, err := downloadClient(0).Do(req.WithContext(ctx))
	if err != nil {
		return stalledOr(parent, ctx, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	body := &stallReader{r: io.LimitReader(resp.Body, maxBinarySize+1), timer: stall}
	n, err := io.Copy(dest, body)
	if err != nil {
		return stalledOr(parent, ctx, err)
	}
	if n > maxBinarySize {
		return fmt.Errorf("file larger than %d MiB — download refused", maxBinarySize>>20)
	}
	// Flush to disk before the rename: a power loss right after it must not
	// leave a truncated binary in place.
	return dest.Sync()
}

// stallTimeout is how long the binary download may go without receiving
// any data before it is abandoned (a variable so tests shorten it).
var stallTimeout = 60 * time.Second

// stallReader restarts timer on every read that returns data.
type stallReader struct {
	r     io.Reader
	timer *time.Timer
}

func (s *stallReader) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	if n > 0 {
		s.timer.Reset(stallTimeout)
	}
	return n, err
}

// stalledOr names a download cancelled by the stall timer (ctx done while
// its parent is not); any other error is returned as it is.
func stalledOr(parent, ctx context.Context, err error) error {
	if parent.Err() == nil && ctx.Err() != nil {
		return fmt.Errorf("no data received for %s: %w", stallTimeout, err)
	}
	return err
}

// verifyChecksum reports an error if the SHA-256 of the file at path does not
// match expectedSHA256. This is the only thing standing between a compromised
// download (bad DNS, MITM'd CDN, tampered release asset) and replacing the
// running binary — Apply must call this before the file is ever executed.
func verifyChecksum(path, expectedSHA256 string) error {
	return fsutil.VerifyFile(path, sha256.New(), expectedSHA256)
}
