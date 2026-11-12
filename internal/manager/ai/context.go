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
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

//go:embed all:templates
var templateFS embed.FS

// Model represents a project type with its own instruction file.
type Model struct {
	Name        string
	Instruction string // path inside templateFS
	// AppliesTo says, in English, which tasks the standard covers. Agents
	// without "@" import support rely on it to decide which file to read
	// before starting a task.
	AppliesTo string
	// Requires names another model this one only makes sense on top of —
	// GenerateSharedFiles pulls it in automatically when it isn't selected.
	Requires string
	// Legacy lists filenames this model's instruction file had under
	// .instructions/ under a previous name: they still count as "active" in
	// DetectActiveModels, and are deleted once the current file is written
	// (or when the model is removed).
	Legacy []string
	// Description is the message key of what the standard covers, shown on
	// the Contextos de IA screens; DescriptionText returns it in the
	// current language.
	Description string
	// Paths lists globs of the files the standard covers: Claude Code gets
	// it as a rule that loads only when it works on a matching file. Empty
	// for a standard tied to a kind of task rather than to file types.
	Paths []string
}

// docsBase is the base documentation standard every "Documentação: *"
// complement builds on.
const docsBase = "Documentação de projeto"

// Base standards other models complement.
const (
	goBase    = "Go"
	tauriBase = "Rust + Tauri v2"
	slimBase  = "PHP + Slim 4"
)

