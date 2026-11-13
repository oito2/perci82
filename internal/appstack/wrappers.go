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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/dev/localbin"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// appTools is the list of tools of the older per-tool wrappers, written for
// AppTypeMoodle/AppTypePHP/AppTypeGeneric, whose project root is mounted at
// /var/www/html.
var appTools = []string{"php", "phpcs", "phpcbf", "phpunit", "composer"}

// nodeAppTools is appTools' equivalent for AppTypeNode, whose project root
// is mounted at /app: just npm.
var nodeAppTools = []string{"npm"}

// wrapperUser is who the PHP-family and PHP+Node wrappers run as. Those
// containers run as root, so the wrappers exec as www-data, whose UID
// matches the host user, keeping vendor/ and node_modules/ owned by the
// host user.
const wrapperUser = "www-data"

// writeAppWrappers creates ~/.local/bin/{php,composer,phpcs,phpcbf,
// phpunit}-<folder> for a PHP-family Container Aplicativo (Moodle/PHP/
// Generic), each pointing at that app's own container, project root at
// /var/www/html.
func writeAppWrappers(folder, htmlDir string, stdout io.Writer) {
	writeWrappers(folder, appTools, htmlDir, "/var/www/html", wrapperUser, stdout)
}

// writeNodeAppWrappers is writeAppWrappers' equivalent for AppTypeNode:
// ~/.local/bin/npm-<folder>, project root at /app. user is "" for a
// Node-only container, which already runs as the UID-matched "node" user,
// and wrapperUser for a PHP+Node one.
func writeNodeAppWrappers(folder, appDir, user string, stdout io.Writer) {
	writeWrappers(folder, nodeAppTools, appDir, "/app", user, stdout)
}

// wrapperMarker is the second line of every wrapper perci writes: only a
// file carrying it (or the layout older versions wrote) is ever replaced or
// removed.
const wrapperMarker = "# perci-wrapper"

// legacyWrapperPrefix is how wrappers written before wrapperMarker existed
// begin.
const legacyWrapperPrefix = "#!/usr/bin/env bash\nCONTAINER="

// isPerciWrapper reports whether the file at path is a wrapper perci wrote.
func isPerciWrapper(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := string(data)
	return strings.Contains(content, "\n"+wrapperMarker+"\n") ||
		(strings.HasPrefix(content, legacyWrapperPrefix) && strings.Contains(content, "docker exec"))
}

// removeAppWrappers deletes every wrapper writeAppWrappers/
// writeNodeAppWrappers may have created for folder, so no <tool>-<folder>
// is left pointing at a missing container. A file at one of those paths
// that is not a perci wrapper is left alone.
func removeAppWrappers(folder string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	for _, tool := range append(append([]string{}, appTools...), nodeAppTools...) {
		path := filepath.Join(home, ".local", "bin", tool+"-"+folder)
		if isPerciWrapper(path) {
			_ = os.Remove(path)
		}
	}
}

// writeWrappers creates ~/.local/bin/<tool>-<folder> for every tool in
// tools, each pointing at that app's own container.
func writeWrappers(folder string, tools []string, hostDir, containerDir, user string, stdout io.Writer) {
	home, err := os.UserHomeDir()
	if err != nil {
		ui.Warning(stdout, i18n.Tr("docker.run.noHome", i18n.Args{"error": err.Error()}))
		return
	}

	localBin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		ui.Warning(stdout, i18n.Tr("docker.run.localBinFailed", i18n.Args{"error": err.Error()}))
		return
	}

	for _, tool := range tools {
		name := tool + "-" + folder
		path := filepath.Join(localBin, name)
		// A file that is not a perci wrapper is never overwritten.
		if _, err := os.Lstat(path); err == nil && !isPerciWrapper(path) {
			ui.Warning(stdout, i18n.Tr("docker.run.wrapperExists", i18n.Args{"path": path, "name": name}))
			continue
		}
		// Each wrapper is removed first so a symlink at path is replaced, not followed.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			ui.Warning(stdout, i18n.Tr("docker.run.wrapperReplaceFailed", i18n.Args{"name": name, "error": err.Error()}))
			continue
		}
		if err := os.WriteFile(path, []byte(buildAppWrapperScript(folder, tool, hostDir, containerDir, user)), 0o755); err != nil {
			ui.Warning(stdout, i18n.Tr("docker.run.wrapperFailed", i18n.Args{"name": name, "error": err.Error()}))
		}
	}

	localbin.EnsureInPath(stdout)
}

func appWrapperSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// buildAppWrapperScript rewrites any argument under the host project path
// to its containerDir equivalent, then execs into the container,
// interactive when attached to a real terminal and non-interactive (docker
// exec -i) otherwise. A non-empty user runs the tool as that user, with
// HOME=/tmp so tools such as npm have a writable cache directory.
func buildAppWrapperScript(container, tool, hostDir, containerDir, user string) string {
	execAs := "()"
	if user != "" {
		execAs = "(-u " + appWrapperSingleQuote(user) + " -e HOME=/tmp)"
	}
	return fmt.Sprintf(`#!/usr/bin/env bash
`+wrapperMarker+`
CONTAINER=%s
WS_HOST=%s
WS_CONT=%s
EXEC_AS=%s
ARGS=()
for arg in "$@"; do
    if [[ "$arg" == "${WS_HOST}/"* ]] || [[ "$arg" == "${WS_HOST}" ]]; then
        ARGS+=("${WS_CONT}${arg#"${WS_HOST}"}")
    else
        ARGS+=("$arg")
    fi
done
if [ -t 0 ] && [ -t 1 ]; then
    exec docker exec -it "${EXEC_AS[@]}" "${CONTAINER}" %s "${ARGS[@]}"
else
    exec docker exec -i "${EXEC_AS[@]}" "${CONTAINER}" %s "${ARGS[@]}"
fi
`, appWrapperSingleQuote(container), appWrapperSingleQuote(hostDir), appWrapperSingleQuote(containerDir), execAs, tool, tool)
}

// The tools of an app's command a checklist turns on or off. phpcs also
// covers phpcbf; moodle-cs registers the Moodle coding standard with them.
const (
	ToolComposer = "composer"
	ToolPHPUnit  = "phpunit"
	ToolPHPCS    = "phpcs"
	ToolMoodleCS = "moodle-cs"
)

// moodleCSInstalledPaths are the phpcs standards the PHP image keeps in
// /opt/phpcs-standards (moodle-cs and the PHPCSExtra/PHPCSUtils standards
// it builds on), registered only by the command of an app with moodle-cs
// on.
const moodleCSInstalledPaths = "/opt/phpcs-standards/moodle-cs,/opt/phpcs-standards/PHPCSExtra,/opt/phpcs-standards/PHPCSUtils"

// AppTools returns the checklist tools an app of appType offers: the PHP
// tools for every type with PHP, plus moodle-cs for Moodle; none for Node.
func AppTools(appType string) []string {
	switch appType {
	case config.AppTypeNode:
		return nil
	case config.AppTypeMoodle:
		return []string{ToolComposer, ToolPHPUnit, ToolPHPCS, ToolMoodleCS}
	default:
		return []string{ToolComposer, ToolPHPUnit, ToolPHPCS}
	}
}

// ToolEnabled reports whether app's type offers tool and its checklist
// leaves it on.
func ToolEnabled(app config.AppContainer, tool string) bool {
	return slices.Contains(AppTools(app.Type), tool) && !slices.Contains(app.DisabledTools, tool)
}

// commandMount pairs a host folder with where the app's container sees it.
type commandMount struct {
	host, container string
}

// appCommandSpec is what an app's command needs to know.
type appCommandSpec struct {
	container string
	user      string         // "" runs as the container's own user
	mounts    []commandMount // the first one is the default working folder
	fallback  string         // what runs when the first argument is no subcommand
	reserved  []string       // the subcommands of the app's type
	enabled   []string       // the subcommands its checklist leaves on
	phpcsArgs []string       // extra arguments of phpcs and phpcbf
}

// appCommandSpecFor builds the command of app, whose project root is under
// workspace.
func appCommandSpecFor(app config.AppContainer, workspace string) appCommandSpec {
	spec := appCommandSpec{container: app.Folder, user: wrapperUser, fallback: "php"}
	phpTools := []string{"composer", "phpunit", "phpcs", "phpcbf"}
	switch app.Type {
	case config.AppTypeNode:
		spec.user, spec.fallback = "", "node"
		spec.mounts = []commandMount{{AppHTMLDir(workspace, app.Folder), "/app"}}
		spec.reserved = []string{"npm"}
	case config.AppTypePHPNode:
		spec.mounts = []commandMount{
			{AppComboAPIDir(workspace, app.Folder), "/var/www/html"},
			{AppComboAppDir(workspace, app.Folder), "/app"},
		}
		spec.reserved = append(slices.Clone(phpTools), "npm")
	default:
		spec.mounts = []commandMount{{AppHTMLDir(workspace, app.Folder), "/var/www/html"}}
		spec.reserved = phpTools
	}
	for _, sub := range spec.reserved {
		tool := sub
		if sub == "phpcbf" {
			tool = ToolPHPCS
		}
		if sub == "npm" || ToolEnabled(app, tool) {
			spec.enabled = append(spec.enabled, sub)
		}
	}
	if ToolEnabled(app, ToolMoodleCS) {
		spec.phpcsArgs = []string{"--runtime-set", "installed_paths", moodleCSInstalledPaths}
	}
	// A workspace reached through a link is also known by its real path,
	// which is what `pwd -P` reports inside it.
	for _, m := range slices.Clone(spec.mounts) {
		if real, err := filepath.EvalSymlinks(m.host); err == nil && real != m.host {
			spec.mounts = append(spec.mounts, commandMount{real, m.container})
		}
	}
	return spec
}

