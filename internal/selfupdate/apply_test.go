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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/version"
)

// withExecutable makes executablePath return a fake binary (content
// "old") in dir for the duration of the test.
func withExecutable(t *testing.T, dir string) string {
	t.Helper()
	exe := filepath.Join(dir, "perci82-system")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := executablePath
	executablePath = func() (string, error) { return exe, nil }
	t.Cleanup(func() { executablePath = orig })
	return exe
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// serveBinary serves body at /bin and returns the release pointing at it.
func serveBinary(t *testing.T, body []byte, checksum string) *Release {
	t.Helper()
	srv := withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bin" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	})
	return &Release{App: "system", Version: "v9.9.9", DownloadURL: srv.URL + "/bin", ChecksumSHA256: checksum}
}

func dryRunExecutor(buf *bytes.Buffer) *executor.Executor {
	return &executor.Executor{DryRun: true, UsePolicyKit: true, Stdout: buf, Stderr: buf}
}

// No leftover download next to the binary, whatever the outcome.
func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(dir, "perci82-*.new"))
	if len(matches) > 0 {
		t.Errorf("temp files left behind: %v", matches)
	}
}

// A writable binary directory: the new binary is renamed into place, 0755,
// with no privilege escalation.
func TestApply_ReplacesBinaryInWritableDir(t *testing.T) {
	dir := t.TempDir()
	exe := withExecutable(t, dir)
	body := []byte("new-binary")
	rel := serveBinary(t, body, sha256Hex(body))

	var buf bytes.Buffer
	if err := Apply(context.Background(), dryRunExecutor(&buf), &buf, rel); err != nil {
		t.Fatalf("Apply: %v\n%s", err, buf.String())
	}
	got, err := os.ReadFile(exe)
	if err != nil || string(got) != "new-binary" {
		t.Fatalf("binary = %q, %v", got, err)
	}
	if info, _ := os.Stat(exe); info.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, want 0755", info.Mode().Perm())
	}
	if strings.Contains(buf.String(), "pkexec") || strings.Contains(buf.String(), "sudo") {
		t.Errorf("no escalation expected:\n%s", buf.String())
	}
	assertNoTempFiles(t, dir)
}

// A checksum mismatch aborts before anything is replaced.
func TestApply_WrongChecksumKeepsBinary(t *testing.T) {
	dir := t.TempDir()
	exe := withExecutable(t, dir)
	rel := serveBinary(t, []byte("tampered"), sha256Hex([]byte("expected")))

	var buf bytes.Buffer
	err := Apply(context.Background(), dryRunExecutor(&buf), &buf, rel)
	if err == nil || !strings.Contains(err.Error(), "integridade") {
		t.Fatalf("err = %v, want an integrity failure", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Errorf("binary changed to %q", got)
	}
	assertNoTempFiles(t, dir)
}

// A download error (HTTP 404) aborts too.
func TestApply_DownloadErrorKeepsBinary(t *testing.T) {
	dir := t.TempDir()
	exe := withExecutable(t, dir)
	rel := serveBinary(t, nil, "00")
	rel.DownloadURL += "-missing"

	var buf bytes.Buffer
	if err := Apply(context.Background(), dryRunExecutor(&buf), &buf, rel); err == nil {
		t.Fatal("expected a download error")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Errorf("binary changed to %q", got)
	}
}

// A read-only binary directory (e.g. /usr/local/bin): the download goes to
// the system temp dir and the swap is ONE privileged install that
// re-verifies the checksum as root.
func TestApply_ReadOnlyDirUsesOnePrivilegedInstall(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	exe := withExecutable(t, dir)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	body := []byte("new-binary")
	sum := sha256Hex(body)
	rel := serveBinary(t, body, sum)

	var buf bytes.Buffer
	if err := Apply(context.Background(), dryRunExecutor(&buf), &buf, rel); err != nil {
		t.Fatalf("Apply: %v\n%s", err, buf.String())
	}
	out := buf.String()
	if n := strings.Count(out, "pkexec"); n != 1 {
		t.Errorf("want exactly one pkexec call, got %d:\n%s", n, out)
	}
	if !strings.Contains(out, sum) || !strings.Contains(out, exe) {
		t.Errorf("privileged install must carry the checksum and the destination:\n%s", out)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Errorf("dry-run must not touch the binary, got %q", got)
	}
}

// Run applies only a strictly newer release.
func TestRun_OnlyNewerVersionsApply(t *testing.T) {
	origVersion := version.Version
	t.Cleanup(func() { version.Version = origVersion })

	for _, tc := range []struct {
		current string
		applied bool
	}{
		{"1.2.3", false}, // same version
		{"2.0.0", false}, // local build newer than the release
		{"1.0.0", true},
	} {
		t.Run(tc.current, func(t *testing.T) {
			version.Version = tc.current
			dir := t.TempDir()
			exe := withExecutable(t, dir)
			asset := assetName("system")
			body := []byte("new-binary")
			var srvURL string
			srv := withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case releasesPath:
					fmt.Fprint(w, releasesJSON(srvURL,
						release{tag: "system-v1.2.3", assets: []string{asset, "checksums.txt"}},
						release{tag: "dev-v9.0.0", assets: []string{assetName("dev"), "checksums.txt"}},
					))
				case "/system-v1.2.3/checksums.txt":
					fmt.Fprintf(w, "%s  %s\n", sha256Hex(body), asset)
				case "/system-v1.2.3/" + asset:
					_, _ = w.Write(body)
				default:
					http.NotFound(w, r)
				}
			})
			srvURL = srv.URL

			var buf bytes.Buffer
			if err := Run(context.Background(), dryRunExecutor(&buf), &buf, "system"); err != nil {
				t.Fatalf("Run: %v\n%s", err, buf.String())
			}
			got, _ := os.ReadFile(exe)
			if applied := string(got) == "new-binary"; applied != tc.applied {
				t.Errorf("applied = %v, want %v\n%s", applied, tc.applied, buf.String())
			}
		})
	}
}

