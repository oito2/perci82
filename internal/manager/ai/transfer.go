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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// TransferKind and TransferVersion identify the AI contexts export file
// and the version of its format this build writes and reads.
const (
	TransferKind    = "perci-ai-contexts"
	TransferVersion = 1
)

// TransferFile is the AI contexts export file: the registered folders
// with their standards and agents.
type TransferFile struct {
	Kind     string            `yaml:"kind" json:"kind"`
	Version  int               `yaml:"version" json:"version"`
	Contexts []TransferContext `yaml:"contexts" json:"contexts"`
}

// TransferContext is one folder of a TransferFile. With Relative, Path is
// relative to the workspace folder (written with "/"); otherwise it is
// absolute.
type TransferContext struct {
	Name     string   `yaml:"name" json:"name"`
	Path     string   `yaml:"path" json:"path"`
	Relative bool     `yaml:"relative,omitempty" json:"relative"`
	Models   []string `yaml:"models,omitempty" json:"models"`
	Agents   []string `yaml:"agents" json:"agents"`
}

// ErrNothingToExport is returned when an export would hold no folder.
var ErrNothingToExport = i18n.NewError("aic.nothingToExport")

// Exportable converts contexts into an export file: a folder inside
// workspace gets its path relative to it, any other keeps its absolute
// path.
func Exportable(contexts []config.AIContext, workspace string) TransferFile {
	f := TransferFile{Kind: TransferKind, Version: TransferVersion, Contexts: []TransferContext{}}
	for _, c := range contexts {
		t := TransferContext{Name: c.Name, Path: c.Path, Models: c.Models, Agents: c.Agents}
		if workspace != "" {
			if rel, err := filepath.Rel(filepath.Clean(workspace), c.Path); err == nil && rel != "." && filepath.IsLocal(rel) {
				t.Path, t.Relative = filepath.ToSlash(rel), true
			}
		}
		f.Contexts = append(f.Contexts, t)
	}
	return f
}

