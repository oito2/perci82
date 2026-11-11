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

package gui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/oito2/perci82/internal/appstack"
	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
	managerai "github.com/oito2/perci82/internal/manager/ai"
	"github.com/oito2/perci82/internal/manager/gitaccount"
	"github.com/oito2/perci82/internal/selfupdate"
	"github.com/oito2/perci82/internal/ui"
	"github.com/oito2/perci82/internal/version"
)

// errSelfManagementUnavailable is returned by the self-update and
// uninstall bindings while selfManagementEnabled is false.
var errSelfManagementUnavailable = errors.New("a atualização e a desinstalação pelo aplicativo ainda não estão disponíveis nesta versão")

// selfManagementEnabled turns on RunSelfUpdate, GetSelfUpdateInfo's
// release lookup and DevHomeService.RunSelfUninstall. It is false in
// builds of this version; tests switch it on.
var selfManagementEnabled = false

// homeCore holds the "Home" bindings both applications share: theme
// list, version, self-update and sidebar logo. SystemHomeService and
// DevHomeService embed it, so its exported methods are bound for both.
type homeCore struct {
	serviceBase
}

// SystemHomeService binds Perci82 System's "Home" category.
type SystemHomeService struct {
	homeCore
}

// DevHomeService binds Perci82 Dev's "Home" category.
type DevHomeService struct {
	homeCore
}

// GetCategories returns Perci82 System's sidebar categories and items.
func (h *SystemHomeService) GetCategories() []MenuCategory {
	return systemCategories()
}

// GetCategories returns Perci82 Dev's sidebar categories and items, with
// one item per registered Git account (an unreadable config lists none).
func (h *DevHomeService) GetCategories() []MenuCategory {
	var accounts []config.GitAccount
	if cfg, err := config.Load(); err == nil {
		accounts = cfg.GitAccounts
	}
	return devCategories(accounts)
}

// GetTheme returns Perci82 System's daisyUI theme (~/.perci/config.yaml,
// field system_gui_theme), falling back to the default when unset or
// unrecognized.
func (h *SystemHomeService) GetTheme() string {
	cfg, err := config.Load()
	if err != nil {
		return config.DefaultGUITheme
	}
	return cfg.SystemGUIThemeOrDefault()
}

// SetTheme persists Perci82 System's theme (field system_gui_theme).
func (h *SystemHomeService) SetTheme(name string) error {
	name = oneOf(name, config.GUIThemes(), config.DefaultGUITheme)
	return config.Update(func(cfg *config.Config) error {
		cfg.SystemGUITheme = name
		return nil
	})
}

// GetTheme returns Perci82 Dev's daisyUI theme (~/.perci/config.yaml,
// field gui_theme), falling back to the default when unset or
// unrecognized.
func (h *DevHomeService) GetTheme() string {
	cfg, err := config.Load()
	if err != nil {
		return config.DefaultGUITheme
	}
	return cfg.GUIThemeOrDefault()
}

// SetTheme persists Perci82 Dev's theme (field gui_theme).
func (h *DevHomeService) SetTheme(name string) error {
	name = oneOf(name, config.GUIThemes(), config.DefaultGUITheme)
	return config.Update(func(cfg *config.Config) error {
		cfg.GUITheme = name
		return nil
	})
}

// GetGUIThemes returns the fixed list of daisyUI themes the GUI offers, so
// the frontend's theme selector doesn't have to duplicate that list by
// hand.
func (h *homeCore) GetGUIThemes() []string {
	return config.GUIThemes()
}

// GetAppVersion returns the running binary's own version (internal/version)
// with no network access — used by "Home :: Visão Geral" (dashboard), which
// must stay useful offline, unlike GetSelfUpdateInfo below.
func (h *homeCore) GetAppVersion() string {
	return version.Version
}

// SelfUpdateInfo is what "Home :: Atualizar Perci" needs to render: the
// installed version, the latest one found on GitHub Releases, and whether
// they differ — or Error when the check itself failed (e.g. no network).
type SelfUpdateInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion,omitempty"`
	UpdateAvailable bool   `json:"updateAvailable"`
	Error           string `json:"error,omitempty"`
}