// bashArray returns items as a bash array literal of single-quoted words.
func bashArray(items []string) string {
	quoted := make([]string, len(items))
	for i, it := range items {
		quoted[i] = appWrapperSingleQuote(it)
	}
	return "(" + strings.Join(quoted, " ") + ")"
}

// buildAppCommandScript is the ~/.local/bin/<app> script of spec. The first
// argument, when it is one of the type's subcommands, picks the tool (a
// subcommand the checklist turned off is refused); anything else runs
// through the fallback (php, or node for a Node app). The working folder
// inside the container follows the current folder when it is inside the
// app's project, and arguments that are host paths into it are rewritten
// to the container's paths. The messages are in the current language.
func buildAppCommandScript(spec appCommandSpec) string {
	hosts := make([]string, len(spec.mounts))
	conts := make([]string, len(spec.mounts))
	for i, m := range spec.mounts {
		hosts[i], conts[i] = m.host, m.container
	}
	execAs := "()"
	if spec.user != "" {
		execAs = "(-u " + appWrapperSingleQuote(spec.user) + " -e HOME=/tmp)"
	}
	return `#!/usr/bin/env bash
` + wrapperMarker + `
` + appCommandMarker + `
CONTAINER=` + appWrapperSingleQuote(spec.container) + `
EXEC_AS=` + execAs + `
HOSTS=` + bashArray(hosts) + `
CONTS=` + bashArray(conts) + `
FALLBACK=` + appWrapperSingleQuote(spec.fallback) + `
RESERVED=` + bashArray(spec.reserved) + `
ENABLED=` + bashArray(spec.enabled) + `
PHPCS_ARGS=` + bashArray(spec.phpcsArgs) + `
MSG_DISABLED=` + appWrapperSingleQuote(i18n.Tr("appcmd.disabled", i18n.Args{"tool": "{tool}", "app": spec.container})) + `
MSG_OUTSIDE=` + appWrapperSingleQuote(i18n.Tr("appcmd.outside", i18n.Args{"dir": "{dir}", "app": spec.container})) + `

# map_path prints the container path of the host path $1, failing when $1
# is outside the app's folders.
map_path() {
    local i
    for i in "${!HOSTS[@]}"; do
        if [[ "$1" == "${HOSTS[$i]}" || "$1" == "${HOSTS[$i]}/"* ]]; then
            printf '%s' "${CONTS[$i]}${1#"${HOSTS[$i]}"}"
            return 0
        fi
    done
    return 1
}

CMD=("$FALLBACK")
if [[ $# -gt 0 ]]; then
    for sub in "${RESERVED[@]}"; do
        [[ "$1" == "$sub" ]] || continue
        on=0
        for e in "${ENABLED[@]}"; do [[ "$e" == "$1" ]] && on=1; done
        if [[ $on == 0 ]]; then
            printf '%s\n' "${MSG_DISABLED//\{tool\}/$1}" >&2
            exit 2
        fi
        CMD=("$1")
        if [[ "$1" == phpcs || "$1" == phpcbf ]]; then CMD+=("${PHPCS_ARGS[@]}"); fi
        shift
        break
    done
fi

if ! WD=$(map_path "$(pwd -P)") && ! WD=$(map_path "$PWD"); then
    WD=${CONTS[0]}
    printf '%s\n' "${MSG_OUTSIDE//\{dir\}/$WD}" >&2
fi

ARGS=()
for arg in "$@"; do
    if mapped=$(map_path "$arg"); then ARGS+=("$mapped"); else ARGS+=("$arg"); fi
done

TTY=-i
if [ -t 0 ] && [ -t 1 ]; then TTY=-it; fi
exec docker exec "$TTY" -w "$WD" "${EXEC_AS[@]}" "$CONTAINER" "${CMD[@]}" "${ARGS[@]}"
`
}