// models is the catalog of supported instruction models, in output order.
var models = []Model{
	{Name: "Linux Bash", Description: "aic.model.bash.desc", Instruction: "templates/instructions/BASH.md",
		AppliesTo: "Writing or reviewing Bash/shell scripts, CLI tools and installers",
		Paths:     []string{"**/*.sh", "**/*.bash"}},
	{Name: goBase, Description: "aic.model.go.desc", Instruction: "templates/instructions/GO.md",
		AppliesTo: "Writing Go code (CLI tools, MCP servers, desktop backends): layout, errors, concurrency, tests, linting, releases",
		Paths:     []string{"**/*.go", "go.mod", "go.sum", ".golangci.yml"}},
	{Name: "Go: Servidor MCP", Description: "aic.model.go-mcp.desc", Instruction: "templates/instructions/GO-MCP.md", Requires: goBase,
		AppliesTo: "Building an MCP server with the official Go SDK — together with GO.md",
		Paths:     []string{"**/*.go"}},
	{Name: "Go: Wails v3", Description: "aic.model.wails.desc", Instruction: "templates/instructions/WAILS.md", Requires: goBase,
		AppliesTo: "Building a Wails v3 desktop app (services, bindings, frontend, packaging) — together with GO.md",
		Paths:     []string{"**/*.go", "frontend/**"}},
	{Name: tauriBase, Description: "aic.model.tauri.desc", Instruction: "templates/instructions/TAURI.md",
		AppliesTo: "Building a Rust + Tauri v2 app (commands, capabilities, CSP, build, updater)",
		Paths:     []string{"src-tauri/**", "**/*.rs"}},
	{Name: "Tauri: Vanilla TS + Vite", Description: "aic.model.tauri-vanilla.desc", Instruction: "templates/instructions/TAURI-VANILLA.md", Requires: tauriBase,
		AppliesTo: "Frontend of a Tauri v2 app in vanilla TypeScript + Vite — together with TAURI.md",
		Paths:     []string{"src/**/*.ts", "index.html", "vite.config.*"}},
	{Name: "Tauri: SvelteKit", Description: "aic.model.tauri-sveltekit.desc", Instruction: "templates/instructions/TAURI-SVELTEKIT.md", Requires: tauriBase,
		AppliesTo: "Frontend of a Tauri v2 app in SvelteKit (static) — together with TAURI.md",
		Paths:     []string{"src/**/*.{ts,svelte}", "svelte.config.*", "vite.config.*"}},
	{Name: "Tauri: React + Vite", Description: "aic.model.tauri-react.desc", Instruction: "templates/instructions/TAURI-REACT.md", Requires: tauriBase,
		AppliesTo: "Frontend of a Tauri v2 app in React + TypeScript + Vite — together with TAURI.md",
		Paths:     []string{"src/**/*.{ts,tsx}", "index.html", "vite.config.*"}},
	{Name: slimBase, Description: "aic.model.php-slim.desc", Instruction: "templates/instructions/PHP-SLIM.md",
		AppliesTo: "Building PHP systems with Slim 4 (layout, routing, middleware, DI, security, tests)",
		Paths:     []string{"**/*.php", "composer.json"}},
	{Name: "Slim: PDO", Description: "aic.model.slim-pdo.desc", Instruction: "templates/instructions/SLIM-PDO.md", Requires: slimBase,
		AppliesTo: "Data access with PDO in a Slim 4 system — together with PHP-SLIM.md",
		Paths:     []string{"**/*.php"}},
	{Name: "Slim: Doctrine DBAL", Description: "aic.model.slim-dbal.desc", Instruction: "templates/instructions/SLIM-DBAL.md", Requires: slimBase,
		AppliesTo: "Data access with Doctrine DBAL in a Slim 4 system — together with PHP-SLIM.md",
		Paths:     []string{"**/*.php"}},
	{Name: "Slim: Eloquent", Description: "aic.model.slim-eloquent.desc", Instruction: "templates/instructions/SLIM-ELOQUENT.md", Requires: slimBase,
		AppliesTo: "Data access with Eloquent (illuminate/database) in a Slim 4 system — together with PHP-SLIM.md",
		Paths:     []string{"**/*.php"}},
	{Name: "Moodle", Description: "aic.model.moodle.desc", Instruction: "templates/instructions/MOODLE.md",
		AppliesTo: "Developing Moodle plugins (file structure, Moodle APIs, architecture)",
		Paths:     []string{"**/*.php", "db/**", "templates/**/*.mustache", "amd/**", "lang/**"}},
	{Name: phpPerciName, Description: "aic.model.php-perci82.desc", Instruction: "templates/instructions/PHP-PERCI82.md",
		AppliesTo: "Running PHP code, its tools and tests, and reaching the database, in this project's Perci82 Dev Docker container",
		Paths:     []string{"**/*.php", "composer.json"}},
	{Name: "Dart + Flutter", Description: "aic.model.flutter.desc", Instruction: "templates/instructions/FLUTTER.md",
		AppliesTo: "Building Dart/Flutter apps (architecture, state, testing, release)",
		Paths:     []string{"**/*.dart", "pubspec.yaml", "analysis_options.yaml"}},
	{Name: "Godot 4 (GDScript)", Description: "aic.model.godot.desc", Instruction: "templates/instructions/GODOT.md",
		AppliesTo: "Building games in Godot 4 with GDScript, targeting Android",
		Paths:     []string{"**/*.gd", "**/*.tscn", "**/*.tres", "project.godot"}},
	{Name: "MySQL/MariaDB", Description: "aic.model.mysql-mariadb.desc", Instruction: "templates/instructions/MYSQL-MARIADB.md",
		AppliesTo: "MySQL/MariaDB schema design, migrations, query tuning and configuration",
		Paths:     []string{"**/*.sql", "**/migrations/**"}},
	{Name: "TypeScript/Node", Description: "aic.model.typescript-node.desc", Instruction: "templates/instructions/TYPESCRIPT-NODE.md",
		AppliesTo: "Writing TypeScript or Node.js code (backends, CLIs, libraries): tsconfig, modules, typing, errors, security, tests",
		Paths:     []string{"**/*.{ts,tsx,js,mjs,cjs}", "package.json", "tsconfig*.json"}},
	{Name: "Docker/Compose", Description: "aic.model.docker.desc", Instruction: "templates/instructions/DOCKER.md",
		AppliesTo: "Writing Dockerfiles and Compose files (images, builds, security, runtime)",
		Paths:     []string{"**/Dockerfile*", "**/*compose*.y*ml", ".dockerignore"}},
	{Name: "CI (GitHub Actions)", Description: "aic.model.github-actions.desc", Instruction: "templates/instructions/GITHUB-ACTIONS.md",
		AppliesTo: "Writing GitHub Actions workflows (triggers, permissions, security, caching, releases)",
		Paths:     []string{".github/workflows/**"}},
	{Name: "Website", Description: "aic.model.website.desc", Instruction: "templates/instructions/WEBSITE.md",
		AppliesTo: "Building websites (structure, accessibility, performance, SEO, hosting)"},
	{Name: "Landing Page Design", Description: "aic.model.landing-page.desc", Instruction: "templates/instructions/LANDING-PAGE.md",
		AppliesTo: "Designing or building landing pages"},
	{Name: docsBase, Description: "aic.model.project-documentation.desc", Instruction: "templates/instructions/PROJECT-DOCUMENTATION.md", Legacy: []string{"DOCUMENTATION-SITE.md"},
		AppliesTo: "Creating or updating project documentation (README, docs/, CONTRIBUTING, CODE_OF_CONDUCT, LICENSE, license headers, GitHub community files, repository metadata, releases)"},
	{Name: "Documentação: Servidor MCP", Description: "aic.model.project-documentation-mcp.desc", Instruction: "templates/instructions/PROJECT-DOCUMENTATION-MCP.md", Requires: docsBase,
		AppliesTo: "Documentation of an MCP server — together with PROJECT-DOCUMENTATION.md"},
	{Name: "Documentação: Plugin Moodle", Description: "aic.model.project-documentation-moodle.desc", Instruction: "templates/instructions/PROJECT-DOCUMENTATION-MOODLE.md", Requires: docsBase,
		AppliesTo: "Documentation of a Moodle plugin — together with PROJECT-DOCUMENTATION.md"},
	{Name: "Documentação: Aplicação Desktop", Description: "aic.model.project-documentation-desktop.desc", Instruction: "templates/instructions/PROJECT-DOCUMENTATION-DESKTOP.md", Requires: docsBase,
		AppliesTo: "Documentation of a desktop application — together with PROJECT-DOCUMENTATION.md"},
	{Name: "Documentação: Aplicação Mobile", Description: "aic.model.project-documentation-mobile.desc", Instruction: "templates/instructions/PROJECT-DOCUMENTATION-MOBILE.md", Requires: docsBase,
		AppliesTo: "Documentation of a mobile application — together with PROJECT-DOCUMENTATION.md"},
	{Name: "Documentação: Aplicação Web", Description: "aic.model.project-documentation-web.desc", Instruction: "templates/instructions/PROJECT-DOCUMENTATION-WEB.md", Requires: docsBase,
		AppliesTo: "Documentation of a web application — together with PROJECT-DOCUMENTATION.md"},
	{Name: "Documentação: Jogo", Description: "aic.model.project-documentation-game.desc", Instruction: "templates/instructions/PROJECT-DOCUMENTATION-GAME.md", Requires: docsBase,
		AppliesTo: "Documentation of a game — together with PROJECT-DOCUMENTATION.md"},
}

