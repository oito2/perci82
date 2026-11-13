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

// Bound methods for "Docker :: Pastas de Desenvolvimento" — a thin layer
// over the development folder mounts of internal/appstack.

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/oito2/perci82/internal/appstack"
	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/i18n"
)

// DevToolsService binds Pastas de Desenvolvimento,
// IA: Contextos, IA: SKILLs, IA: MCPs — one struct, with one group of
// methods per sub-screen.
type DevToolsService struct {
	serviceBase
}

// DevFolderApp is one app container offered by the screen. Type is the
// app's config.AppType*; Root is where a destination is relative to
// (the app's project root on the host, relative to the workspace).
type DevFolderApp struct {
	Folder string `json:"folder"`
	Label  string `json:"label"`
	Type   string `json:"type"`
	Root   string `json:"root"`
}

// DevFolderRow is one mounted folder. Component is the Moodle plugin's
// component, empty for a folder mounted into another app type. Status is
// one of the appstack.Plugin* values; DeclaredAs is set when a Moodle
// plugin's development folder declares another component.
type DevFolderRow struct {
	App        string `json:"app"`
	AppLabel   string `json:"appLabel"`
	Component  string `json:"component"`
	DevPath    string `json:"devPath"`
	DevLabel   string `json:"devLabel"`
	RelPath    string `json:"relPath"`
	DestLabel  string `json:"destLabel"`
	Status     string `json:"status"`
	DeclaredAs string `json:"declaredAs"`
}

// DevFoldersScreen is everything the screen lists.
type DevFoldersScreen struct {
	Apps []DevFolderApp `json:"apps"`
	Rows []DevFolderRow `json:"rows"`
	// OutdatedUnits counts the mounts whose systemd unit predates the
	// checks against links (UpdateDevFolderUnits rewrites them).
	OutdatedUnits int `json:"outdatedUnits"`
}

