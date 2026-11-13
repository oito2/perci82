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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/oito2/perci82/internal/appstack"
	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/distro"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	managerai "github.com/oito2/perci82/internal/manager/ai"
	"github.com/oito2/perci82/internal/manager/gitaccount"
	"github.com/oito2/perci82/internal/selfupdate"
	"github.com/oito2/perci82/internal/theme"
	"github.com/oito2/perci82/internal/ui"
	"github.com/oito2/perci82/internal/version"
	"github.com/oito2/perci82/packaging"
)

// homeCore holds the "Home" bindings both applications share: theme and
// language lists, version and self-update. SystemHomeService and
// DevHomeService embed it, so its exported methods are bound for both.
type homeCore struct {
	serviceBase
}

// SystemHomeService binds Perci82 System's "Home" category.
type SystemHomeService struct {
	homeCore
}

// DevHomeService binds Perci82 Dev's "Home" category. onLanguageChanged,
// set by Run when the tray is bound, receives the locale SetLanguage
// switched to.
type DevHomeService struct {
	homeCore
	onLanguageChanged func(locale string)
}

// GetSections returns Perci82 System's sections, labeled in its language.
func (h *SystemHomeService) GetSections() []Section {
	return systemSections(h.GetLanguage().Locale)
}

// GetSections returns Perci82 Dev's sections, labeled in its language,
// with one "Repositórios" item per registered Git account (an unreadable
// config lists none).
func (h *DevHomeService) GetSections() []Section {
	var accounts []config.GitAccount
	if cfg, err := config.Load(); err == nil {
		accounts = cfg.GitAccounts
	}
	return devSections(h.GetLanguage().Locale, accounts)
}

// GetTheme returns Perci82 System's theme choice (~/.perci/config.yaml,
// field system_gui_theme): "system" or a preset ID; anything else reads
// as "system".
func (h *SystemHomeService) GetTheme() string {
	cfg, err := config.Load()
	if err != nil {
		return theme.SystemChoice
	}
	return theme.ChoiceOrSystem(cfg.SystemGUITheme)
}

// SetTheme persists Perci82 System's theme choice (field
// system_gui_theme); an unknown value is saved as "system".
func (h *SystemHomeService) SetTheme(choice string) error {
	choice = theme.ChoiceOrSystem(choice)
	return config.Update(func(cfg *config.Config) error {
		cfg.SystemGUITheme = choice
		return nil
	})
}

// GetTheme returns Perci82 Dev's theme choice (~/.perci/config.yaml,
// field gui_theme): "system" or a preset ID; anything else reads as
// "system".
func (h *DevHomeService) GetTheme() string {
	cfg, err := config.Load()
	if err != nil {
		return theme.SystemChoice
	}
	return theme.ChoiceOrSystem(cfg.GUITheme)
}

// SetTheme persists Perci82 Dev's theme choice (field gui_theme); an
// unknown value is saved as "system".
func (h *DevHomeService) SetTheme(choice string) error {
	choice = theme.ChoiceOrSystem(choice)
	return config.Update(func(cfg *config.Config) error {
		cfg.GUITheme = choice
		return nil
	})
}

// LanguageState is an application's language: the stored choice and the
// locale it resolves to.
type LanguageState struct {
	// Choice is "system" or a locale.
	Choice string `json:"choice"`
	// Locale is the locale the frontend shows ("pt-BR" or "en").
	Locale string `json:"locale"`
}

// languageState returns the state of a stored language choice; an unknown
// value reads as "system".
func languageState(stored string) LanguageState {
	choice := i18n.ChoiceOrSystem(stored)
	return LanguageState{Choice: choice, Locale: i18n.Resolve(choice)}
}

// GetLanguageOptions returns the language choices: "system" (labeled by
// the frontend), then each supported language under its own name.
func (h *homeCore) GetLanguageOptions() []i18n.Option {
	return i18n.Options()
}

// GetLanguage returns Perci82 System's language (~/.perci/config.yaml,
// field system_language).
func (h *SystemHomeService) GetLanguage() LanguageState {
	cfg, err := config.Load()
	if err != nil {
		return languageState("")
	}
	return languageState(cfg.SystemLanguage)
}

// SetLanguage persists Perci82 System's language choice (field
// system_language; an unknown value is saved as "system"), makes the
// messages built in Go follow it and emits "language-changed" with the
// new state to every window.
func (h *SystemHomeService) SetLanguage(choice string) (LanguageState, error) {
	state := languageState(choice)
	if err := config.Update(func(cfg *config.Config) error {
		cfg.SystemLanguage = state.Choice
		return nil
	}); err != nil {
		return LanguageState{}, err
	}
	i18n.SetLocale(state.Locale)
	h.emit("language-changed", state)
	return state, nil
}

