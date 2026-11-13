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

package prereqs

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/oito2/perci82/internal/dev/localbin"
	"github.com/oito2/perci82/internal/dev/nettimeout"
	"github.com/oito2/perci82/internal/distro"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/shellrc"
	"github.com/oito2/perci82/internal/ui"
)

// Prereq describes a prerequisite managed by perci.gnl.
type Prereq struct {
	Name string
	// Description is the message key of the item's description;
	// DescriptionText returns it in the current language.
	Description string
	ID          string
	// RemoveWarning, when set, is the message key of the text shown for
	// confirmation before the item is uninstalled — for a removal that
	// takes more with it than the item itself.
	RemoveWarning string
	// Requires lists what must be installed before this item: IDs of other
	// Catalogue items, or of externalRequirements.
	Requires []string
}

// DescriptionText returns the item's description in the current language.
func (p Prereq) DescriptionText() string {
	return i18n.Tr(p.Description, nil)
}

// RemoveWarningText returns the item's uninstall warning in the current
// language, or "" when it has none.
func (p Prereq) RemoveWarningText() string {
	if p.RemoveWarning == "" {
		return ""
	}
	return i18n.Tr(p.RemoveWarning, nil)
}

// externalRequirements maps the requirement IDs installed outside this
// catalogue to the message key of their label.
var externalRequirements = map[string]string{
	"go": "prereq.req.go",
}

// Catalogue lists all prerequisites managed by perci.gnl: the items others
// require come first, then the rest by essentiality. Every item comes after
// the Catalogue items it requires.
var Catalogue = []Prereq{
	{Name: "Curl", ID: "curl", Description: "prereq.curl"},
	{Name: "Flatpak", ID: "flatpak", Description: "prereq.flatpak"},
	{Name: "Git", ID: "git", Description: "prereq.git"},
	{Name: "Build Essential", ID: "build-essential", Description: "prereq.build-essential"},
	{Name: "OpenSSL", ID: "openssl", Description: "prereq.openssl"},
	{Name: "Lsof", ID: "lsof", Description: "prereq.lsof"},
	{Name: "CMake", ID: "cmake", Description: "prereq.cmake"},
	{Name: "Ninja", ID: "ninja", Description: "prereq.ninja"},
	{Name: "Clang", ID: "clang", Description: "prereq.clang"},
	{Name: "GTK3 Dev Headers", ID: "gtk3dev", Description: "prereq.gtk3dev"},
	{Name: "GTK4 Dev Headers", ID: "gtk4dev", Description: "prereq.gtk4dev"},
	{Name: "WebKitGTK 6.0 Dev Headers", ID: "webkitgtk6dev", Description: "prereq.webkitgtk6dev"},
	{Name: "WebKitGTK 4.1 Dev Headers", ID: "webkitgtk41dev", Description: "prereq.webkitgtk41dev"},
	{Name: "Libxdo Dev", ID: "libxdo", Description: "prereq.libxdo"},
	{Name: "Wails3 CLI", ID: "wails3", Description: "prereq.wails3",
		Requires: []string{"go"}},
	{Name: "Libsecret Tools", ID: "libsecret", Description: "prereq.libsecret"},
	{Name: "Librsvg Tools", ID: "librsvg", Description: "prereq.librsvg"},
	{Name: "GNOME Keyring", ID: "gnome-keyring", Description: "prereq.gnome-keyring"},
	{Name: "GitHub CLI (gh)", ID: "gh", Description: "prereq.gh",
		Requires: []string{"curl"}},
	{Name: "Docker Engine", ID: "docker", Description: "prereq.docker"},
	{Name: "Node.js", ID: "node", Description: "prereq.node",
		Requires:      []string{"curl"},
		RemoveWarning: "prereq.node.removeWarning"},
	{Name: "Flatpak Builder", ID: "flatpak-builder", Description: "prereq.flatpak-builder",
		Requires: []string{"flatpak"}},
	{Name: "Flatpak Builder Tools", ID: "flatpak-builder-tools", Description: "prereq.flatpak-builder-tools",
		Requires:      []string{"git", "flatpak-builder"},
		RemoveWarning: "prereq.flatpak-builder-tools.removeWarning"},
	{Name: "AppStream", ID: "appstream", Description: "prereq.appstream"},
	{Name: "Libfuse2", ID: "libfuse2", Description: "prereq.libfuse2"},
	{Name: "AppImageTool", ID: "appimagetool", Description: "prereq.appimagetool",
		Requires: []string{"curl"}},
	{Name: "Mkcert", ID: "mkcert", Description: "prereq.mkcert"},
}

