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
	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/i18n"
)

// MenuItem is one screen of a section: an entry of the section's side
// list, or the section's only screen.
type MenuItem struct {
	// Title is the screen title, also the label of its side list entry.
	Title string `json:"title"`
	Desc  string `json:"desc"` // may hold multiple lines (\n) — rendered with line breaks preserved

	// ActionID identifies the screen (SCREENS/SCREEN_FAMILIES in the
	// frontend) and which bound method its "Executar" button calls.
	ActionID string `json:"actionId,omitempty"`

	// Button is the "Executar" button's text when ActionID != "" —
	// screen-specific text (e.g. "Executar atualização"). Empty uses the
	// generic "Executar" label.
	Button string `json:"button,omitempty"`

	// Notes are warnings shown below the screen's buttons. Empty renders no
	// notes section.
	Notes []string `json:"notes,omitempty"`

	// Tab is the label of the screen's first inner tab; empty uses Title.
	Tab string `json:"tab,omitempty"`

	// Group is the heading shown above this entry in the side list when it
	// differs from the previous entry's; empty for no heading.
	Group string `json:"group,omitempty"`

	// AccountID is the Git account an item of the "Repositórios" section
	// opens (ActionID "git-account").
	AccountID string `json:"accountId,omitempty"`
}

// Section is one tab of the window's top tab bar. A section with more
// than one item shows them in a side list.
type Section struct {
	// ID identifies the section across catalog reloads and languages.
	ID string `json:"id"`
	// Label is the tab's text.
	Label string `json:"label"`
	// Icon is the name of the tab's lucide icon.
	Icon  string     `json:"icon"`
	Items []MenuItem `json:"items"`
}

// sectionDef is a section of a static catalog: label, item titles, tabs
// and groups are message keys of the i18n catalog, and so are the item
// descriptions, buttons and notes that have one (any other text is shown
// as is).
type sectionDef struct {
	id    string
	label string
	icon  string
	items []MenuItem
}

// translate returns the section with its message keys resolved in locale.
func (d sectionDef) translate(locale string) Section {
	s := Section{ID: d.id, Label: i18n.T(locale, d.label, nil), Icon: d.icon}
	for _, it := range d.items {
		it.Title = i18n.T(locale, it.Title, nil)
		it.Desc = i18n.T(locale, it.Desc, nil)
		if it.Button != "" {
			it.Button = i18n.T(locale, it.Button, nil)
		}
		if len(it.Notes) > 0 {
			notes := make([]string, len(it.Notes))
			for i, n := range it.Notes {
				notes[i] = i18n.T(locale, n, nil)
			}
			it.Notes = notes
		}
		if it.Tab != "" {
			it.Tab = i18n.T(locale, it.Tab, nil)
		}
		if it.Group != "" {
			it.Group = i18n.T(locale, it.Group, nil)
		}
		s.Items = append(s.Items, it)
	}
	return s
}

// translateAll resolves a static catalog in locale.
func translateAll(defs []sectionDef, locale string) []Section {
	out := make([]Section, 0, len(defs))
	for _, d := range defs {
		out = append(out, d.translate(locale))
	}
	return out
}

// systemSections returns Perci82 System's sections in locale.
func systemSections(locale string) []Section {
	return translateAll(systemCatalog, locale)
}

// devSections returns Perci82 Dev's sections in locale; the
// "repositories" section ends with one item per Git account, in
// accounts' order, under the "Contas" group.
func devSections(locale string, accounts []config.GitAccount) []Section {
	sections := translateAll(devCatalog, locale)
	for i := range sections {
		if sections[i].ID != "repositories" {
			continue
		}
		group := i18n.T(locale, "nav.group.accounts", nil)
		for _, acc := range accounts {
			sections[i].Items = append(sections[i].Items, MenuItem{
				Title:     acc.Name,
				Desc:      i18n.T(locale, "nav.account.desc", i18n.Args{"name": acc.Name}),
				ActionID:  "git-account",
				AccountID: acc.ID,
				Tab:       acc.Name,
				Group:     group,
			})
		}
	}
	return sections
}