// DevFolderCheck is the result of checking the form: Locked means the
// Moodle plugin's type and name came from the development folder's
// version.php; OK means Criar can run and the folder will appear at
// DestLabel; otherwise Message says why not.
type DevFolderCheck struct {
	Locked    bool   `json:"locked"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	OK        bool   `json:"ok"`
	DestLabel string `json:"destLabel"`
	Message   string `json:"message"`
}

// appLabel is "Name (folder)", or the folder alone when they match.
func appLabel(a config.AppContainer) string {
	if a.Name == "" || a.Name == a.Folder {
		return a.Folder
	}
	return a.Name + " (" + a.Folder + ")"
}

// GetDevFolders lists the app containers and every mounted folder with its
// current status.
func (t *DevToolsService) GetDevFolders() (DevFoldersScreen, error) {
	cfg, err := config.Load()
	if err != nil {
		return DevFoldersScreen{}, err
	}
	screen := DevFoldersScreen{Apps: []DevFolderApp{}, Rows: []DevFolderRow{}}
	labels := map[string]string{}
	workspace, _ := cfg.Workspace()
	for _, a := range cfg.Docker.Apps {
		labels[a.Folder] = appLabel(a)
		root := relativeToWorkspace(workspace, appstack.AppHTMLDir(workspace, a.Folder))
		screen.Apps = append(screen.Apps, DevFolderApp{Folder: a.Folder, Label: labels[a.Folder], Type: a.Type, Root: root})
	}
	for _, st := range appstack.MoodlePluginStatuses(context.Background(), t.exe, cfg) {
		label := labels[st.App]
		if label == "" {
			label = st.App
		}
		screen.Rows = append(screen.Rows, DevFolderRow{
			App:        st.App,
			AppLabel:   label,
			Component:  st.Component,
			DevPath:    st.DevPath,
			DevLabel:   relativeToWorkspace(workspace, st.DevPath),
			RelPath:    st.RelPath,
			DestLabel:  filepath.Join(st.App, st.RelPath),
			Status:     st.Status,
			DeclaredAs: st.DeclaredAs,
		})
	}
	screen.OutdatedUnits = appstack.OutdatedMountUnits(cfg)
	return screen, nil
}

// relativeToWorkspace returns path relative to workspace, or path itself
// when it is outside it.
func relativeToWorkspace(workspace, path string) string {
	if workspace == "" {
		return path
	}
	if resolved, err := filepath.EvalSymlinks(workspace); err == nil {
		workspace = resolved
	}
	rel, err := filepath.Rel(workspace, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return rel
}

// PickDevFolder opens the native folder picker for the development
// folder.
func (t *DevToolsService) PickDevFolder() (string, error) {
	return t.pickFolder(i18n.Tr("devfolder.pickTitle", nil))
}

// GetMoodlePluginTypes lists, sorted, the plugin types the Moodle in
// folder accepts.
func (t *DevToolsService) GetMoodlePluginTypes(folder string) ([]string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	workspace, err := cfg.Workspace()
	if err != nil {
		return nil, err
	}
	if app, found := appstack.FindAppByFolder(cfg.Docker.Apps, folder); !found || app.Type != config.AppTypeMoodle {
		return nil, errors.New(i18n.Tr("devfolder.notMoodle", i18n.Args{"app": folder}))
	}
	types, err := appstack.MoodlePluginTypes(workspace, folder)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(types))
	for k := range types {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

// requireDevFolder refuses a development folder that isn't an existing
// absolute folder picked in the native dialog.
func requireDevFolder(dev string) error {
	if err := requireFolder(dev); err != nil {
		return err
	}
	return requireApprovedPath(dev)
}

// InspectDevFolder checks the form: ptype and name for a Moodle app (when
// dev/version.php declares a component, its type and name replace them),
// relPath for the other app types.
func (t *DevToolsService) InspectDevFolder(dev, folder, ptype, name, relPath string) (DevFolderCheck, error) {
	if err := requireDevFolder(dev); err != nil {
		return DevFolderCheck{}, err
	}
	cfg, err := config.Load()
	if err != nil {
		return DevFolderCheck{}, err
	}
	check := DevFolderCheck{Type: ptype, Name: name}
	if folder == "" {
		check.Message = i18n.Tr("devfolder.chooseApp", nil)
		return check, nil
	}
	app, found := appstack.FindAppByFolder(cfg.Docker.Apps, folder)
	moodle := found && app.Type == config.AppTypeMoodle
	if moodle {
		if declType, declName := appstack.DevPluginComponent(dev); declType != "" {
			check.Locked, check.Type, check.Name = true, declType, declName
		}
		if check.Type == "" || check.Name == "" {
			check.Message = i18n.Tr("devfolder.enterPlugin", nil)
			return check, nil
		}
	}
	target := appstack.DevFolderTarget{PluginType: check.Type, PluginName: check.Name, RelPath: relPath}
	plan, err := appstack.PlanDevFolder(cfg, dev, folder, target)
	if err != nil {
		check.Message = capitalizeFirst(err.Error())
		return check, nil
	}
	check.OK = true
	check.DestLabel = filepath.Join(folder, plan.RelPath)
	return check, nil
}

// capitalizeFirst returns s with its first letter in upper case.
func capitalizeFirst(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

// AddDevFolder mounts dev into the app in folder (one password prompt): as
// the plugin ptype_name in a Moodle app, at relPath in the other apps.
func (t *DevToolsService) AddDevFolder(dev, folder, ptype, name, relPath string) error {
	if err := requireDevFolder(dev); err != nil {
		return err
	}
	target := appstack.DevFolderTarget{PluginType: ptype, PluginName: name, RelPath: relPath}
	return t.runAction(func(stdout io.Writer) error {
		return appstack.AddDevFolder(context.Background(), t.exe, stdout, dev, folder, target)
	})
}

// ApplyDevFolders restarts the app container in folder (and Nginx, for an
// app it serves) so they see the current mounts.
func (t *DevToolsService) ApplyDevFolders(folder string) error {
	return t.runAction(func(stdout io.Writer) error {
		return appstack.ApplyDevFolders(context.Background(), t.exe, stdout, folder)
	})
}

// RemoveDevFolder unmounts the recorded folder at relPath in the app in
// folder (one password prompt) and restarts the containers. Only a
// recorded mount is accepted; the development folder is never touched.
func (t *DevToolsService) RemoveDevFolder(folder, relPath string) error {
	return t.runAction(func(stdout io.Writer) error {
		return appstack.RemoveDevFolder(context.Background(), t.exe, stdout, folder, relPath)
	})
}

// UpdateDevFolderUnits rewrites the systemd units of the mounts that
// predate the checks against links (one password prompt).
func (t *DevToolsService) UpdateDevFolderUnits() error {
	return t.runAction(func(stdout io.Writer) error {
		return appstack.UpdateMountUnits(context.Background(), t.exe, stdout)
	})
}