// GetLanguage returns Perci82 Dev's language (~/.perci/config.yaml, field
// language).
func (h *DevHomeService) GetLanguage() LanguageState {
	cfg, err := config.Load()
	if err != nil {
		return languageState("")
	}
	return languageState(cfg.Language)
}

// SetLanguage persists Perci82 Dev's language choice (field language; an
// unknown value is saved as "system"), makes the messages built in Go
// follow it, emits "language-changed" with the new state to every window
// and calls onLanguageChanged, when set, with the new locale.
func (h *DevHomeService) SetLanguage(choice string) (LanguageState, error) {
	state := languageState(choice)
	if err := config.Update(func(cfg *config.Config) error {
		cfg.Language = state.Choice
		return nil
	}); err != nil {
		return LanguageState{}, err
	}
	i18n.SetLocale(state.Locale)
	h.emit("language-changed", state)
	if h.onLanguageChanged != nil {
		h.onLanguageChanged(state.Locale)
	}
	return state, nil
}

// ThemeOption is one choice of the theme selector.
type ThemeOption struct {
	// ID is the value GetTheme/SetTheme use.
	ID string `json:"id"`
	// Label is the name shown to the user.
	Label string `json:"label"`
	// Scheme is the CSS scheme name of a preset in normal contrast; empty
	// for "system", which follows GetSystemScheme.
	Scheme string `json:"scheme"`
	// Dark reports whether a preset is dark.
	Dark bool `json:"dark"`
}

// GetThemeOptions returns the theme choices: "Definido pelo sistema",
// then the oito2 presets.
func (h *homeCore) GetThemeOptions() []ThemeOption {
	options := []ThemeOption{{ID: theme.SystemChoice, Label: "Definido pelo sistema"}}
	for _, p := range theme.Presets() {
		options = append(options, ThemeOption{ID: p.ID, Label: p.Label, Scheme: p.Scheme, Dark: p.Dark})
	}
	return options
}

// GetSystemScheme returns the scheme "Definido pelo sistema" resolves to
// from the desktop's current appearance settings; changes arrive in the
// "system-scheme-changed" event.
func (h *homeCore) GetSystemScheme(ctx context.Context) theme.SystemScheme {
	return readSystemSettings(ctx).Resolve()
}

// readSystemSettings reads the desktop's appearance settings — a variable
// so tests never depend on the machine's desktop.
var readSystemSettings = theme.ReadSystem

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

// selfUpdateInfo checks GitHub Releases for app's newest release
// (internal/selfupdate.LatestRelease) and compares it with the running
// version; RunSelfUpdate redoes the lookup before applying, never trusting
// this earlier check. A 10s timeout keeps a dead network from hanging the
// screen open.
func (h *homeCore) selfUpdateInfo(app string) SelfUpdateInfo {
	info := SelfUpdateInfo{CurrentVersion: version.Version}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rel, err := latestRelease(ctx, app)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	info.LatestVersion = rel.Version
	info.UpdateAvailable = selfupdate.IsNewer(rel.Version, version.Version)
	return info
}

// latestRelease finds an application's newest release — a variable so
// tests never reach GitHub.
var latestRelease = selfupdate.LatestRelease

// runSelfUpdate checks for and applies an update of app —
// internal/selfupdate.Run, which redoes the release lookup itself.
func (h *homeCore) runSelfUpdate(app string) error {
	return h.runAction(func(stdout io.Writer) error {
		return selfupdate.Run(context.Background(), h.exe, stdout, app)
	})
}

// GetSelfUpdateInfo reports the running and newest published versions of
// Perci82 System (tags system-v*).
func (h *SystemHomeService) GetSelfUpdateInfo() SelfUpdateInfo {
	return h.selfUpdateInfo(packaging.AppSystem)
}

// RunSelfUpdate updates Perci82 System to its newest release; the
// "Atualizar" button click is itself the confirmation.
func (h *SystemHomeService) RunSelfUpdate() error {
	return h.runSelfUpdate(packaging.AppSystem)
}

// GetSelfUpdateInfo reports the running and newest published versions of
// Perci82 Dev (tags dev-v*).
func (h *DevHomeService) GetSelfUpdateInfo() SelfUpdateInfo {
	return h.selfUpdateInfo(packaging.AppDev)
}

