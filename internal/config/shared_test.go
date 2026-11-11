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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/oito2/perci82/internal/config"
)

// legacyConfig is a config.yaml written by Perci 1.9.0 with every field
// set.
const legacyConfig = "testdata/perci-1.9/config.yaml"

// installConfig writes data as config.yaml under a fresh HOME and returns
// the file's path.
func installConfig(t *testing.T, data []byte) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".perci")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// useApp makes Update write only app's keys until the test ends.
func useApp(t *testing.T, app config.App) {
	t.Helper()
	config.SetApp(app)
	t.Cleanup(func() { config.SetApp("") })
}

// topLevel splits a block-style YAML mapping into its top-level entries,
// keyed by key, each with its exact bytes: the top-level comment lines
// right above the key, the key line and its nested lines.
func topLevel(t *testing.T, data []byte) map[string]string {
	t.Helper()
	entries := map[string]string{}
	key, comments := "", ""
	for line := range strings.SplitAfterSeq(string(data), "\n") {
		switch {
		case line == "":
			continue
		case line[0] == '#':
			comments += line
			continue
		case line[0] != ' ' && line[0] != '\n':
			k, _, ok := strings.Cut(line, ":")
			if !ok {
				t.Fatalf("unexpected top-level line %q", line)
			}
			key = k
		}
		entries[key] += comments + line
		comments = ""
	}
	if comments != "" {
		entries["#trailing"] = comments
	}
	return entries
}