// Preset is a project type: the models it checks at once. Hint, when set,
// tells the user what still has to be chosen by hand.
type Preset struct {
	Name   string
	Models []string
	// Hint is the message key of what to add by hand, or "".
	Hint string
}

// modelLabelKeys maps the standards whose name is in Portuguese to the
// message key of their label; the other names are shown as they are.
// The names themselves identify the standards in config.yaml and in
// export files, so they never change.
var modelLabelKeys = map[string]string{
	docsBase:                          "aic.model.docs",
	"Documentação: Servidor MCP":      "aic.model.docsMCP",
	"Documentação: Plugin Moodle":     "aic.model.docsMoodle",
	"Documentação: Aplicação Desktop": "aic.model.docsDesktop",
	"Documentação: Aplicação Mobile":  "aic.model.docsMobile",
	"Documentação: Aplicação Web":     "aic.model.docsWeb",
	"Documentação: Jogo":              "aic.model.docsGame",
}

// presetLabelKeys maps each project type to the message key of its label.
var presetLabelKeys = map[string]string{
	"Servidor MCP em Go":         "aic.preset.goMCP",
	"Aplicativo Go + Wails":      "aic.preset.wails",
	"Plugin Moodle":              "aic.preset.moodle",
	"Aplicativo Rust + Tauri v2": "aic.preset.tauri",
	"Website":                    "aic.preset.website",
	"Sistema PHP + Slim":         "aic.preset.slim",
	"Jogo Godot (Android)":       "aic.preset.godot",
}

// ModelLabel returns how the screens name the standard called name, in the
// current language.
func ModelLabel(name string) string {
	if key, ok := modelLabelKeys[name]; ok {
		return i18n.Tr(key, nil)
	}
	return name
}

// DescriptionText returns what the standard covers, in the current
// language.
func (m Model) DescriptionText() string {
	return i18n.Tr(m.Description, nil)
}

// LabelText returns the project type's label in the current language.
func (p Preset) LabelText() string {
	if key, ok := presetLabelKeys[p.Name]; ok {
		return i18n.Tr(key, nil)
	}
	return p.Name
}

