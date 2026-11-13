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

// Package antigravityide manages the Google Antigravity IDE. Install and
// Update take a local .tar.gz path chosen by the user. Only the commands
// that need root run with RequiresSudo; the rest run as the current user.
package antigravityide

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

const (
	installDir       = "/opt/antigravity"
	defaultBinName   = "antigravity"
	sandboxPermWant  = "4755"
	iconExtractedRel = "icon.png"
)

// excludedSuffixes lists the extensions findBinary ignores when looking for
// the app binary.
var excludedSuffixes = []string{".so", ".json", ".png", ".pak", ".dll", ".dat", ".bin", ".desktop"}

// validBinName restricts findBinary's candidates to plain filenames, so a
// tarball entry name containing newlines or other bytes can never end up
// inside the .desktop file writeDesktopEntry generates (Exec=<path> %U).
var validBinName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func desktopFile(home string) string {
	return filepath.Join(home, ".local", "share", "applications", "antigravity.desktop")
}

func iconFile(home string) string {
	return filepath.Join(home, ".local", "share", "icons", "hicolor", "512x512", "apps", "antigravity.png")
}

// Installed reports whether Antigravity IDE is currently installed, using a
// plain os.Stat on the install directory.
func Installed() bool {
	info, err := os.Stat(installDir)
	return err == nil && info.IsDir()
}

// Install performs a first install; it fails if installDir already exists
// (use Update to replace an existing installation).
func Install(ctx context.Context, exe *executor.Executor, stdout io.Writer, tarballPath string) error {
	return doInstallOrUpdate(ctx, exe, stdout, tarballPath, false)
}

// Update replaces an existing installation with the contents of a new
// tarball.
func Update(ctx context.Context, exe *executor.Executor, stdout io.Writer, tarballPath string) error {
	return doInstallOrUpdate(ctx, exe, stdout, tarballPath, true)
}

func doInstallOrUpdate(ctx context.Context, exe *executor.Executor, stdout io.Writer, tarballPath string, update bool) error {
	if _, err := os.Stat(tarballPath); err != nil {
		return errors.New(i18n.Tr("dev.fileNotFound", i18n.Args{"path": tarballPath}))
	}
	if !update && Installed() {
		return errors.New(i18n.Tr("agide.exists", i18n.Args{"dir": installDir}))
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home folder: %w", err)
	}

	workdir, err := os.MkdirTemp("", "antigravity-extract-*")
	if err != nil {
		return fmt.Errorf("create temporary folder: %w", err)
	}
	defer func() { _ = os.RemoveAll(workdir) }()

	userOpts := executor.Options{Stdout: stdout, Stderr: stdout}

	ui.Info(stdout, i18n.Tr("agide.run.extract", i18n.Args{"file": tarballPath}))
	if err := exe.Run(ctx, userOpts, "tar", "-xzf", tarballPath, "-C", workdir); err != nil {
		return fmt.Errorf("extract tarball: %w", err)
	}

	candidateDir, binName, err := findBinary(workdir)
	if err != nil {
		return err
	}
	if err := exe.Run(ctx, userOpts, "chmod", "+x", filepath.Join(candidateDir, binName)); err != nil {
		ui.Warning(stdout, i18n.Tr("agide.run.chmodFailed", i18n.Args{"error": err.Error()}))
	}

	if candidateDir == workdir {
		return i18n.NewError("agide.noRoot")
	}
	sandboxSum, err := fileSHA256(filepath.Join(candidateDir, "chrome-sandbox"))
	if err != nil {
		return fmt.Errorf("chrome-sandbox not found in the tarball: %w", err)
	}

	// One privileged batch (one pkexec prompt) that never trusts the
	// extracted tree past the password dialog: the tree is still writable by
	// the user while the dialog is open, so root copies it into a root-owned
	// staging dir, strips any setuid/setgid bit that came with the archive,
	// re-verifies chrome-sandbox against the checksum taken above before
	// making it setuid root, and only then swaps it in — keeping the previous
	// install until the new one is fully in place.
	sandbox := filepath.Join(installDir, "chrome-sandbox")
	steps := []executor.PrivilegedStep{{
		Announce: i18n.Tr("agide.run.install", i18n.Args{"dir": installDir}),
		Name:     "sh",
		Args:     []string{"-c", installScript, "sh", installDir, candidateDir, sandboxSum, sandboxPermWant},
	}}
	if err := exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, steps); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	if err := verifySandboxPerms(ctx, exe, sandbox); err != nil {
		ui.Warning(stdout, err.Error())
	}

	if iconPath, err := extractIcon(workdir); err != nil {
		ui.Warning(stdout, i18n.Tr("agide.run.iconFailed", i18n.Args{"error": err.Error()}))
	} else {
		installIcon(stdout, home, iconPath)
	}

	if err := writeDesktopEntry(home, binName); err != nil {
		ui.Warning(stdout, i18n.Tr("agide.run.menuFailed", i18n.Args{"error": err.Error()}))
	} else {
		ui.Info(stdout, i18n.Tr("agide.run.menuCreated", i18n.Args{"path": desktopFile(home)}))
	}

	refreshDesktopDB(ctx, exe, stdout, home)

	ui.Success(stdout, i18n.Tr("agide.run.installed", i18n.Args{"dir": installDir}))
	return nil
}

