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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTestServer points apiBaseURL/httpClient/trustedDownloadHost at an
// httptest.TLSServer for the duration of the test, restoring the real
// values afterward. The server itself is configured by the caller.
func withTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)

	origAPIBase, origClient, origHost := apiBaseURL, httpClient, trustedDownloadHost
	host, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}

	apiBaseURL = srv.URL
	httpClient = srv.Client()
	trustedDownloadHost = host.Host

	t.Cleanup(func() {
		apiBaseURL, httpClient, trustedDownloadHost = origAPIBase, origClient, origHost
	})
	return srv
}

// release is one entry of the test release list.
type release struct {
	tag               string
	draft, prerelease bool
	assets            []string // asset names, served under the server URL
}

// releasesJSON renders releases as the GitHub API's release list, with
// every asset URL under base.
func releasesJSON(base string, releases ...release) string {
	var parts []string
	for _, r := range releases {
		var assets []string
		for _, a := range r.assets {
			assets = append(assets, fmt.Sprintf(`{"name":%q,"browser_download_url":%q}`, a, base+"/"+r.tag+"/"+a))
		}
		parts = append(parts, fmt.Sprintf(`{"tag_name":%q,"draft":%v,"prerelease":%v,"assets":[%s]}`,
			r.tag, r.draft, r.prerelease, strings.Join(assets, ",")))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

const releasesPath = "/repos/oito2/perci82/releases"

// Among the monorepo's releases, each application gets its own highest
// published version: other applications' tags, drafts, prereleases and
// malformed tags are skipped, and versions compare numerically.
func TestLatestRelease_PicksTheAppsNewestRelease(t *testing.T) {
	sysAsset, devAsset := assetName("system"), assetName("dev")
	binContent := []byte("fake-binary-content")
	wantChecksum := sha256Hex(binContent)

	var srv *httptest.Server
	srv = withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case releasesPath:
			if r.URL.Query().Get("per_page") != "100" {
				t.Errorf("query = %q, want per_page=100", r.URL.RawQuery)
			}
			fmt.Fprint(w, releasesJSON(srv.URL,
				release{tag: "dev-v9.0.0", assets: []string{devAsset, "checksums.txt"}},
				release{tag: "system-v1.2.3", assets: []string{sysAsset, "checksums.txt"}},
				release{tag: "system-v1.2.10", assets: []string{sysAsset, "checksums.txt"}},
				release{tag: "system-v1.9.0", draft: true, assets: []string{sysAsset, "checksums.txt"}},
				release{tag: "system-v2.0.0", prerelease: true, assets: []string{sysAsset, "checksums.txt"}},
				release{tag: "system-v3.0.0-rc.1", assets: []string{sysAsset, "checksums.txt"}},
				release{tag: "system-v4.0", assets: []string{sysAsset, "checksums.txt"}},
				release{tag: "v5.0.0", assets: []string{sysAsset, "checksums.txt"}},
			))
		case "/system-v1.2.10/checksums.txt":
			fmt.Fprintf(w, "%s  %s\ndeadbeef  some-other-file\n", wantChecksum, sysAsset)
		default:
			http.NotFound(w, r)
		}
	})

	rel, err := LatestRelease(context.Background(), "system")
	if err != nil {
		t.Fatalf("LatestRelease(system): %v", err)
	}
	if rel.App != "system" || rel.Version != "v1.2.10" {
		t.Errorf("got %s %s, want system v1.2.10", rel.App, rel.Version)
	}
	if want := srv.URL + "/system-v1.2.10/" + sysAsset; rel.DownloadURL != want {
		t.Errorf("DownloadURL = %q, want %q", rel.DownloadURL, want)
	}
	if rel.ChecksumSHA256 != "" {
		t.Errorf("LatestRelease must not resolve the checksum, got %q", rel.ChecksumSHA256)
	}
	if err := resolveChecksum(context.Background(), rel); err != nil {
		t.Fatalf("resolveChecksum: %v", err)
	}
	if rel.ChecksumSHA256 != wantChecksum {
		t.Errorf("ChecksumSHA256 = %q, want %q", rel.ChecksumSHA256, wantChecksum)
	}

	dev, err := LatestRelease(context.Background(), "dev")
	if err != nil || dev.Version != "v9.0.0" || dev.DownloadURL != srv.URL+"/dev-v9.0.0/"+devAsset {
		t.Errorf("LatestRelease(dev) got %+v, %v", dev, err)
	}
}

// An application without a published release gets a clear error.
func TestLatestRelease_NoReleaseOfTheApp(t *testing.T) {
	var srv *httptest.Server
	srv = withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, releasesJSON(srv.URL, release{tag: "dev-v1.0.0", assets: []string{assetName("dev")}}))
	})
	_, err := LatestRelease(context.Background(), "system")
	if err == nil || !strings.Contains(err.Error(), "Perci82 System") {
		t.Errorf("got %v, want an error naming Perci82 System", err)
	}
}

func TestLatestRelease_UnknownApp(t *testing.T) {
	hits := 0
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) { hits++ })
	if _, err := LatestRelease(context.Background(), "other"); err == nil || hits != 0 {
		t.Errorf("got err=%v with %d requests, want an error and no request", err, hits)
	}
}

// A plain version check never requests checksums.txt.
func TestLatestRelease_DoesNotDownloadChecksums(t *testing.T) {
	var srv *httptest.Server
	checksumHits := 0
	srv = withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == releasesPath {
			fmt.Fprint(w, releasesJSON(srv.URL, release{tag: "dev-v1.2.3", assets: []string{assetName("dev"), "checksums.txt"}}))
			return
		}
		checksumHits++
	})
	if _, err := LatestRelease(context.Background(), "dev"); err != nil {
		t.Fatal(err)
	}
	if checksumHits != 0 {
		t.Errorf("checksums.txt requested %d times, want 0", checksumHits)
	}
}