// HintText returns the project type's hint in the current language, or "".
func (p Preset) HintText() string {
	if p.Hint == "" {
		return ""
	}
	return i18n.Tr(p.Hint, nil)
}

// presets lists the project types, in display order.
var presets = []Preset{
	{Name: "Servidor MCP em Go", Models: []string{goBase, "Go: Servidor MCP", docsBase, "Documentação: Servidor MCP"}},
	{Name: "Aplicativo Go + Wails", Models: []string{goBase, "Go: Wails v3", docsBase, "Documentação: Aplicação Desktop"}},
	{Name: "Plugin Moodle", Models: []string{"Moodle", phpPerciName, "MySQL/MariaDB", docsBase, "Documentação: Plugin Moodle"}},
	{Name: "Aplicativo Rust + Tauri v2", Models: []string{tauriBase, docsBase, "Documentação: Aplicação Desktop"},
		Hint: "aic.preset.tauri.hint"},
	{Name: "Website", Models: []string{"Website", docsBase, "Documentação: Aplicação Web"},
		Hint: "aic.preset.website.hint"},
	{Name: "Sistema PHP + Slim", Models: []string{slimBase, phpPerciName, "MySQL/MariaDB", docsBase, "Documentação: Aplicação Web"},
		Hint: "aic.preset.slim.hint"},
	{Name: "Jogo Godot (Android)", Models: []string{"Godot 4 (GDScript)", docsBase, "Documentação: Jogo"}},
}

// Presets returns the project types.
func Presets() []Preset {
	return presets
}

// withRequired returns active plus every model an active one Requires that
// wasn't selected, in catalog order (so AGENTS.md always references a base
// before its complements), and the names of the models it added.
func withRequired(active []Model) (resolved []Model, added []string) {
	selected := make(map[string]bool, len(active))
	for _, m := range active {
		selected[m.Name] = true
	}
	for _, m := range active {
		if m.Requires != "" && !selected[m.Requires] {
			selected[m.Requires] = true
			added = append(added, m.Requires)
		}
	}
	if len(added) == 0 {
		return active, nil
	}
	for _, m := range models {
		if selected[m.Name] {
			resolved = append(resolved, m)
		}
	}
	return resolved, added
}

// Models returns the list of all supported project models.
func Models() []Model {
	return models
}

// DetectActiveModels checks the .instructions/ directory under dir and
// returns which models already have their instruction file present. dir
// is the folder the user picked (empty means the process's current
// directory — same convention as internal/manager/repo and
// internal/manager/gitignore).
func DetectActiveModels(dir string) map[string]bool {
	present := make(map[string]bool, len(models))
	for _, m := range models {
		for _, name := range append([]string{filepath.Base(m.Instruction)}, m.Legacy...) {
			if _, err := os.Stat(filepath.Join(dir, ".instructions", name)); err == nil {
				present[m.Name] = true
				break
			}
		}
	}
	return present
}

var (
	moodleVersionRe    = regexp.MustCompile(`\$version\s*=\s*([\d.]+)\s*;`)
	moodleReleaseRe    = regexp.MustCompile(`\$release\s*=\s*'([^']+)'\s*;`)
	moodleMajorMinorRe = regexp.MustCompile(`^(\d+\.\d+)`)
	pluginRequiresRe   = regexp.MustCompile(`\$plugin->requires\s*=\s*([\d.]+)\s*;`)
)

// moodleUnknown fills MOODLE.md's placeholders when no version.php gives
// them, so the file never carries raw {{…}} markers.
const moodleUnknown = "unknown (read `$plugin->requires` in the plugin's version.php, or ask the user)"

// maxVersionPHPSize caps how much of version.php is read — it's a project
// file the user controls, not something perci generates, so a corrupted
// file or a symlink pointing somewhere huge shouldn't be read in full before
// the regexes above even run. 1 MiB is generously above any real version.php.
const maxVersionPHPSize = 1 << 20

// readCapped reads at most maxVersionPHPSize bytes from path.
func readCapped(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(io.LimitReader(f, maxVersionPHPSize))
}