// InstalledMap returns which prerequisites are currently installed.
// It checks every Catalogue entry in parallel, one goroutine per item,
// since several checks spawn a subprocess.
func InstalledMap(ctx context.Context, exe *executor.Executor) map[string]bool {
	result := make(map[string]bool, len(Catalogue))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, p := range Catalogue {
		wg.Add(1)
		go func(p Prereq) {
			defer wg.Done()
			installed := isInstalled(ctx, exe, p.ID)
			mu.Lock()
			result[p.Name] = installed
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	return result
}

// MissingRequirements returns the labels of p's requirements that aren't
// installed. installed is InstalledMap's result (keyed by Name).
func MissingRequirements(ctx context.Context, exe *executor.Executor, p Prereq, installed map[string]bool) []string {
	var missing []string
	for _, req := range p.Requires {
		if label, ok := externalRequirements[req]; ok {
			if !exe.CommandAvailable(ctx, req) {
				missing = append(missing, i18n.Tr(label, nil))
			}
			continue
		}
		for _, other := range Catalogue {
			if other.ID == req && !installed[other.Name] {
				missing = append(missing, other.Name)
			}
		}
	}
	return missing
}

// Installed reports whether the Catalogue item named name is installed.
func Installed(ctx context.Context, exe *executor.Executor, name string) bool {
	for _, p := range Catalogue {
		if p.Name == name {
			return isInstalled(ctx, exe, p.ID)
		}
	}
	return false
}

func isInstalled(ctx context.Context, exe *executor.Executor, id string) bool {
	which := func(cmd string) bool { return exe.CommandAvailable(ctx, cmd) }
	pkgConfig := func(module string) bool {
		_, err := exe.Output(ctx, executor.Options{}, "pkg-config", "--exists", module)
		return err == nil
	}
	switch id {
	case "git":
		return which("git")
	case "curl":
		return which("curl")
	case "build-essential":
		return which("gcc") && which("g++") && which("make")
	case "openssl":
		return which("openssl") && pkgConfig("openssl")
	case "lsof":
		return which("lsof")
	case "cmake":
		return which("cmake")
	case "ninja":
		return which("ninja") || which("ninja-build")
	case "clang":
		return which("clang")
	case "gtk3dev":
		return pkgConfig("gtk+-3.0")
	case "gtk4dev":
		return pkgConfig("gtk4")
	case "webkitgtk6dev":
		return pkgConfig("webkitgtk-6.0")
	case "webkitgtk41dev":
		return pkgConfig("webkit2gtk-4.1")
	case "libxdo":
		_, err := os.Stat("/usr/include/xdo.h")
		return err == nil
	case "wails3":
		if which("wails3") {
			return true
		}
		gobin := goBinDir(ctx, exe)
		if gobin == "" {
			return false
		}
		_, err := os.Stat(filepath.Join(gobin, "wails3"))
		return err == nil
	case "libsecret":
		return which("secret-tool") && pkgConfig("libsecret-1")
	case "librsvg":
		return which("rsvg-convert") && pkgConfig("librsvg-2.0")
	case "gnome-keyring":
		return which("gnome-keyring")
	case "gh":
		return which("gh")
	case "docker":
		return which("docker")
	case "node":
		script := `export NVM_DIR="$HOME/.nvm"; [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"; command -v node && command -v npm`
		_, err := exe.Output(ctx, executor.Options{}, "bash", "-c", script)
		return err == nil
	case "flatpak":
		return which("flatpak")
	case "flatpak-builder":
		return which("flatpak-builder")
	case "flatpak-builder-tools":
		return flatpakToolsInstalled()
	case "appstream":
		return which("appstreamcli")
	case "libfuse2":
		if distro.Detect() == distro.Fedora {
			_, err := exe.Output(ctx, executor.Options{}, "rpm", "-q", "fuse-libs")
			return err == nil
		}
		_, err := exe.Output(ctx, executor.Options{}, "dpkg", "-s", "libfuse2t64")
		return err == nil
	case "appimagetool":
		return which("appimagetool")
	case "mkcert":
		return which("mkcert")
	}
	return false
}

// InstallOne installs a single prerequisite.
func InstallOne(ctx context.Context, exe *executor.Executor, stdout io.Writer, p Prereq) error {
	return installFor(ctx, exe, stdout, p, distro.Detect())
}

// installFor is InstallOne for an explicit distro family.
func installFor(ctx context.Context, exe *executor.Executor, stdout io.Writer, p Prereq, family string) error {
	if err := requireRelease(); err != nil {
		return err
	}
	opts := executor.Options{RequiresSudo: true, Stdout: stdout, Stderr: stdout}
	switch p.ID {
	case "wails3":
		return installWails3(ctx, exe, stdout)
	case "gh":
		return installGH(ctx, exe, stdout, family, opts)
	case "docker":
		return installDocker(ctx, exe, stdout, family)
	case "node":
		return installNode(ctx, exe, stdout)
	case "appimagetool":
		return installAppImageTool(ctx, exe, stdout, opts)
	case "flatpak-builder-tools":
		return installFlatpakTools(ctx, exe, stdout, family)
	}
	if pkgs := packagesFor(p.ID, family); pkgs != nil {
		return distro.InstallPkgs(ctx, exe, stdout, family, pkgs...)
	}
	return fmt.Errorf("no installer for %s", p.Name)
}

// UninstallOne uninstalls a single prerequisite.
func UninstallOne(ctx context.Context, exe *executor.Executor, stdout io.Writer, p Prereq) error {
	return uninstallFor(ctx, exe, stdout, p, distro.Detect())
}

// uninstallFor is UninstallOne for an explicit distro family.
func uninstallFor(ctx context.Context, exe *executor.Executor, stdout io.Writer, p Prereq, family string) error {
	opts := executor.Options{RequiresSudo: true, Stdout: stdout, Stderr: stdout}
	switch p.ID {
	case "wails3":
		return uninstallWails3(ctx, exe)
	case "gh":
		return uninstallGH(ctx, exe, stdout, family, opts)
	case "node":
		return uninstallNode(stdout)
	case "appimagetool":
		return exe.Run(ctx, opts, "rm", "-f", "--", appImageToolDest)
	case "flatpak-builder-tools":
		return uninstallFlatpakTools(ctx, exe, stdout)
	}
	if pkgs := removalPackagesFor(p.ID, family); pkgs != nil {
		return RemovePackages(ctx, exe, opts, family, pkgs...)
	}
	return fmt.Errorf("no uninstaller for %s", p.Name)
}

// UpdateOne updates a single installed prerequisite: its packages through
// the package manager, Node.js to the latest LTS, and the items pinned to
// a version (Wails3 CLI, AppImageTool, Flatpak Builder Tools) by
// reinstalling that version.
func UpdateOne(ctx context.Context, exe *executor.Executor, stdout io.Writer, p Prereq) error {
	return updateFor(ctx, exe, stdout, p, distro.Detect())
}

// updateFor is UpdateOne for an explicit distro family.
func updateFor(ctx context.Context, exe *executor.Executor, stdout io.Writer, p Prereq, family string) error {
	if err := requireRelease(); err != nil {
		return err
	}
	switch p.ID {
	case "wails3":
		return installWails3(ctx, exe, stdout)
	case "node":
		return updateNode(ctx, exe, stdout)
	case "appimagetool":
		return installAppImageTool(ctx, exe, stdout, executor.Options{RequiresSudo: true, Stdout: stdout, Stderr: stdout})
	case "flatpak-builder-tools":
		return installFlatpakTools(ctx, exe, stdout, family)
	}
	if pkgs := packagesFor(p.ID, family); pkgs != nil {
		return upgradePkgs(ctx, exe, stdout, family, pkgs...)
	}
	return fmt.Errorf("no updater for %s", p.Name)
}

// packagesFor returns the distro packages of item id on family, or nil for
// the items not made of distro packages (Wails3 CLI, Node.js,
// AppImageTool, Flatpak Builder Tools).
func packagesFor(id, family string) []string {
	fedora := family == distro.Fedora
	pick := func(debian, fedoraPkg string) []string {
		if fedora {
			return []string{fedoraPkg}
		}
		return []string{debian}
	}
	switch id {
	case "git", "curl", "lsof", "cmake", "clang", "gnome-keyring", "flatpak", "flatpak-builder", "appstream", "gh":
		return []string{id}
	case "ninja":
		return []string{"ninja-build"}
	case "build-essential":
		if fedora {
			return []string{"gcc", "gcc-c++", "make"}
		}
		return []string{"build-essential"}
	case "openssl":
		return []string{"openssl", pick("libssl-dev", "openssl-devel")[0]}
	case "webkitgtk41dev":
		return pick("libwebkit2gtk-4.1-dev", "webkit2gtk4.1-devel")
	case "libxdo":
		return pick("libxdo-dev", "libxdo-devel")
	case "gtk3dev":
		return pick("libgtk-3-dev", "gtk3-devel")
	case "gtk4dev":
		return pick("libgtk-4-dev", "gtk4-devel")
	case "webkitgtk6dev":
		return pick("libwebkitgtk-6.0-dev", "webkitgtk6.0-devel")
	case "libsecret":
		if fedora {
			return []string{"libsecret", "libsecret-devel", "git-credential-libsecret"}
		}
		return []string{"libsecret-1-0", "libsecret-1-dev", "libsecret-tools"}
	case "librsvg":
		return []string{librsvgPkg(family), pick("librsvg2-dev", "librsvg2-devel")[0]}
	case "libfuse2":
		return []string{libfuse2Pkg(family)}
	case "mkcert":
		return mkcertPkgs(family)
	case "docker":
		return dockerPkgs(family)
	}
	return nil
}

// removalPackagesFor returns the packages removed with item id on family:
// packagesFor without what other software on the system depends on — the
// libsecret runtime library and the openssl command, which the system's
// certificate handling needs.
func removalPackagesFor(id, family string) []string {
	pkgs := packagesFor(id, family)
	switch id {
	case "libsecret":
		// The runtime library stays: the desktop (GNOME Shell, gvfs, KWallet,
		// keyrings) depends on it. On Fedora it is the libsecret package.
		return slices.DeleteFunc(slices.Clone(pkgs), func(p string) bool { return p == "libsecret-1-0" || p == "libsecret" })
	case "openssl":
		return slices.DeleteFunc(slices.Clone(pkgs), func(p string) bool { return p == "openssl" })
	}
	return pkgs
}

// upgradePkgs upgrades pkgs to their latest available version without
// installing anything new — on Debian the package index is refreshed in
// the same privileged batch (one password prompt).
func upgradePkgs(ctx context.Context, exe *executor.Executor, stdout io.Writer, family string, pkgs ...string) error {
	switch family {
	case distro.Debian:
		return exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, []executor.PrivilegedStep{
			{Name: "apt-get", Env: distro.AptEnv, Args: distro.AptArgs("update", "-q")},
			{Name: "apt-get", Env: distro.AptEnv, Args: distro.AptArgs(append([]string{"install", "--only-upgrade", "-y", "--"}, pkgs...)...)},
		})
	case distro.Fedora:
		return exe.Run(ctx, executor.Options{RequiresSudo: true, Stdout: stdout, Stderr: stdout}, "dnf", append([]string{"upgrade", "-y", "--"}, pkgs...)...)
	default:
		return distro.UnsupportedFamilyError(family)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func librsvgPkg(family string) string {
	if family == distro.Fedora {
		return "librsvg2-tools"
	}
	return "librsvg2-bin"
}

func libfuse2Pkg(family string) string {
	if family == distro.Fedora {
		return "fuse-libs"
	}
	return "libfuse2t64"
}

// mkcertPkgs returns mkcert plus the NSS tools package it needs to install
// its local CA into Firefox's trust store; the package name depends on the
// distro family.
func mkcertPkgs(family string) []string {
	if family == distro.Fedora {
		return []string{"mkcert", "nss-tools"}
	}
	return []string{"mkcert", "libnss3-tools"}
}

// dockerPkgs returns the distribution's own Docker Engine, Compose v2 and
// Buildx packages for family. Compose v2 is docker-compose-v2 on Ubuntu and
// its derivatives and docker-compose on Debian, where docker-compose-v2
// does not exist.
func dockerPkgs(family string) []string {
	switch family {
	case distro.Fedora:
		// Fedora's own packages, not Docker Inc.'s repository names.
		return []string{"moby-engine", "docker-compose", "docker-buildx"}
	default:
		compose := "docker-compose"
		if aptHasPackage("docker-compose-v2") {
			compose = "docker-compose-v2"
		}
		return []string{"docker.io", compose, "docker-buildx"}
	}
}

// requireRelease refuses installs and updates on a distribution release
// older than the package names here exist on (a variable so tests choose
// the answer).
var requireRelease = distro.RequireSupportedRelease

// aptHasPackage reports whether apt knows a package named name (a
// variable so tests choose the answer).
var aptHasPackage = func(name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "apt-cache", "show", name).Run() == nil
}

// aptShell renders an unattended apt-get command line for a bash script.
func aptShell(args ...string) string {
	words := []string{"apt-get"}
	for _, a := range distro.AptArgs(args...) {
		words = append(words, executor.ShellQuote(a))
	}
	return strings.Join(words, " ")
}

// ── GitHub CLI ────────────────────────────────────────────────────────────────

func installGH(ctx context.Context, exe *executor.Executor, stdout io.Writer, family string, opts executor.Options) error {
	ui.Info(stdout, i18n.Tr("prereq.run.installGH", nil))
	switch family {
	case distro.Debian:
		script := `set -Eeuo pipefail
curl -fsSL ` + nettimeout.CurlShellFlags + ` https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" > /etc/apt/sources.list.d/github-cli.list
` + aptShell("update", "-q") + `
` + aptShell("install", "-y", "--", "gh")
		opts.Env = append(slices.Clone(opts.Env), distro.AptEnv...)
		return exe.Run(ctx, opts, "bash", "-c", script)
	case distro.Fedora:
		// gh is in Fedora's own repositories, so no extra repo is added.
		return exe.Run(ctx, opts, "dnf", "install", "-y", "--", "gh")
	default:
		return fmt.Errorf("unsupported distro family for GitHub CLI: %s", family)
	}
}

func uninstallGH(ctx context.Context, exe *executor.Executor, stdout io.Writer, family string, opts executor.Options) error {
	_ = stdout
	switch family {
	case distro.Debian:
		if err := CheckRemoval(ctx, exe, family, "gh"); err != nil {
			return err
		}
		script := `set -Eeuo pipefail
` + aptShell("purge", "-y", "--", "gh") + `
rm -f /etc/apt/sources.list.d/github-cli.list /usr/share/keyrings/githubcli-archive-keyring.gpg
` + aptShell("update", "-q")
		opts.Env = append(slices.Clone(opts.Env), distro.AptEnv...)
		return exe.Run(ctx, opts, "bash", "-c", script)
	case distro.Fedora:
		return RemovePackages(ctx, exe, opts, family, "gh")
	default:
		return fmt.Errorf("unsupported distro family for GitHub CLI: %s", family)
	}
}

// ── Docker ────────────────────────────────────────────────────────────────────

// installDocker installs the engine, enables the service and adds the user
// to the docker group as one privileged batch, so the password is asked
// once. The group membership is checked beforehand, unprivileged.
func installDocker(ctx context.Context, exe *executor.Executor, stdout io.Writer, family string) error {
	ui.Info(stdout, i18n.Tr("prereq.run.installDocker", nil))
	var steps []executor.PrivilegedStep
	switch family {
	case distro.Debian:
		steps = []executor.PrivilegedStep{
			{Name: "apt-get", Env: distro.AptEnv, Args: distro.AptArgs("update", "-q")},
			{Name: "apt-get", Env: distro.AptEnv, Args: distro.AptArgs(append([]string{"install", "-y", "--"}, dockerPkgs(family)...)...)},
		}
	case distro.Fedora:
		steps = []executor.PrivilegedStep{{Name: "dnf", Args: append([]string{"install", "-y", "--"}, dockerPkgs(family)...)}}
	default:
		return fmt.Errorf("unsupported distro family for Docker: %s", family)
	}
	steps = append(steps, executor.PrivilegedStep{
		Announce: i18n.Tr("prereq.run.dockerService", nil),
		Soft:     true, WarnMessage: i18n.Tr("prereq.run.dockerServiceFailed", nil),
		Name: "systemctl", Args: []string{"enable", "--now", "docker"},
	})

	user := executor.CurrentUser()
	addToGroup := false
	if user != "" {
		out, _ := exe.Output(ctx, executor.Options{}, "groups", "--", user)
		addToGroup = !slices.Contains(strings.Fields(out), "docker")
	}
	if addToGroup {
		steps = append(steps, executor.PrivilegedStep{
			Announce: i18n.Tr("prereq.run.dockerGroup", i18n.Args{"user": user}),
			Soft:     true, WarnMessage: i18n.Tr("prereq.run.dockerGroupFailed", i18n.Args{"user": user}),
			Name: "usermod", Args: []string{"-aG", "docker", "--", user},
		})
	}

	if err := exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, steps); err != nil {
		return err
	}
	if addToGroup {
		ui.Warning(stdout, i18n.Tr("prereq.run.dockerRelogin", nil))
	}
	if family == distro.Fedora {
		ui.Warning(stdout, i18n.Tr("prereq.run.dockerSELinux", nil))
	}
	return nil
}

// ── Node.js ───────────────────────────────────────────────────────────────────

// nvmVersion is the nvm release tag whose installer is downloaded, and
// nvmInstallSHA256 the SHA-256 of that installer (install.sh).
const (
	nvmVersion       = "v0.40.8"
	nvmInstallSHA256 = "48a0eee9a60e07422dce0eb5774754c83889570ca1ee2566c516acbe8af03a9e"
)

// nvmRCLines are the lines nvm's installer appends to a shell rc file.
var nvmRCLines = []string{
	`export NVM_DIR="$HOME/.nvm"`,
	`[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm`,
	`[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion`,
}

// nodeInstallScript installs nvm (when missing) at the pinned release,
// running its installer only once its SHA-256 matches, and the latest
// Node.js LTS through it. -u (nounset) is not set, because nvm.sh
// references variables that may be unset.
const nodeInstallScript = `set -Eeo pipefail
export NVM_DIR="$HOME/.nvm"
if [ ! -s "$NVM_DIR/nvm.sh" ]; then
    installer=$(mktemp)
    trap 'rm -f -- "$installer"' EXIT
    curl -fsSL ` + nettimeout.CurlShellFlags + ` -o "$installer" https://raw.githubusercontent.com/nvm-sh/nvm/` + nvmVersion + `/install.sh
    printf '%s  %s\n' ` + nvmInstallSHA256 + ` "$installer" | sha256sum -c --status - || { echo "nvm installer checksum mismatch" >&2; exit 1; }
    bash "$installer"
    export NVM_DIR="$HOME/.nvm"
fi
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"
nvm install --lts
nvm use --lts
`

func installNode(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	checkScript := `export NVM_DIR="$HOME/.nvm"; [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"; command -v node && command -v npm`
	if _, e := exe.Output(ctx, executor.Options{}, "bash", "-c", checkScript); e == nil {
		ui.Info(stdout, i18n.Tr("prereq.run.nodeReady", nil))
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home folder: %w", err)
	}
	_, statErr := os.Stat(filepath.Join(home, ".nvm", "nvm.sh"))
	freshNVM := statErr != nil

	ui.Info(stdout, i18n.Tr("prereq.run.installNode", nil))
	ctx, cancel := nettimeout.Context(ctx)
	defer cancel()
	if err := exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, "bash", "-c", nodeInstallScript); err != nil {
		return fmt.Errorf("install Node.js: %w", err)
	}

	localbin.EnsureInPath(stdout)

	if freshNVM {
		ui.Warning(stdout, i18n.Tr("prereq.run.nodeRestartTerminal", nil))
	}
	return nil
}