// assertOnlyChanged fails unless after differs from before exactly in the
// top-level entries named in changed; every other entry must keep its
// bytes.
func assertOnlyChanged(t *testing.T, before, after []byte, changed ...string) {
	t.Helper()
	b, a := topLevel(t, before), topLevel(t, after)
	for key, val := range b {
		if slices.Contains(changed, key) {
			continue
		}
		if a[key] != val {
			t.Errorf("entry %q changed:\nbefore:\n%s\nafter:\n%s", key, val, a[key])
		}
	}
	for key := range a {
		if _, ok := b[key]; !ok && !slices.Contains(changed, key) {
			t.Errorf("entry %q appeared", key)
		}
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// A config.yaml written by Perci 1.9 loads with every field, and each
// application writing one of its keys leaves every other entry with the
// same bytes — the other application's keys (with a field a newer version
// added inside one), the legacy mascot keys, a key this version doesn't
// know and the comment above it.
func TestUpdate_Perci19ConfigKeepsOtherEntries(t *testing.T) {
	legacy := readFile(t, legacyConfig)
	extra := "# kept by every update\nfuture_key:\n    nested: [1, 2]\n"
	// A field a newer version added inside a key: kept unless the key
	// itself is written.
	nested := "      ssh_key: /home/pessoa/.ssh/id_ed25519_github-exemplo\n"
	if !strings.Contains(string(legacy), nested) {
		t.Fatal("fixture changed: ssh_key line not found")
	}
	legacy = []byte(strings.Replace(string(legacy), nested, nested+"      future_field: kept\n", 1))
	original := append(append([]byte{}, legacy...), extra...)

	cases := []struct {
		name string
		app  config.App
		edit func(*config.Config)
		key  string
	}{
		{"System theme", config.AppSystem, func(c *config.Config) { c.SystemGUITheme = "retro" }, "system_gui_theme"},
		{"System flatpak scope", config.AppSystem, func(c *config.Config) { c.FlatpakScope = "system" }, "flatpak_scope"},
		{"System post-install", config.AppSystem, func(c *config.Config) {
			c.PostinstallApplied["debian"] = []string{"contrib"}
		}, "postinstall_applied"},
		{"Dev theme", config.AppDev, func(c *config.Config) { c.GUITheme = "dark" }, "gui_theme"},
		{"Dev tray", config.AppDev, func(c *config.Config) { c.TrayEnabled = false }, "tray_enabled"},
		{"Dev docker", config.AppDev, func(c *config.Config) { c.Docker.Apps = c.Docker.Apps[:1] }, "docker"},
		{"Dev git accounts", config.AppDev, func(c *config.Config) { c.GitAccounts[0].HasToken = false }, "git_accounts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := installConfig(t, original)
			useApp(t, tc.app)

			loaded, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			if len(loaded.GitAccounts) != 2 || len(loaded.Docker.Apps) != 3 || loaded.SidebarLogo != "pink" ||
				loaded.PostinstallMachineID == "" || len(loaded.AIContexts) != 2 || !loaded.TrayEnabled {
				t.Fatalf("Perci 1.9 config loaded incompletely: %+v", loaded)
			}

			if err := config.Update(func(c *config.Config) error { tc.edit(c); return nil }); err != nil {
				t.Fatal(err)
			}
			after := readFile(t, path)
			assertOnlyChanged(t, original, after, tc.key)
			if !strings.Contains(string(after), extra) {
				t.Errorf("unknown key or its comment lost:\n%s", after)
			}
		})
	}
}

// Removing every value of an owned key removes the key from the file and
// nothing else.
func TestUpdate_ClearedKeyIsRemoved(t *testing.T) {
	original := readFile(t, legacyConfig)
	path := installConfig(t, original)
	useApp(t, config.AppDev)

	if err := config.Update(func(c *config.Config) error { c.AIContexts = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	after := readFile(t, path)
	if strings.Contains(string(after), "ai_contexts:") {
		t.Errorf("ai_contexts still in the file:\n%s", after)
	}
	assertOnlyChanged(t, original, after, "ai_contexts")
}

// A change to a key of the other application, or to a legacy key, is
// refused and nothing is written.
func TestUpdate_ForeignKeyRefused(t *testing.T) {
	cases := []struct {
		app  config.App
		edit func(*config.Config)
		key  string
	}{
		{config.AppSystem, func(c *config.Config) { c.GUITheme = "dark" }, "gui_theme"},
		{config.AppSystem, func(c *config.Config) { c.WorkspacePath = "/elsewhere" }, "workspace_path"},
		{config.AppDev, func(c *config.Config) { c.FlatpakScope = "system" }, "flatpak_scope"},
		{config.AppDev, func(c *config.Config) { c.SystemGUITheme = "dark" }, "system_gui_theme"},
		{config.AppDev, func(c *config.Config) { c.SidebarLogo = "blue" }, "sidebar_logo"},
		{config.AppSystem, func(c *config.Config) { c.AppIcon = "blue" }, "app_icon"},
	}
	for _, tc := range cases {
		t.Run(string(tc.app)+" "+tc.key, func(t *testing.T) {
			original := readFile(t, legacyConfig)
			path := installConfig(t, original)
			useApp(t, tc.app)

			err := config.Update(func(c *config.Config) error {
				tc.edit(c)
				// An owned key changed together with the foreign one is not
				// written either.
				if tc.app == config.AppSystem {
					c.FlatpakScope = "system"
				} else {
					c.TrayEnabled = false
				}
				return nil
			})
			var foreign *config.ForeignKeyError
			if !errors.As(err, &foreign) || foreign.Key != tc.key || foreign.App != tc.app {
				t.Fatalf("err = %v, want *ForeignKeyError for %q", err, tc.key)
			}
			if got := readFile(t, path); string(got) != string(original) {
				t.Errorf("file changed after a refused update:\n%s", got)
			}
		})
	}
}

// An update that changes nothing leaves the file untouched, not even
// rewritten.
func TestUpdate_NoChangeDoesNotWrite(t *testing.T) {
	path := installConfig(t, readFile(t, legacyConfig))
	useApp(t, config.AppSystem)
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := config.Update(func(c *config.Config) error { c.FlatpakScope = "user"; return nil }); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Error("config.yaml was rewritten by an update that changed nothing")
	}
}

// Without a config.yaml, an update creates one holding only what changed.
func TestUpdate_CreatesFileWithChangedKeysOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	useApp(t, config.AppSystem)

	if err := config.Update(func(c *config.Config) error { c.FlatpakScope = "user"; return nil }); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(home, ".perci", "config.yaml"))
	if string(got) != "flatpak_scope: user\n" {
		t.Errorf("config.yaml = %q, want only flatpak_scope", got)
	}
}

