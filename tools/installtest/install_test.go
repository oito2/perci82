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

// Package installtest tests install.sh, the Perci82 installer at the
// repository root, by running its functions in bash.
package installtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot is the repository root, relative to this package.
const repoRoot = "../.."

// installer is install.sh's absolute path.
func installer(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not installed")
	}
	abs, err := filepath.Abs(filepath.Join(repoRoot, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// env is a clean environment with HOME set to home.
func env(home string) []string {
	return []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "LC_ALL=C"}
}

// bash runs script with install.sh sourced first and returns its output and
// error.
func bash(t *testing.T, home, script string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "bash", append([]string{"-c", `source "$1" && ` + script, "bash", installer(t)}, args...)...)
	cmd.Env = env(home)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// mustBash is bash for scripts that must succeed.
func mustBash(t *testing.T, home, script string, args ...string) string {
	t.Helper()
	out, err := bash(t, home, script, args...)
	if err != nil {
		t.Fatalf("bash: %v\n%s", err, out)
	}
	return out
}

// detached runs install.sh with args and no controlling terminal (setsid),
// so /dev/tty can't be opened.
func detached(t *testing.T, args ...string) (string, error) {
	t.Helper()
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid not installed")
	}
	cmd := exec.CommandContext(t.Context(), "setsid", append([]string{"-w", "bash", installer(t)}, args...)...)
	cmd.Env = env(t.TempDir())
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Sourcing the installer defines its functions without installing.
func TestSourcingRunsNothing(t *testing.T) {
	if got := mustBash(t, t.TempDir(), `declare -F main >/dev/null && echo defined`); got != "defined\n" {
		t.Errorf("got %q, want only the check's output", got)
	}
}

func TestPackagesFor(t *testing.T) {
	for args, want := range map[string]string{
		"system":     "perci82-system\n",
		"dev":        "perci82-dev\n",
		"all":        "perci82-system\nperci82-dev\n",
		"DEV system": "perci82-system\nperci82-dev\n",
		"dev dev":    "perci82-dev\n",
	} {
		got := mustBash(t, t.TempDir(), `packages_for "${@:2}"`, strings.Fields(args)...)
		if got != want {
			t.Errorf("packages_for %s = %q, want %q", args, got, want)
		}
	}
	if out, err := bash(t, t.TempDir(), `packages_for system nope`); err == nil || !strings.Contains(out, "Unknown app 'nope'") {
		t.Errorf("packages_for accepted an unknown app: %v\n%s", err, out)
	}
}

func TestMain_Arguments(t *testing.T) {
	if out, err := detached(t, "--help"); err != nil || !strings.Contains(out, "Usage: install.sh") {
		t.Errorf("--help: %v\n%s", err, out)
	}
	if out, err := detached(t, "nope"); err == nil || !strings.Contains(out, "Unknown app 'nope'") {
		t.Errorf("an unknown app was accepted: %v\n%s", err, out)
	}
	out, err := detached(t)
	if err == nil || !strings.Contains(out, "No terminal to ask on") {
		t.Errorf("no arguments and no terminal: %v\n%s", err, out)
	}
	if strings.Count(out, "Error at line") > 0 {
		t.Errorf("a handled error was reported as a failure:\n%s", out)
	}
}

// The fingerprint the installer requires is the published key's, and a
// file without that key is refused.
func TestVerifyKey(t *testing.T) {
	if _, err := exec.LookPath("gpg"); err != nil {
		t.Skip("gpg not installed")
	}
	key, err := filepath.Abs(filepath.Join(repoRoot, "packaging", "repo", "perci82.asc"))
	if err != nil {
		t.Fatal(err)
	}
	if out := mustBash(t, t.TempDir(), `verify_key "$2" && echo ok`, key); out != "ok\n" {
		t.Errorf("the published key was refused: %q", out)
	}
	other := filepath.Join(t.TempDir(), "other.asc")
	if err := os.WriteFile(other, []byte("not a key\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := bash(t, t.TempDir(), `verify_key "$2"`, other); err == nil {
		t.Errorf("a file without the key was accepted:\n%s", out)
	}
}

// The repository files the installer writes point at the site and key
// paths that scripts/build-pages.sh publishes.
func TestRepositoryFilesMatchTheSite(t *testing.T) {
	site, err := os.ReadFile(filepath.Join(repoRoot, "scripts", "build-pages.sh"))
	if err != nil {
		t.Fatal(err)
	}
	install, err := os.ReadFile(filepath.Join(repoRoot, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"APT_KEYRING", "RPM_KEY"} {
		re := regexp.MustCompile(`(?m)^readonly ` + name + `=.*$`)
		if a, b := re.Find(install), re.Find(site); a == nil || string(a) != string(b) {
			t.Errorf("%s differs: install.sh %q, build-pages.sh %q", name, a, b)
		}
	}
	if !strings.Contains(string(site), `SITE_URL="${SITE_URL:-https://oito2.github.io/perci82}"`) ||
		!strings.Contains(string(install), `readonly SITE_URL="https://oito2.github.io/perci82"`) {
		t.Error("the installer and the site use different site addresses")
	}

	apt := mustBash(t, t.TempDir(), `apt_sources_content`)
	for _, want := range []string{"URIs: https://oito2.github.io/perci82/apt\n", "Suites: stable\n", "Components: main\n", "Signed-By: /usr/share/keyrings/perci82-archive-keyring.asc\n"} {
		if !strings.Contains(apt, want) {
			t.Errorf("apt source lacks %q:\n%s", want, apt)
		}
	}
	dnf := mustBash(t, t.TempDir(), `dnf_repo_content`)
	for _, want := range []string{"baseurl=https://oito2.github.io/perci82/rpm\n", "gpgcheck=1\n", "repo_gpgcheck=1\n", "gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-perci82\n"} {
		if !strings.Contains(dnf, want) {
			t.Errorf("dnf repository lacks %q:\n%s", want, dnf)
		}
	}
}

// The old Perci is found by its binary, its alias only when it is a
// symlink, its menu entry, icons and tray autostart entry.
func TestLegacyPerciPaths(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	write := func(rel string) string {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	list := `legacy_system_paths "$2"; legacy_user_paths "$HOME"`

	if got := mustBash(t, home, list, root); got != "" {
		t.Fatalf("nothing installed, got %q", got)
	}

	binary := write("usr/local/bin/prci")
	alias := filepath.Join(root, "usr/local/bin/perci")
	if err := os.Symlink("prci", alias); err != nil {
		t.Fatal(err)
	}
	desktop := write("usr/share/applications/perci.desktop")
	pixmap := write("usr/share/pixmaps/perci.png")
	icon := write("usr/share/icons/hicolor/64x64/apps/perci.png")
	write("usr/share/icons/hicolor/64x64/apps/io.github.oito2.perci82.dev.png")
	write("usr/local/bin/perci82-dev")
	autostart := filepath.Join(home, ".config/autostart/perci.desktop")
	if err := os.MkdirAll(filepath.Dir(autostart), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(autostart, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	got := mustBash(t, home, list, root)
	want := strings.Join([]string{binary, alias, desktop, pixmap, icon, autostart}, "\n") + "\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}

	// A regular file named perci is someone else's: left alone.
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	write("usr/local/bin/perci")
	if got := mustBash(t, home, `legacy_system_paths "$2"`, root); strings.Contains(got, alias) {
		t.Errorf("a regular perci file was listed:\n%s", got)
	}
}

// Without a terminal to ask on, the old Perci is kept and the user is
// told which files to remove.
func TestLegacyPerciKeptWithoutTerminal(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid not installed")
	}
	root := t.TempDir()
	binary := filepath.Join(root, "usr/local/bin/prci")
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "setsid", "-w", "bash", "-c", `source "$1" && remove_legacy_perci "$2" "$HOME"`, "bash", installer(t), root)
	cmd.Env = env(t.TempDir())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("remove_legacy_perci: %v\n%s", err, out)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Errorf("the old Perci was removed without asking: %v", err)
	}
	if !strings.Contains(string(out), binary) || !strings.Contains(string(out), "No terminal to ask on") {
		t.Errorf("output does not list the kept files:\n%s", out)
	}
}