// RunSelfUpdate updates Perci82 Dev to its newest release; the
// "Atualizar" button click is itself the confirmation.
func (h *DevHomeService) RunSelfUpdate() error {
	return h.runSelfUpdate(packaging.AppDev)
}

// SystemInfo is the "Sistema" and "Ambiente" cards of Perci82 System's
// Visão Geral.
type SystemInfo struct {
	// System is the post-install profile's name of a supported system
	// ("ZorinOS 18.1 Lite"), otherwise the distribution's name and version;
	// "" when unknown.
	System string `json:"system"`
	// Desktop is the desktop environment's name ("GNOME"); "" when unknown.
	Desktop string `json:"desktop"`
}

// GetSystemInfo returns the detected system and desktop environment.
func (h *SystemHomeService) GetSystemInfo() SystemInfo {
	info := SystemInfo{Desktop: distro.DesktopName()}
	if profile := detectPostinstallProfile(); profile != nil {
		info.System = profile.Label
		return info
	}
	info.System = strings.TrimSpace(distro.DisplayName() + " " + distro.VersionID())
	return info
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

// RunSelfUninstall removes Perci82 System — its binary, menu entry and
// icons and, with removeConfig, its settings (selfupdate.Uninstall) — in
// the Execução tab; a successful uninstall then closes the window.
func (h *SystemHomeService) RunSelfUninstall(removeConfig bool) error {
	return h.runUninstall(func(w io.Writer) error {
		return uninstallApp(context.Background(), h.exe, w, packaging.AppSystem, selfupdate.UninstallOptions{RemoveConfig: removeConfig})
	})
}

// runUninstall runs body, an uninstall, like runAction (runMu
// serialization, action lock, panic recovery, "action-done") and closes
// the application once body reports selfupdate.ErrUninstalled, which
// counts as success: the window closes a short moment after the
// "action-done" event, time enough to read the success line in the
// terminal panel.
func (h *homeCore) runUninstall(body func(w io.Writer) error) error {
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
			err := body(w)
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
	return h.pickFolder(i18n.Tr("settings.workspace.pickTitle", nil))
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

// GetProjectNotesPath returns the persisted project notes folder, "" when
// unset or when the config can't be read.
func (h *DevHomeService) GetProjectNotesPath() string {
	cfg, err := config.Load()
	if err != nil {
		return ""
	}
	return cfg.ProjectNotesPath
}

// PickProjectNotesFolder opens the native folder picker for the project
// notes folder.
func (h *DevHomeService) PickProjectNotesFolder() (string, error) {
	return h.pickFolder(i18n.Tr("notes.pickTitle", nil))
}

// SetProjectNotesPath persists the project notes folder chosen via
// PickProjectNotesFolder, or clears it with "". Only a path picked in that
// dialog this session is accepted: the AI contexts give the coding agents
// write access to it.
func (h *DevHomeService) SetProjectNotesPath(path string) error {
	if path != "" {
		if err := requireApprovedPath(path); err != nil {
			return err
		}
	}
	return config.Update(func(cfg *config.Config) error {
		cfg.ProjectNotesPath = path
		return nil
	})
}

// PickUninstallBackupPath opens the native "Salvar Como" dialog for the
// uninstall's Docker settings backup — it differs from PickExportConfigPath
// only in its title/default filename; both save through the same
// appstack.ExportConfig underneath.
func (h *DevHomeService) PickUninstallBackupPath() (string, error) {
	return approvePath(h.wailsApp.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:    i18n.Tr("devuninstall.pickDocker", nil),
		Filename: "perci-backup-" + time.Now().Format("2006-01-02-150405") + ".yaml",
		Filters:  []application.FileFilter{{DisplayName: "YAML (*.yaml)", Pattern: "*.yaml;*.yml"}},
	}).PromptForSingleSelection())
}

// PickUninstallReposBackupPath opens the native "Salvar Como" dialog for
// the uninstall's repositories backup — the file "Repositórios ::
// Gerenciar Repositórios" imports.
func (h *DevHomeService) PickUninstallReposBackupPath() (string, error) {
	return approvePath(h.wailsApp.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:    i18n.Tr("devuninstall.pickRepos", nil),
		Filename: "perci-repositorios-" + time.Now().Format("2006-01-02-150405") + ".yaml",
		Filters:  []application.FileFilter{{DisplayName: "YAML (*.yaml)", Pattern: "*.yaml;*.yml"}},
	}).PromptForSingleSelection())
}

