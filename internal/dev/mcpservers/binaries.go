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

package mcpservers

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"strings"
	"time"

	"github.com/oito2/perci82/internal/dev/nettimeout"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/selfupdate"
	"github.com/oito2/perci82/internal/ui"
)

// binaryServer is a server shipped as a release binary of a GitHub
// repository: the latest stable release's "<name>_linux_<arch>" asset is
// installed as ~/.local/bin/<name> and the agents launch it by that path,
// with no arguments.
type binaryServer struct {
	repo string // "owner/repo"
	name string // binary name, also the asset prefix
}

// binaryServers maps each catalog slug installed from a release binary to
// its repository.
var binaryServers = map[string]binaryServer{
	"build82": {repo: "oito2/mcp-build82", name: "build82"},
	"sync82":  {repo: "oito2/mcp-sync82", name: "sync82"},
}

// IsBinary reports whether slug is installed from a release binary.
func IsBinary(slug string) bool {
	_, ok := binaryServers[slug]
	return ok
}

// BinaryInfo identifies a binary Perci installed: the release tag it came
// from and the SHA-256 of the file. A file at the binary's path is Perci's
// only while its SHA-256 matches.
type BinaryInfo struct {
	Version string
	SHA256  string
}

// BinaryPath returns where slug's binary is installed (~/.local/bin/<name>).
func BinaryPath(slug string) (string, error) {
	b, ok := binaryServers[slug]
	if !ok {
		return "", fmt.Errorf("%s is not installed from a release binary", slug)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".local", "bin", b.name), nil
}

// The variables below are replaced by tests: the GitHub API root, the
// hosts release assets may be downloaded from, and the HTTP client.
var (
	releaseAPIBase      = "https://api.github.com"
	trustedAssetHost    = "github.com"
	trustedAssetSuffix  = ".githubusercontent.com"
	releaseHTTPClient   = &http.Client{Timeout: 30 * time.Second}
	maxBinaryDownload   = int64(256 << 20)
	stableReleaseTag    = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
	errForeignBinary    = i18n.NewError("mcp.binary.foreign")
	checksumsAssetName  = "checksums.txt"
	maxChecksumsFileLen = int64(1 << 20)
)

// binaryRelease is the part of a GitHub release used to install a binary.
type binaryRelease struct {
	version      string
	asset        string
	binaryURL    string
	checksumsURL string
}

// latestBinaryRelease reads b's latest stable release (GitHub's
// releases/latest skips drafts and prereleases) and finds the binary for
// the running architecture and the release's checksums.txt.
func latestBinaryRelease(ctx context.Context, b binaryServer) (binaryRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPIBase+"/repos/"+b.repo+"/releases/latest", nil)
	if err != nil {
		return binaryRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := releaseHTTPClient.Do(req)
	if err != nil {
		return binaryRelease{}, fmt.Errorf("query latest release of %s: %w", b.repo, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return binaryRelease{}, fmt.Errorf("query latest release of %s: GitHub API returned status %d", b.repo, resp.StatusCode)
	}
	var payload struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&payload); err != nil {
		return binaryRelease{}, fmt.Errorf("decode latest release of %s: %w", b.repo, err)
	}
	if !stableReleaseTag.MatchString(payload.TagName) {
		return binaryRelease{}, fmt.Errorf("latest release of %s has an unexpected tag %q", b.repo, payload.TagName)
	}
	rel := binaryRelease{version: payload.TagName, asset: b.name + "_linux_" + runtime.GOARCH}
	for _, a := range payload.Assets {
		switch a.Name {
		case rel.asset:
			rel.binaryURL = a.URL
		case checksumsAssetName:
			rel.checksumsURL = a.URL
		}
	}
	if rel.binaryURL == "" {
		return binaryRelease{}, fmt.Errorf("asset %s not found in release %s of %s", rel.asset, rel.version, b.repo)
	}
	if rel.checksumsURL == "" {
		return binaryRelease{}, fmt.Errorf("%s not found in release %s of %s: there is nothing to verify the binary against", checksumsAssetName, rel.version, b.repo)
	}
	for _, u := range []string{rel.binaryURL, rel.checksumsURL} {
		if err := checkAssetURL(u); err != nil {
			return binaryRelease{}, err
		}
	}
	return rel, nil
}