// withInstallDirs makes installDirs return dirs for the duration of the
// test.
func withInstallDirs(t *testing.T, dirs ...string) {
	t.Helper()
	orig := installDirs
	installDirs = func(string) []string { return dirs }
	t.Cleanup(func() { installDirs = orig })
}

// withMenuPaths makes uninstallMenuPaths return paths for the duration of
// the test.
func withMenuPaths(t *testing.T, paths ...string) {
	t.Helper()
	orig := uninstallMenuPaths
	uninstallMenuPaths = func(string) []string { return paths }
	t.Cleanup(func() { uninstallMenuPaths = orig })
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

// Uninstall removes a user-owned binary without escalating, keeps the
// settings unless asked, and reports ErrUninstalled; root-owned menu files
// go in one privileged batch.
func TestUninstall_UserOwnedBinary(t *testing.T) {
	for _, removeConfig := range []bool{false, true} {
		t.Run(fmt.Sprint("removeConfig=", removeConfig), func(t *testing.T) {
			cfgDir := writeConfig(t, "flatpak_scope: user\n")
			dir := t.TempDir()
			exe := withExecutable(t, dir)
			withInstallDirs(t, dir)

			// Menu files: one the user can delete, and two in a read-only
			// directory that need root.
			userEntry := filepath.Join(t.TempDir(), "io.github.oito2.perci82.system.desktop")
			sysDir := t.TempDir()
			sysIcons := []string{filepath.Join(sysDir, "a.png"), filepath.Join(sysDir, "b.png")}
			for _, p := range append([]string{userEntry}, sysIcons...) {
				if err := os.WriteFile(p, nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Chmod(sysDir, 0o555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(sysDir, 0o755) })
			withMenuPaths(t, append([]string{userEntry}, sysIcons...)...)

			var buf bytes.Buffer
			err := Uninstall(context.Background(), dryRunExecutor(&buf), &buf, "system", UninstallOptions{RemoveConfig: removeConfig})
			if !errors.Is(err, ErrUninstalled) {
				t.Fatalf("err = %v, want ErrUninstalled", err)
			}
			if _, statErr := os.Lstat(exe); !os.IsNotExist(statErr) {
				t.Errorf("%s still exists", exe)
			}
			if _, statErr := os.Stat(cfgDir); os.IsNotExist(statErr) != removeConfig {
				t.Errorf("~/.perci removed = %v, want %v\n%s", os.IsNotExist(statErr), removeConfig, buf.String())
			}
			if _, statErr := os.Stat(userEntry); !os.IsNotExist(statErr) {
				t.Error("a user-removable menu file must be removed directly")
			}
			// The root-owned files go in ONE privileged batch — never one
			// prompt per file (skipped as root, which needs no batch).
			if os.Geteuid() != 0 {
				out := buf.String()
				if n := strings.Count(out, "pkexec"); n != 1 || !strings.Contains(out, sysIcons[0]) || !strings.Contains(out, sysIcons[1]) {
					t.Errorf("want one pkexec call removing both icons, got %d:\n%s", n, out)
				}
			}
		})
	}
}

// "Também remover configurações" removes only the application's own keys
// and data folders; ~/.perci goes only when nothing else needs it.
func TestUninstall_RemoveConfigKeepsWhatOthersNeed(t *testing.T) {
	cases := []struct {
		name      string
		config    string
		installed string // another binary next to the removed one
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
			if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
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
			withMenuPaths(t)

			var buf bytes.Buffer
			opts := UninstallOptions{RemoveConfig: true, DataDirs: []string{"appstack", "git", "../escape", ""}}
			if err := Uninstall(context.Background(), dryRunExecutor(&buf), &buf, "dev", opts); !errors.Is(err, ErrUninstalled) {
				t.Fatalf("err = %v", err)
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

// Binary and menu files that all need root are removed by ONE privileged
// batch.
func TestUninstall_AllPrivilegedInOneBatch(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	exe := withExecutable(t, dir)
	sysDir := t.TempDir()
	icon := filepath.Join(sysDir, "a.png")
	if err := os.WriteFile(icon, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{dir, sysDir} {
		if err := os.Chmod(d, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(d, 0o755) })
	}
	withMenuPaths(t, icon)

	var buf bytes.Buffer
	if err := Uninstall(context.Background(), dryRunExecutor(&buf), &buf, "system", UninstallOptions{}); !errors.Is(err, ErrUninstalled) {
		t.Fatalf("err = %v, want ErrUninstalled", err)
	}
	out := buf.String()
	if n := strings.Count(out, "pkexec"); n != 1 {
		t.Errorf("want exactly one pkexec call, got %d:\n%s", n, out)
	}
	for _, p := range []string{exe, icon} {
		if !strings.Contains(out, p) {
			t.Errorf("batch must remove %s:\n%s", p, out)
		}
	}
}

// Each application removes only its own menu entry and icons.
func TestUninstallMenuPaths_OnlyTheAppsFiles(t *testing.T) {
	paths := uninstallMenuPaths("dev")
	want := []string{
		"/usr/share/applications/io.github.oito2.perci82.dev.desktop",
		"/usr/share/icons/hicolor/512x512/apps/io.github.oito2.perci82.dev.png",
		"/usr/share/icons/hicolor/scalable/apps/io.github.oito2.perci82.dev.svg",
		"/usr/share/icons/hicolor/symbolic/apps/io.github.oito2.perci82.dev-symbolic.svg",
	}
	for _, w := range want {
		if !slices.Contains(paths, w) {
			t.Errorf("%s missing from %v", w, paths)
		}
	}
	for _, p := range paths {
		if !strings.Contains(p, "io.github.oito2.perci82.dev") {
			t.Errorf("%s is not a Perci82 Dev file", p)
		}
	}
}

func TestUninstall_UnknownApp(t *testing.T) {
	dir := t.TempDir()
	exe := withExecutable(t, dir)
	var buf bytes.Buffer
	if err := Uninstall(context.Background(), dryRunExecutor(&buf), &buf, "other", UninstallOptions{}); err == nil || errors.Is(err, ErrUninstalled) {
		t.Fatalf("err = %v, want a plain error", err)
	}
	if _, err := os.Stat(exe); err != nil {
		t.Errorf("binary removed for an unknown app: %v", err)
	}
}

// While the binary needs the privileged batch, a user-writable menu entry
// is not removed directly: canceling the prompt keeps both.
func TestUninstall_MenuWaitsForPrivilegedBinary(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	withExecutable(t, dir)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	entry := filepath.Join(t.TempDir(), "io.github.oito2.perci82.system.desktop")
	if err := os.WriteFile(entry, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	withMenuPaths(t, entry)

	var buf bytes.Buffer
	_ = Uninstall(context.Background(), dryRunExecutor(&buf), &buf, "system", UninstallOptions{})
	if _, err := os.Stat(entry); err != nil {
		t.Errorf("the menu entry was removed directly: %v", err)
	}
	if !strings.Contains(buf.String(), entry) {
		t.Errorf("the menu entry must go in the privileged batch:\n%s", buf.String())
	}
}

// Apply refuses a release with no checksum source, before downloading.
func TestApply_RefusesReleaseWithoutChecksum(t *testing.T) {
	dir := t.TempDir()
	exe := withExecutable(t, dir)
	rel := serveBinary(t, []byte("new-binary"), "")
	var buf bytes.Buffer
	if err := Apply(context.Background(), dryRunExecutor(&buf), &buf, rel); err == nil {
		t.Fatal("Apply must refuse a release without checksum")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Errorf("binary changed: %q", got)
	}
	assertNoTempFiles(t, dir)
}

// A development build is never overwritten by the self-update.
func TestRun_RefusesDevBuild(t *testing.T) {
	origVersion := version.Version
	t.Cleanup(func() { version.Version = origVersion })
	version.Version = "dev"
	dir := t.TempDir()
	exe := withExecutable(t, dir)
	hits := 0
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) { hits++ })

	var buf bytes.Buffer
	if err := Run(context.Background(), dryRunExecutor(&buf), &buf, "system"); err == nil {
		t.Fatal("Run must refuse a dev build")
	}
	if hits != 0 {
		t.Errorf("no request expected, got %d", hits)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Errorf("binary changed: %q", got)
	}
}
