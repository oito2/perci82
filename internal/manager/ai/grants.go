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

package ai

// Grants: what a context lets the coding agents reach beyond the project —
// the project notes folder (read and write) and the commands of the
// project's Docker app (run without asking). Each agent gets them in its
// own per-project settings; entries Perci wrote that no longer apply are
// taken out, entries the user wrote stay.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// grants is what a context lets the agents reach: notesDir ("" for none)
// and the app commands they may run without asking.
type grants struct {
	notesDir string
	commands []string
}

// Per-project settings files of the agents, relative to the project.
var (
	geminiSettingsPath = filepath.Join(".gemini", "settings.json")
	codexConfigPath    = filepath.Join(".codex", "config.toml")
	codexRulesPath     = filepath.Join(".codex", "rules", "perci.rules")
)

// appCommandName matches the name of an app command (an app folder name).
var appCommandName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// appCommandMarker is the line that marks Perci82 Dev's command of a
// Docker app in ~/.local/bin.
const appCommandMarker = "# perci-app-command"

// isAppCommand reports whether name is the command of a Docker app Perci82
// Dev set up (~/.local/bin/<name> carrying appCommandMarker). A variable
// so tests decide which commands exist.
var isAppCommand = func(name string) bool {
	home, err := os.UserHomeDir()
	if err != nil || !appCommandName.MatchString(name) {
		return false
	}
	data, err := os.ReadFile(filepath.Join(home, ".local", "bin", name))
	return err == nil && strings.Contains(string(data), "\n"+appCommandMarker+"\n")
}

var (
	claudeEditRule = regexp.MustCompile(`^Edit\(/(/.+)/\*\*\)$`)
	claudeBashRule = regexp.MustCompile(`^Bash\(([A-Za-z0-9_-]+) \*\)$`)
	geminiShell    = regexp.MustCompile(`^run_shell_command\(([A-Za-z0-9_-]+)\)$`)
)

// listEdit sets the Perci entries of the string list at keys (a path of
// object keys) in a JSON document: add are written once each, and the
// entries managed reports as Perci's that are not in add are taken out.
type listEdit struct {
	keys    []string
	add     []string
	managed func(string) bool
}

// applyListEdit applies e to doc; changed is false when nothing changed.
// A value along the path with another type is an error.
func applyListEdit(doc map[string]any, e listEdit) (changed bool, err error) {
	parents := []map[string]any{doc}
	cur := doc
	for _, k := range e.keys[:len(e.keys)-1] {
		next := map[string]any{}
		if raw, ok := cur[k]; ok {
			if next, ok = raw.(map[string]any); !ok {
				return false, fmt.Errorf("%q is not an object", k)
			}
		}
		cur = next
		parents = append(parents, cur)
	}
	last := e.keys[len(e.keys)-1]
	var list []any
	if raw, ok := cur[last]; ok {
		if list, ok = raw.([]any); !ok {
			return false, fmt.Errorf("%q is not a list", last)
		}
	}
	kept := list[:0:0]
	for _, v := range list {
		if s, ok := v.(string); ok && !slices.Contains(e.add, s) && e.managed(s) {
			changed = true
			continue
		}
		kept = append(kept, v)
	}
	for _, s := range e.add {
		if !slices.Contains(kept, any(s)) {
			kept = append(kept, s)
			changed = true
		}
	}
	if !changed {
		return false, nil
	}
	// Write back from the innermost object out, dropping what is empty.
	var value any = kept
	if len(kept) == 0 {
		value = nil
	}
	for i := len(e.keys) - 1; i >= 0; i-- {
		obj := parents[i]
		if value == nil {
			delete(obj, e.keys[i])
		} else {
			obj[e.keys[i]] = value
		}
		if i > 0 {
			value = obj
			if len(obj) == 0 {
				value = nil
			}
		}
	}
	return true, nil
}

