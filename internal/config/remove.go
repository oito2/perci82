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

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/oito2/perci82/internal/fsutil"
)

// RemoveOwnedKeys removes from config.yaml every top-level key app owns,
// keeping every other key (and its comments) as it is, under the same
// lock as Update. empty reports whether the file is left with no key
// other than the legacy ones (a missing file counts as empty).
func RemoveOwnedKeys(app App) (empty bool, err error) {
	owned := ownedKeys[app]
	if len(owned) == 0 {
		return false, fmt.Errorf("unknown application %q", app)
	}

	mu.Lock()
	defer mu.Unlock()

	unlock, err := lockConfig()
	if err != nil {
		return false, err
	}
	defer unlock()

	path, err := configPath()
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read config: %w", err)
	}
	doc, err := parseDocument(data)
	if err != nil {
		return false, err
	}
	root := doc.Content[0]

	removed := false
	kept := root.Content[:0:0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if slices.Contains(owned, root.Content[i].Value) {
			removed = true
			continue
		}
		kept = append(kept, root.Content[i], root.Content[i+1])
	}
	root.Content = kept

	empty = true
	for i := 0; i < len(kept); i += 2 {
		if !slices.Contains(legacyKeys, kept[i].Value) {
			empty = false
			break
		}
	}
	if !removed {
		return empty, nil
	}
	out, err := encodeNode(doc)
	if err != nil {
		return false, err
	}
	if err := fsutil.WriteThroughLink(path, out, 0o600); err != nil {
		return false, fmt.Errorf("write config: %w", err)
	}
	return empty, nil
}

// RemoveDir removes config.yaml and the lock files from ~/.perci, then
// the folder itself when nothing else is left in it. A folder still
// holding other files is kept, without error.
func RemoveDir() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	for _, name := range []string{configFile, configLockFile, actionLockFile} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", name, err)
		}
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read config dir: %w", err)
	}
	if len(entries) > 0 {
		return nil
	}
	if err := os.Remove(dir); err != nil {
		return fmt.Errorf("remove config dir: %w", err)
	}
	return nil
}