// checkAssetURL accepts only https URLs on GitHub's download hosts.
func checkAssetURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("download URL must use https, got %q", u.Scheme)
	}
	if u.Host != trustedAssetHost && !strings.HasSuffix(u.Host, trustedAssetSuffix) {
		return fmt.Errorf("untrusted download host: %q", u.Host)
	}
	return nil
}

// assetClient is the client for release assets: every redirect hop is
// checked with checkAssetURL. It has no overall timeout; the caller's
// context bounds the download.
func assetClient() *http.Client {
	return &http.Client{
		Transport: releaseHTTPClient.Transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects downloading %s", via[0].URL)
			}
			return checkAssetURL(req.URL.String())
		},
	}
}

// getAsset starts the download of an asset URL and returns its body.
func getAsset(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := assetClient().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download of %s returned status %d", rawURL, resp.StatusCode)
	}
	return resp.Body, nil
}

// releaseChecksum returns the SHA-256 checksums.txt lists for rel's binary.
func releaseChecksum(ctx context.Context, rel binaryRelease) (string, error) {
	body, err := getAsset(ctx, rel.checksumsURL)
	if err != nil {
		return "", err
	}
	defer func() { _ = body.Close() }()
	scanner := bufio.NewScanner(io.LimitReader(body, maxChecksumsFileLen))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == rel.asset {
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no entry for %s in %s", rel.asset, checksumsAssetName)
}

// fileSHA256 returns the hex SHA-256 of the file at path.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// binaryState reports whether a file exists at path and whether it is the
// one Perci installed (its SHA-256 equals owned.SHA256).
func binaryState(path string, owned BinaryInfo) (exists, ours bool, err error) {
	if _, err := os.Lstat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, false, nil
		}
		return false, false, err
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return true, false, err
	}
	return true, owned.SHA256 != "" && strings.EqualFold(sum, owned.SHA256), nil
}

// installRelease downloads rel's binary next to path, verifies it against
// checksums.txt and renames it over path with mode 0755.
func installRelease(ctx context.Context, stdout io.Writer, rel binaryRelease, path string) (BinaryInfo, error) {
	sum, err := releaseChecksum(ctx, rel)
	if err != nil {
		return BinaryInfo{}, fmt.Errorf("read checksum of %s: %w", rel.asset, err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return BinaryInfo{}, fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return BinaryInfo{}, fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	ui.Info(stdout, i18n.Tr("mcp.binary.downloading", i18n.Args{"asset": rel.asset, "version": rel.version}))
	if err := downloadInto(ctx, rel.binaryURL, tmp); err != nil {
		_ = tmp.Close()
		return BinaryInfo{}, fmt.Errorf("download %s: %w", rel.asset, err)
	}
	if err := tmp.Close(); err != nil {
		return BinaryInfo{}, fmt.Errorf("close temporary file: %w", err)
	}
	if err := fsutil.VerifyFile(tmpPath, sha256.New(), sum); err != nil {
		return BinaryInfo{}, fmt.Errorf("verify %s: %w", rel.asset, err)
	}
	ui.Success(stdout, i18n.Tr("mcp.binary.verified", nil))
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return BinaryInfo{}, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return BinaryInfo{}, fmt.Errorf("install %s: %w", path, err)
	}
	ui.Success(stdout, i18n.Tr("mcp.binary.installed", i18n.Args{"path": path, "version": rel.version}))
	return BinaryInfo{Version: rel.version, SHA256: sum}, nil
}

// downloadInto writes the asset at rawURL into dest, refusing a body
// larger than maxBinaryDownload, and flushes it to disk.
func downloadInto(ctx context.Context, rawURL string, dest *os.File) error {
	body, err := getAsset(ctx, rawURL)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }()
	n, err := io.Copy(dest, io.LimitReader(body, maxBinaryDownload+1))
	if err != nil {
		return err
	}
	if n > maxBinaryDownload {
		return fmt.Errorf("file larger than %d MiB", maxBinaryDownload>>20)
	}
	return dest.Sync()
}