// overviewSection and settingsSection are the first and last sections of
// both applications; the item descriptions differ per application.
func overviewSection(desc string) sectionDef {
	return sectionDef{id: "overview", label: "nav.overview", icon: "house", items: []MenuItem{{
		// "dashboard" mode (#dashboard-container — also the screen shown
		// before the sections load).
		Title:    "nav.overview",
		Desc:     desc,
		ActionID: "dashboard",
	}}}
}

func settingsSection(desc string) sectionDef {
	return sectionDef{id: "settings", label: "nav.settings", icon: "settings", items: []MenuItem{{
		// "self-config" mode (#selfconfig-container).
		Title:    "nav.settings",
		Desc:     desc,
		ActionID: "self-config",
	}}}
}

// systemCatalog is Perci82 System's static catalog.
var systemCatalog = []sectionDef{
	overviewSection("nav.overview.systemDesc"),
	{id: "postinstall", label: "nav.postinstall", icon: "wand", items: []MenuItem{{
		// "checklist com dependência" mode (#postinstall-container); Desc
		// and Button stay empty.
		Title:    "nav.postinstall",
		ActionID: "postinstall",
		Tab:      "nav.tab.items",
	}}},
	{id: "update", label: "nav.update", icon: "refreshCw", items: []MenuItem{{
		// Simple mode (#simple-container) with the "Sistema" status bar.
		Title:    "nav.update.title",
		Desc:     "nav.update.desc",
		ActionID: "system-update",
		Button:   "nav.update.button",
		Tab:      "nav.tab.items",
	}}},
	{id: "fonts", label: "nav.fonts", icon: "type", items: []MenuItem{{
		// "checklist simples" mode (#multiselect-container): the checkbox
		// reflects the current state, with no dependency between items.
		Title:    "nav.fonts.title",
		Desc:     "nav.fonts.desc",
		ActionID: "fonts",
		Tab:      "nav.tab.items",
	}}},
	{id: "templates", label: "nav.templates", icon: "fileText", items: []MenuItem{{
		// "checklist simples" mode, same as fonts.
		Title:    "nav.templates.title",
		Desc:     "nav.templates.desc",
		ActionID: "templates",
		Tab:      "nav.tab.items",
	}}},
	{id: "flatpaks", label: "nav.flatpaks", icon: "package", items: []MenuItem{{
		// "checklist simples" mode, same as fonts.
		Title:    "nav.flatpaks",
		Desc:     "nav.flatpaks.desc",
		ActionID: "flatpak-apps",
		Tab:      "nav.tab.items",
	}}},
	settingsSection("nav.settings.systemDesc"),
}