// GetSelfUpdateInfo checks GitHub Releases (internal/selfupdate.
// LatestRelease) and compares against the running version; RunSelfUpdate
// redoes the lookup before applying, never trusting this earlier check. A
// 10s timeout keeps a dead network from hanging the screen open
// indefinitely (LatestRelease itself has no timeout of its own). While
// selfManagementEnabled is false it does no lookup and reports
// errSelfManagementUnavailable in Error.
func (h *homeCore) GetSelfUpdateInfo() SelfUpdateInfo {
	info := SelfUpdateInfo{CurrentVersion: version.Version}
	if !selfManagementEnabled {
		info.Error = errSelfManagementUnavailable.Error()
		return info
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rel, err := selfupdate.LatestRelease(ctx)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	info.LatestVersion = rel.Version
	info.UpdateAvailable = selfupdate.IsNewer(rel.Version, version.Version)
	return info
}

// RunSelfUpdate checks for and applies an update — internal/selfupdate.Run.
// The GUI's "Atualizar agora" button click is itself the confirmation.
// Recalculates the latest release itself instead of trusting
// GetSelfUpdateInfo's earlier result. Returns errSelfManagementUnavailable
// while selfManagementEnabled is false.
func (h *homeCore) RunSelfUpdate() error {
	if !selfManagementEnabled {
		return errSelfManagementUnavailable
	}
	return h.runAction(func(stdout io.Writer) error {
		return selfupdate.Run(context.Background(), h.exe, stdout)
	})
}

// GetFlatpakScope returns the persisted Flatpak install scope, "user" or
// "system" (also for an unset scope or an unreadable config).
func (h *SystemHomeService) GetFlatpakScope() string {
	cfg, err := config.Load()
	if err != nil || cfg.FlatpakScope != "user" {
		return "system"
	}
	return "user"
}

// SetFlatpakScope persists the Flatpak install scope — applies instantly
// on select, synchronously like SetWorkspacePath.
func (h *SystemHomeService) SetFlatpakScope(scope string) error {
	if scope != "user" {
		scope = "system"
	}
	return config.Update(func(cfg *config.Config) error {
		cfg.FlatpakScope = scope
		return nil
	})
}

// RunSelfUninstall always returns errSelfManagementUnavailable: Perci82
// System has no uninstall of its own yet.
func (h *SystemHomeService) RunSelfUninstall(removeConfig bool) error {
	return errSelfManagementUnavailable
}

// GetWorkspacePath returns the persisted workspace folder, "" when unset
// or when the config can't be read.
func (h *DevHomeService) GetWorkspacePath() string {
	cfg, err := config.Load()
	if err != nil {
		return ""
	}
	return cfg.WorkspacePath
}

// PickWorkspaceFolder opens the native folder picker for the workspace
// path, with CanCreateDirectories enabled.
func (h *DevHomeService) PickWorkspaceFolder() (string, error) {
	return h.pickFolder("Selecionar pasta do workspace")
}

// SetWorkspacePath persists the workspace folder path chosen via
// PickWorkspaceFolder (expanding a leading ~). Synchronous, no Execução
// tab involved. Always reloads the config first: don't clobber other
// fields (e.g. Docker containers) another process may have changed while
// the GUI was open.
//
// Only a path picked in that dialog this session is accepted
// (requireApprovedPath): Docker creates folders under the workspace and
// mounts it into containers, so the webview must not be able to point it
// anywhere it likes.
func (h *DevHomeService) SetWorkspacePath(path string) error {
	if expanded, expandErr := config.ExpandPath(path); expandErr == nil {
		path = expanded
	}
	if err := requireApprovedPath(path); err != nil {
		return err
	}
	return config.Update(func(cfg *config.Config) error {
		cfg.WorkspacePath = path
		return nil
	})
}

// GetSidebarLogo returns the sidebar mascot variant ("blue"/"pink") an
// earlier Perci version saved, the default when unset or unreadable.
func (h *homeCore) GetSidebarLogo() string {
	cfg, err := config.Load()
	if err != nil {
		return config.DefaultSidebarLogo
	}
	return cfg.SidebarLogoOrDefault()
}

// PickUninstallBackupPath opens the native "Salvar Como" dialog for the
// uninstall's "Fazer backup das configurações" checkbox — it differs from
// PickExportConfigPath only in its title/default filename; both save
// through the same appstack.ExportConfig underneath.
func (h *DevHomeService) PickUninstallBackupPath() (string, error) {
	return approvePath(h.wailsApp.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:    "Salvar backup das configurações",
		Filename: "perci-backup-" + time.Now().Format("2006-01-02-150405") + ".yaml",
		Filters:  []application.FileFilter{{DisplayName: "YAML (*.yaml)", Pattern: "*.yaml;*.yml"}},
	}).PromptForSingleSelection())
}

