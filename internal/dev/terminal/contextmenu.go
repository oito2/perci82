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

package terminal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// SyncContextMenuEntries makes the "Abrir no <terminal>" entries match
// installed (terminal name → installed, as returned by InstalledMap): added
// for every installed terminal, removed for the others. Idempotent.
func SyncContextMenuEntries(installed map[string]bool, stdout io.Writer) {
	for _, t := range Catalogue {
		if installed[t.Name] {
			AddContextMenuEntries(t, stdout)
		} else {
			RemoveContextMenuEntries(t, stdout)
		}
	}
}

// AddContextMenuEntries creates "Open here" entries for Nautilus, Nemo and Dolphin.
func AddContextMenuEntries(t Terminal, stdout io.Writer) {
	if t.DirFlag == "" {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		ui.Warning(stdout, i18n.Tr("terminal.run.menuNoHome", i18n.Args{"error": err.Error()}))
		return
	}
	ui.Info(stdout, i18n.Tr("terminal.run.menuAdd", i18n.Args{"name": t.Name}))
	writeContextMenuFile(nautilusScriptPath(t, home), nautilusScript(t), 0o755, "Nautilus", stdout)
	writeContextMenuFile(nemoScriptPath(t, home), nemoScript(t), 0o755, "Nemo", stdout)
	writeContextMenuFile(dolphinMenuPath(t, home), dolphinMenu(t), 0o755, "Dolphin", stdout)
}

// RemoveContextMenuEntries removes "Open here" entries for all file managers.
func RemoveContextMenuEntries(t Terminal, stdout io.Writer) {
	if t.DirFlag == "" {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	paths := []string{dolphinMenuPath(t, home)}
	for _, locale := range i18n.Locales() {
		paths = append(paths, nautilusScriptPathIn(t, home, locale), nemoScriptPathIn(t, home, locale))
	}
	for _, p := range paths {
		_ = os.Remove(p)
		_ = os.Remove(p + ".perci-bak")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeContextMenuFile(path, content string, perm os.FileMode, label string, stdout io.Writer) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		ui.Warning(stdout, i18n.Tr("terminal.run.menuDirFailed", i18n.Args{"name": label, "error": err.Error()}))
		return
	}
	if err := fsutil.WriteGenerated(path, []byte(content), perm); err != nil {
		ui.Warning(stdout, i18n.Tr("terminal.run.menuWriteFailed", i18n.Args{"name": label, "error": err.Error()}))
	}
}

func execCmd(t Terminal) string {
	if t.MenuExec != "" {
		return t.MenuExec
	}
	return t.Cmd
}

func menuSlug(t Terminal) string {
	return strings.ToLower(strings.ReplaceAll(t.Name, " ", "-"))
}

// shellescapeWords quotes each whitespace-separated word in s individually
// via executor.ShellQuote.
func shellescapeWords(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = executor.ShellQuote(w)
	}
	return strings.Join(words, " ")
}

// ── Nautilus ─────────────────────────────────────────────────────────────────

// openInLabel is the "Open in <terminal>" label of the file managers' menu
// entries in locale.
func openInLabel(t Terminal, locale string) string {
	return i18n.T(locale, "terminal.openIn", i18n.Args{"name": t.Name})
}

// nautilusScriptPath is the Nautilus script of t in the current language.
func nautilusScriptPath(t Terminal, home string) string {
	return nautilusScriptPathIn(t, home, i18n.Locale())
}

// nautilusScriptPathIn is the Nautilus script of t in locale — the script's
// file name is the label the menu shows.
func nautilusScriptPathIn(t Terminal, home, locale string) string {
	return filepath.Join(home, ".local", "share", "nautilus", "scripts", openInLabel(t, locale))
}

func nautilusScript(t Terminal) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
URI="${NAUTILUS_SCRIPT_CURRENT_URI:-${NEMO_SCRIPT_CURRENT_URI:-}}"
DIR=$(python3 -c "import sys,urllib.parse; u=sys.argv[1]; print(urllib.parse.unquote(u[7:] if u.startswith('file://') else u))" "$URI" 2>/dev/null)
[[ -z "$DIR" ]] && DIR="$HOME"
%s %s "$DIR"
`, shellescapeWords(execCmd(t)), shellescapeWords(t.DirFlag))
}

// ── Nemo ─────────────────────────────────────────────────────────────────────

// nemoScriptPath is the Nemo script of t in the current language.
func nemoScriptPath(t Terminal, home string) string {
	return nemoScriptPathIn(t, home, i18n.Locale())
}

// nemoScriptPathIn is the Nemo script of t in locale — the script's file
// name is the label the menu shows.
func nemoScriptPathIn(t Terminal, home, locale string) string {
	return filepath.Join(home, ".local", "share", "nemo", "scripts", openInLabel(t, locale))
}

func nemoScript(t Terminal) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
URI="${NEMO_SCRIPT_CURRENT_URI:-${NAUTILUS_SCRIPT_CURRENT_URI:-}}"
DIR=$(python3 -c "import sys,urllib.parse; u=sys.argv[1]; print(urllib.parse.unquote(u[7:] if u.startswith('file://') else u))" "$URI" 2>/dev/null)
[[ -z "$DIR" ]] && DIR="$HOME"
%s %s "$DIR"
`, shellescapeWords(execCmd(t)), shellescapeWords(t.DirFlag))
}

// ── Dolphin ───────────────────────────────────────────────────────────────────

func dolphinMenuPath(t Terminal, home string) string {
	return filepath.Join(home, ".local", "share", "kio", "servicemenus", "perci-"+menuSlug(t)+".desktop")
}

func dolphinMenu(t Terminal) string {
	actionID := "perci-open-" + menuSlug(t)
	return fmt.Sprintf(`[Desktop Entry]
Type=Service
ServiceTypes=KonqPopupMenu/Plugin
MimeType=inode/directory;
Actions=%s;

[Desktop Action %s]
Name=%s
Icon=utilities-terminal
Exec=%s %s %%f
`, actionID, actionID, openInLabel(t, i18n.Locale()), shellescapeWords(execCmd(t)), shellescapeWords(t.DirFlag))
}
