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
	"regexp"
	"slices"
	"strings"

	"github.com/oito2/perci82/internal/config"
)

// BehatPrefix is the database table prefix of a Behat test site, distinct
// from Moodle's default "mdl_" and from the usual PHPUnit "phpu_".
const BehatPrefix = "bht_"

// BehatDataroot is the Behat test site's data folder inside a Moodle
// app's container: a subfolder of the app's moodledata mount, the only
// folder of the container that persists and is writable.
const BehatDataroot = "/var/www/data/behat"

// BehatConfigFile is the behat.yml Moodle's Behat initialization writes
// under BehatDataroot, which the behat runner takes with --config.
const BehatConfigFile = BehatDataroot + "/behatrun/behat/behat.yml"

// validComponent matches a Moodle frankenstyle component name (e.g.
// "local_example"), the only component names written into the commands of
// a BehatSetup.
var validComponent = regexp.MustCompile(`^[a-z][a-z0-9]*_[a-z0-9_]+$`)

// BehatSetup is what a Moodle app with its Behat option on needs in its
// config.php and on the command line to run Behat.
type BehatSetup struct {
	// Host is the test site's host name and WWWRoot its URL
	// ($CFG->behat_wwwroot).
	Host    string `json:"host"`
	WWWRoot string `json:"wwwroot"`
	// ConfigPHP holds the lines to add to the app's config.php, before the
	// require_once of lib/setup.php.
	ConfigPHP string `json:"configPhp"`
	// InitCommand initializes (or updates) the test site, through the app's
	// command.
	InitCommand string `json:"initCommand"`
	// RunCommands run the Behat scenarios of each plugin component mounted
	// in the app, or of a placeholder component when none is mounted.
	RunCommands []string `json:"runCommands"`
	// Selenium reports whether the shared Selenium container is set up:
	// ConfigPHP then points Behat at it, and the run commands no longer
	// exclude the @javascript scenarios.
	Selenium bool `json:"selenium"`
}

// BehatSetupFor returns app's BehatSetup, and false when app has no Behat
// test site (BehatHost). cfg provides the Selenium state and the plugin
// mounts of the app.
func BehatSetupFor(cfg *config.Config, app config.AppContainer) (BehatSetup, bool) {
	host, ok := BehatHost(app)
	if !ok {
		return BehatSetup{}, false
	}
	s := BehatSetup{
		Host:     host,
		WWWRoot:  "http://" + host,
		Selenium: cfg.Docker.SeleniumCreated,
	}

	var b strings.Builder
	b.WriteString("// Behat.\n")
	b.WriteString("$CFG->behat_wwwroot  = '" + s.WWWRoot + "';\n")
	b.WriteString("$CFG->behat_prefix   = '" + BehatPrefix + "';\n")
	b.WriteString("$CFG->behat_dataroot = '" + BehatDataroot + "';\n")
	if s.Selenium {
		b.WriteString("$CFG->behat_profiles = [\n")
		b.WriteString("    'default' => [\n")
		b.WriteString("        'browser' => 'chrome',\n")
		b.WriteString("        'wd_host' => '" + SeleniumWebDriverURL + "',\n")
		b.WriteString("    ],\n")
		b.WriteString("];\n")
	}
	s.ConfigPHP = b.String()

	admin := "admin"
	if app.MoodleVersion == MoodleVersion51Plus || app.MoodleVersion == "" {
		admin = "public/admin"
	}
	s.InitCommand = app.Folder + " " + admin + "/tool/behat/cli/init.php"

	var components []string
	for _, m := range cfg.MoodlePlugins {
		if m.App == app.Folder && validComponent.MatchString(m.Component) && !slices.Contains(components, m.Component) {
			components = append(components, m.Component)
		}
	}
	if len(components) == 0 {
		components = []string{"local_example"}
	}
	for _, c := range components {
		tags := "@" + c
		if !s.Selenium {
			tags += "&&~@javascript"
		}
		s.RunCommands = append(s.RunCommands, app.Folder+" vendor/bin/behat --config "+BehatConfigFile+" --tags='"+tags+"'")
	}
	return s, true
}