// PickUninstallReposBackupPath opens the native "Salvar Como" dialog for
// the uninstall's "Fazer backup dos repositórios" checkbox — the file
// "Repositórios :: Gerenciar Repositórios" imports.
func (h *DevHomeService) PickUninstallReposBackupPath() (string, error) {
	return approvePath(h.wailsApp.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:    "Salvar backup dos repositórios",
		Filename: "perci-repositorios-" + time.Now().Format("2006-01-02-150405") + ".yaml",
		Filters:  []application.FileFilter{{DisplayName: "YAML (*.yaml)", Pattern: "*.yaml;*.yml"}},
	}).PromptForSingleSelection())
}

// PickUninstallAIContextsBackupPath opens the native "Salvar Como" dialog
// for the uninstall's "Fazer backup dos contextos de IA" checkbox — the
// file "Dev Tools :: IA: Contextos" imports.
func (h *DevHomeService) PickUninstallAIContextsBackupPath() (string, error) {
	return approvePath(h.wailsApp.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:    "Salvar backup dos contextos de IA",
		Filename: "perci-contextos-ia-" + time.Now().Format("2006-01-02-150405") + ".yaml",
		Filters:  []application.FileFilter{{DisplayName: "YAML (*.yaml)", Pattern: "*.yaml;*.yml"}},
	}).PromptForSingleSelection())
}

// RunSelfUninstall removes the Perci binary — internal/selfupdate.Uninstall.
// Four steps run first, all optional:
//
//   - backupPath != "": exports cfg.Docker and the Moodle plugin mounts to
//     that path first (appstack.ExportConfig) — reimportable later via
//     "Docker :: Exportar/Importar".
//   - reposBackupPath != "": exports the Git accounts and their
//     repositories to that path (gitaccount.Exportable/WriteTransfer) —
//     reimportable via "Repositórios :: Gerenciar Repositórios".
//   - aiBackupPath != "": exports the registered AI context folders to
//     that path (managerai.Exportable/WriteTransfer) — reimportable via
//     "Dev Tools :: IA: Contextos"; the context files in those folders are
//     never touched by the uninstall.
//   - removeDocker: removes every registered container (apps, then
//     MariaDB, then Nginx — apps first so nothing routes through infra
//     that's about to disappear) via appstack.DeleteApp/DeleteMariaDB/
//     DeleteNginx — disk data (workspace/localhost/...) is left
//     untouched.
//
// A failed backup aborts the uninstall before anything is removed; only
// "nothing to export" is a warning.
//
// With removeConfig, once the binary is removed, the includeIf rules,
// gitconfig files and API tokens of the Git accounts go too
// (gitaccount.Forget); their SSH keys and repositories stay.
//
// Doesn't go through runAction: a successful uninstall must also close the
// GUI window, and runAction's generic "ok/error" event has no notion of
// that, so the goroutine/event/quit sequence is inlined here — with the
// same runMu serialization, panic recovery and emitDone.
// ErrUninstalled is success, not a failure, so it's translated to a nil
// error (ok:true) before the "action-done" event is sent; the window
// closes a short moment later (time enough to read the success line in
// the terminal panel). Returns errSelfManagementUnavailable, before
// anything else, while selfManagementEnabled is false.
func (h *DevHomeService) RunSelfUninstall(removeConfig, removeDocker bool, backupPath, reposBackupPath, aiBackupPath string) error {
	if !selfManagementEnabled {
		return errSelfManagementUnavailable
	}
	// A backup path not chosen in its picker refuses the whole uninstall —
	// checked before anything destructive starts.
	for _, p := range []string{backupPath, reposBackupPath, aiBackupPath} {
		if p != "" {
			if err := requireApprovedPath(p); err != nil {
				return err
			}
		}
	}
	if !runMu.TryLock() {
		return errActionRunning
	}
	go func() {
		defer runMu.Unlock()
		beginAction()
		defer endAction()
		w := h.newEventWriter()
		quit := false
		err := runRecovered(func() error {
			err := h.selfUninstall(w, removeConfig, removeDocker, backupPath, reposBackupPath, aiBackupPath)
			if errors.Is(err, selfupdate.ErrUninstalled) {
				quit = true
				return nil
			}
			return err
		})
		w.Flush()
		h.emitDone(err)
		if quit {
			time.Sleep(uninstallQuitDelay)
			quitAllowed.Store(true)
			h.wailsApp.Quit()
		}
	}()
	return nil
}

// uninstallQuitDelay leaves the success line readable in the terminal
// panel before the window closes.
const uninstallQuitDelay = 1500 * time.Millisecond

// uninstallPerci removes the binary (and optionally ~/.perci) — a variable
// so tests never remove the test binary.
var uninstallPerci = selfupdate.Uninstall

