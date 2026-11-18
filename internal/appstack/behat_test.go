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

package appstack

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
)

func behatMoodle(folder, version string) config.AppContainer {
	return config.AppContainer{
		Folder: folder, URL: strings.ToLower(folder) + ".localhost", Type: config.AppTypeMoodle,
		PHPVersion: "8.3", MoodleVersion: version, Behat: true,
	}
}

func TestBehatHostFor(t *testing.T) {
	long57 := strings.Repeat("a", 57)
	cases := []struct {
		folder string
		want   string
		ok     bool
	}{
		{"mdle-uvv", "mdle-uvv-behat.test", true},
		{"MDLE", "mdle-behat.test", true},
		{long57, long57 + "-behat.test", true},
		{long57 + "a", "", false},
		{"mdle_uvv", "", false},
		{"-x", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.folder, func(t *testing.T) {
			got, ok := BehatHostFor(tc.folder)
			if got != tc.want || ok != tc.ok {
				t.Errorf("BehatHostFor(%q) = %q, %v; want %q, %v", tc.folder, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestBehatHost(t *testing.T) {
	on := behatMoodle("mdle", "4.4-4.5")
	off := on
	off.Behat = false
	php := on
	php.Type = config.AppTypePHP
	if host, ok := BehatHost(on); !ok || host != "mdle-behat.test" {
		t.Errorf("BehatHost(on) = %q, %v; want mdle-behat.test, true", host, ok)
	}
	for name, app := range map[string]config.AppContainer{"off": off, "php": php} {
		if host, ok := BehatHost(app); ok {
			t.Errorf("BehatHost(%s) = %q, true; want none", name, host)
		}
	}
}

func TestValidateApp_Behat(t *testing.T) {
	cases := []struct {
		name string
		app  config.AppContainer
		ok   bool
	}{
		{"moodle", behatMoodle("mdle", "5.1+"), true},
		{"underscore folder", behatMoodle("mdle_x", "5.1+"), false},
		{"long folder", behatMoodle(strings.Repeat("m", 58), "5.1+"), false},
		{"php app", config.AppContainer{Folder: "loja", URL: "loja.localhost", Type: config.AppTypePHP, PHPVersion: "8.3", Behat: true}, false},
		{"selenium folder", config.AppContainer{Folder: "selenium", URL: "selenium.localhost", Type: config.AppTypePHP, PHPVersion: "8.3"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateApp(tc.app); (err == nil) != tc.ok {
				t.Errorf("ValidateApp = %v, want ok=%v", err, tc.ok)
			}
		})
	}
}

func TestCheckURLUnique_BehatHost(t *testing.T) {
	upper := behatMoodle("MDLE", "5.1+")
	upper.URL = "outro.localhost"
	if err := checkURLUnique(behatMoodle("mdle", "5.1+"), []config.AppContainer{upper}); err == nil {
		t.Error("two apps sharing a Behat host name were accepted")
	}
	upper.Behat = false
	if err := checkURLUnique(behatMoodle("mdle", "5.1+"), []config.AppContainer{upper}); err != nil {
		t.Errorf("checkURLUnique = %v, want nil when the other app has no Behat site", err)
	}
}

func TestBuildNginxConf_Behat(t *testing.T) {
	cases := []struct {
		version string
		root    string
		marker  string
	}{
		{"5.1+", appHTMLMount + "/mdle/public", "try_files $uri $uri/ /r.php$is_args$args;"},
		{"5.0", appHTMLMount + "/mdle", "try_files $uri $uri/ /r.php;"},
		{"4.4-4.5", appHTMLMount + "/mdle", "location ~ [^/]\\.php(/|$) {"},
	}
	for _, tc := range cases {
		t.Run(tc.version, func(t *testing.T) {
			conf := BuildNginxConf([]config.AppContainer{behatMoodle("mdle", tc.version)})
			_, block, ok := strings.Cut(conf, "server_name mdle-behat.test;")
			if !ok {
				t.Fatalf("no Behat server block in:\n%s", conf)
			}
			head := conf[:strings.Index(conf, "server_name mdle-behat.test;")]
			head = head[strings.LastIndex(head, "server {"):]
			if !strings.Contains(head, "listen 80;") || strings.Contains(head, "443") {
				t.Errorf("Behat block must listen on HTTP only:\n%s", head)
			}
			for _, want := range []string{"root " + tc.root + ";", tc.marker, "set $backend mdle:9000;", "/vendor/"} {
				if !strings.Contains(block, want) {
					t.Errorf("Behat block missing %q:\n%s", want, block)
				}
			}
			for _, unwanted := range []string{"/behat/", "/fixtures/", "ssl_certificate", "{{FOLDER}}"} {
				if strings.Contains(block, unwanted) {
					t.Errorf("Behat block contains %q:\n%s", unwanted, block)
				}
			}
			if strings.Count(conf, "server_name mdle.localhost;") != 1 {
				t.Errorf("the app's own server block is missing:\n%s", conf)
			}
		})
	}

	off := behatMoodle("mdle", "5.1+")
	off.Behat = false
	if conf := BuildNginxConf([]config.AppContainer{off}); strings.Contains(conf, "-behat.test") {
		t.Errorf("Behat block written with the option off:\n%s", conf)
	}
}

func TestBehatAliases(t *testing.T) {
	off := behatMoodle("zeta", "5.1+")
	off.Behat = false
	apps := []config.AppContainer{behatMoodle("mdle", "5.1+"), off, behatMoodle("alfa", "4.4-4.5"), reloadApps[0]}
	got := behatAliases(apps)
	want := []string{"alfa-behat.test", "mdle-behat.test"}
	if !slices.Equal(got, want) {
		t.Errorf("behatAliases = %v, want %v", got, want)
	}
}

func TestParseBehatAliases(t *testing.T) {
	cases := []struct {
		out  string
		want []string
		ok   bool
	}{
		{"null\n", nil, true},
		{`["nginx","a1b2c3","b-behat.test","a-behat.test"]`, []string{"a-behat.test", "b-behat.test"}, true},
		{"running", nil, false},
	}
	for _, tc := range cases {
		got, err := parseBehatAliases(tc.out)
		if (err == nil) != tc.ok || !slices.Equal(got, tc.want) {
			t.Errorf("parseBehatAliases(%q) = %v, %v; want %v, ok=%v", tc.out, got, err, tc.want, tc.ok)
		}
	}
}

// fakeAliasDocker puts a `docker` script first on PATH that reports the
// Nginx container as running with the JSON alias list aliases, succeeds on
// every other call and logs every call to the returned file.
func fakeAliasDocker(t *testing.T, aliases string) string {
	t.Helper()
	bin := t.TempDir()
	log := filepath.Join(bin, "calls.log")
	script := `#!/bin/sh
printf '%s\n' "$*" >> '` + log + `'
case "$*" in
*Aliases*) printf '%s\n' '` + aliases + `' ;;
inspect*) echo running ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func TestSyncNginxAliases(t *testing.T) {
	apps := []config.AppContainer{behatMoodle("mdle", "5.1+")}
	cases := []struct {
		name      string
		apps      []config.AppContainer
		aliases   string
		reconnect string
	}{
		{"unchanged", apps, `["nginx","mdle-behat.test"]`, ""},
		{"added", apps, `["nginx"]`, "network connect --alias mdle-behat.test -- " + NetworkName + " " + NginxContainerName},
		{"removed", nil, `["nginx","mdle-behat.test"]`, "network connect -- " + NetworkName + " " + NginxContainerName},
		{"none wanted, unreadable", nil, `not json`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			log := fakeAliasDocker(t, tc.aliases)
			var buf bytes.Buffer
			syncNginxAliases(t.Context(), executor.New(&buf, &buf), &buf, tc.apps)
			data, _ := os.ReadFile(log)
			calls := string(data)
			if tc.reconnect == "" {
				if strings.Contains(calls, "network disconnect") || strings.Contains(calls, "network connect") {
					t.Errorf("unexpected reconnection:\n%s", calls)
				}
				return
			}
			disconnect := strings.Index(calls, "network disconnect -- "+NetworkName+" "+NginxContainerName)
			connect := strings.Index(calls, tc.reconnect)
			if disconnect < 0 || connect < disconnect {
				t.Errorf("want disconnect then %q, got:\n%s", tc.reconnect, calls)
			}
		})
	}
}

func TestSeleniumRunArgs(t *testing.T) {
	got := strings.Join(seleniumRunArgs(), " ")
	for _, want := range []string{
		"--name selenium", "--network " + NetworkName, "--restart no", "--shm-size 2g",
		"-e SE_NODE_MAX_SESSIONS=2", "-e SE_NODE_OVERRIDE_MAX_SESSIONS=true",
		"-p 127.0.0.1:7900:7900", "-- " + SeleniumImage,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("seleniumRunArgs() missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, ":4444") {
		t.Errorf("the WebDriver port must not be published: %q", got)
	}
	if !strings.HasSuffix(got, SeleniumImage) || strings.HasSuffix(SeleniumImage, ":latest") {
		t.Errorf("the image must come last with a fixed tag: %q", got)
	}
}

func TestCreateAndDeleteSelenium(t *testing.T) {
	stackHome(t)
	log := fakeStack(t, stackOpts{status: "running"})
	exe, buf := testExe()
	if err := CreateSelenium(t.Context(), exe, buf); err != nil {
		t.Fatalf("CreateSelenium: %v\n%s", err, buf)
	}
	if !strings.Contains(calls(t, log), "run -d --name selenium") {
		t.Errorf("selenium container not created:\n%s", calls(t, log))
	}
	cfg, err := config.Load()
	if err != nil || !cfg.Docker.SeleniumCreated {
		t.Fatalf("SeleniumCreated = false after CreateSelenium (err %v)", err)
	}
	if err := DeleteSelenium(t.Context(), exe, buf); err != nil {
		t.Fatalf("DeleteSelenium: %v\n%s", err, buf)
	}
	if !strings.Contains(calls(t, log), "rm -f -- selenium") {
		t.Errorf("selenium container not removed:\n%s", calls(t, log))
	}
	if cfg, _ := config.Load(); cfg.Docker.SeleniumCreated {
		t.Error("SeleniumCreated still true after DeleteSelenium")
	}
}

func TestTransferSelection_Selenium(t *testing.T) {
	e := ExportedConfig{Docker: config.DockerConfig{SeleniumCreated: true}}
	if sel := SelectAll(e); !sel.Selenium {
		t.Error("SelectAll left Selenium out")
	}
	part, err := e.Filter(TransferSelection{Selenium: true})
	if err != nil || !part.Docker.SeleniumCreated || part.empty() {
		t.Errorf("Filter(Selenium) = %+v, %v; want Selenium only", part, err)
	}
	if _, err := (ExportedConfig{}).Filter(TransferSelection{Selenium: true}); err == nil {
		t.Error("Filter accepted Selenium missing from the file")
	}
}

func TestAllImagesTarget_Selenium(t *testing.T) {
	stackHome(t)
	if err := config.Update(func(c *config.Config) error {
		c.Docker.SeleniumCreated = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	target, err := AllImagesTarget()
	if err != nil || !target.Selenium {
		t.Errorf("AllImagesTarget = %+v, %v; want Selenium included", target, err)
	}
	if (ImageUpdatePlan{Selenium: true}).Empty() {
		t.Error("a plan recreating Selenium reports itself empty")
	}
}

func TestBehatSetupFor(t *testing.T) {
	cfg := &config.Config{MoodlePlugins: []config.MoodlePluginMount{
		{App: "mdle", Component: "gradeimport_hercules"},
		{App: "mdle", Component: "gradeimport_hercules"},
		{App: "outro", Component: "local_x"},
		{App: "mdle", Component: "bad; rm"},
	}}
	app := behatMoodle("mdle", "4.4-4.5")

	s, ok := BehatSetupFor(cfg, app)
	if !ok {
		t.Fatal("BehatSetupFor = false for a Moodle app with Behat on")
	}
	if s.WWWRoot != "http://mdle-behat.test" || s.Selenium {
		t.Errorf("WWWRoot/Selenium = %q/%v", s.WWWRoot, s.Selenium)
	}
	for _, want := range []string{"behat_wwwroot  = 'http://mdle-behat.test'", "behat_prefix   = 'bht_'", "behat_dataroot = '/var/www/data/behat'"} {
		if !strings.Contains(s.ConfigPHP, want) {
			t.Errorf("ConfigPHP missing %q:\n%s", want, s.ConfigPHP)
		}
	}
	if strings.Contains(s.ConfigPHP, "behat_profiles") {
		t.Errorf("behat_profiles without Selenium:\n%s", s.ConfigPHP)
	}
	if s.InitCommand != "mdle admin/tool/behat/cli/init.php" {
		t.Errorf("InitCommand = %q", s.InitCommand)
	}
	wantRun := []string{"mdle vendor/bin/behat --config " + BehatConfigFile + " --tags='@gradeimport_hercules&&~@javascript'"}
	if !slices.Equal(s.RunCommands, wantRun) {
		t.Errorf("RunCommands = %q, want %q", s.RunCommands, wantRun)
	}

	cfg.Docker.SeleniumCreated = true
	cfg.MoodlePlugins = nil
	s, _ = BehatSetupFor(cfg, behatMoodle("mdle", "5.1+"))
	if !strings.Contains(s.ConfigPHP, "'wd_host' => 'http://selenium:4444/wd/hub'") || !s.Selenium {
		t.Errorf("ConfigPHP without the Selenium profile:\n%s", s.ConfigPHP)
	}
	if s.InitCommand != "mdle public/admin/tool/behat/cli/init.php" {
		t.Errorf("InitCommand (5.1+) = %q", s.InitCommand)
	}
	if len(s.RunCommands) != 1 || !strings.HasSuffix(s.RunCommands[0], "--tags='@local_example'") {
		t.Errorf("RunCommands (Selenium, no plugin) = %q", s.RunCommands)
	}

	off := app
	off.Behat = false
	if _, ok := BehatSetupFor(cfg, off); ok {
		t.Error("BehatSetupFor = true with the option off")
	}
}
