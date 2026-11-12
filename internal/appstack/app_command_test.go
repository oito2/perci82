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

package appstack

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oito2/perci82/internal/config"
)

// runAppCommand writes app's command for workspace into a folder with a
// fake docker that prints its arguments one per line, runs it from dir
// with args, and returns docker's arguments, stderr and the exit code.
func runAppCommand(t *testing.T, app config.AppContainer, workspace, dir string, args ...string) (dockerArgs []string, stderr string, code int) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not installed")
	}
	bin := t.TempDir()
	writeFile(t, filepath.Join(bin, "docker"), "#!/bin/sh\nprintf '%s\\n' \"$@\"\n", 0o755)
	script := filepath.Join(bin, app.Folder)
	writeFile(t, script, buildAppCommandScript(appCommandSpecFor(app, workspace)), 0o755)
	cmd := exec.Command(script, args...)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + bin + ":/usr/bin:/bin", "PWD=" + dir}
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(out.String(), "\n"), "\n"), errOut.String(), code
}

// projectDir creates and returns folder inside the project root of app.
func projectDir(t *testing.T, root string, parts ...string) string {
	t.Helper()
	dir := filepath.Join(append([]string{root}, parts...)...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAppCommand_PHPApp(t *testing.T) {
	ws, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := AppHTMLDir(ws, "loja")
	sub := projectDir(t, root, "modules", "pagamento")

	got, _, code := runAppCommand(t, lojaApp, ws, sub, "composer", "install")
	want := []string{"exec", "-i", "-w", "/var/www/html/modules/pagamento", "-u", "www-data", "-e", "HOME=/tmp", "loja", "composer", "install"}
	if code != 0 || strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("composer inside a subfolder:\n got %q\nwant %q", got, want)
	}

	// Anything else runs through php, with host paths into the app
	// rewritten.
	got, _, _ = runAppCommand(t, lojaApp, ws, root, filepath.Join(root, "teste.php"), "/etc/hosts")
	if strings.Join(got[len(got)-3:], " ") != "php /var/www/html/teste.php /etc/hosts" {
		t.Errorf("php fallback = %q", got)
	}

	// Outside the app the command runs at the project root and says so.
	got, stderr, _ := runAppCommand(t, lojaApp, ws, ws, "-r", "echo 1;")
	if got[3] != "/var/www/html" || !strings.Contains(stderr, "fora do app") {
		t.Errorf("outside the app: args %q, stderr %q", got, stderr)
	}

	// phpcs without moodle-cs gets no extra standards.
	got, _, _ = runAppCommand(t, lojaApp, ws, root, "phpcs", ".")
	if strings.Contains(strings.Join(got, " "), "installed_paths") {
		t.Errorf("phpcs of a PHP app registered moodle-cs: %q", got)
	}
}

func TestAppCommand_ChecklistTurnsToolsOff(t *testing.T) {
	ws := t.TempDir()
	app := lojaApp
	app.DisabledTools = []string{ToolPHPCS}
	root := projectDir(t, AppHTMLDir(ws, "loja"))
	for _, sub := range []string{"phpcs", "phpcbf"} {
		got, stderr, code := runAppCommand(t, app, ws, root, sub, ".")
		if code != 2 || !strings.Contains(stderr, sub+": ferramenta não ativada neste app (loja)") || got[0] != "" {
			t.Errorf("%s turned off: code %d, stderr %q, docker %q", sub, code, stderr, got)
		}
	}
	if got, _, code := runAppCommand(t, app, ws, root, "composer"); code != 0 || got[len(got)-1] != "composer" {
		t.Errorf("composer stays on: %q", got)
	}
}

func TestAppCommand_MoodleCSRegistersTheStandards(t *testing.T) {
	ws := t.TempDir()
	app := config.AppContainer{Folder: "mdle", Type: config.AppTypeMoodle}
	root := projectDir(t, AppHTMLDir(ws, "mdle"))
	got, _, _ := runAppCommand(t, app, ws, root, "phpcs", "--standard=moodle", "local/x")
	want := "mdle phpcs --runtime-set installed_paths " + moodleCSInstalledPaths + " --standard=moodle local/x"
	if !strings.HasSuffix(strings.Join(got, " "), want) {
		t.Errorf("moodle phpcs = %q", got)
	}
	app.DisabledTools = []string{ToolMoodleCS}
	got, _, _ = runAppCommand(t, app, ws, root, "phpcbf", ".")
	if strings.Contains(strings.Join(got, " "), "installed_paths") {
		t.Errorf("moodle-cs off still registers the standards: %q", got)
	}
}

func TestAppCommand_NodeAndComboApps(t *testing.T) {
	ws := t.TempDir()
	node := config.AppContainer{Folder: "site", Type: config.AppTypeNode}
	got, _, _ := runAppCommand(t, node, ws, projectDir(t, AppHTMLDir(ws, "site"), "src"), "server.js")
	if strings.Join(got, " ") != "exec -i -w /app/src site node server.js" {
		t.Errorf("node fallback = %q", got)
	}
	if got, _, _ = runAppCommand(t, node, ws, AppHTMLDir(ws, "site"), "npm", "test"); strings.Join(got[len(got)-2:], " ") != "npm test" {
		t.Errorf("node npm = %q", got)
	}

	combo := config.AppContainer{Folder: "painel", Type: config.AppTypePHPNode}
	got, _, _ = runAppCommand(t, combo, ws, projectDir(t, AppComboAppDir(ws, "painel"), "web"), "npm", "run", "dev")
	if got[3] != "/app/web" || strings.Join(got[len(got)-3:], " ") != "npm run dev" {
		t.Errorf("combo npm = %q", got)
	}
	got, _, _ = runAppCommand(t, combo, ws, projectDir(t, AppComboAPIDir(ws, "painel"), "src"), "phpunit")
	if got[3] != "/var/www/html/src" || got[len(got)-1] != "phpunit" {
		t.Errorf("combo phpunit = %q", got)
	}
}

func TestInstallAppCommand_ReplacesTheOldWrappers(t *testing.T) {
	ws := stackHome(t, lojaApp)
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() { lookPath = exec.LookPath })
	bin := filepath.Join(os.Getenv("HOME"), ".local", "bin")
	var out bytes.Buffer
	writeAppWrappers("loja", AppHTMLDir(ws, "loja"), &out)
	if !HasLegacyWrappers("loja") {
		t.Fatal("the old wrappers were not detected")
	}
	if err := MigrateAppCommand(&out, "loja"); err != nil {
		t.Fatal(err)
	}
	if HasLegacyWrappers("loja") {
		t.Error("the old wrappers survived the migration")
	}
	if b, _ := os.ReadFile(filepath.Join(bin, "loja")); !strings.Contains(string(b), appCommandMarker) {
		t.Errorf("the command was not written:\n%s", b)
	}
	removeAppCommand("loja")
	if _, err := os.Stat(filepath.Join(bin, "loja")); !os.IsNotExist(err) {
		t.Error("removeAppCommand kept the command")
	}
}