// Every top-level key Config writes belongs to exactly one application or
// is a legacy key nobody writes.
func TestOwnedKeys_CoverEveryField(t *testing.T) {
	system, dev := config.OwnedKeys(config.AppSystem), config.OwnedKeys(config.AppDev)
	legacy := config.LegacyKeys()
	for _, key := range system {
		if slices.Contains(dev, key) {
			t.Errorf("key %q is owned by both applications", key)
		}
	}

	full := config.Config{
		WorkspacePath: "w", FlatpakScope: "user", GUITheme: "dark", SystemGUITheme: "dark",
		SidebarLogo: "pink", AppIcon: "pink", TrayEnabled: true, PostinstallMachineID: "m",
		Docker:             config.DockerConfig{NginxCreated: true},
		GitAccounts:        []config.GitAccount{{ID: "a"}},
		AgentSkills:        []config.AgentSkillInstall{{Slug: "s"}},
		MCPServers:         []config.MCPServerInstall{{Slug: "m"}},
		MoodlePlugins:      []config.MoodlePluginMount{{App: "a"}},
		AIContexts:         []config.AIContext{{Path: "/p"}},
		PostinstallApplied: map[string][]string{"p": {"a"}},
	}
	var n yaml.Node
	if err := n.Encode(full); err != nil {
		t.Fatal(err)
	}
	if len(n.Content) == 0 {
		t.Fatal("nothing encoded")
	}
	for i := 0; i < len(n.Content); i += 2 {
		key := n.Content[i].Value
		owners := 0
		for _, set := range [][]string{system, dev, legacy} {
			if slices.Contains(set, key) {
				owners++
			}
		}
		if owners != 1 {
			t.Errorf("key %q has %d owners, want 1", key, owners)
		}
	}
}

// Two processes — one per application — updating their own keys at the
// same time lose no update: config.lock serializes them and each re-reads
// the file inside the lock.
func TestUpdate_ConcurrentProcessesLoseNothing(t *testing.T) {
	if os.Getenv("PERCI82_CONFIG_HELPER") != "" {
		return
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	const rounds = 40

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, app := range []config.App{config.AppSystem, config.AppDev} {
		wg.Go(func() {
			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestHelperUpdater$")
			cmd.Env = append(os.Environ(), "PERCI82_CONFIG_HELPER="+string(app), "PERCI82_CONFIG_ROUNDS="+strconv.Itoa(rounds))
			if out, err := cmd.CombinedOutput(); err != nil {
				errs <- fmt.Errorf("%s helper: %w\n%s", app, err, out)
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(cfg.PostinstallApplied["helper"]); got != rounds {
		t.Errorf("System updates kept = %d, want %d", got, rounds)
	}
	if got := len(cfg.AIContexts); got != rounds {
		t.Errorf("Dev updates kept = %d, want %d", got, rounds)
	}
}

// TestHelperUpdater runs only as TestUpdate_ConcurrentProcessesLoseNothing's
// child process: it appends PERCI82_CONFIG_ROUNDS entries to a key of the
// application named in PERCI82_CONFIG_HELPER, one Update each.
func TestHelperUpdater(t *testing.T) {
	app := config.App(os.Getenv("PERCI82_CONFIG_HELPER"))
	if app == "" {
		t.Skip("helper process only")
	}
	rounds, err := strconv.Atoi(os.Getenv("PERCI82_CONFIG_ROUNDS"))
	if err != nil {
		t.Fatal(err)
	}
	config.SetApp(app)
	for i := range rounds {
		err := config.Update(func(c *config.Config) error {
			if app == config.AppSystem {
				if c.PostinstallApplied == nil {
					c.PostinstallApplied = map[string][]string{}
				}
				c.PostinstallApplied["helper"] = append(c.PostinstallApplied["helper"], strconv.Itoa(i))
			} else {
				c.AIContexts = append(c.AIContexts, config.AIContext{Name: strconv.Itoa(i), Path: "/p/" + strconv.Itoa(i)})
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// action.lock is exclusive across holders: a second LockActions is refused
// with the holder's name until the first is released.
func TestLockActions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	release, err := config.LockActions(config.AppSystem)
	if err != nil {
		t.Fatal(err)
	}
	_, err = config.LockActions(config.AppDev)
	var busy *config.ActionBusyError
	if !errors.As(err, &busy) || busy.Holder != config.AppSystem {
		t.Fatalf("second lock: err = %v, want *ActionBusyError held by system", err)
	}
	if !strings.Contains(err.Error(), "Perci82 System") {
		t.Errorf("message %q doesn't name the holder", err)
	}

	release()
	release2, err := config.LockActions(config.AppDev)
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	release2()
}