func uninstallNode(stdout io.Writer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home folder: %w", err)
	}
	if err := os.RemoveAll(filepath.Join(home, ".nvm")); err != nil {
		return fmt.Errorf("remove ~/.nvm: %w", err)
	}
	candidates := shellrc.Dedup([]string{
		filepath.Join(home, ".bashrc"), filepath.Join(home, ".bash_profile"), filepath.Join(home, ".profile"),
		filepath.Join(home, ".zshrc"), shellrc.File(home),
	})
	for _, line := range nvmRCLines {
		for _, res := range shellrc.RemoveEntry(candidates, "", line) {
			if res.Err != nil {
				ui.Warning(stdout, i18n.Tr("prereq.run.rcFailed", i18n.Args{"path": res.Path, "error": res.Err.Error()}))
				continue
			}
			ui.Info(stdout, i18n.Tr("prereq.run.nvmLineRemoved", i18n.Args{"path": res.Path}))
		}
	}
	ui.Success(stdout, i18n.Tr("prereq.run.nodeRemoved", nil))
	return nil
}

// nodeUpdateScript installs the latest Node.js LTS through nvm, carrying
// over the global npm packages of nvm's default version (the one in use
// when there is no default), and makes the LTS the default for new shells.
// Nothing is installed when that version already is the latest LTS: the
// script then prints $PERCI_UP_TO_DATE with {version} replaced.
const nodeUpdateScript = `set -Eeo pipefail
export NVM_DIR="$HOME/.nvm"
. "$NVM_DIR/nvm.sh"
current="$(nvm version default)"
if [ "$current" = "N/A" ]; then
    current="$(nvm current)"
fi
latest="$(nvm version-remote --lts)"
if [ "$current" = "$latest" ]; then
    printf '%s\n' "${PERCI_UP_TO_DATE//\{version\}/$current}"
    exit 0
fi
nvm install --lts --reinstall-packages-from="$current"
nvm alias default 'lts/*'
`