func TestInstallAppCommand_NameTaken(t *testing.T) {
	ws := stackHome(t, lojaApp)
	bin := filepath.Join(os.Getenv("HOME"), ".local", "bin")

	// Another program called like the app, elsewhere on PATH.
	lookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	t.Cleanup(func() { lookPath = exec.LookPath })
	var out bytes.Buffer
	if installAppCommand(lojaApp, ws, &out) {
		t.Fatal("the command was written over another program's name")
	}
	if _, err := os.Stat(filepath.Join(bin, "loja")); !os.IsNotExist(err) {
		t.Error("~/.local/bin/loja was created")
	}
	if !HasLegacyWrappers("loja") || !strings.Contains(out.String(), "/usr/bin/loja") {
		t.Errorf("the old wrappers were not kept, or no warning:\n%s", out.String())
	}

	// A file of the user's own in ~/.local/bin is never overwritten.
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	writeFile(t, filepath.Join(bin, "loja"), "#!/bin/sh\necho mine\n", 0o755)
	if installAppCommand(lojaApp, ws, &out) {
		t.Fatal("the user's own file was replaced")
	}
	if b, _ := os.ReadFile(filepath.Join(bin, "loja")); string(b) != "#!/bin/sh\necho mine\n" {
		t.Errorf("the user's file changed: %q", b)
	}
}

func TestValidateApp_Tools(t *testing.T) {
	moodle := config.AppContainer{Folder: "mdle", URL: "mdle.localhost", Type: config.AppTypeMoodle, PHPVersion: "8.3", MoodleVersion: "5.1+"}
	moodle.DisabledTools = []string{ToolMoodleCS, ToolComposer}
	if err := ValidateApp(moodle); err != nil {
		t.Errorf("valid Moodle tools refused: %v", err)
	}
	php := lojaApp
	php.DisabledTools = []string{ToolMoodleCS}
	if err := ValidateApp(php); err == nil {
		t.Error("moodle-cs accepted for a PHP app")
	}
	node := config.AppContainer{Folder: "site", URL: "site.localhost", Type: config.AppTypeNode, NodeVersion: "24", DevCommand: "npm run dev", DevPort: 3000, DisabledTools: []string{ToolComposer}}
	if err := ValidateApp(node); err == nil {
		t.Error("a PHP tool accepted for a Node app")
	}
}

func TestPHPImage_PinsTheMoodleStandards(t *testing.T) {
	for _, want := range []string{
		"moodle-cs/archive/refs/tags/v3.7.0.tar.gz\" 9e4c4c111137f6630f97f82e37b348d2f2a1ed8db927c1ca5e7edb625e16fc6b moodle-cs",
		"PHPCSExtra/archive/refs/tags/1.5.1.tar.gz\" ee20b0d5f1f295dba7f37b259283c6c7544b57803600c9b825d10806ad614c1c PHPCSExtra",
		"PHPCSUtils/archive/refs/tags/1.2.3.tar.gz\" 1bd2198fefb8e9081ce35332127aca50f01cd4106d6f7f6bd28c0d507041d6c9 PHPCSUtils",
		"sha256sum -c -",
	} {
		if !strings.Contains(phpImageDockerfile, want) {
			t.Errorf("Dockerfile lacks %q", want)
		}
	}
	for _, dir := range strings.Split(moodleCSInstalledPaths, ",") {
		if !strings.Contains(phpImageDockerfile, "/opt/phpcs-standards/") || !strings.HasPrefix(dir, "/opt/phpcs-standards/") {
			t.Errorf("standard %s is not where the image keeps them", dir)
		}
	}
}