// WriteTransfer writes f to path as YAML, atomically, readable only by the
// user.
func WriteTransfer(path string, f TransferFile) error {
	if len(f.Contexts) == 0 {
		return ErrNothingToExport
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("generate yaml: %w", err)
	}
	if err := fsutil.WriteFileAtomic(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// LoadTransfer reads and validates the export file at path. Unknown keys
// are refused, so a file of another kind is not mistaken for one.
func LoadTransfer(path string) (TransferFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TransferFile{}, fmt.Errorf("read %s: %w", path, err)
	}
	var f TransferFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil || f.Kind != TransferKind {
		return TransferFile{}, errors.New(i18n.Tr("aic.notTransferFile", i18n.Args{"path": path}))
	}
	if err := f.Validate(); err != nil {
		return TransferFile{}, fmt.Errorf("%s: %w", i18n.Tr("aic.invalidFile", nil), err)
	}
	return f, nil
}

// Validate checks the format version and every folder: a valid name
// (normalizeName), unique in the file; a clean path, relative and inside
// the workspace when Relative, absolute otherwise, unique in the file.
// Standards and agents this version does not know are not an error here
// (PlanImport reports them).
func (f TransferFile) Validate() error {
	switch {
	case f.Version == 0:
		return i18n.NewError("git.noVersion")
	case f.Version > TransferVersion:
		return errors.New(i18n.Tr("git.newerVersion", i18n.Args{"version": f.Version, "supported": TransferVersion}))
	}
	var seen []config.AIContext
	for _, t := range f.Contexts {
		name, err := normalizeName(seen, t.Name, "")
		if err != nil {
			return fmt.Errorf("folder %q: %w", t.Path, err)
		}
		if name != t.Name {
			return fmt.Errorf("name with spaces at its ends: %q", t.Name)
		}
		if err := checkTransferPath(t); err != nil {
			return fmt.Errorf("context %q: %w", t.Name, err)
		}
		// A relative path never starts with "/", so it never equals an
		// absolute one.
		if slices.ContainsFunc(seen, func(c config.AIContext) bool { return c.Path == t.Path }) {
			return fmt.Errorf("context %q: the folder %s appears more than once", t.Name, t.Path)
		}
		seen = append(seen, config.AIContext{Name: name, Path: t.Path})
	}
	return nil
}

// checkTransferPath requires a clean path: relative, below the workspace
// and written with "/" when t.Relative; absolute otherwise.
func checkTransferPath(t TransferContext) error {
	if t.Relative {
		if t.Path == "" || t.Path == "." || !filepath.IsLocal(filepath.FromSlash(t.Path)) || filepath.ToSlash(filepath.Clean(t.Path)) != t.Path {
			return fmt.Errorf("invalid relative path: %q", t.Path)
		}
		return nil
	}
	if !filepath.IsAbs(t.Path) || filepath.Clean(t.Path) != t.Path {
		return fmt.Errorf("invalid path: %q", t.Path)
	}
	return nil
}

// resolvePath is t's folder on this machine: its absolute path, or its
// relative path joined to workspace.
func (t TransferContext) resolvePath(workspace string) string {
	if t.Relative {
		return filepath.Join(workspace, filepath.FromSlash(t.Path))
	}
	return t.Path
}

// Import states of a folder (ImportContext.State).
const (
	ImportApply      = "apply"      // the folder exists: its context is generated and it is registered
	ImportPending    = "pending"    // the folder does not exist: registered as pending
	ImportRegistered = "registered" // the folder is already registered here: kept as it is, skipped
	ImportInvalid    = "invalid"    // cannot be imported (Reason)
)

// ImportContext is one folder of an export file checked against this
// machine: Path is its folder here, Models and Agents the ones this
// version knows, Unknown the standards and agents it does not (left out),
// Reason why it cannot be imported (State ImportInvalid).
type ImportContext struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Models  []string `json:"models"`
	Agents  []string `json:"agents"`
	Unknown []string `json:"unknown"`
	State   string   `json:"state"`
	Reason  string   `json:"reason"`
}

// ImportPick names one folder of an export file (by its name in the file)
// to import as Name; Path, when set, replaces the folder the file gives.
type ImportPick struct {
	Source string `json:"source"`
	Name   string `json:"name"`
	Path   string `json:"path"`
}

// knownSelection splits t's standards and agents into the ones this
// version knows and the ones it does not.
func knownSelection(t TransferContext) (ms, ids, unknown []string) {
	ms, unknown = []string{}, []string{}
	for _, name := range t.Models {
		if slices.ContainsFunc(models, func(m Model) bool { return m.Name == name }) {
			ms = append(ms, name)
		} else {
			unknown = append(unknown, name)
		}
	}
	ids = []string{}
	for _, id := range t.Agents {
		if slices.ContainsFunc(agents, func(a Agent) bool { return a.ID == id }) {
			ids = append(ids, id)
		} else {
			unknown = append(unknown, id)
		}
	}
	return ms, ids, unknown
}

// planOne checks t, to be placed at path, against the registered
// contexts.
func planOne(t TransferContext, path string, contexts []config.AIContext) ImportContext {
	ms, ids, unknown := knownSelection(t)
	ic := ImportContext{Name: t.Name, Path: path, Models: ms, Agents: ids, Unknown: unknown}
	switch {
	case len(ids) == 0:
		ic.State, ic.Reason = ImportInvalid, i18n.Tr("aic.noKnownAgent", nil)
	case slices.ContainsFunc(contexts, func(c config.AIContext) bool { return c.Path == path }):
		ic.State, ic.Reason = ImportRegistered, i18n.Tr("aic.alreadyRegistered", nil)
	default:
		info, err := os.Stat(path)
		switch {
		case err == nil && !info.IsDir():
			ic.State, ic.Reason = ImportInvalid, i18n.Tr("aic.notAFolder", nil)
		case err == nil:
			ic.State = ImportApply
		default:
			ic.State = ImportPending
		}
	}
	return ic
}