func TestLatestRelease_MissingAsset(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, releasesJSON("https://example.invalid", release{tag: "system-v1.2.3", assets: []string{"perci82-system-linux-riscv64"}}))
	})
	if _, err := LatestRelease(context.Background(), "system"); err == nil {
		t.Error("a release without the running architecture's asset must fail")
	}
}

// A release without checksums.txt is still reported (the version check
// works); only applying it is refused.
func TestLatestRelease_MissingChecksumsFileStillReported(t *testing.T) {
	var srv *httptest.Server
	srv = withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, releasesJSON(srv.URL, release{tag: "system-v1.2.3", assets: []string{assetName("system")}}))
	})

	rel, err := LatestRelease(context.Background(), "system")
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.Version != "v1.2.3" || rel.ChecksumsURL != "" {
		t.Errorf("rel = %+v", rel)
	}
	if err := resolveChecksum(context.Background(), rel); err == nil {
		t.Fatal("resolving the checksum of a release without checksums.txt must fail")
	}
}

func TestResolveChecksum_NotListedForAsset(t *testing.T) {
	srv := withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/checksums.txt" {
			fmt.Fprint(w, "deadbeef  algum-outro-arquivo\n")
		}
	})
	rel := &Release{App: "system", Version: "v1.2.3", ChecksumsURL: srv.URL + "/checksums.txt"}
	if err := resolveChecksum(context.Background(), rel); err == nil {
		t.Error("esperava erro por checksum do asset não estar listado em checksums.txt, mas passou")
	}
}

func TestLatestRelease_APIError(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if _, err := LatestRelease(context.Background(), "system"); err == nil {
		t.Error("esperava erro por status 500 da API, mas passou")
	}
}

func TestFetchChecksum(t *testing.T) {
	sum := sha256.Sum256([]byte("conteudo-amd64"))
	wantHash := hex.EncodeToString(sum[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  prci-linux-amd64\ndeadbeef  prci-linux-arm64\n", wantHash)
	}))
	defer srv.Close()

	got, err := fetchChecksum(context.Background(), srv.URL, "prci-linux-amd64")
	if err != nil {
		t.Fatalf("fetchChecksum: %v", err)
	}
	if got != wantHash {
		t.Errorf("fetchChecksum() = %q, want %q", got, wantHash)
	}

	if _, err := fetchChecksum(context.Background(), srv.URL, "prci-linux-inexistente"); err == nil {
		t.Error("esperava erro para asset não listado, mas passou")
	}
}

func TestVerifyChecksum(t *testing.T) {
	content := []byte("binario-de-teste")
	sum := sha256.Sum256(content)
	goodHash := hex.EncodeToString(sum[:])

	tmp, err := os.CreateTemp(t.TempDir(), "checksum-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}

	if err := verifyChecksum(tmp.Name(), goodHash); err != nil {
		t.Errorf("esperava verificação OK com hash correto, deu erro: %v", err)
	}

	if err := verifyChecksum(tmp.Name(), "0000000000000000000000000000000000000000000000000000000000ff"); err == nil {
		t.Error("esperava erro com hash adulterado, mas passou")
	}

	if err := verifyChecksum(filepath.Join(t.TempDir(), "nao-existe"), goodHash); err == nil {
		t.Error("esperava erro ao verificar arquivo inexistente, mas passou")
	}
}

func TestValidateDownloadURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"github release asset", "https://github.com/oito2/perci82/releases/download/v1.0.0/prci-linux-amd64", false},
		{"githubusercontent cdn", "https://objects.githubusercontent.com/foo/bar", false},
		{"http scheme rejected", "http://github.com/oito2/perci82/releases/download/v1.0.0/prci-linux-amd64", true},
		{"untrusted host rejected", "https://evil.example.com/prci-linux-amd64", true},
		{"lookalike host rejected", "https://github.com.evil.com/x", true},
		{"malformed url rejected", "://not a url", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDownloadURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDownloadURL(%q) err=%v, wantErr=%v", tt.url, err, tt.wantErr)
			}
		})
	}
}

// A redirect to a host outside the trusted set must be refused on the hop
// itself, not only when the first URL is validated.
func TestDownloadFile_RejectsRedirectToUntrustedHost(t *testing.T) {
	srv := withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example.com/prci", http.StatusFound)
	})
	f, err := os.CreateTemp(t.TempDir(), "dl-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	err = downloadFile(context.Background(), srv.URL+"/prci", f)
	if err == nil || !strings.Contains(err.Error(), "não confiável") {
		t.Fatalf("expected an untrusted-host error, got %v", err)
	}
}

func TestDownloadFile_RefusesOversizedBody(t *testing.T) {
	srv := withTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		chunk := make([]byte, 1<<20)
		for i := 0; i <= maxBinarySize>>20; i++ {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	})
	f, err := os.CreateTemp(t.TempDir(), "dl-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	err = downloadFile(context.Background(), srv.URL+"/prci", f)
	if err == nil || !strings.Contains(err.Error(), "maior que") {
		t.Fatalf("expected an explicit size error, got %v", err)
	}
}

func TestVerifyChecksum_CaseInsensitive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bin")
	content := []byte("perci")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	if err := verifyChecksum(path, " "+strings.ToUpper(hex.EncodeToString(sum[:]))+"\n"); err != nil {
		t.Fatalf("uppercase/whitespace checksum should match: %v", err)
	}
}
