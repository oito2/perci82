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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot is the repository root, relative to this package.
const repoRoot = "../.."

func TestCommittedInstallersUpToDate(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"-check", "-dir", repoRoot}, &stderr); code != 0 {
		t.Errorf("got exit %d: %s", code, stderr.String())
	}
}

func TestRun_WritesThenChecks(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	tpl := "#!/bin/bash\n@GENERATED_NOTICE@\nname=@BINARY@ title='@TITLE@' app=@APP@\n"
	if err := os.WriteFile(filepath.Join(dir, templatePath), []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if code := run([]string{"-check", "-dir", dir}, &stderr); code != 1 {
		t.Fatalf("check before writing got exit %d, want 1", code)
	}
	if code := run([]string{"-dir", dir}, &stderr); code != 0 {
		t.Fatalf("write got exit %d: %s", code, stderr.String())
	}
	got, err := os.ReadFile(filepath.Join(dir, "install-dev.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "#!/bin/bash\n" + generatedNotice + "\nname=perci82-dev title='Perci82 Dev' app=dev\n"; string(got) != want {
		t.Errorf("install-dev.sh got %q, want %q", got, want)
	}
	if fi, err := os.Stat(filepath.Join(dir, "install-system.sh")); err != nil || fi.Mode().Perm() != 0o755 {
		t.Errorf("install-system.sh mode got %v (err %v), want 0755", fi.Mode().Perm(), err)
	}
	stderr.Reset()
	if code := run([]string{"-check", "-dir", dir}, &stderr); code != 0 {
		t.Fatalf("check after writing got exit %d: %s", code, stderr.String())
	}
}

func TestRender_UnknownPlaceholder(t *testing.T) {
	if _, err := render("x=@NOPE@", map[string]string{"APP": "dev"}); err == nil || !strings.Contains(err.Error(), "NOPE") {
		t.Errorf("got %v, want an error naming NOPE", err)
	}
}

// bash runs script with the installer at path sourced first, in a clean
// environment with HOME set to home, and returns its output.
func bash(t *testing.T, path, home, script string) string {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not installed")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "bash", "-c", `source "$1" && `+script, "bash", abs)
	cmd.Env = []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "LC_ALL=C"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash: %v\n%s", err, out)
	}
	return string(out)
}

const releases = `[
	{"tag_name":"dev-v9.0.0","draft":false,"prerelease":false,"assets":[]},
	{"tag_name":"system-v1.2.3","draft":false,"prerelease":false,"assets":[]},
	{"tag_name":"system-v1.2.10","draft":false,"prerelease":false,"assets":[
		{"name":"perci82-system-linux-amd64","browser_download_url":"https://github.com/oito2/perci82/releases/download/system-v1.2.10/perci82-system-linux-amd64"}]},
	{"tag_name":"system-v1.9.0","draft":true,"prerelease":false,"assets":[]},
	{"tag_name":"system-v2.0.0","draft":false,"prerelease":true,"assets":[]},
	{"tag_name":"system-v3.0.0-rc.1","draft":false,"prerelease":false,"assets":[]},
	{"tag_name":"v5.0.0","draft":false,"prerelease":false,"assets":[]}
]`

// The installers pick each application's newest published release and
// read assets and checksums the same way.
func TestInstallers_ReleaseSelection(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not installed")
	}
	home := t.TempDir()
	cases := map[string]string{"install-system.sh": "system-v1.2.10", "install-dev.sh": "dev-v9.0.0"}
	for file, want := range cases {
		t.Run(file, func(t *testing.T) {
			path := filepath.Join(repoRoot, file)
			got := bash(t, path, home, `select_release '`+releases+`' "$TAG_PREFIX" | jq -r .tag_name`)
			if strings.TrimSpace(got) != want {
				t.Errorf("selected %q, want %q", got, want)
			}
		})
	}
}