// PickUninstallAIContextsBackupPath opens the native "Salvar Como" dialog
// for the uninstall's AI contexts backup — the file "Contextos de IA ::
// Exportar/Importar" imports.
func (h *DevHomeService) PickUninstallAIContextsBackupPath() (string, error) {
	return approvePath(h.wailsApp.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:    i18n.Tr("devuninstall.pickAI", nil),
		Filename: "perci-contextos-ia-" + time.Now().Format("2006-01-02-150405") + ".yaml",
		Filters:  []application.FileFilter{{DisplayName: "YAML (*.yaml)", Pattern: "*.yaml;*.yml"}},
	}).PromptForSingleSelection())
}

// RunSelfUninstall removes Perci82 Dev — its binary, menu entry and icons
// (selfupdate.Uninstall). Four steps run first, all optional:
//
//   - backupPath != "": exports cfg.Docker and the Moodle plugin mounts to
//     that path first (appstack.ExportConfig) — reimportable later via
//     "Docker :: Exportar/Importar".
//   - reposBackupPath != "": exports the Git accounts and their
//     repositories to that path (gitaccount.Exportable/WriteTransfer) —
//     reimportable via "Repositórios :: Gerenciar Repositórios".
//   - aiBackupPath != "": exports the registered AI context folders to
//     that path (managerai.Exportable/WriteTransfer) — reimportable via
//     "Contextos de IA"; the context files in those folders are never
//     touched by the uninstall.
//   - removeDocker: removes every registered container (apps, then
//     MariaDB, then Nginx — apps first so nothing routes through infra
//     that's about to disappear) via appstack.DeleteApp/DeleteMariaDB/
//     DeleteNginx — disk data (workspace/localhost/...) is left
//     untouched.
//
// A failed backup aborts the uninstall before anything is removed; only
// "nothing to export" is a warning.
//
// With removeConfig, Perci82 Dev's settings and its folders under ~/.perci
// (devDataDirs) go too, and, once the binary is removed, the includeIf
// rules, gitconfig files and API tokens of the Git accounts
// (gitaccount.Forget); their SSH keys and repositories stay.
//
// A successful uninstall closes the window (runUninstall).
func (h *DevHomeService) RunSelfUninstall(removeConfig, removeDocker bool, backupPath, reposBackupPath, aiBackupPath string) error {
	// A backup path not chosen in its picker refuses the whole uninstall —
	// checked before anything destructive starts.
	for _, p := range []string{backupPath, reposBackupPath, aiBackupPath} {
		if p != "" {
			if err := requireApprovedPath(p); err != nil {
				return err
			}
		}
	}
	return h.runUninstall(func(w io.Writer) error {
		return h.selfUninstall(w, removeConfig, removeDocker, backupPath, reposBackupPath, aiBackupPath)
	})
}

// uninstallQuitDelay leaves the success line readable in the terminal
// panel before the window closes.
const uninstallQuitDelay = 1500 * time.Millisecond

// uninstallApp removes an application (selfupdate.Uninstall) — a variable
// so tests never remove the test binary.
var uninstallApp = selfupdate.Uninstall

