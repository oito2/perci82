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

// Project notes: with the project_notes_path setting, each project with a
// context gets a notes folder outside its repository,
// <project_notes_path>/<project folder name>/, holding PROJECT.md (the
// project rules, in place of .instructions/PROJECT.md), notes/, plans/ and
// references/. AGENTS.md points the agents at it, and each agent's files
// load its PROJECT.md and let the agent read and write the folder.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// notesSubdirs are the folders of a notes folder besides PROJECT.md.
var notesSubdirs = []string{"notes", "plans", "references"}

// notesPlaceholder stands for the notes folder in the templates.
const notesPlaceholder = "{{NOTES_DIR}}"

// NotesDir returns the notes folder of the project in dir under the
// project_notes_path setting (see notesNames), or "" when the setting is
// empty or the config can't be read.
func NotesDir(dir string) string {
	cfg, err := config.Load()
	if err != nil || cfg.ProjectNotesPath == "" {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	return filepath.Join(cfg.ProjectNotesPath, notesNames(cfg, abs)[abs])
}

// notesNames gives every registered context, and the folder extra when it
// is not registered (extra may be ""), its notes folder name: the recorded
// one, otherwise the project folder's name, with a -2, -3... suffix when
// a context before it in the list (or a recorded name) already took it.
// Two projects in different places with the same folder name never share
// notes.
func notesNames(cfg *config.Config, extra string) map[string]string {
	names := map[string]string{}
	taken := map[string]bool{}
	for _, c := range cfg.AIContexts {
		if c.Notes != "" {
			names[c.Path] = c.Notes
			taken[c.Notes] = true
		}
	}
	paths := make([]string, 0, len(cfg.AIContexts)+1)
	for _, c := range cfg.AIContexts {
		paths = append(paths, c.Path)
	}
	paths = append(paths, extra)
	for _, p := range paths {
		if p == "" || names[p] != "" {
			continue
		}
		base := filepath.Base(p)
		name := base
		for n := 2; taken[name]; n++ {
			name = fmt.Sprintf("%s-%d", base, n)
		}
		names[p] = name
		taken[name] = true
	}
	return names
}

// FreezeNotes records in every context of cfg the notes folder name it
// uses now, when a notes folder is set, so that adding or removing another
// context never moves its notes.
func FreezeNotes(cfg *config.Config) {
	if cfg.ProjectNotesPath == "" {
		return
	}
	names := notesNames(cfg, "")
	for i := range cfg.AIContexts {
		if cfg.AIContexts[i].Notes == "" {
			cfg.AIContexts[i].Notes = names[cfg.AIContexts[i].Path]
		}
	}
}

// isNotesFolder reports whether path is a notes folder Perci set up: it
// holds PROJECT.md and every one of notesSubdirs.
func isNotesFolder(path string) bool {
	if !filepath.IsAbs(path) {
		return false
	}
	if info, err := os.Stat(filepath.Join(path, "PROJECT.md")); err != nil || !info.Mode().IsRegular() {
		return false
	}
	for _, sub := range notesSubdirs {
		if info, err := os.Stat(filepath.Join(path, sub)); err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}

// withNotes rewrites the references to .instructions/PROJECT.md in text
// (template wording, "@./" and "@" imports) to notesDir's PROJECT.md;
// text stays as it is when notesDir is "".
func withNotes(text, notesDir string) string {
	if notesDir == "" {
		return text
	}
	rules := filepath.Join(notesDir, "PROJECT.md")
	text = strings.ReplaceAll(text, "./"+filepath.ToSlash(projectRulesPath), rules)
	return strings.ReplaceAll(text, filepath.ToSlash(projectRulesPath), rules)
}

// notesSection is the AGENTS.md section describing notesDir.
func notesSection(notesDir string) (string, error) {
	tpl, err := readTpl("templates/NOTES.md")
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(strings.TrimRight(tpl, "\n"), notesPlaceholder, notesDir), nil
}

// notesPathRe matches an absolute path in a code span: what a notes
// section and the rules of an AGENTS.md written for a notes folder name.
var notesPathRe = regexp.MustCompile("`/[^`]+`")

// genericNotesLine returns l with every absolute path in a code span put
// back as the templates word it: a notes PROJECT.md as
// .instructions/PROJECT.md, any other path as the notes placeholder. It
// recognizes the lines of an AGENTS.md written for a notes folder other
// than the current one.
func genericNotesLine(l string) string {
	return notesPathRe.ReplaceAllStringFunc(l, func(m string) string {
		if strings.HasSuffix(m, "/PROJECT.md`") {
			return "`" + filepath.ToSlash(projectRulesPath) + "`"
		}
		return "`" + notesPlaceholder + "`"
	})
}

// normalizeNotes turns an AGENTS.md written for notesDir back into the
// wording of the templates, so its generated lines are recognized.
func normalizeNotes(text, notesDir string) string {
	if notesDir == "" {
		return text
	}
	text = strings.ReplaceAll(text, filepath.Join(notesDir, "PROJECT.md"), filepath.ToSlash(projectRulesPath))
	return strings.ReplaceAll(text, notesDir, notesPlaceholder)
}

// legacyRules returns the content of dir's .instructions/PROJECT.md and
// whether it holds anything besides the untouched template; exists is
// false when there is no such file.
func legacyRules(dir string) (content string, own, exists bool, err error) {
	data, err := os.ReadFile(filepath.Join(dir, projectRulesPath))
	if os.IsNotExist(err) {
		return "", false, false, nil
	}
	if err != nil {
		return "", false, false, fmt.Errorf("read %s: %w", projectRulesPath, err)
	}
	tpl, err := readTpl("templates/PROJECT.md")
	if err != nil {
		return "", false, false, err
	}
	return string(data), strings.TrimSpace(string(data)) != strings.TrimSpace(tpl), true, nil
}

// movedHeading heads a block appended to the notes PROJECT.md.
func movedHeading(from string) string {
	return "## Moved from " + from + " (" + time.Now().Format("2006-01-02") + ")"
}

// notesRules returns the PROJECT.md content of notesDir after the project
// in dir moves its rules there: the current file (or the template), plus
// the rules of dir's .instructions/PROJECT.md and the rules written by
// hand in dir's AGENTS.md (when overwrite) it doesn't hold yet. changed is
// false when the file would stay as it is.
func notesRules(dir, notesDir string, overwrite bool) (content string, changed bool, err error) {
	path := filepath.Join(notesDir, "PROJECT.md")
	current, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}
	content = string(current)
	if !exists {
		if content, err = readTpl("templates/PROJECT.md"); err != nil {
			return "", false, err
		}
	}
	add := func(from, block string) {
		block = strings.TrimSpace(block)
		if block == "" || strings.Contains(content, block) {
			return
		}
		content = strings.TrimRight(content, "\n") + "\n\n" + movedHeading(from) + "\n\n" + block + "\n"
	}
	legacy, own, _, err := legacyRules(dir)
	if err != nil {
		return "", false, err
	}
	switch {
	case own && !exists:
		// A new notes PROJECT.md is the project's own file, moved.
		content = legacy
	case own:
		// Only the project's rules join: the template's own text is in the
		// notes PROJECT.md already.
		tpl, err := readTpl("templates/PROJECT.md")
		if err != nil {
			return "", false, err
		}
		add(filepath.ToSlash(projectRulesPath), strings.TrimPrefix(strings.TrimSpace(legacy), strings.TrimSpace(tpl)))
	}
	if overwrite {
		if agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md")); err == nil {
			moved, err := projectContent(string(agents), notesDir)
			if err != nil {
				return "", false, err
			}
			add("AGENTS.md", moved)
		}
	}
	return content, !exists || content != string(current), nil
}