func TestInstallers_Helpers(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not installed")
	}
	path := filepath.Join(repoRoot, "install-system.sh")
	home := t.TempDir()
	script := `
		r=$(select_release '` + releases + `' system-v)
		printf 'asset=%s\n' "$(asset_url "$r" perci82-system-linux-amd64)"
		printf 'missing=[%s]\n' "$(asset_url "$r" nope)"
		printf 'none=[%s]\n' "$(select_release '[]' system-v)"
		sums=$'abc123  perci82-system-linux-amd64\ndef456 *checksums.txt\n'
		printf 'sum=%s star=%s absent=[%s]\n' "$(checksum_for "$sums" perci82-system-linux-amd64)" "$(checksum_for "$sums" checksums.txt)" "$(checksum_for "$sums" other)"
		for u in https://github.com/x https://objects.githubusercontent.com/y http://github.com/x https://github.com.evil.example/x https://evilgithubusercontent.com/x; do
			if trusted_url "$u"; then printf 'trusted %s\n' "$u"; else printf 'refused %s\n' "$u"; fi
		done
	`
	got := bash(t, path, home, script)
	for _, want := range []string{
		"asset=https://github.com/oito2/perci82/releases/download/system-v1.2.10/perci82-system-linux-amd64",
		"missing=[]",
		"none=[]",
		"sum=abc123 star=def456 absent=[]",
		"trusted https://github.com/x",
		"trusted https://objects.githubusercontent.com/y",
		"refused http://github.com/x",
		"refused https://github.com.evil.example/x",
		"refused https://evilgithubusercontent.com/x",
	} {
		if !strings.Contains(got, want+"\n") {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// The old Perci is found by its binary, its alias only when it is a
// symlink, its menu entry, icons and tray autostart entry.
func TestInstallers_LegacyPerciPaths(t *testing.T) {
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
	path := filepath.Join(repoRoot, "install-dev.sh")

	if got := bash(t, path, home, `legacy_system_paths '`+root+`'; legacy_user_paths "$HOME"`); got != "" {
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

	got := bash(t, path, home, `legacy_system_paths '`+root+`'; legacy_user_paths "$HOME"`)
	want := strings.Join([]string{binary, alias, desktop, pixmap, icon, autostart}, "\n") + "\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}

	// A regular file named perci is someone else's: left alone.
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	write("usr/local/bin/perci")
	if got := bash(t, path, home, `legacy_system_paths '`+root+`'`); strings.Contains(got, alias) {
		t.Errorf("a regular perci file was listed:\n%s", got)
	}
}

// Sourcing an installer defines its functions without installing.
func TestInstallers_SourcingRunsNothing(t *testing.T) {
	for _, file := range []string{"install-system.sh", "install-dev.sh"} {
		got := bash(t, filepath.Join(repoRoot, file), t.TempDir(), `declare -F main >/dev/null && echo defined`)
		if got != "defined\n" {
			t.Errorf("%s: got %q, want only the check's output", file, got)
		}
	}
}

// Without a terminal to ask on, the old Perci is kept and the user is
// told which files to remove.
func TestInstallers_LegacyPerciKeptWithoutTerminal(t *testing.T) {
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
	abs, err := filepath.Abs(filepath.Join(repoRoot, "install-system.sh"))
	if err != nil {
		t.Fatal(err)
	}
	// setsid drops the controlling terminal, so /dev/tty can't be opened.
	cmd := exec.CommandContext(t.Context(), "setsid", "-w", "bash", "-c", `source "$1" && remove_legacy_perci "$2" "$HOME"`, "bash", abs, root)
	cmd.Env = []string{"HOME=" + t.TempDir(), "PATH=" + os.Getenv("PATH"), "LC_ALL=C"}
	cmd.Stdin = nil
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

// The installers check the attestation of checksums.txt, which covers the
// binary and the menu archive, and require it to come from the release
// workflow; the menu archive only ever puts menu entries and icons in /usr.
func TestInstallers_AttestationAndMenuArchive(t *testing.T) {
	for _, file := range []string{"install-system.sh", "install-dev.sh"} {
		t.Run(file, func(t *testing.T) {
			path := filepath.Join(repoRoot, file)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{
				`verify_attestation "$_tmpdir/checksums.txt"`,
				`--wildcards "share/applications/*.desktop" "share/icons/hicolor/*"`,
			} {
				if !strings.Contains(string(data), want) {
					t.Errorf("%s lacks %s", file, want)
				}
			}

			home := t.TempDir()
			bin := t.TempDir()
			log := filepath.Join(home, "gh.log")
			gh := "#!/bin/sh\necho \"$@\" >> " + log + "\nexit 0\n"
			if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(gh), 0o755); err != nil {
				t.Fatal(err)
			}
			if _, err := exec.LookPath("bash"); err != nil {
				t.Skip("bash not installed")
			}
			abs, _ := filepath.Abs(path)
			cmd := exec.CommandContext(t.Context(), "bash", "-c", `source "$1" && verify_attestation /tmp/checksums.txt`, "bash", abs)
			cmd.Env = []string{"HOME=" + home, "PATH=" + bin + ":" + os.Getenv("PATH"), "LC_ALL=C"}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("verify_attestation: %v\n%s", err, out)
			}
			got, err := os.ReadFile(log)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(got), "attestation verify /tmp/checksums.txt --repo oito2/perci82 --signer-workflow oito2/perci82/.github/workflows/release.yml") {
				t.Errorf("gh ran with:\n%s", got)
			}
		})
	}
}