func updateNode(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	ui.Info(stdout, i18n.Tr("prereq.run.updateNode", nil))
	ctx, cancel := nettimeout.Context(ctx)
	defer cancel()
	opts := executor.Options{Stdout: stdout, Stderr: stdout, Env: []string{"PERCI_UP_TO_DATE=" + i18n.Tr("prereq.run.nodeUpToDate", nil)}}
	if err := exe.Run(ctx, opts, "bash", "-c", nodeUpdateScript); err != nil {
		return fmt.Errorf("update Node.js: %w", err)
	}
	return nil
}

// ── AppImageTool ──────────────────────────────────────────────────────────────

const appImageToolDest = "/usr/local/bin/appimagetool"

// appImageToolVersion is the appimagetool release tag that is downloaded
// and checked against the release's published checksum (see
// appImageToolChecksum).
const appImageToolVersion = "1.9.1"

// githubAPIBase/githubDownloadBase are GitHub's API and download roots,
// variables so tests can point them at a local server.
var (
	githubAPIBase      = "https://api.github.com"
	githubDownloadBase = "https://github.com"
)

func appImageToolArch() string {
	if runtime.GOARCH == "arm64" {
		return "aarch64"
	}
	return "x86_64"
}

// appImageToolChecksum fetches the SHA-256 digest that the GitHub API
// lists for the release asset assetName, independently of the download.
func appImageToolChecksum(ctx context.Context, assetName string) (string, error) {
	apiURL := githubAPIBase + "/repos/AppImage/appimagetool/releases/tags/" + appImageToolVersion
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("query the GitHub API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var payload struct {
		Assets []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	for _, a := range payload.Assets {
		if a.Name != assetName {
			continue
		}
		sum, ok := strings.CutPrefix(a.Digest, "sha256:")
		if !ok {
			return "", fmt.Errorf("unexpected digest format for %s: %q", assetName, a.Digest)
		}
		return sum, nil
	}
	return "", fmt.Errorf("asset %s not found in release %s", assetName, appImageToolVersion)
}

// installAppImageTool downloads appimagetool to an unprivileged temp file
// and verifies its checksum before anything runs elevated. The final step
// is executor.PrivilegedInstall: root copies the file into a root-owned
// staged copy, re-verifies the checksum on it and renames it into
// /usr/local/bin as root:root 0755.
func installAppImageTool(ctx context.Context, exe *executor.Executor, stdout io.Writer, opts executor.Options) error {
	assetName := "appimagetool-" + appImageToolArch() + ".AppImage"
	toolURL := githubDownloadBase + "/AppImage/appimagetool/releases/download/" + appImageToolVersion + "/" + assetName

	ui.Info(stdout, i18n.Tr("prereq.run.appimageChecksum", i18n.Args{"version": appImageToolVersion}))
	checksum, err := appImageToolChecksum(ctx, assetName)
	if err != nil {
		return fmt.Errorf("get the appimagetool checksum: %w", err)
	}

	tmp, err := os.CreateTemp("", "appimagetool-*.AppImage")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	ui.Info(stdout, i18n.Tr("prereq.run.appimageDownload", nil))
	if err := exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, "curl", nettimeout.CurlArgs("-fsSL", "-o", tmpPath, "--", toolURL)...); err != nil {
		return fmt.Errorf("download appimagetool: %w", err)
	}

	ui.Info(stdout, i18n.Tr("prereq.run.appimageVerify", nil))
	if err := fsutil.VerifyFile(tmpPath, sha256.New(), checksum); err != nil {
		return fmt.Errorf("appimagetool integrity check failed: %w", err)
	}

	if err := exe.PrivilegedInstall(ctx, opts, tmpPath, appImageToolDest, checksum, "0755"); err != nil {
		return fmt.Errorf("install appimagetool: %w", err)
	}
	return nil
}