// checkFileFolder marks plan invalid when its folder, as the import file
// gives it, is one Perci must not write agent files into without the user
// choosing it in the file picker: the home folder, a folder above it, a
// hidden folder of it (~/.claude, ~/.config...), or a relative folder that
// a link leads out of the workspace.
func checkFileFolder(plan *ImportContext, t TransferContext, workspace string) {
	if plan.State != ImportApply && plan.State != ImportPending {
		return
	}
	if reason := fileFolderProblem(plan.Path, t.Relative, workspace); reason != "" {
		plan.State, plan.Reason = ImportInvalid, reason
	}
}

// fileFolderProblem is checkFileFolder's reason for path, or "".
func fileFolderProblem(path string, relative bool, workspace string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return i18n.Tr("aic.folderTooWide", i18n.Args{"path": path})
	}
	home = resolveExisting(filepath.Clean(home))
	real := resolveExisting(path)
	if rel, err := filepath.Rel(real, home); err == nil && (rel == "." || filepath.IsLocal(rel)) {
		return i18n.Tr("aic.folderTooWide", i18n.Args{"path": path})
	}
	if rel, err := filepath.Rel(home, real); err == nil && filepath.IsLocal(rel) {
		if first, _, _ := strings.Cut(filepath.ToSlash(rel), "/"); strings.HasPrefix(first, ".") {
			return i18n.Tr("aic.folderTooWide", i18n.Args{"path": path})
		}
	}
	if relative {
		ws := resolveExisting(filepath.Clean(workspace))
		if rel, err := filepath.Rel(ws, real); err != nil || !filepath.IsLocal(rel) {
			return i18n.Tr("aic.folderOutsideWorkspace", i18n.Args{"path": path})
		}
	}
	return ""
}

// resolveExisting resolves the links of the deepest existing part of
// path and joins the rest to it.
func resolveExisting(path string) string {
	for p := path; ; p = filepath.Dir(p) {
		if real, err := filepath.EvalSymlinks(p); err == nil {
			rest, err := filepath.Rel(p, path)
			if err != nil {
				return path
			}
			return filepath.Join(real, rest)
		}
		if filepath.Dir(p) == p {
			return path
		}
	}
}

// PlanImport checks every folder of f against this machine, with paths
// (keyed by the folder's name in f) replacing the folders the file gives
// and relative paths placed under workspace — without applying anything.
func PlanImport(f TransferFile, contexts []config.AIContext, workspace string, paths map[string]string) []ImportContext {
	out := make([]ImportContext, len(f.Contexts))
	for i, t := range f.Contexts {
		path := t.resolvePath(workspace)
		picked := false
		if p := paths[t.Name]; p != "" {
			path, picked = filepath.Clean(p), true
		}
		out[i] = planOne(t, path, contexts)
		if !picked {
			checkFileFolder(&out[i], t, workspace)
		}
	}
	return out
}

// importStep is one folder an Import handles.
type importStep struct {
	plan ImportContext
	name string
}

