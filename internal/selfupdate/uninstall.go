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
	"io"
	"os"
	"path/filepath"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// installDirs returns the folders where an installed Perci82 or Perci
// binary is looked for: the one holding the running binary, /usr/bin and
// /usr/local/bin — a variable so tests never look at the real system.
var installDirs = func(exeDir string) []string {
	return []string{exeDir, packagedBinDir, "/usr/local/bin"}
}

// UninstallOptions are the choices of an uninstall.
type UninstallOptions struct {
	// RemoveConfig removes the application's own settings from
	// ~/.perci/config.yaml and its DataDirs.
	RemoveConfig bool
	// DataDirs are folders under ~/.perci only this application writes,
	// removed with RemoveConfig.
	DataDirs []string
}

// Uninstall removes, with opts.RemoveConfig, app's settings and data
// folders; ~/.perci itself goes only when no other Perci82 application nor
// the old Perci is installed and no other setting is left. The binary,
// menu entry and icons belong to app's package: Uninstall ends by printing
// the command that removes it (RemoveCommand).
func Uninstall(stdout io.Writer, app string, opts UninstallOptions) error {
	if err := checkApp(app); err != nil {
		return err
	}
	if opts.RemoveConfig {
		exeDir := ""
		if exe, err := executablePath(); err == nil {
			exeDir = filepath.Dir(exe)
		}
		removeSettings(stdout, app, opts.DataDirs, exeDir)
	}
	ui.Success(stdout, i18n.Tr("selfuninstall.done", i18n.Args{"app": appName(app)}))
	if cmd := RemoveCommand(app); cmd != "" {
		ui.Info(stdout, i18n.Tr("selfuninstall.removePackage", i18n.Args{"command": cmd}))
	}
	return nil
}

// removeSettings removes app's keys from config.yaml and its data folders
// under ~/.perci, then ~/.perci itself when the config is left empty and
// no other Perci82 application nor the old Perci is installed. Failures
// only warn.
func removeSettings(stdout io.Writer, app string, dataDirs []string, exeDir string) {
	empty, err := config.RemoveOwnedKeys(config.App(app))
	if err != nil {
		ui.Warning(stdout, i18n.Tr("selfuninstall.settingsFailed", i18n.Args{"error": err.Error()}))
		return
	}
	ui.Info(stdout, i18n.Tr("selfuninstall.settingsRemoved", i18n.Args{"app": appName(app)}))

	home, err := os.UserHomeDir()
	if err != nil {
		ui.Warning(stdout, i18n.Tr("selfuninstall.homeFailed", i18n.Args{"error": err.Error()}))
		return
	}
	for _, dir := range dataDirs {
		if dir == "" || !filepath.IsLocal(dir) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(home, ".perci", dir)); err != nil {
			ui.Warning(stdout, i18n.Tr("selfuninstall.dirFailed", i18n.Args{"dir": "~/.perci/" + dir, "error": err.Error()}))
			continue
		}
		ui.Info(stdout, i18n.Tr("selfuninstall.dirRemoved", i18n.Args{"dir": "~/.perci/" + dir}))
	}

	if other := installedApp(app, exeDir); other != "" {
		ui.Info(stdout, i18n.Tr("selfuninstall.keptOther", i18n.Args{"app": other}))
		return
	}
	if !empty {
		ui.Info(stdout, i18n.Tr("selfuninstall.keptConfig", nil))
		return
	}
	if err := config.RemoveDir(); err != nil {
		ui.Warning(stdout, i18n.Tr("selfuninstall.dirFailed", i18n.Args{"dir": "~/.perci", "error": err.Error()}))
		return
	}
	ui.Info(stdout, i18n.Tr("selfuninstall.dirRemoved", i18n.Args{"dir": "~/.perci"}))
}

// installedApp returns the name of another installed application that
// shares ~/.perci (the other Perci82 application or the old Perci), or ""
// when none is found in installDirs.
func installedApp(app, exeDir string) string {
	type candidate struct{ binary, name string }
	var candidates []candidate
	for _, other := range Apps {
		if other != app {
			candidates = append(candidates, candidate{BinaryName(other), appName(other)})
		}
	}
	candidates = append(candidates, candidate{"prci", "Perci"})
	for _, dir := range installDirs(exeDir) {
		if dir == "" {
			continue
		}
		for _, c := range candidates {
			if _, err := os.Lstat(filepath.Join(dir, c.binary)); err == nil {
				return c.name
			}
		}
	}
	return ""
}