// installScript is doInstallOrUpdate's privileged batch. $1 = installDir,
// $2 = extracted app dir (user-owned), $3 = expected chrome-sandbox SHA-256,
// $4 = chrome-sandbox mode. Positional parameters keep every value out of
// the script text. The tree is copied into a staging dir root creates
// (0700), so a link put in place of $2 or of chrome-sandbox while the
// password dialog is open can never make chown, chmod or sha256sum act on
// a file outside it: $2 must be a directory, chrome-sandbox a regular file,
// and chown never follows links. The previous install is kept as $1.old
// until the new one is in place and is moved back when the final rename
// fails.
const installScript = `set -e
new="$1.new"; old="$1.old"
rm -rf -- "$new"
if [ ! -d "$2" ] || [ -L "$2" ]; then echo "The extracted app folder is not a directory." >&2; exit 1; fi
mkdir -m 0700 -- "$new"
cp -a --no-preserve=ownership -- "$2/." "$new/"
chown -hR root:root -- "$new"
chmod -R u-s,g-s -- "$new"
sb="$new/chrome-sandbox"
if [ ! -f "$sb" ] || [ -L "$sb" ]; then rm -rf -- "$new"; echo "chrome-sandbox is not a regular file." >&2; exit 1; fi
printf '%s  %s\n' "$3" "$sb" | sha256sum -c --status - || { rm -rf -- "$new"; echo "chrome-sandbox changed after it was verified" >&2; exit 1; }
chmod 0755 -- "$new"
chmod "$4" -- "$sb"
if [ -e "$1" ]; then rm -rf -- "$old"; mv -- "$1" "$old"; fi
if ! mv -- "$new" "$1"; then
  rm -rf -- "$new"
  if [ -e "$old" ]; then mv -- "$old" "$1"; fi
  echo "Failed to put the new version in place; the previous install was restored." >&2
  exit 1
fi
rm -rf -- "$old"`

// fileSHA256 returns the hex SHA-256 of the regular file at path.
func fileSHA256(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", path)
	}
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

// findBinary finds files whose name starts with "antigravity"
// (case-insensitive) up to 3 levels under workdir, excluding known
// non-binary extensions. When more than one candidate matches, prefers the
// shallowest one that has a "chrome-sandbox" sibling (the real Electron app
// root), falling back to the first candidate found.
func findBinary(workdir string) (dir, name string, err error) {
	var candidates []string
	walkErr := filepath.WalkDir(workdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort: unreadable entries are skipped
		}
		rel, relErr := filepath.Rel(workdir, path)
		if relErr != nil {
			return nil
		}
		depth := strings.Count(rel, string(filepath.Separator))
		if d.IsDir() {
			if depth >= 3 {
				return filepath.SkipDir
			}
			return nil
		}
		if depth > 3 {
			return nil
		}
		lower := strings.ToLower(d.Name())
		if !strings.HasPrefix(lower, "antigravity") {
			return nil
		}
		if !validBinName.MatchString(d.Name()) {
			return nil // name outside the expected charset — discard the candidate
		}
		for _, suf := range excludedSuffixes {
			if strings.HasSuffix(lower, suf) {
				return nil
			}
		}
		candidates = append(candidates, path)
		return nil
	})
	if walkErr != nil {
		return "", "", fmt.Errorf("walk the extracted files: %w", walkErr)
	}
	if len(candidates) == 0 {
		return "", "", i18n.NewError("agide.noBinary")
	}
	if len(candidates) == 1 {
		return filepath.Dir(candidates[0]), filepath.Base(candidates[0]), nil
	}

	best, bestDepth := "", -1
	for _, c := range candidates {
		dir := filepath.Dir(c)
		if _, err := os.Stat(filepath.Join(dir, "chrome-sandbox")); err != nil {
			continue
		}
		rel, _ := filepath.Rel(workdir, c)
		depth := strings.Count(rel, string(filepath.Separator))
		if best == "" || depth < bestDepth {
			best, bestDepth = c, depth
		}
	}
	if best == "" {
		best = candidates[0]
	}
	return filepath.Dir(best), filepath.Base(best), nil
}