// detectMoodleVars reads the values of MOODLE.md's placeholders from
// projectDir: from a Moodle checkout's version.php (or public/version.php,
// the Moodle 5.1+ layout) — its $version and $release — or from a plugin's
// version.php — the Moodle version its $plugin->requires names. ok is false
// when neither is found.
func detectMoodleVars(projectDir string) (vars map[string]string, ok bool) {
	data, err := readCapped(filepath.Join(projectDir, "version.php"))
	if err != nil {
		data, err = readCapped(filepath.Join(projectDir, "public", "version.php"))
		if err != nil {
			return nil, false
		}
	}

	if m := pluginRequiresRe.FindSubmatch(data); m != nil {
		return map[string]string{
			"MOODLE_VERSION":     "the release `$plugin->requires` (" + string(m[1]) + ") names",
			"MOODLE_FULLVERSION": string(m[1]),
		}, true
	}

	versionMatch := moodleVersionRe.FindSubmatch(data)
	releaseMatch := moodleReleaseRe.FindSubmatch(data)
	if versionMatch == nil || releaseMatch == nil {
		return nil, false
	}
	majorMinor := moodleMajorMinorRe.FindString(string(releaseMatch[1]))
	if majorMinor == "" {
		return nil, false
	}
	return map[string]string{
		"MOODLE_VERSION":     majorMinor,
		"MOODLE_FULLVERSION": string(versionMatch[1]),
	}, true
}

// moodleVarsFor is detectMoodleVars of dir ("" for the current folder),
// with moodleUnknown for both placeholders when nothing was detected.
func moodleVarsFor(dir string) (vars map[string]string, detected bool) {
	if dir == "" {
		dir = "."
	}
	if vars, ok := detectMoodleVars(dir); ok {
		return vars, true
	}
	return map[string]string{"MOODLE_VERSION": moodleUnknown, "MOODLE_FULLVERSION": moodleUnknown}, false
}

// moodleVars is moodleVarsFor, telling the user when nothing was detected.
func moodleVars(stdout io.Writer, dir string) map[string]string {
	vars, detected := moodleVarsFor(dir)
	if !detected {
		ui.Warning(stdout, i18n.Tr("aic.run.noVersionPHP", nil))
	}
	return vars
}

// standardsHeader opens AGENTS.md's list of active standards. The list
// names each file without an "@" import, so no agent loads every standard
// on every session: each entry says which tasks it covers, and agents are
// told to read the file themselves before such a task. With claudeRules,
// it adds that Claude Code loads the file-type standards by itself.
func standardsHeader(claudeRules bool) string {
	h := "\n\n## Language-Specific Standards\n\nThe standards below are **mandatory** for the tasks they cover. **Before starting a task one of them applies to, read the whole file.** When several apply, follow all of them. They complement the rules above and never override them, nor `.instructions/PROJECT.md`.\n"
	if claudeRules {
		h += "\nClaude Code loads the standards that cover file types by itself, from `.claude/rules/perci/`, when it works on a matching file.\n"
	}
	return h
}

// GenerateSharedFiles writes, for the agents named by agentIDs (see
// Agents), AGENTS.md listing every active model, each active model's
// instruction file under dir/.instructions/, the project rules file
// .instructions/PROJECT.md (created once, see ensureProjectRules) and each
// agent's own files: CLAUDE.md, the path-scoped .claude/rules/perci/*.md
// and the .claude/settings.json read rules for Claude Code; GEMINI.md and
// .aiexclude for Gemini; the project rules entry of opencode.json for
// OpenCode. Codex and Antigravity read AGENTS.md as is. The files of the
// agents not named are removed while they are still what Perci wrote. If
// overwrite is true, existing files are replaced without confirmation. dir
// empty means the current directory. Progress and warnings go to stdout
// (the GUI's terminal panel).
func GenerateSharedFiles(dir string, active []Model, agentIDs []string, overwrite bool, stdout io.Writer) error {
	return generate(dir, dir, active, agentIDs, overwrite, stdout)
}

