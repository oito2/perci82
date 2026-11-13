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

// Bound methods for "Skills & MCPs :: MCPs" — a thin layer over
// internal/dev/mcpservers. config.yaml's MCPServers field records which
// servers are registered, per scope, and MCPBinaries the server binaries
// Perci installed.

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/dev/mcpservers"
	"github.com/oito2/perci82/internal/i18n"
)

// PickMCPFolder opens the native folder picker for a "Local" install
// target.
func (t *DevToolsService) PickMCPFolder() (string, error) {
	return t.pickFolder(i18n.Tr("scope.pickProject", nil))
}

// PickMCPServerParam opens the right native dialog for slug's extra
// parameter — a folder picker for "filesystem" (allowed directory), a file
// picker filtered to TOML files for "dbhub" (its dbhub.toml). Returns an
// error if slug doesn't take a parameter or left the catalog.
func (t *DevToolsService) PickMCPServerParam(slug string) (string, error) {
	sv, ok := mcpservers.BySlug(slug)
	if !ok || sv.Retired {
		return "", fmt.Errorf("unknown MCP server: %s", slug)
	}
	switch sv.Param {
	case mcpservers.ParamDirectory:
		return t.pickFolder(sv.ParamLabelText())
	case mcpservers.ParamFile:
		return approvePath(t.wailsApp.Dialog.OpenFile().
			SetTitle(sv.ParamLabelText()).
			AddFilter("TOML (*.toml)", "*.toml").
			PromptForSingleSelection())
	}
	return "", fmt.Errorf("%s takes no extra parameter", slug)
}

// MCPServerRow is one row of "Skills & MCPs :: MCPs". Agents lists, by
// name, the agents that accepted the registration (empty for a record
// written before they were recorded); Partial is set when some agent that
// applies to the scope is missing. Retired marks an entry that left the
// catalog, listed only while registered so it can be removed. Version is
// the release of the binary Perci installed, for a server shipped as one.
type MCPServerRow struct {
	Slug       string   `json:"slug"`
	Name       string   `json:"name"`
	Note       string   `json:"note"`
	Retired    bool     `json:"retired"`
	Installed  bool     `json:"installed"`
	Partial    bool     `json:"partial"`
	Agents     []string `json:"agents"`
	Param      string   `json:"param"`     // persisted value (if installed) — pre-fills "Update"
	ParamKind  string   `json:"paramKind"` // "" | "directory" | "file"
	ParamLabel string   `json:"paramLabel"`
	Manual     bool     `json:"manual"` // true = Godot Studio: no Install/Update/Remove, just a link/instructions
	ManualURL  string   `json:"manualUrl"`
	ManualNote string   `json:"manualNote"`
	Version    string   `json:"version"`
}

// findMCPServerInstall is findAgentSkillInstall's counterpart for MCP
// server records.
func findMCPServerInstall(installs []config.MCPServerInstall, slug string, global bool, folder string) int {
	return indexOfScoped(installs, mcpServerKey, slug, global, folder)
}

// mcpBinaryOf returns the binary Perci installed for slug, or a zero
// BinaryInfo when there is no record.
func mcpBinaryOf(cfg *config.Config, slug string) mcpservers.BinaryInfo {
	for _, b := range cfg.MCPBinaries {
		if b.Slug == slug {
			return mcpservers.BinaryInfo{Version: b.Version, SHA256: b.SHA256}
		}
	}
	return mcpservers.BinaryInfo{}
}