// verifySandboxPerms checks that chrome-sandbox has the setuid bit (4755,
// applied by the RunSudoSequence batch in doInstallOrUpdate), without which
// Electron's sandboxed renderer process refuses to start. Read-only, no
// privilege needed.
func verifySandboxPerms(ctx context.Context, exe *executor.Executor, sandbox string) error {
	perms, err := exe.Output(ctx, executor.Options{}, "stat", "-c", "%a", sandbox)
	if err != nil {
		return fmt.Errorf("check the chrome-sandbox permissions: %w", err)
	}
	if strings.TrimSpace(perms) != sandboxPermWant {
		return errors.New("failed to set the setuid permissions of chrome-sandbox")
	}
	return nil
}

// extractIcon copies icon.png out of the installed app's Electron
// app.asar into workdir and returns its path.
func extractIcon(workdir string) (string, error) {
	data, err := readAsarFile(filepath.Join(installDir, "resources", "app.asar"), iconExtractedRel)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(workdir, iconExtractedRel)
	if err := os.WriteFile(dest, data, 0o600); err != nil {
		return "", err
	}
	return dest, nil
}

func installIcon(stdout io.Writer, home, extractedIconPath string) {
	dest := iconFile(home)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		ui.Warning(stdout, i18n.Tr("agide.run.iconDirFailed", i18n.Args{"error": err.Error()}))
		return
	}
	data, err := os.ReadFile(extractedIconPath)
	if err != nil {
		ui.Warning(stdout, i18n.Tr("agide.run.iconReadFailed", i18n.Args{"error": err.Error()}))
		return
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		ui.Warning(stdout, i18n.Tr("agide.run.iconInstallFailed", i18n.Args{"error": err.Error()}))
		return
	}
	ui.Info(stdout, i18n.Tr("agide.run.iconInstalled", i18n.Args{"path": dest}))
}

// writeDesktopEntry writes the application menu entry for binName.
func writeDesktopEntry(home, binName string) error {
	content := fmt.Sprintf(`[Desktop Entry]
Name=Antigravity
Comment=Google Antigravity IDE
Exec=%s %%U
Terminal=false
Type=Application
Icon=antigravity
Categories=Development;IDE;
StartupNotify=true
StartupWMClass=Antigravity
MimeType=x-scheme-handler/antigravity;
`, filepath.Join(installDir, binName))

	path := desktopFile(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o755)
}

// refreshDesktopDB refreshes the desktop and icon caches. Entirely
// best-effort: a missing update-desktop-database/gtk-update-icon-cache/
// desktop-file-validate is not fatal.
func refreshDesktopDB(ctx context.Context, exe *executor.Executor, stdout io.Writer, home string) {
	opts := executor.Options{Stdout: stdout, Stderr: stdout}
	_ = exe.Run(ctx, opts, "update-desktop-database", filepath.Join(home, ".local", "share", "applications"))
	_ = exe.Run(ctx, opts, "gtk-update-icon-cache", "-t", filepath.Join(home, ".local", "share", "icons", "hicolor"))
	if err := exe.Run(ctx, opts, "desktop-file-validate", desktopFile(home)); err != nil {
		ui.Warning(stdout, i18n.Tr("agide.run.desktopInvalid", i18n.Args{"error": err.Error()}))
	}
}

// Uninstall removes the installation, the desktop entry and the icon.
// User config and cache (~/.config/Antigravity, ~/.cache/Antigravity,
// ~/.config/antigravity-updater) are left in place.
func Uninstall(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	sudo := executor.Options{RequiresSudo: true, Stdout: stdout, Stderr: stdout}

	ui.Info(stdout, i18n.Tr("dev.removingDir", i18n.Args{"dir": installDir}))
	if err := exe.Run(ctx, sudo, "rm", "-rf", "--", installDir); err != nil {
		return fmt.Errorf("remove %s: %w", installDir, err)
	}

	if home, err := os.UserHomeDir(); err == nil {
		_ = os.Remove(desktopFile(home))
		_ = os.Remove(iconFile(home))
		refreshDesktopDB(ctx, exe, stdout, home)
	}

	ui.Success(stdout, i18n.Tr("agide.run.uninstalled", nil))
	return nil
}