// ── Wails3 CLI ────────────────────────────────────────────────────────────────

// wails3Version is the Wails CLI version installed by `go install`.
const wails3Version = "v3.0.0-beta.28"

// goBinDir resolves where `go install` puts binaries: $GOBIN if set,
// otherwise $GOPATH/bin. Returns "" when neither can be determined.
func goBinDir(ctx context.Context, exe *executor.Executor) string {
	if out, err := exe.Output(ctx, executor.Options{}, "go", "env", "GOBIN"); err == nil {
		if dir := strings.TrimSpace(out); dir != "" {
			return dir
		}
	}
	if out, err := exe.Output(ctx, executor.Options{}, "go", "env", "GOPATH"); err == nil {
		if dir := strings.TrimSpace(out); dir != "" {
			return filepath.Join(dir, "bin")
		}
	}
	return ""
}

// installWails3 requires Go to already be installed. It needs no sudo,
// since `go install` writes under the user's own GOPATH/GOBIN.
func installWails3(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	if !exe.CommandAvailable(ctx, "go") {
		return i18n.NewError("prereq.wails3NoGo")
	}
	ui.Info(stdout, i18n.Tr("prereq.run.installWails3", i18n.Args{"version": wails3Version}))
	pkg := "github.com/wailsapp/wails/v3/cmd/wails3@" + wails3Version
	ctx, cancel := nettimeout.Context(ctx)
	defer cancel()
	if err := exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, "go", "install", pkg); err != nil {
		return fmt.Errorf("install wails3: %w", err)
	}
	if !exe.CommandAvailable(ctx, "wails3") {
		ui.Warning(stdout, i18n.Tr("prereq.run.wails3NotInPath", nil))
	}
	return nil
}

func uninstallWails3(ctx context.Context, exe *executor.Executor) error {
	gobin := goBinDir(ctx, exe)
	if gobin == "" {
		return i18n.NewError("prereq.goBinNotFound")
	}
	if err := os.Remove(filepath.Join(gobin, "wails3")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove wails3: %w", err)
	}
	return nil
}
