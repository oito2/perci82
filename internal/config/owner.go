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

package config

import (
	"slices"

	"github.com/oito2/perci82/internal/i18n"
)

// App identifies the Perci82 application running in this process. Each
// application owns a fixed set of top-level keys of config.yaml and
// Update writes only those.
type App string

// The Perci82 applications.
const (
	AppSystem App = "system"
	AppDev    App = "dev"
)

// DisplayName returns the application's name as shown to the user.
func (a App) DisplayName() string {
	switch a {
	case AppSystem:
		return "Perci82 System"
	case AppDev:
		return "Perci82 Dev"
	}
	return "Perci82"
}

// ownedKeys lists, per application, the top-level config.yaml keys it
// writes. Keys listed here without a Config field yet are reserved for
// that application.
var ownedKeys = map[App][]string{
	AppSystem: {
		"postinstall_applied",
		"postinstall_machine_id",
		"flatpak_scope",
		"system_gui_theme",
		"system_language",
	},
	AppDev: {
		"workspace_path",
		"docker",
		"git_accounts",
		"agent_skills",
		"mcp_servers",
		"mcp_binaries",
		"moodle_plugins",
		"ai_contexts",
		"tray_enabled",
		"gui_theme",
		"language",
		"project_notes_path",
	},
}

// legacyKeys are top-level keys written by earlier Perci versions that no
// application writes any more. Update keeps them as they are.
var legacyKeys = []string{"sidebar_logo", "app_icon"}

// LegacyKeys returns a copy of the top-level keys no application writes.
func LegacyKeys() []string {
	return slices.Clone(legacyKeys)
}

// OwnedKeys returns a copy of the top-level keys app writes.
func OwnedKeys(app App) []string {
	return slices.Clone(ownedKeys[app])
}

// currentApp is the application whose keys Update may write; empty
// allows every key. Guarded by mu.
var currentApp App

// SetApp makes Update write only the top-level keys app owns. It is
// called once, at startup, before any Update.
func SetApp(app App) {
	mu.Lock()
	defer mu.Unlock()
	currentApp = app
}

// ForeignKeyError is returned by Update when its mutation changed a
// top-level key the running application doesn't own. Nothing is written.
type ForeignKeyError struct {
	App App
	Key string
}

func (e *ForeignKeyError) Error() string {
	return i18n.Tr("config.foreignKey", i18n.Args{"app": e.App.DisplayName(), "key": e.Key})
}

// checkOwnership returns a *ForeignKeyError for the first key of changed
// that app doesn't own. An empty app owns every key.
func checkOwnership(app App, changed []string) error {
	if app == "" {
		return nil
	}
	for _, key := range changed {
		if !slices.Contains(ownedKeys[app], key) {
			return &ForeignKeyError{App: app, Key: key}
		}
	}
	return nil
}