// appCommandMarker is the third line of an app's command, telling it apart
// from the older <tool>-<app> wrappers.
const appCommandMarker = "# perci-app-command"

// lookPath finds a command on PATH — a variable so tests decide what is
// installed.
var lookPath = exec.LookPath

// commandCollision returns the path of an existing command called name that
// is not a perci wrapper, or "" when ~/.local/bin/<name> can be written.
func commandCollision(name, localBin string) string {
	path := filepath.Join(localBin, name)
	if _, err := os.Lstat(path); err == nil {
		if isPerciWrapper(path) {
			return ""
		}
		return path
	}
	if found, err := lookPath(name); err == nil && found != path {
		return found
	}
	return ""
}

// installAppCommand writes ~/.local/bin/<app> for app and removes its older
// <tool>-<app> wrappers. When another command already has that name, the
// command is not written, a warning says so, and the older wrappers are
// written instead, so the tools stay reachable.
func installAppCommand(app config.AppContainer, workspace string, stdout io.Writer) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		ui.Warning(stdout, i18n.Tr("docker.run.noHome", i18n.Args{"error": err.Error()}))
		return false
	}
	localBin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		ui.Warning(stdout, i18n.Tr("docker.run.localBinFailed", i18n.Args{"error": err.Error()}))
		return false
	}
	if other := commandCollision(app.Folder, localBin); other != "" {
		ui.Warning(stdout, i18n.Tr("appcmd.run.collision", i18n.Args{"name": app.Folder, "path": other}))
		writeLegacyWrappers(app, workspace, stdout)
		return false
	}
	path := filepath.Join(localBin, app.Folder)
	// Removed first so a symlink at path is replaced, not followed.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		ui.Warning(stdout, i18n.Tr("docker.run.wrapperReplaceFailed", i18n.Args{"name": app.Folder, "error": err.Error()}))
		return false
	}
	if err := os.WriteFile(path, []byte(buildAppCommandScript(appCommandSpecFor(app, workspace))), 0o755); err != nil {
		ui.Warning(stdout, i18n.Tr("docker.run.wrapperFailed", i18n.Args{"name": app.Folder, "error": err.Error()}))
		return false
	}
	removeAppWrappers(app.Folder)
	ui.Info(stdout, i18n.Tr("appcmd.run.installed", i18n.Args{"name": app.Folder}))
	localbin.EnsureInPath(stdout)
	return true
}

// writeLegacyWrappers writes the older <tool>-<app> wrappers of app.
func writeLegacyWrappers(app config.AppContainer, workspace string, stdout io.Writer) {
	switch app.Type {
	case config.AppTypeNode:
		writeNodeAppWrappers(app.Folder, AppHTMLDir(workspace, app.Folder), "", stdout)
	case config.AppTypePHPNode:
		writeAppWrappers(app.Folder, AppComboAPIDir(workspace, app.Folder), stdout)
		writeNodeAppWrappers(app.Folder, AppComboAppDir(workspace, app.Folder), wrapperUser, stdout)
	default:
		writeAppWrappers(app.Folder, AppHTMLDir(workspace, app.Folder), stdout)
	}
}

// removeAppCommand deletes ~/.local/bin/<folder> when it is a perci
// wrapper.
func removeAppCommand(folder string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	if path := filepath.Join(home, ".local", "bin", folder); isPerciWrapper(path) {
		_ = os.Remove(path)
	}
}

// HasLegacyWrappers reports whether folder still has an older <tool>-<app>
// wrapper perci wrote.
func HasLegacyWrappers(folder string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	for _, tool := range append(slices.Clone(appTools), nodeAppTools...) {
		if isPerciWrapper(filepath.Join(home, ".local", "bin", tool+"-"+folder)) {
			return true
		}
	}
	return false
}

// MigrateAppCommand replaces the older <tool>-<app> wrappers of the app in
// folder with its command, without recreating its container.
func MigrateAppCommand(stdout io.Writer, folder string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	// Validated: the folder name becomes a file name in ~/.local/bin.
	app, err := registeredApp(folder)
	if err != nil {
		return err
	}
	workspace, err := resolveWorkspace(cfg)
	if err != nil {
		return err
	}
	if !installAppCommand(app, workspace, stdout) {
		return errors.New(i18n.Tr("appcmd.notMigrated", i18n.Args{"name": folder}))
	}
	if ToolEnabled(app, ToolMoodleCS) {
		ui.Info(stdout, i18n.Tr("appcmd.run.moodleCSRecreate", nil))
	}
	return nil
}