// EnsureBinary makes slug's binary available at BinaryPath. A file already
// there that Perci installed (owned) is kept as is; a file Perci did not
// install is never replaced and the call fails. Otherwise the latest
// release is downloaded. fresh reports whether a binary was installed now.
func EnsureBinary(ctx context.Context, exe *executor.Executor, stdout io.Writer, slug string, owned BinaryInfo) (info BinaryInfo, fresh bool, err error) {
	b, ok := binaryServers[slug]
	if !ok {
		return BinaryInfo{}, false, fmt.Errorf("%s is not installed from a release binary", slug)
	}
	path, err := BinaryPath(slug)
	if err != nil {
		return BinaryInfo{}, false, err
	}
	exists, ours, err := binaryState(path, owned)
	if err != nil {
		return BinaryInfo{}, false, fmt.Errorf("check %s: %w", path, err)
	}
	if exists && ours {
		ui.Info(stdout, i18n.Tr("mcp.binary.present", i18n.Args{"path": path, "version": owned.Version}))
		return owned, false, nil
	}
	if exists {
		return BinaryInfo{}, false, fmt.Errorf("%s: %w", path, errForeignBinary)
	}
	if exe.DryRun {
		ui.Info(stdout, i18n.Tr("mcp.binary.dryRun", i18n.Args{"path": path}))
		return BinaryInfo{}, false, nil
	}
	ctx, cancel := nettimeout.Context(ctx)
	defer cancel()
	rel, err := latestBinaryRelease(ctx, b)
	if err != nil {
		return BinaryInfo{}, false, err
	}
	info, err = installRelease(ctx, stdout, rel, path)
	if err != nil {
		return BinaryInfo{}, false, err
	}
	return info, true, nil
}

// UpdateBinary replaces slug's binary with the latest release when that
// release is newer than owned.Version, or when the binary is missing. A
// file Perci did not install is never replaced. Returns the binary in place
// afterwards.
func UpdateBinary(ctx context.Context, exe *executor.Executor, stdout io.Writer, slug string, owned BinaryInfo) (BinaryInfo, error) {
	b, ok := binaryServers[slug]
	if !ok {
		return BinaryInfo{}, fmt.Errorf("%s is not installed from a release binary", slug)
	}
	path, err := BinaryPath(slug)
	if err != nil {
		return BinaryInfo{}, err
	}
	exists, ours, err := binaryState(path, owned)
	if err != nil {
		return BinaryInfo{}, fmt.Errorf("check %s: %w", path, err)
	}
	if exists && !ours {
		return BinaryInfo{}, fmt.Errorf("%s: %w", path, errForeignBinary)
	}
	if exe.DryRun {
		ui.Info(stdout, i18n.Tr("mcp.binary.dryRun", i18n.Args{"path": path}))
		return owned, nil
	}
	ctx, cancel := nettimeout.Context(ctx)
	defer cancel()
	rel, err := latestBinaryRelease(ctx, b)
	if err != nil {
		return BinaryInfo{}, err
	}
	if exists && !selfupdate.IsNewer(rel.version, owned.Version) {
		ui.Success(stdout, i18n.Tr("mcp.binary.upToDate", i18n.Args{"name": b.name, "version": owned.Version}))
		return owned, nil
	}
	return installRelease(ctx, stdout, rel, path)
}

// RemoveBinary deletes slug's binary when it is the one Perci installed
// (owned); a missing file is fine, and a file Perci did not install is
// kept with a warning.
func RemoveBinary(stdout io.Writer, exe *executor.Executor, slug string, owned BinaryInfo) error {
	path, err := BinaryPath(slug)
	if err != nil {
		return err
	}
	exists, ours, err := binaryState(path, owned)
	if err != nil {
		return fmt.Errorf("check %s: %w", path, err)
	}
	switch {
	case !exists:
		return nil
	case !ours:
		ui.Warning(stdout, i18n.Tr("mcp.binary.keptForeign", i18n.Args{"path": path}))
		return nil
	case exe.DryRun:
		ui.Info(stdout, i18n.Tr("mcp.binary.dryRun", i18n.Args{"path": path}))
		return nil
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	ui.Success(stdout, i18n.Tr("mcp.binary.removed", i18n.Args{"path": path}))
	return nil
}
