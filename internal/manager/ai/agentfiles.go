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

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// claudeRulesDir holds the path-scoped Claude Code rules Perci writes, one
// per standard that covers file types; Claude Code loads each only when it
// works on a file its paths match.
var claudeRulesDir = filepath.Join(".claude", "rules", "perci")

// claudeRule is the rule file of model: its paths as YAML frontmatter,
// then the standard itself (content, already rendered).
func claudeRule(model Model, content string) string {
	var b strings.Builder
	b.WriteString("---\npaths:\n")
	for _, p := range model.Paths {
		fmt.Fprintf(&b, "  - %q\n", p)
	}
	b.WriteString("---\n\n")
	b.WriteString(content)
	return b.String()
}

// syncClaudeRules writes the rule of every model of active that covers
// file types, and removes the rules of the other models while they are
// still exactly what Perci wrote (an edited rule is kept, with a warning).
// rendered holds each active model's standard as written to .instructions/.
// With active nil every rule Perci wrote is removed.
func syncClaudeRules(dir string, active []Model, rendered map[string]string, overwrite bool, stdout io.Writer) error {
	rulesDir := filepath.Join(dir, claudeRulesDir)
	keep := map[string]bool{}
	for _, m := range active {
		if len(m.Paths) == 0 {
			continue
		}
		name := filepath.Base(m.Instruction)
		keep[name] = true
		if err := os.MkdirAll(rulesDir, 0o755); err != nil {
			return err
		}
		if err := WriteFile(filepath.Join(rulesDir, name), claudeRule(m, rendered[m.Name]), overwrite, stdout); err != nil {
			return err
		}
	}
	for _, m := range models {
		name := filepath.Base(m.Instruction)
		if keep[name] || len(m.Paths) == 0 {
			continue
		}
		path := filepath.Join(rulesDir, name)
		if _, err := os.Lstat(path); err != nil {
			continue
		}
		content, err := renderInstruction(dir, m)
		if err != nil {
			return err
		}
		generated := []string{claudeRule(m, content)}
		if data, err := os.ReadFile(path); err == nil && earlierRule(m, data) {
			generated = append(generated, string(data))
		}
		removeIfPristine(path, stdout, generated...)
	}
	// Leaves the folder only when nothing else lives in it.
	_ = os.Remove(rulesDir)
	return nil
}

// earlierRule reports whether data is model's rule as an earlier Perci
// version wrote it: a paths frontmatter followed by a standard whose
// SHA-256 is in templateHistory.
func earlierRule(model Model, data []byte) bool {
	rest, ok := bytes.CutPrefix(data, []byte("---\npaths:\n"))
	if !ok {
		return false
	}
	_, body, ok := bytes.Cut(rest, []byte("\n---\n\n"))
	if !ok {
		return false
	}
	sum := sha256.Sum256(body)
	return slices.Contains(templateHistory[filepath.Base(model.Instruction)], hex.EncodeToString(sum[:]))
}

// removeIfPristine deletes the regular file at path when its content is
// exactly one of generated; an edited copy is kept, with a warning.
func removeIfPristine(path string, stdout io.Writer, generated ...string) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if !slices.Contains(generated, string(data)) {
		ui.Warning(stdout, i18n.Tr("aic.run.notGenerated", i18n.Args{"path": path}))
		return
	}
	if err := os.Remove(path); err != nil {
		ui.Warning(stdout, i18n.Tr("aic.run.removeFailed", i18n.Args{"path": path, "error": err.Error()}))
		return
	}
	ui.Info(stdout, i18n.Tr("aic.run.removed", i18n.Args{"path": path}))
}

// writeStub writes the stub at name (CLAUDE.md, GEMINI.md) unless it
// already resolves to AGENTS.md, which gives the agent the rules as is —
// writing the stub through it would replace AGENTS.md.
func writeStub(dir, name, content string, overwrite bool, stdout io.Writer) error {
	if sameFile(filepath.Join(dir, name), filepath.Join(dir, "AGENTS.md")) {
		ui.Info(stdout, i18n.Tr("aic.run.symlinkKept", i18n.Args{"name": name}))
		return nil
	}
	return WriteFile(filepath.Join(dir, name), content, overwrite, stdout)
}

// openCodeConfig is OpenCode's project configuration file and
// openCodeSchema the schema a new one declares.
const (
	openCodeConfig = "opencode.json"
	openCodeSchema = "https://opencode.ai/config.json"
)

// openCodeRulesEntry is the instructions entry that makes OpenCode load
// the project rules (it does not follow AGENTS.md references by itself).
var openCodeRulesEntry = filepath.ToSlash(projectRulesPath)