// GetMCPServerRows lists the whole catalog with its registered state for
// the given scope (global, or local at folder).
func (t *DevToolsService) GetMCPServerRows(global bool, folder string) ([]MCPServerRow, error) {
	if err := requireKnownFolderUnlessGlobal(global, folder); err != nil {
		return nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	rows := make([]MCPServerRow, 0, len(mcpservers.Catalogue))
	for _, sv := range mcpservers.Catalogue {
		row := MCPServerRow{
			Slug: sv.Slug, Name: sv.Name, Note: sv.NoteText(), Retired: sv.Retired, Agents: []string{},
			ParamKind: string(sv.Param), ParamLabel: sv.ParamLabelText(),
			Manual: sv.Manual, ManualURL: sv.ManualURL, ManualNote: sv.ManualNoteText(),
		}
		if idx := findMCPServerInstall(cfg.MCPServers, sv.Slug, global, folder); idx >= 0 {
			rec := cfg.MCPServers[idx]
			row.Installed = true
			row.Param = rec.Param
			row.Partial = isPartial(rec.Agents, mcpservers.ExpectedAgents(sv.Slug, global))
			row.Agents = agentLabels(rec.Agents, mcpservers.AgentLabel)
			row.Version = mcpBinaryOf(cfg, sv.Slug).Version
		} else if sv.Retired {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// recordMCPServerInstall updates config.yaml's MCPServers to reflect
// whether (slug, global, folder) is now registered, and with which agents —
// replacing any existing record for that same key.
func recordMCPServerInstall(slug string, global bool, folder, param string, agents []string, installed bool) error {
	return config.Update(func(cfg *config.Config) error {
		filtered := withoutScoped(cfg.MCPServers, mcpServerKey, slug, global, folder)
		if installed {
			filtered = append(filtered, config.MCPServerInstall{Slug: slug, Global: global, Folder: folder, Param: param, Agents: agents})
		}
		cfg.MCPServers = filtered
		return nil
	})
}

// recordMCPBinary replaces config.yaml's record of slug's binary with info,
// or drops it when info is zero.
func recordMCPBinary(slug string, info mcpservers.BinaryInfo) error {
	return config.Update(func(cfg *config.Config) error {
		kept := make([]config.MCPBinary, 0, len(cfg.MCPBinaries))
		for _, b := range cfg.MCPBinaries {
			if b.Slug != slug {
				kept = append(kept, b)
			}
		}
		if info != (mcpservers.BinaryInfo{}) {
			kept = append(kept, config.MCPBinary{Slug: slug, Version: info.Version, SHA256: info.SHA256})
		}
		cfg.MCPBinaries = kept
		return nil
	})
}

// mcpRegisteredElsewhere reports whether slug is still registered in a
// scope other than (global, folder).
func mcpRegisteredElsewhere(cfg *config.Config, slug string, global bool, folder string) bool {
	for i, r := range cfg.MCPServers {
		if r.Slug == slug && i != findMCPServerInstall(cfg.MCPServers, slug, global, folder) {
			return true
		}
	}
	return false
}

// removeUnusedMCPBinary deletes slug's binary and its record when no scope
// other than (global, folder) still registers slug.
func (t *DevToolsService) removeUnusedMCPBinary(stdout io.Writer, slug string, global bool, folder string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if mcpRegisteredElsewhere(cfg, slug, global, folder) {
		return nil
	}
	if err := mcpservers.RemoveBinary(stdout, t.exe, slug, mcpBinaryOf(cfg, slug)); err != nil {
		return err
	}
	return recordMCPBinary(slug, mcpservers.BinaryInfo{})
}

// InstallMCPServer registers slug with every supported agent and records
// it, with the agents that accepted it, in config.yaml on success. A server
// shipped as a release binary is installed into ~/.local/bin first; when no
// agent accepts it and no other scope uses it, the binary is removed again.
func (t *DevToolsService) InstallMCPServer(slug string, global bool, folder, param string) error {
	if err := requireKnownFolderUnlessGlobal(global, folder); err != nil {
		return err
	}
	if err := requireMCPParam(slug, global, folder, param); err != nil {
		return err
	}
	return t.runAction(func(stdout io.Writer) error {
		ctx := context.Background()
		fresh := false
		if mcpservers.IsBinary(slug) {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			info, installed, err := mcpservers.EnsureBinary(ctx, t.exe, stdout, slug, mcpBinaryOf(cfg, slug))
			if err != nil {
				return err
			}
			if installed {
				fresh = true
				if err := recordMCPBinary(slug, info); err != nil {
					return err
				}
			}
		}
		agents, err := mcpservers.Install(ctx, t.exe, stdout, slug, global, folder, param)
		if err != nil {
			if fresh {
				if cleanErr := t.removeUnusedMCPBinary(stdout, slug, global, folder); cleanErr != nil {
					return errors.Join(err, cleanErr)
				}
			}
			return err
		}
		return recordMCPServerInstall(slug, global, folder, param, agents, true)
	})
}

// RemoveMCPServer unregisters slug from every supported agent and clears
// its config.yaml record on success. The binary of a server shipped as one
// is deleted when no other scope registers it.
func (t *DevToolsService) RemoveMCPServer(slug string, global bool, folder string) error {
	if err := requireKnownFolderUnlessGlobal(global, folder); err != nil {
		return err
	}
	return t.runAction(func(stdout io.Writer) error {
		if err := mcpservers.Remove(context.Background(), t.exe, stdout, slug, global, folder); err != nil {
			return err
		}
		if mcpservers.IsBinary(slug) {
			if err := t.removeUnusedMCPBinary(stdout, slug, global, folder); err != nil {
				return err
			}
		}
		return recordMCPServerInstall(slug, global, folder, "", nil, false)
	})
}

// UpdateMCPServer refreshes slug's registration and keeps config.yaml's
// stored Param and agents in sync (in case the user changed the parameter
// before updating). A server shipped as a release binary first gets the
// latest release when it is newer than the installed one.
func (t *DevToolsService) UpdateMCPServer(slug string, global bool, folder, param string) error {
	if err := requireKnownFolderUnlessGlobal(global, folder); err != nil {
		return err
	}
	if err := requireMCPParam(slug, global, folder, param); err != nil {
		return err
	}
	return t.runAction(func(stdout io.Writer) error {
		ctx := context.Background()
		if mcpservers.IsBinary(slug) {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			owned := mcpBinaryOf(cfg, slug)
			info, err := mcpservers.UpdateBinary(ctx, t.exe, stdout, slug, owned)
			if err != nil {
				return err
			}
			if info != owned {
				if err := recordMCPBinary(slug, info); err != nil {
					return err
				}
			}
		}
		agents, err := mcpservers.Update(ctx, t.exe, stdout, slug, global, folder, param)
		if err != nil {
			if errors.Is(err, mcpservers.ErrNotReinstalled) {
				// Removed everywhere, re-registered nowhere: the list must
				// not keep showing it as installed.
				if recErr := recordMCPServerInstall(slug, global, folder, "", nil, false); recErr != nil {
					return errors.Join(err, recErr)
				}
			}
			return err
		}
		return recordMCPServerInstall(slug, global, folder, param, agents, true)
	})
}