// devCatalog is Perci82 Dev's static catalog; devSections adds the Git
// accounts to the "repositories" section.
var devCatalog = []sectionDef{
	overviewSection("Veja o resumo dos contêineres Docker e a versão do Perci82 Dev."),
	{id: "environment", label: "nav.environment", icon: "code", items: []MenuItem{
		{
			// Item table mode (#itemtable-container): one row per item
			// with its own Instalar/Atualizar/Desinstalar buttons.
			Title:    "nav.prereqs",
			Desc:     "nav.prereqs.desc",
			ActionID: "prereqs",
			Tab:      "nav.tab.items",
		},
		{
			// Item table mode.
			Title:    "nav.sdks",
			Desc:     "nav.sdks.desc",
			ActionID: "sdks",
			Tab:      "nav.tab.items",
		},
		{
			// "checklist simples" mode (#multiselect-container).
			Title:    "nav.ai",
			Desc:     "nav.ai.desc",
			ActionID: "ai-apps",
			Tab:      "nav.tab.items",
		},
		{
			// "checklist simples" mode plus the "Aplicar Starship" /
			// "Remover Starship" buttons (MULTISELECT_SCREENS extras).
			Title:    "nav.terminals",
			Desc:     "nav.terminals.desc",
			ActionID: "terminals",
			Tab:      "nav.tab.items",
		},
		{
			// Item table mode.
			Title:    "nav.ides",
			Desc:     "nav.ides.desc",
			ActionID: "ides",
			Tab:      "nav.tab.items",
		},
		{
			// Local file install/uninstall mode (#fileapp-container), with
			// no Atualizar button: Android Studio updates itself.
			Title:    "nav.androidStudio",
			Desc:     "nav.androidStudio.desc",
			Notes:    []string{"nav.androidStudio.note", "nav.androidStudio.noteUpdate"},
			ActionID: "androidstudio",
			Tab:      "nav.tab.items",
		},
		{
			// Local file install/update/uninstall mode (#fileapp-container).
			Title:    "nav.antigravityIDE",
			Desc:     "nav.antigravityIDE.desc",
			ActionID: "antigravityide",
			Tab:      "nav.tab.items",
		},
	}},
	{id: "docker", label: "nav.docker", icon: "container", items: []MenuItem{
		{
			// "criar container" mode (#dockercreate-container): a "Tipo de
			// contêiner" select with per-type fields; Nginx/MariaDB are
			// disabled once they exist (singletons).
			Title:    "nav.dockerCreate",
			Desc:     "nav.dockerCreate.desc",
			ActionID: "docker-create",
			Tab:      "nav.tab.newContainer",
		},
		{
			// "gerenciar containers" mode (#dockerlist-container): a table
			// with live status and per-row actions.
			Title:    "nav.dockerManage",
			Desc:     "nav.dockerManage.desc",
			ActionID: "docker-manage",
			Tab:      "nav.tab.containers",
		},
		{
			// "pastas de desenvolvimento" mode (#moodleplugins-container):
			// a development folder mounted into an app container, plus the
			// "Pastas montadas" inner tab.
			Title:    "nav.devFolders",
			Desc:     "nav.devFolders.desc",
			ActionID: "moodleplugins",
			Tab:      "nav.devFolders",
		},
		{
			// "exportar/importar" mode (#transfer-container), plus the
			// "Importar" inner tab.
			Title:    "nav.exportImport",
			Desc:     "nav.exportImport.desc",
			ActionID: "docker-transfer",
			Tab:      "nav.tab.export",
		},
	}},
	{id: "repositories", label: "nav.repositories", icon: "gitBranch", items: []MenuItem{
		{
			// "contas git" mode (#gitaccounts-container): the account table
			// and its form, plus the "Identidade padrão" inner tab.
			Title:    "nav.gitAccounts",
			Desc:     "nav.gitAccounts.desc",
			ActionID: "git-accounts",
			Tab:      "nav.tab.accounts",
		},
		{
			// "gerenciar repositórios" mode (#gittransfer-container): the
			// Exportar checklist, plus the "Importar" inner tab.
			Title:    "nav.gitTransfer",
			Desc:     "nav.gitTransfer.desc",
			ActionID: "git-transfer",
			Tab:      "nav.tab.export",
		},
	}},
	{id: "ai-contexts", label: "nav.aiContexts", icon: "brain", items: []MenuItem{
		{
			// #aicdefine-container: the folder, agents and standards of one
			// context.
			Title:    "nav.aicDefine",
			Desc:     "nav.aicDefine.desc",
			ActionID: "aicontext-define",
			Tab:      "nav.aicDefine",
		},
		{
			// #aicontext-container: the registered folders.
			Title:    "nav.aicManage",
			Desc:     "nav.aicManage.desc",
			ActionID: "aicontext",
			Tab:      "nav.tab.contexts",
		},
		{
			// #aictransfer-container: export and import the registered
			// folders.
			Title:    "nav.exportImport",
			Desc:     "nav.aicTransfer.desc",
			ActionID: "aicontext-transfer",
			Tab:      "nav.exportImport",
		},
	}},
	{id: "skills-mcps", label: "nav.skillsMCPs", icon: "plug", items: []MenuItem{
		{
			// "IA: SKILLs" mode (#agentskills-container): skills.sh skills
			// (internal/dev/agentskills).
			Title:    "nav.skills",
			Desc:     "Atalhos para instalar skills de agente de IA disponibilizadas pelo skills.sh",
			ActionID: "agentskills",
			Tab:      "nav.skills",
		},
		{
			// "IA: MCPs" mode (#mcpservers-container): MCP servers
			// registered with Claude Code/Codex/Antigravity.
			Title:    "nav.mcps",
			Desc:     "Registre servidores MCP para Claude Code, Codex e Antigravity.",
			ActionID: "mcpservers",
			Tab:      "nav.mcps",
		},
	}},
	settingsSection("Idioma, tema, workspace e bandeja do sistema."),
}