// readOpenCodeConfig parses dir's opencode.json: nil (and no error) when
// it does not exist.
func readOpenCodeConfig(dir string) (map[string]any, error) {
	data, err := os.ReadFile(filepath.Join(dir, openCodeConfig))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	doc := map[string]any{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber() // keeps numbers exactly as written
	if err := dec.Decode(&doc); err != nil || doc == nil {
		return nil, fmt.Errorf("%s is not a valid JSON object", openCodeConfig)
	}
	return doc, nil
}

// instructionsOf returns doc's "instructions" list, or an error when it
// has another type.
func instructionsOf(doc map[string]any) ([]any, error) {
	raw, ok := doc["instructions"]
	if !ok {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf(`"instructions" of %s is not a list`, openCodeConfig)
	}
	return list, nil
}

// writeOpenCodeConfig writes doc as dir's opencode.json.
func writeOpenCodeConfig(dir string, doc map[string]any, stdout io.Writer) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return err
	}
	path := filepath.Join(dir, openCodeConfig)
	if err := fsutil.WriteGenerated(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	ui.Info(stdout, i18n.Tr("aic.run.updated", i18n.Args{"path": path}))
	return nil
}

// isOpenCodeRulesEntry reports whether the instructions entry v loads
// project rules Perci set up: .instructions/PROJECT.md, or the PROJECT.md
// of a notes folder.
func isOpenCodeRulesEntry(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	if s == openCodeRulesEntry {
		return true
	}
	return filepath.Base(s) == "PROJECT.md" && isNotesFolder(filepath.Dir(s))
}

// writeOpenCodeConfigFor makes dir's opencode.json load the project rules
// (entry: .instructions/PROJECT.md, or the notes folder's PROJECT.md) and
// hold the permissions of g; entry "" takes Perci's instructions entry and
// permissions out, and removes a file left with nothing else. Every other
// key is kept; a file that can't be merged is left alone with a warning.
func writeOpenCodeConfigFor(dir, entry string, g grants, stdout io.Writer) error {
	if entry != "" {
		if _, err := os.Stat(filepath.Join(dir, "opencode.jsonc")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, openCodeConfig)); os.IsNotExist(err) {
				ui.Warning(stdout, i18n.Tr("aic.run.opencodeJsonc", i18n.Args{"entry": entry}))
				return nil
			}
		}
	}
	doc, err := readOpenCodeConfig(dir)
	if err == nil && doc == nil {
		if entry == "" {
			return nil
		}
		doc = map[string]any{"$schema": openCodeSchema}
	}
	var list []any
	if err == nil {
		list, err = instructionsOf(doc)
	}
	if err != nil {
		if entry != "" {
			ui.Warning(stdout, i18n.Tr("aic.run.opencodeBad", i18n.Args{"error": err.Error()}))
		}
		return nil
	}

	changed := false
	kept := list[:0:0]
	for _, v := range list {
		if v != any(entry) && isOpenCodeRulesEntry(v) {
			changed = true
			continue
		}
		kept = append(kept, v)
	}
	if entry != "" && !slices.Contains(kept, any(entry)) {
		kept = append(kept, entry)
		changed = true
	}
	if len(kept) == 0 {
		delete(doc, "instructions")
	} else {
		doc["instructions"] = kept
	}
	permChanged, err := setOpenCodePermissions(doc, openCodePermissions(g))
	if err != nil {
		ui.Warning(stdout, i18n.Tr("aic.run.opencodeBad", i18n.Args{"error": err.Error()}))
		return nil
	}
	path := filepath.Join(dir, openCodeConfig)
	switch {
	case !changed && !permChanged:
		if entry != "" {
			ui.Info(stdout, i18n.Tr("aic.run.exists", i18n.Args{"path": path}))
		}
		return nil
	case len(doc) == 0 || (len(doc) == 1 && doc["$schema"] == openCodeSchema):
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
		ui.Info(stdout, i18n.Tr("aic.run.removed", i18n.Args{"path": path}))
		return nil
	}
	return writeOpenCodeConfig(dir, doc, stdout)
}

// openCodeHasProjectRules reports whether dir's opencode.json loads the
// project rules.
func openCodeHasProjectRules(dir string) bool {
	doc, err := readOpenCodeConfig(dir)
	if err != nil || doc == nil {
		return false
	}
	list, err := instructionsOf(doc)
	return err == nil && slices.ContainsFunc(list, isOpenCodeRulesEntry)
}