// generate is GenerateSharedFiles writing into dir while taking the list
// of files git tracks from gitDir (the project itself when dir is a
// preview copy of it).
func generate(dir, gitDir string, active []Model, agentIDs []string, overwrite bool, stdout io.Writer) error {
	selected, err := agentSet(agentIDs)
	if err != nil {
		return err
	}
	active, added := withRequired(active)
	for _, name := range added {
		ui.Info(stdout, i18n.Tr("aic.run.baseAdded", i18n.Args{"name": ModelLabel(name)}))
	}
	active, dropped := applicable(gitDir, active)
	if dropped {
		ui.Warning(stdout, i18n.Tr("aic.run.noApp", nil))
	}
	g := grants{}

	// Rules hand-written into AGENTS.md move to PROJECT.md before AGENTS.md
	// is regenerated: the notes folder's when the project has one, written
	// only on a real run (a preview copy never reaches the notes folder).
	notesDir := NotesDir(gitDir)
	g.notesDir = notesDir
	switch {
	case notesDir == "":
		if err := ensureProjectRules(dir, overwrite, stdout); err != nil {
			return err
		}
	case dir == gitDir:
		if err := ensureNotes(dir, notesDir, overwrite, stdout); err != nil {
			return err
		}
	}

	// Each referenced standard on disk.
	rendered := make(map[string]string, len(active))
	for _, m := range active {
		if m.Name == phpPerciName {
			// Written from the settings of the Docker app holding the real
			// project (gitDir), also on a preview copy.
			content, command, _, err := phpGuideFor(gitDir)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Join(dir, ".instructions"), 0o755); err != nil {
				return err
			}
			if err := WriteFile(filepath.Join(dir, ".instructions", filepath.Base(m.Instruction)), content, overwrite, stdout); err != nil {
				return err
			}
			rendered[m.Name] = content
			g.commands = append(g.commands, command)
			continue
		}
		var vars map[string]string
		if m.Name == "Moodle" {
			vars = moodleVars(stdout, dir)
		}
		if err := WriteInstruction(dir, m, overwrite, stdout, vars); err != nil {
			return err
		}
		content, err := readTpl(m.Instruction)
		if err != nil {
			return err
		}
		rendered[m.Name] = substitute(content, vars)
	}

	agentsContent, err := agentsFile(active, selected, notesDir)
	if err != nil {
		return err
	}
	if err := WriteFile(filepath.Join(dir, "AGENTS.md"), agentsContent, overwrite, stdout); err != nil {
		return err
	}

	if err := writeAgentFiles(dir, gitDir, selected, active, rendered, g, overwrite, stdout); err != nil {
		return err
	}

	// Remove instruction files for models that were active before this run
	// but aren't selected anymore. Only a copy still identical to what Perci
	// generates is deleted: DetectActiveModels goes by filename, so a
	// hand-written or edited BASH.md is "active" too, and deleting it would
	// lose the user's work.
	detected := DetectActiveModels(dir)
	activeNow := make(map[string]bool, len(active))
	for _, m := range active {
		activeNow[m.Name] = true
	}
	for _, m := range models {
		if !detected[m.Name] || activeNow[m.Name] {
			continue
		}
		current := filepath.Join(dir, ".instructions", filepath.Base(m.Instruction))
		if _, err := os.Stat(current); err == nil && !isPristine(dir, m) {
			ui.Warning(stdout, i18n.Tr("aic.run.editedKept", i18n.Args{"path": ".instructions/" + filepath.Base(m.Instruction)}))
			if overwrite {
				removeLegacy(dir, m, stdout)
			}
			continue
		}
		if _, err := os.Stat(current); err != nil && !overwrite {
			// Only an old-named copy is left (Model.Legacy): same rule as
			// WriteInstruction — removed only on an overwrite run.
			continue
		}
		RemoveInstruction(dir, m, stdout)
	}

	return nil
}

// agentsFile is the AGENTS.md content for the models of active (already
// resolved by withRequired) and the agents of selected: the base rules,
// the list of standards (one line per model, the file named in a code
// span so no agent imports it), the notes section when notesDir is not ""
// (the rules then name notesDir's PROJECT.md) and the marker naming the
// agents.
func agentsFile(active []Model, selected map[string]bool, notesDir string) (string, error) {
	rawBasic, err := readTpl("templates/BASIC.md")
	if err != nil {
		return "", err
	}
	claudeRules := false
	var list strings.Builder
	for _, m := range active {
		fmt.Fprintf(&list, "\n- %s: `.instructions/%s`", m.AppliesTo, filepath.Base(m.Instruction))
		claudeRules = claudeRules || (selected[AgentClaude] && len(m.Paths) > 0)
	}
	content := strings.TrimRight(rawBasic, "\n")
	if len(active) > 0 {
		content += standardsHeader(claudeRules) + list.String()
	}
	content = withNotes(content, notesDir)
	if notesDir != "" {
		section, err := notesSection(notesDir)
		if err != nil {
			return "", err
		}
		content += "\n\n" + section
	}
	return content + "\n\n" + agentsMarker(selected) + "\n", nil
}

