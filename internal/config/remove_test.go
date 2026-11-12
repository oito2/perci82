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

package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oito2/perci82/internal/config"
)

// Removing one application's keys leaves the other application's keys,
// unknown keys and comments exactly as they were.
func TestRemoveOwnedKeys_OnlyTheAppsKeys(t *testing.T) {
	path := installConfig(t, []byte("flatpak_scope: user\n# kept comment\ngui_theme: nordDark\nfuture_key:\n    nested: 1\nsystem_language: en\n"))
	empty, err := config.RemoveOwnedKeys(config.AppSystem)
	if err != nil || empty {
		t.Fatalf("RemoveOwnedKeys(system) got empty=%v err=%v, want false, nil", empty, err)
	}
	got := string(readFile(t, path))
	for _, gone := range []string{"flatpak_scope", "system_language"} {
		if strings.Contains(got, gone) {
			t.Errorf("%s still in config:\n%s", gone, got)
		}
	}
	for _, kept := range []string{"# kept comment", "gui_theme: nordDark", "future_key:", "nested: 1"} {
		if !strings.Contains(got, kept) {
			t.Errorf("%q missing from config:\n%s", kept, got)
		}
	}
}

func TestRemoveOwnedKeys_Empty(t *testing.T) {
	cases := []struct {
		name, data string
		app        config.App
		want       bool
	}{
		{"only the app's keys", "language: en\ntray_enabled: true\n", config.AppDev, true},
		{"legacy keys only are left", "sidebar_logo: pink\nlanguage: en\n", config.AppDev, true},
		{"the other app's keys are left", "language: en\nflatpak_scope: user\n", config.AppDev, false},
		{"an unknown key is left", "language: en\nfuture_key: 1\n", config.AppDev, false},
		{"nothing to remove", "flatpak_scope: user\n", config.AppDev, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			installConfig(t, []byte(tc.data))
			got, err := config.RemoveOwnedKeys(tc.app)
			if err != nil || got != tc.want {
				t.Errorf("got empty=%v err=%v, want %v, nil", got, err, tc.want)
			}
		})
	}
}

func TestRemoveOwnedKeys_NoFileAndUnknownApp(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if empty, err := config.RemoveOwnedKeys(config.AppSystem); err != nil || !empty {
		t.Errorf("missing config got empty=%v err=%v, want true, nil", empty, err)
	}
	if _, err := config.RemoveOwnedKeys("other"); err == nil {
		t.Error("unknown application got nil error")
	}
}

func TestRemoveDir(t *testing.T) {
	path := installConfig(t, []byte("language: en\n"))
	dir := filepath.Dir(path)
	if _, err := config.RemoveOwnedKeys(config.AppDev); err != nil { // creates config.lock
		t.Fatal(err)
	}
	other := filepath.Join(dir, "appstack")
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := config.RemoveDir(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("config.yaml left behind: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.lock")); !os.IsNotExist(err) {
		t.Errorf("config.lock left behind: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("a folder with other files was removed: %v", err)
	}

	if err := os.Remove(other); err != nil {
		t.Fatal(err)
	}
	if err := config.RemoveDir(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("empty ~/.perci left behind: %v", err)
	}
	if err := config.RemoveDir(); err != nil {
		t.Errorf("second RemoveDir: %v", err)
	}
}