// selfUninstall is RunSelfUninstall's body: optional backups, optional
// Docker cleanup, then selfupdate.Uninstall (whose ErrUninstalled means
// success). The tray's autostart entry — and, with removeConfig, the Git
// accounts' rules and tokens — are removed only once the uninstall itself
// succeeded.
func (h *DevHomeService) selfUninstall(w io.Writer, removeConfig, removeDocker bool, backupPath, reposBackupPath, aiBackupPath string) error {
	ctx := context.Background()

	if backupPath != "" {
		if err := appstack.ExportConfig(backupPath); err != nil {
			if !errors.Is(err, appstack.ErrNothingToExport) {
				return fmt.Errorf("o backup das configurações falhou, nada foi removido: %w", err)
			}
			ui.Warning(w, "Nada para exportar no backup: "+err.Error())
		} else {
			ui.Success(w, "Backup salvo em "+backupPath+".")
		}
	}
	if reposBackupPath != "" {
		if err := backupRepositories(ctx, h.exe, reposBackupPath); err != nil {
			if !errors.Is(err, gitaccount.ErrNothingToExport) {
				return fmt.Errorf("o backup dos repositórios falhou, nada foi removido: %w", err)
			}
			ui.Warning(w, "Nada para exportar no backup dos repositórios: "+err.Error())
		} else {
			ui.Success(w, "Backup dos repositórios salvo em "+reposBackupPath+".")
		}
	}
	if aiBackupPath != "" {
		if err := backupAIContexts(aiBackupPath); err != nil {
			if !errors.Is(err, managerai.ErrNothingToExport) {
				return fmt.Errorf("o backup dos contextos de IA falhou, nada foi removido: %w", err)
			}
			ui.Warning(w, "Nada para exportar no backup dos contextos de IA: "+err.Error())
		} else {
			ui.Success(w, "Backup dos contextos de IA salvo em "+aiBackupPath+".")
		}
	}

	if removeDocker {
		if cfg, err := config.Load(); err == nil {
			for _, app := range cfg.Docker.Apps {
				if delErr := appstack.DeleteApp(ctx, h.exe, w, app.Folder); delErr != nil {
					ui.Warning(w, "Falha ao remover contêiner "+app.Folder+": "+delErr.Error())
				}
			}
			if delErr := appstack.DeleteMariaDB(ctx, h.exe, w); delErr != nil {
				ui.Warning(w, "Falha ao remover contêiner MariaDB: "+delErr.Error())
			}
			if delErr := appstack.DeleteNginx(ctx, h.exe, w); delErr != nil {
				ui.Warning(w, "Falha ao remover contêiner Nginx: "+delErr.Error())
			}
		} else {
			ui.Warning(w, "Falha ao carregar configuração para remover contêineres Docker: "+err.Error())
		}
	}

	// The accounts are read before ~/.perci goes away with removeConfig.
	var accounts []config.GitAccount
	if removeConfig {
		if cfg, err := config.Load(); err == nil {
			accounts = cfg.GitAccounts
		} else {
			ui.Warning(w, "Falha ao carregar as contas Git: "+err.Error())
		}
	}

	err := uninstallPerci(ctx, h.exe, w, removeConfig)
	if errors.Is(err, selfupdate.ErrUninstalled) {
		if fErr := gitaccount.Forget(ctx, h.exe, w, accounts); fErr != nil {
			ui.Warning(w, "Falha ao remover as regras das contas Git: "+fErr.Error())
		}
		// Tray autostart (~/.config/autostart/perci.desktop) — removed regardless
		// of TrayEnabled: with the binary gone, it would be a "ghost" autostart
		// entry (same class of problem as the app-menu .desktop entry, handled
		// inside selfupdate.Uninstall).
		if rmErr := removeAutostartDesktopFile(); rmErr != nil {
			ui.Warning(w, "Falha ao remover autostart da bandeja: "+rmErr.Error())
		}
	}
	return err
}

// backupRepositories writes every Git account and its repositories to
// path, in the format of "Repositórios :: Gerenciar Repositórios".
func backupRepositories(ctx context.Context, exe *executor.Executor, path string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	file, _, err := gitaccount.Exportable(ctx, exe, cfg.GitAccounts)
	if err != nil {
		return err
	}
	return gitaccount.WriteTransfer(path, file)
}

// backupAIContexts writes every registered AI context folder to path, in
// the format of "Dev Tools :: IA: Contextos" (folders inside the
// workspace with a path relative to it).
func backupAIContexts(path string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ws, err := cfg.Workspace()
	if err != nil {
		return err
	}
	return managerai.WriteTransfer(path, managerai.Exportable(cfg.AIContexts, ws))
}