// ensureNotes sets up notesDir for the project in dir: its folders, its
// PROJECT.md (notesRules) and, when dir has a .instructions/PROJECT.md,
// that file's move: its rules join the notes PROJECT.md and it is renamed
// to PROJECT.md.perci-bak (deleted when it was the untouched template).
func ensureNotes(dir, notesDir string, overwrite bool, stdout io.Writer) error {
	for _, sub := range append([]string{""}, notesSubdirs...) {
		if err := os.MkdirAll(filepath.Join(notesDir, sub), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", filepath.Join(notesDir, sub), err)
		}
	}
	content, changed, err := notesRules(dir, notesDir, overwrite)
	if err != nil {
		return err
	}
	path := filepath.Join(notesDir, "PROJECT.md")
	if changed {
		if err := fsutil.WriteGenerated(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		ui.Info(stdout, i18n.Tr("aic.run.updated", i18n.Args{"path": path}))
	}
	_, own, exists, err := legacyRules(dir)
	if err != nil || !exists {
		return err
	}
	legacyPath := filepath.Join(dir, projectRulesPath)
	if own {
		if err := os.Rename(legacyPath, legacyPath+".perci-bak"); err != nil {
			return fmt.Errorf("rename %s: %w", legacyPath, err)
		}
	} else if err := os.Remove(legacyPath); err != nil {
		return fmt.Errorf("remove %s: %w", legacyPath, err)
	}
	ui.Info(stdout, i18n.Tr("notes.run.moved", i18n.Args{"from": legacyPath, "to": path}))
	return nil
}

// notesChanges tells what ensureNotes would change for the project in dir:
// the notes PROJECT.md (by its absolute path) and dir's
// .instructions/PROJECT.md. Nothing is written.
func notesChanges(dir, notesDir string) ([]Change, error) {
	if notesDir == "" {
		return nil, nil
	}
	var out []Change
	path := filepath.Join(notesDir, "PROJECT.md")
	_, changed, err := notesRules(dir, notesDir, true)
	if err != nil {
		return nil, err
	}
	if changed {
		action := "update"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			action = "create"
		}
		out = append(out, Change{Path: path, Action: action})
	}
	if _, _, exists, err := legacyRules(dir); err != nil {
		return nil, err
	} else if exists {
		out = append(out, Change{Path: filepath.ToSlash(projectRulesPath), Action: "remove"})
	}
	return out, nil
}
