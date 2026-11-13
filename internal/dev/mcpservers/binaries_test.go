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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/oito2/perci82/internal/executor"
)

// releaseServer is a fake GitHub API + asset host for one repository's
// latest release: tag, binary content and the checksum listed for it.
type releaseServer struct {
	srv      *httptest.Server
	tag      string
	body     []byte
	checksum string
	hits     atomic.Int32
}

// newReleaseServer starts a TLS server answering releases/latest for
// oito2/mcp-build82 with tag, serving body as the binary and a
// checksums.txt with checksum (the real SHA-256 of body when empty), and
// points the package's release variables at it.
func newReleaseServer(t *testing.T, tag string, body []byte, checksum string) *releaseServer {
	t.Helper()
	if checksum == "" {
		sum := sha256.Sum256(body)
		checksum = hex.EncodeToString(sum[:])
	}
	rs := &releaseServer{tag: tag, body: body, checksum: checksum}
	asset := "build82_linux_" + runtime.GOARCH
	rs.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rs.hits.Add(1)
		base := "https://" + r.Host
		switch r.URL.Path {
		case "/repos/oito2/mcp-build82/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name": rs.tag,
				"assets": []map[string]string{
					{"name": asset, "browser_download_url": base + "/dl/" + asset},
					{"name": "checksums.txt", "browser_download_url": base + "/dl/checksums.txt"},
				},
			})
		case "/dl/" + asset:
			_, _ = w.Write(rs.body)
		case "/dl/checksums.txt":
			_, _ = io.WriteString(w, rs.checksum+"  "+asset+"\n0000  other_asset\n")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(rs.srv.Close)
	u, _ := url.Parse(rs.srv.URL)
	prevAPI, prevHost, prevClient := releaseAPIBase, trustedAssetHost, releaseHTTPClient
	releaseAPIBase, trustedAssetHost, releaseHTTPClient = rs.srv.URL, u.Host, rs.srv.Client()
	t.Cleanup(func() { releaseAPIBase, trustedAssetHost, releaseHTTPClient = prevAPI, prevHost, prevClient })
	return rs
}

func sumOf(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func binaryHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return filepath.Join(home, ".local", "bin", "build82")
}

func TestEnsureBinary_InstallsVerifiedLatestRelease(t *testing.T) {
	path := binaryHome(t)
	newReleaseServer(t, "v1.3.0", []byte("binary v1.3.0"), "")
	var out strings.Builder
	info, fresh, err := EnsureBinary(t.Context(), executor.New(nil, nil), &out, "build82", BinaryInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if !fresh || info.Version != "v1.3.0" || info.SHA256 != sumOf([]byte("binary v1.3.0")) {
		t.Errorf("got info %+v fresh %v", info, fresh)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, want 0755", st.Mode().Perm())
	}
	if data, _ := os.ReadFile(path); string(data) != "binary v1.3.0" {
		t.Errorf("content = %q", data)
	}
}

func TestEnsureBinary_ChecksumMismatchLeavesNothing(t *testing.T) {
	path := binaryHome(t)
	newReleaseServer(t, "v1.3.0", []byte("tampered"), sumOf([]byte("original")))
	_, _, err := EnsureBinary(t.Context(), executor.New(nil, nil), io.Discard, "build82", BinaryInfo{})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v, want a checksum mismatch", err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 0 {
		t.Errorf("left behind: %v", entries)
	}
}

func TestEnsureBinary_KeepsForeignAndReusesOwned(t *testing.T) {
	path := binaryHome(t)
	rs := newReleaseServer(t, "v1.3.0", []byte("new"), "")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("installed by hand"), 0o755); err != nil {
		t.Fatal(err)
	}
	exe := executor.New(nil, nil)

	_, _, err := EnsureBinary(t.Context(), exe, io.Discard, "build82", BinaryInfo{})
	if !errors.Is(err, errForeignBinary) {
		t.Fatalf("err = %v, want errForeignBinary", err)
	}
	if data, _ := os.ReadFile(path); string(data) != "installed by hand" {
		t.Errorf("a foreign binary was changed: %q", data)
	}

	owned := BinaryInfo{Version: "v1.2.0", SHA256: sumOf([]byte("installed by hand"))}
	info, fresh, err := EnsureBinary(t.Context(), exe, io.Discard, "build82", owned)
	if err != nil || fresh || info != owned {
		t.Errorf("owned binary: info %+v fresh %v err %v", info, fresh, err)
	}
	if rs.hits.Load() != 0 {
		t.Errorf("the network was queried %d times", rs.hits.Load())
	}
}