// writeAgentFiles writes the files of the selected agents, with what g
// grants them, and removes those of the others while they are still what
// Perci wrote.
func writeAgentFiles(dir, gitDir string, selected map[string]bool, active []Model, rendered map[string]string, g grants, overwrite bool, stdout io.Writer) error {
	claudeStub, err := readTpl("templates/CLAUDE-STUB.md")
	if err != nil {
		return err
	}
	oldClaudeStub, err := readTpl("templates/legacy/CLAUDE-STUB-1.0.md")
	if err != nil {
		return err
	}
	geminiStub, err := readTpl("templates/GEMINI-STUB.md")
	if err != nil {
		return err
	}
	patterns, err := ignorePatterns(gitDir)
	if err != nil {
		return err
	}
	aiexclude, err := ignoreFile("templates/ignore/aiexclude-header", patterns)
	if err != nil {
		return err
	}
	geminiignore, err := ignoreFile("templates/ignore/geminiignore-header", patterns)
	if err != nil {
		return err
	}

	if selected[AgentClaude] {
		if err := writeStub(dir, "CLAUDE.md", withNotes(claudeStub, g.notesDir), overwrite, stdout); err != nil {
			return err
		}
		if err := syncClaudeRules(dir, active, rendered, overwrite, stdout); err != nil {
			return err
		}
		// Claude Code does not read an ignore file at all: its equivalent
		// read rules live in its settings.
		if err := writeClaudeSettings(dir, denyRules(patterns), stdout); err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(dir, ".claudeignore")); err == nil {
			ui.Warning(stdout, i18n.Tr("aic.run.claudeignore", nil))
		}
	} else {
		removeIfPristine(filepath.Join(dir, "CLAUDE.md"), stdout, claudeStub, oldClaudeStub, withNotes(claudeStub, NotesDir(gitDir)))
		if err := syncClaudeRules(dir, nil, nil, overwrite, stdout); err != nil {
			return err
		}
		if err := writeClaudeSettings(dir, nil, stdout); err != nil {
			return err
		}
		// Only empty folders go.
		_ = os.Remove(filepath.Join(dir, ".claude", "rules"))
		_ = os.Remove(filepath.Join(dir, ".claude"))
	}

	if selected[AgentGemini] {
		if err := writeStub(dir, "GEMINI.md", withNotes(geminiStub, g.notesDir), overwrite, stdout); err != nil {
			return err
		}
		// .aiexclude is read by Gemini Code Assist, .geminiignore by Gemini CLI.
		if err := WriteFile(filepath.Join(dir, ".aiexclude"), aiexclude, overwrite, stdout); err != nil {
			return err
		}
		if err := WriteFile(filepath.Join(dir, ".geminiignore"), geminiignore, overwrite, stdout); err != nil {
			return err
		}
	} else {
		removeIfPristine(filepath.Join(dir, "GEMINI.md"), stdout, geminiStub, withNotes(geminiStub, NotesDir(gitDir)))
		oldIgnores := []string{aiexclude}
		for _, path := range legacyIgnoreFiles {
			old, err := readTpl(path)
			if err != nil {
				return err
			}
			oldIgnores = append(oldIgnores, old)
		}
		removeIfPristine(filepath.Join(dir, ".aiexclude"), stdout, oldIgnores...)
		removeIfPristine(filepath.Join(dir, ".geminiignore"), stdout, geminiignore)
	}

	// OpenCode gets no read rules (its documented settings don't say how a
	// path pattern is matched); its "permissions" list only grants what g
	// holds.
	if selected[AgentOpenCode] {
		if err := writeOpenCodeConfigFor(dir, withNotes(openCodeRulesEntry, g.notesDir), g, stdout); err != nil {
			return err
		}
	} else if err := writeOpenCodeConfigFor(dir, "", grants{}, stdout); err != nil {
		return err
	}

	// What each agent may reach beyond the project; Antigravity reads
	// AGENTS.md only and has no per-project setting for it.
	pick := func(id string) grants {
		if selected[id] {
			return g
		}
		return grants{}
	}
	if err := writeClaudeGrants(dir, pick(AgentClaude), stdout); err != nil {
		return err
	}
	if err := writeGeminiGrants(dir, pick(AgentGemini), stdout); err != nil {
		return err
	}
	return writeCodexGrants(dir, pick(AgentCodex), stdout)
}

// sameFile reports whether a and b exist and are the same file after
// following links.
func sameFile(a, b string) bool {
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(ia, ib)
}

