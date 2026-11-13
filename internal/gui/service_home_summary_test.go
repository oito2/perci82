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
	"os"
	"path/filepath"
	"testing"

	"github.com/oito2/perci82/internal/config"
)

// The repositories summary counts the accounts and the repositories under
// their folders; a missing folder counts none.
func TestGetRepositorySummary(t *testing.T) {
	isolateActions(t)
	home := os.Getenv("HOME")
	work := filepath.Join(home, "work")
	for _, repo := range []string{"a", "b", filepath.Join("group", "c")} {
		if err := os.MkdirAll(filepath.Join(work, repo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.Update(func(c *config.Config) error {
		c.GitAccounts = []config.GitAccount{
			{ID: "work", Folder: work},
			{ID: "gone", Folder: filepath.Join(home, "missing")},
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	got, err := (&DevHomeService{}).GetRepositorySummary(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got != (RepositorySummary{Accounts: 2, Repos: 3}) {
		t.Errorf("got %+v, want 2 accounts and 3 repositories", got)
	}
}