// devDataDirs returns the folders under ~/.perci, only Perci82 Dev
// writes, that go with its settings: the Git accounts' gitconfig files
// always, and the Docker stack files only when the containers go too,
// since they still mount them. The MariaDB dumps in appstack/backups are
// the user's data and always stay; w tells where.
func devDataDirs(w io.Writer, removeDocker bool) []string {
	dirs := []string{"git"}
	if !removeDocker {
		ui.Info(w, i18n.Tr("devuninstall.appstackKept", nil))
		return dirs
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return dirs
	}
	appstackDir := filepath.Join(home, ".perci", "appstack")
	backups := filepath.Join(appstackDir, "backups")
	if _, err := os.Lstat(backups); err != nil {
		return append(dirs, "appstack")
	}
	entries, err := os.ReadDir(appstackDir)
	if err != nil {
		return dirs
	}
	for _, e := range entries {
		if e.Name() != "backups" {
			dirs = append(dirs, filepath.Join("appstack", e.Name()))
		}
	}
	ui.Info(w, i18n.Tr("devuninstall.backupsKept", i18n.Args{"path": backups}))
	return dirs
}

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
				return fmt.Errorf("%s: %w", i18n.Tr("devuninstall.backupFailed.docker", nil), err)
			}
			ui.Warning(w, i18n.Tr("devuninstall.nothing.docker", i18n.Args{"error": err.Error()}))
		} else {
			ui.Success(w, i18n.Tr("devuninstall.saved.docker", i18n.Args{"path": backupPath}))
		}
	}
	if reposBackupPath != "" {
		if err := backupRepositories(ctx, h.exe, reposBackupPath); err != nil {
			if !errors.Is(err, gitaccount.ErrNothingToExport) {
				return fmt.Errorf("%s: %w", i18n.Tr("devuninstall.backupFailed.repos", nil), err)
			}
			ui.Warning(w, i18n.Tr("devuninstall.nothing.repos", i18n.Args{"error": err.Error()}))
		} else {
			ui.Success(w, i18n.Tr("devuninstall.saved.repos", i18n.Args{"path": reposBackupPath}))
		}
	}
	if aiBackupPath != "" {
		if err := backupAIContexts(aiBackupPath); err != nil {
			if !errors.Is(err, managerai.ErrNothingToExport) {
				return fmt.Errorf("%s: %w", i18n.Tr("devuninstall.backupFailed.ai", nil), err)
			}
			ui.Warning(w, i18n.Tr("devuninstall.nothing.ai", i18n.Args{"error": err.Error()}))
		} else {
			ui.Success(w, i18n.Tr("devuninstall.saved.ai", i18n.Args{"path": aiBackupPath}))
		}
	}

	if removeDocker {
		if cfg, err := config.Load(); err == nil {
			for _, app := range cfg.Docker.Apps {
				if delErr := appstack.DeleteApp(ctx, h.exe, w, app.Folder); delErr != nil {
					ui.Warning(w, i18n.Tr("devuninstall.containerFailed", i18n.Args{"name": app.Folder, "error": delErr.Error()}))
				}
			}
			if delErr := appstack.DeleteMariaDB(ctx, h.exe, w); delErr != nil {
				ui.Warning(w, i18n.Tr("devuninstall.containerFailed", i18n.Args{"name": "MariaDB", "error": delErr.Error()}))
			}
			if delErr := appstack.DeleteNginx(ctx, h.exe, w); delErr != nil {
				ui.Warning(w, i18n.Tr("devuninstall.containerFailed", i18n.Args{"name": "Nginx", "error": delErr.Error()}))
			}
		} else {
			ui.Warning(w, i18n.Tr("devuninstall.dockerConfigFailed", i18n.Args{"error": err.Error()}))
		}
	}

	// The accounts are read before ~/.perci goes away with removeConfig.
	var accounts []config.GitAccount
	if removeConfig {
		if cfg, err := config.Load(); err == nil {
			accounts = cfg.GitAccounts
		} else {
			ui.Warning(w, i18n.Tr("devuninstall.accountsFailed", i18n.Args{"error": err.Error()}))
		}
	}

	var dataDirs []string
	if removeConfig {
		dataDirs = devDataDirs(w, removeDocker)
	}
	err := uninstallApp(ctx, h.exe, w, packaging.AppDev, selfupdate.UninstallOptions{RemoveConfig: removeConfig, DataDirs: dataDirs})
	if errors.Is(err, selfupdate.ErrUninstalled) {
		if fErr := gitaccount.Forget(ctx, h.exe, w, accounts); fErr != nil {
			ui.Warning(w, i18n.Tr("devuninstall.forgetFailed", i18n.Args{"error": fErr.Error()}))
		}
		// Tray autostart (~/.config/autostart/io.github.oito2.perci82.dev.desktop) — removed regardless
		// of TrayEnabled: with the binary gone, it would be a "ghost" autostart
		// entry (same class of problem as the app-menu .desktop entry, handled
		// inside selfupdate.Uninstall).
		if rmErr := removeAutostartDesktopFile(); rmErr != nil {
			ui.Warning(w, i18n.Tr("devuninstall.autostartFailed", i18n.Args{"error": rmErr.Error()}))
		}
	}
	return err
}

// RepositorySummary counts the registered Git accounts and the
// repositories found in their folders.
type RepositorySummary struct {
	Accounts int `json:"accounts"`
	Repos    int `json:"repos"`
}

// GetRepositorySummary counts the Git accounts and the repositories under
// their folders, without running git or reaching the network. A folder
// that cannot be read counts no repository.
func (h *DevHomeService) GetRepositorySummary(ctx context.Context) (RepositorySummary, error) {
	cfg, err := config.Load()
	if err != nil {
		return RepositorySummary{}, err
	}
	sum := RepositorySummary{Accounts: len(cfg.GitAccounts)}
	for _, acc := range cfg.GitAccounts {
		if dirs, err := gitaccount.Discover(ctx, acc.Folder); err == nil {
			sum.Repos += len(dirs)
		}
	}
	return sum, nil
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
// the format of "Contextos de IA :: Exportar/Importar" (folders inside the
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