// WriteFile writes content to path, returning an error if it fails.
// If overwrite is false and the file exists, it skips writing it and does not error.
// Otherwise it goes through fsutil.WriteGenerated: written atomically
// through a symlink at path, with a .perci-bak of a different previous
// version.
func WriteFile(path, content string, overwrite bool, stdout io.Writer) error {
	if _, err := os.Stat(path); err == nil && !overwrite {
		return nil
	}
	if err := fsutil.WriteGenerated(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	ui.Info(stdout, i18n.Tr("aic.run.created", i18n.Args{"path": path}))
	return nil
}

// WriteInstruction writes the instruction file for a given model under
// dir/.instructions/.
func WriteInstruction(dir string, model Model, overwrite bool, stdout io.Writer, vars map[string]string) error {
	instrDir := filepath.Join(dir, ".instructions")
	if err := os.MkdirAll(instrDir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(instrDir, filepath.Base(model.Instruction))
	content, err := readTpl(model.Instruction)
	if err != nil {
		return err
	}
	content = substitute(content, vars)
	if err := WriteFile(dest, content, overwrite, stdout); err != nil {
		return err
	}
	// Without overwrite the caller asked not to clobber customizations, and
	// a legacy copy may be one — leave it for an explicit overwrite run.
	if overwrite {
		removeLegacy(dir, model, stdout)
	}
	return nil
}

// removeLegacy deletes the pre-rename copies of model's instruction file
// under dir/.instructions/ (see Model.Legacy).
func removeLegacy(dir string, model Model, stdout io.Writer) {
	for _, name := range model.Legacy {
		dest := filepath.Join(dir, ".instructions", name)
		if err := os.Remove(dest); err == nil {
			ui.Info(stdout, i18n.Tr("aic.run.removedOld", i18n.Args{"path": dest}))
		}
	}
}

// RemoveInstruction deletes the instruction file of a model under dir/.instructions/.
func RemoveInstruction(dir string, model Model, stdout io.Writer) {
	dest := filepath.Join(dir, ".instructions", filepath.Base(model.Instruction))
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		ui.Warning(stdout, i18n.Tr("aic.run.removeFailed", i18n.Args{"path": dest, "error": err.Error()}))
	} else if err == nil {
		ui.Info(stdout, i18n.Tr("aic.run.removed", i18n.Args{"path": dest}))
	}
	removeLegacy(dir, model, stdout)
}

// isPristine reports whether dir's copy of model's instruction file is
// still exactly what Perci writes: the template itself, or — for MOODLE.md
// — the template with this folder's version.php values substituted.
func isPristine(dir string, model Model) bool {
	got, err := os.ReadFile(filepath.Join(dir, ".instructions", filepath.Base(model.Instruction)))
	if err != nil {
		return false
	}
	tpl, err := readTpl(model.Instruction)
	if err != nil {
		return false
	}
	if string(got) == tpl {
		return true
	}
	if model.Name == "Moodle" {
		vars, _ := moodleVarsFor(dir)
		return string(got) == substitute(tpl, vars)
	}
	if model.Name == phpPerciName {
		return signedUnedited(string(got))
	}
	return false
}

// ignoreFile is the ignore file made of the header template at header and
// patterns.
func ignoreFile(header, patterns string) (string, error) {
	h, err := readTpl(header)
	if err != nil {
		return "", err
	}
	return h + patterns, nil
}

// substitute replaces each {{KEY}} of content with vars[KEY].
func substitute(content string, vars map[string]string) string {
	for k, v := range vars {
		content = strings.ReplaceAll(content, "{{"+k+"}}", v)
	}
	return content
}

// renderInstruction is model's standard as Perci writes it for dir: for
// MOODLE.md, with the placeholders filled (moodleVarsFor).
func renderInstruction(dir string, model Model) (string, error) {
	content, err := readTpl(model.Instruction)
	if err != nil {
		return "", err
	}
	if model.Name == "Moodle" {
		vars, _ := moodleVarsFor(dir)
		content = substitute(content, vars)
	}
	if model.Name == phpPerciName {
		guide, _, ok, err := phpGuideFor(dir)
		if err != nil || !ok {
			return "", fmt.Errorf("no Docker app holds %s", dir)
		}
		content = guide
	}
	return content, nil
}

func readTpl(path string) (string, error) {
	data, err := templateFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("template %s not found: %w", path, err)
	}
	return string(data), nil
}
