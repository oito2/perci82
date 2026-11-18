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
	"strings"
	"testing"

	"github.com/oito2/perci82/internal/config"
)

func TestContainerFormRequest_BehatOnlyForMoodle(t *testing.T) {
	moodle := ContainerFormRequest{Kind: config.AppTypeMoodle, Behat: true}
	if !moodle.toAppContainer().Behat {
		t.Error("Behat dropped from a Moodle app")
	}
	php := ContainerFormRequest{Kind: config.AppTypePHP, Behat: true}
	if php.toAppContainer().Behat {
		t.Error("Behat kept on a PHP app")
	}
}

func TestDockerService_BehatAndSelenium(t *testing.T) {
	isolateActions(t)
	behat := config.AppContainer{Folder: "mdle", URL: "mdle.localhost", Type: config.AppTypeMoodle, PHPVersion: "8.3", MoodleVersion: "5.1+", Behat: true}
	registerApp(t, behat)
	registerApp(t, config.AppContainer{Folder: "loja", URL: "loja.localhost", Type: config.AppTypePHP, PHPVersion: "8.3"})
	if err := config.Update(func(c *config.Config) error {
		c.Docker.SeleniumCreated = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	fakeDockerOnPath(t, "running")
	svc := realDockerService(t)

	rows := map[string]ContainerRow{}
	for _, row := range svc.GetContainerRows() {
		rows[row.Folder] = row
	}
	if row, ok := rows["selenium"]; !ok || row.Family != kindSelenium || row.CanEdit {
		t.Errorf("Selenium row = %+v, %v; want a non-editable selenium row", row, ok)
	}
	if !rows["mdle"].Behat || rows["loja"].Behat {
		t.Errorf("Behat flags: mdle=%v loja=%v; want true/false", rows["mdle"].Behat, rows["loja"].Behat)
	}

	edit, err := svc.GetContainerForEdit(kindApp, "mdle")
	if err != nil || !edit.Behat {
		t.Errorf("GetContainerForEdit = %+v, %v; want Behat on", edit, err)
	}

	setup, err := svc.GetBehatSetup("mdle")
	if err != nil || setup.WWWRoot != "http://mdle-behat.test" || !strings.Contains(setup.ConfigPHP, "wd_host") {
		t.Errorf("GetBehatSetup(mdle) = %+v, %v", setup, err)
	}
	if _, err := svc.GetBehatSetup("loja"); err == nil {
		t.Error("GetBehatSetup accepted an app without Behat")
	}
	if _, err := svc.GetBehatSetup("nada"); err == nil {
		t.Error("GetBehatSetup accepted an unregistered app")
	}
}
