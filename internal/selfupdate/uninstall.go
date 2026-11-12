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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/ui"
	"github.com/oito2/perci82/packaging"
)

// ErrUninstalled is returned by Uninstall after the binary is successfully removed.
var ErrUninstalled = errors.New("aplicativo desinstalado")

// applicationsDir is where the installers put the application menu
// entries.
const applicationsDir = "/usr/share/applications"

// uninstallMenuPaths returns app's application menu entry and hicolor
// icons — a variable so tests never touch the real system paths.
var uninstallMenuPaths = func(app string) []string {
	paths := []string{filepath.Join(applicationsDir, packaging.IconName(app)+".desktop")}
	paths = append(paths, menuIconPaths(app)...)
	return append(paths,
		filepath.Join(hicolorDir, "scalable", "apps", packaging.IconName(app)+".svg"),
		filepath.Join(hicolorDir, "symbolic", "apps", packaging.IconName(app)+"-symbolic.svg"),
	)
}

// installDirs returns the folders where an installed Perci82 or Perci
// binary is looked for: the one holding the running binary and
// /usr/local/bin — a variable so tests never look at the real system.
var installDirs = func(exeDir string) []string {
	return []string{exeDir, "/usr/local/bin"}
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

// Uninstall removes app's binary, its application menu entry and icons
// and, with opts.RemoveConfig, its settings and data folders; ~/.perci
// itself goes only when no Perci82 application nor the old Perci is
// installed any more and no other setting is left. It returns
// ErrUninstalled once the binary is removed.
func Uninstall(ctx context.Context, exe *executor.Executor, stdout io.Writer, app string, opts UninstallOptions) error {
	if err := checkApp(app); err != nil {
		return err
	}
	currentExe, err := executablePath()
	if err != nil {
		return fmt.Errorf("localizar binário: %w", err)
	}

	ui.Info(stdout, "Removendo binário: "+currentExe)

	// Everything that cannot be removed without privilege is collected into
	// ONE privileged batch, so the password is asked at most once: the binary
	// (a failure there aborts the uninstall), then the menu entry/icons
	// (Soft: a failure only warns).
	var steps []executor.PrivilegedStep
	binaryNeedsRoot := false
	if err := os.Remove(currentExe); err != nil {
		ui.Info(stdout, "Não foi possível remover o binário diretamente ("+err.Error()+"). Será usado privilégio de administrador.")
		binaryNeedsRoot = true
		steps = append(steps, executor.PrivilegedStep{Name: "rm", Args: []string{"-f", "--", currentExe}})
	}

	// Application menu entry + icons, so no .desktop entry pointing at the
	// removed binary is left behind as a "ghost" in the menu. While the
	// binary still needs the privileged batch, nothing is removed directly:
	// a canceled password prompt then leaves the binary and its menu entry
	// together.
	var privileged []string
	for _, path := range uninstallMenuPaths(app) {
		if binaryNeedsRoot {
			if _, statErr := os.Lstat(path); statErr == nil {
				privileged = append(privileged, path)
			}
			continue
		}
		if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
			privileged = append(privileged, path)
		}
	}
	if len(privileged) > 0 {
		steps = append(steps,
			executor.PrivilegedStep{
				Soft:        true,
				WarnMessage: "Falha ao remover a entrada do menu de aplicativos.",
				Name:        "rm",
				Args:        append([]string{"-f", "--"}, privileged...),
			},
			iconCacheStep(), // drop the removed icons from the cache in the same prompt
		)
	}

	if len(steps) > 0 {
		if sudoErr := exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, steps); sudoErr != nil {
			if binaryNeedsRoot {
				return fmt.Errorf("remover binário: %w", sudoErr)
			}
			ui.Warning(stdout, "Falha ao remover itens que exigem privilégio: "+sudoErr.Error())
		}
	}

	if opts.RemoveConfig {
		removeSettings(stdout, app, opts.DataDirs, filepath.Dir(currentExe))
	}

	ui.Success(stdout, appName(app)+" desinstalado com sucesso.")
	return ErrUninstalled
}

// removeSettings removes app's keys from config.yaml and its data folders
// under ~/.perci, then ~/.perci itself when the config is left empty and
// no other Perci82 application nor the old Perci is installed. Failures
// only warn: the binary is already gone.
func removeSettings(stdout io.Writer, app string, dataDirs []string, exeDir string) {
	empty, err := config.RemoveOwnedKeys(config.App(app))
	if err != nil {
		ui.Warning(stdout, "Falha ao remover as configurações: "+err.Error())
		return
	}
	ui.Info(stdout, "Configurações do "+appName(app)+" removidas de ~/.perci/config.yaml.")

	home, err := os.UserHomeDir()
	if err != nil {
		ui.Warning(stdout, "Falha ao localizar a pasta pessoal: "+err.Error())
		return
	}
	for _, dir := range dataDirs {
		if dir == "" || !filepath.IsLocal(dir) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(home, ".perci", dir)); err != nil {
			ui.Warning(stdout, "Falha ao remover ~/.perci/"+dir+": "+err.Error())
			continue
		}
		ui.Info(stdout, "~/.perci/"+dir+" removida.")
	}

	if other := installedApp(app, exeDir); other != "" {
		ui.Info(stdout, "~/.perci mantida: o "+other+" continua instalado.")
		return
	}
	if !empty {
		ui.Info(stdout, "~/.perci mantida: o config.yaml ainda tem outras configurações.")
		return
	}
	if err := config.RemoveDir(); err != nil {
		ui.Warning(stdout, "Falha ao remover ~/.perci: "+err.Error())
		return
	}
	ui.Info(stdout, "~/.perci removida.")
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
		for _, c := range candidates {
			if _, err := os.Lstat(filepath.Join(dir, c.binary)); err == nil {
				return c.name
			}
		}
	}
	return ""
}