func TestUpdateBinary(t *testing.T) {
	path := binaryHome(t)
	rs := newReleaseServer(t, "v1.3.0", []byte("v1.3.0 build"), "")
	exe := executor.New(nil, nil)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("v1.2.0 build"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := BinaryInfo{Version: "v1.2.0", SHA256: sumOf([]byte("v1.2.0 build"))}

	info, err := UpdateBinary(t.Context(), exe, io.Discard, "build82", old)
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "v1.3.0" || info.SHA256 != sumOf([]byte("v1.3.0 build")) {
		t.Errorf("info = %+v", info)
	}

	// Same version: nothing is downloaded again.
	rs.body = []byte("must not be downloaded")
	again, err := UpdateBinary(t.Context(), exe, io.Discard, "build82", info)
	if err != nil || again != info {
		t.Errorf("up to date: info %+v err %v", again, err)
	}
	if data, _ := os.ReadFile(path); string(data) != "v1.3.0 build" {
		t.Errorf("content = %q", data)
	}

	// A file Perci did not install is never replaced.
	if _, err := UpdateBinary(t.Context(), exe, io.Discard, "build82", old); !errors.Is(err, errForeignBinary) {
		t.Errorf("err = %v, want errForeignBinary", err)
	}
}

func TestRemoveBinary(t *testing.T) {
	path := binaryHome(t)
	exe := executor.New(nil, nil)
	if err := RemoveBinary(io.Discard, exe, "build82", BinaryInfo{}); err != nil {
		t.Errorf("missing binary: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RemoveBinary(io.Discard, exe, "build82", BinaryInfo{Version: "v1.0.0", SHA256: sumOf([]byte("other"))}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("a foreign binary was removed: %v", err)
	}
	if err := RemoveBinary(io.Discard, exe, "build82", BinaryInfo{Version: "v1.0.0", SHA256: sumOf([]byte("bin"))}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("owned binary still there: %v", err)
	}
}

func TestLatestBinaryRelease_Refusals(t *testing.T) {
	binaryHome(t)
	b := binaryServers["build82"]

	rs := newReleaseServer(t, "v1.4.0-rc1", []byte("x"), "")
	if _, err := latestBinaryRelease(t.Context(), b); err == nil {
		t.Error("a prerelease tag was accepted")
	}

	rs.tag = "v1.4.0"
	trustedAssetHost = "github.com"
	if _, err := latestBinaryRelease(t.Context(), b); err == nil || !strings.Contains(err.Error(), "untrusted download host") {
		t.Errorf("err = %v, want an untrusted host", err)
	}
}

func TestCreatedByPerci_Binary(t *testing.T) {
	path := binaryHome(t)
	if !createdByPerci("sync82", strings.Replace(path, "build82", "sync82", 1), nil) {
		t.Error("the installed binary path with no arguments is Perci's")
	}
	if createdByPerci("build82", path, []string{"--x"}) {
		t.Error("an entry with arguments is not Perci's")
	}
	if createdByPerci("build82", "/home/u/go/bin/build82", nil) {
		t.Error("another binary path is not Perci's")
	}
}

// Install registers a release binary by its path, with no arguments, and
// Remove leaves an entry another tool registered under the same name.
func TestInstallBinary_RegistersThePath(t *testing.T) {
	env := newAgentsEnv(t)
	path := filepath.Join(env.home, ".local", "bin", "build82")
	exe := executor.New(nil, nil)

	agents, err := Install(t.Context(), exe, io.Discard, "build82", true, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 3 {
		t.Errorf("agents = %v", agents)
	}
	log, _ := os.ReadFile(env.log)
	if !strings.Contains(string(log), "mcp add --scope user build82 -- "+path+"\n") ||
		!strings.Contains(string(log), "mcp add build82 -- "+path+"\n") {
		t.Errorf("calls:\n%s", log)
	}
	doc, err := readJSONObject(filepath.Join(env.home, ".gemini", "config", "mcp_config.json"))
	if err != nil {
		t.Fatal(err)
	}
	entry := doc["mcpServers"].(map[string]any)["build82"].(map[string]any)
	if entry["command"] != path || len(entry["args"].([]any)) != 0 {
		t.Errorf("antigravity entry = %v", entry)
	}

	env.registerEverywhere(t, "sync82", true, "", "/home/u/go/bin/sync82")
	if _, err := Install(t.Context(), exe, io.Discard, "sync82", true, "", ""); err != nil {
		t.Logf("install over foreign entries: %v", err)
	}
	if log, _ := os.ReadFile(env.log); strings.Contains(string(log), "mcp add --scope user sync82") {
		t.Error("a foreign Claude Code entry was replaced")
	}
}