// planPicks checks picks against f and the registered contexts: each pick
// names a folder of f once, gets a valid name not used here nor by another
// pick, and a folder that can be imported (ImportApply or ImportPending)
// not picked twice. A replacement Path must be absolute.
func planPicks(f TransferFile, picks []ImportPick, contexts []config.AIContext, workspace string) ([]importStep, error) {
	if len(picks) == 0 {
		return nil, i18n.NewError("aic.noneSelected")
	}
	taken := slices.Clone(contexts)
	var steps []importStep
	for _, p := range picks {
		i := slices.IndexFunc(f.Contexts, func(t TransferContext) bool { return t.Name == p.Source })
		if i < 0 {
			return nil, errors.New(i18n.Tr("aic.notInFile", i18n.Args{"name": p.Source}))
		}
		if slices.ContainsFunc(steps, func(s importStep) bool { return s.plan.Name == p.Source }) {
			return nil, fmt.Errorf("%q selected more than once", p.Source)
		}
		t := f.Contexts[i]
		path := t.resolvePath(workspace)
		if p.Path != "" {
			if !filepath.IsAbs(p.Path) {
				return nil, fmt.Errorf("%q: path is not absolute: %q", p.Source, p.Path)
			}
			path = filepath.Clean(p.Path)
		}
		plan := planOne(t, path, contexts)
		if p.Path == "" {
			checkFileFolder(&plan, t, workspace)
		}
		if plan.State != ImportApply && plan.State != ImportPending {
			return nil, fmt.Errorf("%q: %s", p.Source, plan.Reason)
		}
		if slices.ContainsFunc(steps, func(s importStep) bool { return s.plan.Path == path }) {
			return nil, errors.New(i18n.Tr("aic.folderTwice", i18n.Args{"name": p.Source, "path": path}))
		}
		name, err := normalizeName(taken, p.Name, path)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", p.Source, err)
		}
		taken = append(taken, config.AIContext{Name: name, Path: path})
		steps = append(steps, importStep{plan: plan, name: name})
	}
	return steps, nil
}

// CheckImport tells whether Import would accept picks, without changing
// anything (planPicks).
func CheckImport(f TransferFile, picks []ImportPick, contexts []config.AIContext, workspace string) error {
	_, err := planPicks(f, picks, contexts, workspace)
	return err
}

// PreviewImport tells, for each pick, the files Import would change in its
// folder; a pending folder has no change.
func PreviewImport(f TransferFile, picks []ImportPick, contexts []config.AIContext, workspace string) ([]ImportContext, [][]Change, error) {
	steps, err := planPicks(f, picks, contexts, workspace)
	if err != nil {
		return nil, nil, err
	}
	plans := make([]ImportContext, len(steps))
	changes := make([][]Change, len(steps))
	for i, s := range steps {
		plans[i] = s.plan
		plans[i].Name = s.name
		changes[i] = []Change{}
		if s.plan.State == ImportApply {
			if changes[i], err = Preview(s.plan.Path, ModelsNamed(s.plan.Models), s.plan.Agents); err != nil {
				return nil, nil, fmt.Errorf("%s: %w", s.name, err)
			}
		}
	}
	return plans, changes, nil
}

// Import registers each picked folder of f under its new name: an existing
// folder first gets its context generated (GenerateSharedFiles), a missing
// one is registered as pending. Folders already registered here are never
// picked (planPicks refuses them). Picks are checked before anything is
// written; a folder that then fails is reported and the next one still
// runs, the import ending with executor.ErrCompletedWithWarnings.
func Import(stdout io.Writer, f TransferFile, picks []ImportPick, workspace string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	steps, err := planPicks(f, picks, cfg.AIContexts, workspace)
	if err != nil {
		return err
	}
	failed := false
	for i, s := range steps {
		ui.Step(stdout, i+1, len(steps), s.name)
		if len(s.plan.Unknown) > 0 {
			ui.Warning(stdout, i18n.Tr("aic.run.unknownLeftOut", i18n.Args{"names": strings.Join(s.plan.Unknown, ", ")}))
		}
		if s.plan.State == ImportApply {
			if err := GenerateSharedFiles(s.plan.Path, ModelsNamed(s.plan.Models), s.plan.Agents, true, stdout); err != nil {
				ui.Err(stdout, s.plan.Path+": "+err.Error())
				failed = true
				continue
			}
		} else {
			ui.Info(stdout, i18n.Tr("aic.run.pending", i18n.Args{"path": s.plan.Path}))
		}
		if err := RecordContext(s.name, s.plan.Path, s.plan.Models, s.plan.Agents); err != nil {
			ui.Err(stdout, s.plan.Path+": "+err.Error())
			failed = true
		}
	}
	if failed {
		return executor.ErrCompletedWithWarnings
	}
	return nil
}