// writeJSONGrants applies edits to the JSON settings file at path: it is
// created when something is to be added, and removed when it is left
// empty. A file that can't be read as a JSON object, or whose values have
// other types, is left as it is with a warning.
func writeJSONGrants(path string, edits []listEdit, stdout io.Writer) error {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	adding := slices.ContainsFunc(edits, func(e listEdit) bool { return len(e.add) > 0 })
	if len(existing) == 0 && !adding {
		return nil
	}
	doc := map[string]any{}
	if len(bytes.TrimSpace(existing)) > 0 {
		dec := json.NewDecoder(bytes.NewReader(existing))
		dec.UseNumber()
		err := dec.Decode(&doc)
		if err == nil && doc == nil {
			err = errors.New("does not hold a JSON object")
		}
		if err != nil {
			ui.Warning(stdout, i18n.Tr("notes.run.settingsBad", i18n.Args{"path": path, "error": err.Error()}))
			return nil
		}
	}
	changed := false
	for _, e := range edits {
		c, err := applyListEdit(doc, e)
		if err != nil {
			ui.Warning(stdout, i18n.Tr("notes.run.settingsBad", i18n.Args{"path": path, "error": err.Error()}))
			return nil
		}
		changed = changed || c
	}
	if !changed {
		return nil
	}
	if len(doc) == 0 {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
		ui.Info(stdout, i18n.Tr("aic.run.removed", i18n.Args{"path": path}))
		removeEmptyParents(filepath.Dir(path), 2)
		return nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := writeMergedConfig(path, buf.Bytes()); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	ui.Info(stdout, i18n.Tr("aic.run.updated", i18n.Args{"path": path}))
	return nil
}

// removeEmptyParents removes dir and up to levels-1 of its parents while
// they are empty.
func removeEmptyParents(dir string, levels int) {
	for i := 0; i < levels; i++ {
		if os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

// writeClaudeGrants gives Claude Code, in dir's .claude/settings.json, the
// notes folder as an additional directory it may edit and the app commands
// as Bash rules it runs without asking. g empty takes Perci's entries out.
func writeClaudeGrants(dir string, g grants, stdout io.Writer) error {
	var dirs, allow []string
	if g.notesDir != "" {
		dirs = []string{g.notesDir}
		allow = append(allow, "Edit(/"+g.notesDir+"/**)")
	}
	for _, c := range g.commands {
		allow = append(allow, "Bash("+c+" *)")
	}
	return writeJSONGrants(filepath.Join(dir, claudeSettingsPath), []listEdit{
		{keys: []string{"permissions", "additionalDirectories"}, add: dirs, managed: isNotesFolder},
		{keys: []string{"permissions", "allow"}, add: allow, managed: func(s string) bool {
			if m := claudeEditRule.FindStringSubmatch(s); m != nil {
				return isNotesFolder(m[1])
			}
			if m := claudeBashRule.FindStringSubmatch(s); m != nil {
				return isAppCommand(m[1])
			}
			return false
		}},
	}, stdout)
}

// writeGeminiGrants gives Gemini, in dir's .gemini/settings.json, the
// notes folder as an included directory and the app commands as shell
// commands it runs without asking. g empty takes Perci's entries out.
func writeGeminiGrants(dir string, g grants, stdout io.Writer) error {
	var dirs, allowed []string
	if g.notesDir != "" {
		dirs = []string{g.notesDir}
	}
	for _, c := range g.commands {
		allowed = append(allowed, "run_shell_command("+c+")")
	}
	return writeJSONGrants(filepath.Join(dir, geminiSettingsPath), []listEdit{
		{keys: []string{"context", "includeDirectories"}, add: dirs, managed: isNotesFolder},
		{keys: []string{"tools", "allowed"}, add: allowed, managed: func(s string) bool {
			m := geminiShell.FindStringSubmatch(s)
			return m != nil && isAppCommand(m[1])
		}},
	}, stdout)
}

// openCodePermission is one entry of the "permissions" list of
// opencode.json.
type openCodePermission struct {
	action, resource string
}

// openCodePermissions returns the "permissions" entries g needs: the notes
// folder as an external directory OpenCode may edit, and each app command
// as a shell command it runs without asking.
func openCodePermissions(g grants) []openCodePermission {
	var out []openCodePermission
	if g.notesDir != "" {
		out = append(out,
			openCodePermission{"external_directory", g.notesDir + "/*"},
			openCodePermission{"edit", g.notesDir + "/*"})
	}
	for _, c := range g.commands {
		out = append(out, openCodePermission{"shell", c + " *"})
	}
	return out
}

// openCodePermissionManaged reports whether the "permissions" entry v is
// one Perci writes for some notes folder or app command.
func openCodePermissionManaged(v any) bool {
	obj, ok := v.(map[string]any)
	if !ok || obj["effect"] != "allow" || len(obj) != 3 {
		return false
	}
	resource, _ := obj["resource"].(string)
	switch obj["action"] {
	case "external_directory", "edit":
		return strings.HasSuffix(resource, "/*") && isNotesFolder(strings.TrimSuffix(resource, "/*"))
	case "shell":
		return strings.HasSuffix(resource, " *") && isAppCommand(strings.TrimSuffix(resource, " *"))
	}
	return false
}

// setOpenCodePermissions sets Perci's entries of doc's "permissions" list
// to want; changed is false when nothing changed.
func setOpenCodePermissions(doc map[string]any, want []openCodePermission) (changed bool, err error) {
	var list []any
	if raw, ok := doc["permissions"]; ok {
		if list, ok = raw.([]any); !ok {
			return false, errors.New(`"permissions" is not a list`)
		}
	}
	is := func(v any, p openCodePermission) bool {
		obj, ok := v.(map[string]any)
		return ok && obj["action"] == p.action && obj["resource"] == p.resource && obj["effect"] == "allow"
	}
	wanted := func(v any) bool {
		return slices.ContainsFunc(want, func(p openCodePermission) bool { return is(v, p) })
	}
	kept := list[:0:0]
	for _, v := range list {
		if !wanted(v) && openCodePermissionManaged(v) {
			changed = true
			continue
		}
		kept = append(kept, v)
	}
	for _, p := range want {
		if !slices.ContainsFunc(kept, func(v any) bool { return is(v, p) }) {
			kept = append(kept, map[string]any{"action": p.action, "resource": p.resource, "effect": "allow"})
			changed = true
		}
	}
	if len(kept) == 0 {
		delete(doc, "permissions")
	} else {
		doc["permissions"] = kept
	}
	return changed, nil
}

// codexMarker is the first line of the Codex files Perci writes.
const codexMarker = "# Generated by Perci82 Dev"

// writeCodexGrants writes, for Codex, the notes folder as a writable root
// in dir's .codex/config.toml and the app commands as allowed prefixes in
// .codex/rules/perci.rules. Both files are Perci's own: one without
// codexMarker is left as it is with a warning, and each is removed when g
// leaves it nothing to hold.
func writeCodexGrants(dir string, g grants, stdout io.Writer) error {
	config := ""
	if g.notesDir != "" {
		config = codexMarker + ": lets Codex write the project notes folder.\n\n[sandbox_workspace_write]\nwritable_roots = [" + strconv.Quote(g.notesDir) + "]\n"
	}
	rules := ""
	if len(g.commands) > 0 {
		var b strings.Builder
		b.WriteString(codexMarker + ": the commands of this project's Docker app run without asking.\n")
		for _, c := range g.commands {
			fmt.Fprintf(&b, "prefix_rule(pattern=[%s], decision=\"allow\")\n", strconv.Quote(c))
		}
		rules = b.String()
	}
	if err := writeOwnFile(filepath.Join(dir, codexConfigPath), config, stdout); err != nil {
		return err
	}
	if err := writeOwnFile(filepath.Join(dir, codexRulesPath), rules, stdout); err != nil {
		return err
	}
	removeEmptyParents(filepath.Join(dir, ".codex", "rules"), 2)
	return nil
}

// writeOwnFile writes content to path, or removes path when content is
// "", as long as path is missing or starts with codexMarker; another file
// is left as it is, with a warning when content is not "".
func writeOwnFile(path, content string, stdout io.Writer) error {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	exists := err == nil
	if exists && !strings.HasPrefix(string(existing), codexMarker) {
		if content != "" {
			ui.Warning(stdout, i18n.Tr("notes.run.notOurs", i18n.Args{"path": path}))
		}
		return nil
	}
	switch {
	case content == "" && exists:
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
		ui.Info(stdout, i18n.Tr("aic.run.removed", i18n.Args{"path": path}))
	case content != "" && string(existing) != content:
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
		}
		if err := fsutil.WriteGenerated(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		ui.Info(stdout, i18n.Tr("aic.run.updated", i18n.Args{"path": path}))
	}
	return nil
}
