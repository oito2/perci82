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

package ide

import (
	"context"
	"sync"

	"github.com/oito2/perci82/internal/dev/localbin"
	"github.com/oito2/perci82/internal/dev/prereqs"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
)

// IDE describes an editor managed by perci.gnl.
type IDE struct {
	Name string
	Cmd  string
	// Description is the message key of the editor's description;
	// DescriptionText returns it in the current language.
	Description string
	// Requires lists the Names of the prereqs.Catalogue items its installer
	// needs.
	Requires []string
	// RemoveWarning is the message key of the text shown in the uninstall
	// confirmation; RemoveWarningText returns it in the current language.
	RemoveWarning string
}

// DescriptionText returns the editor's description in the current
// language.
func (e IDE) DescriptionText() string {
	return i18n.Tr(e.Description, nil)
}

// RemoveWarningText returns the editor's uninstall warning in the current
// language.
func (e IDE) RemoveWarningText() string {
	return i18n.Tr(e.RemoveWarning, nil)
}

// Catalogue lists all IDEs managed by perci.gnl.
var Catalogue = []IDE{
	{Name: "Zed Editor", Cmd: "zed",
		Description:   "ide.zed",
		Requires:      []string{"Curl"},
		RemoveWarning: "ide.zed.removeWarning"},
	{Name: "VS Code", Cmd: "code",
		Description:   "ide.code",
		RemoveWarning: "ide.code.removeWarning"},
	{Name: "VSCodium", Cmd: "codium",
		Description:   "ide.codium",
		RemoveWarning: "ide.codium.removeWarning"},
}

// MissingRequirements returns the labels of e's requirements that aren't
// installed.
func MissingRequirements(ctx context.Context, exe *executor.Executor, e IDE) []string {
	var missing []string
	for _, req := range e.Requires {
		if !prereqs.Installed(ctx, exe, req) {
			missing = append(missing, i18n.Tr("dev.requirement", i18n.Args{"name": req}))
		}
	}
	return missing
}

// InstalledMap returns which IDEs are currently installed (by Name),
// detected through localbin.Which.
func InstalledMap(ctx context.Context, exe *executor.Executor) map[string]bool {
	result := make(map[string]bool, len(Catalogue))
	var mu sync.Mutex
	// Concurrent: a command not on PATH costs a shell sourcing nvm.
	_ = executor.RunConcurrent(ctx, Catalogue, len(Catalogue), func(e IDE) {
		if localbin.Which(ctx, exe, e.Cmd) {
			mu.Lock()
			result[e.Name] = true
			mu.Unlock()
		}
	})
	return result
}
