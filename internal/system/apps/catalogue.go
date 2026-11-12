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

package apps

import "github.com/oito2/perci82/internal/i18n"

// App describes a Flatpak application managed by Perci82 System.
type App struct {
	Name string
	// Kind is the message key of what the application is for ("Photo
	// viewer"), shown after its name; empty for none.
	Kind            string
	FlatID          string
	FlatpakOverride []string // extra args passed to "flatpak override <FlatID> <args...>" after install
}

// Label returns the application's name followed by what it is for, in the
// current language.
func (a App) Label() string {
	if a.Kind == "" {
		return a.Name
	}
	return a.Name + " - " + i18n.Tr(a.Kind, nil)
}

// Catalogue lists all Flatpak apps available for installation.
// To add a new app, append a new App entry here.
var Catalogue = []App{
	{Name: "Loupe", Kind: "apps.kind.photoViewer", FlatID: "org.gnome.Loupe"},
	{Name: "Celluloid", Kind: "apps.kind.videoPlayer", FlatID: "io.github.celluloid_player.Celluloid"},
	{Name: "VLC", Kind: "apps.kind.videoPlayer", FlatID: "org.videolan.VLC"},
	{Name: "Vinyl", Kind: "apps.kind.musicPlayer", FlatID: "page.codeberg.M23Snezhok.Vinyl"},
	{Name: "Calculator", Kind: "apps.kind.calculator", FlatID: "org.gnome.Calculator"},
	{Name: "Gradia", Kind: "apps.kind.screenshot", FlatID: "be.alexandervanhee.gradia"},
	{Name: "Apostrophe", Kind: "apps.kind.markdown", FlatID: "org.gnome.gitlab.somas.Apostrophe"},
	{Name: "Eyedropper", Kind: "apps.kind.colorPicker", FlatID: "com.github.finefindus.eyedropper"},
	{Name: "Gear Lever", Kind: "apps.kind.appImages", FlatID: "it.mijorus.gearlever"},
	{Name: "AppImage Pool", FlatID: "io.github.prateekmedia.appimagepool"},
	{Name: "Web Apps", FlatID: "net.codelogistics.webapps"},
	{Name: "Flatseal", Kind: "apps.kind.flatpakPermissions", FlatID: "com.github.tchx84.Flatseal"},
	{Name: "Bazaar", FlatID: "io.github.kolunmi.Bazaar"},
	{Name: "Zen Browser", FlatID: "app.zen_browser.zen"},
	{Name: "Firefox", FlatID: "org.mozilla.firefox"},
	{Name: "Chromium", FlatID: "org.chromium.Chromium"},
	{Name: "FileZilla", FlatID: "org.filezillaproject.Filezilla"},
	{Name: "Inkscape", FlatID: "org.inkscape.Inkscape"},
	{Name: "Krita", FlatID: "org.kde.krita"},
	{Name: "Penpot", FlatID: "com.authormore.penpotdesktop"},
	{Name: "LibreOffice", FlatID: "org.libreoffice.LibreOffice"},
	{Name: "OnlyOffice", FlatID: "org.onlyoffice.desktopeditors"},
	{Name: "AnyDesk", FlatID: "com.anydesk.Anydesk"},
	{Name: "DBeaver Community", FlatID: "io.dbeaver.DBeaverCommunity", FlatpakOverride: []string{"--share=network"}},
	{Name: "Beekeeper Studio", FlatID: "io.beekeeperstudio.Studio"},
	{Name: "Meld", Kind: "apps.kind.fileCompare", FlatID: "org.gnome.meld"},
	{Name: "Tomatillo", Kind: "apps.kind.pomodoro", FlatID: "io.github.diegopvlk.Tomatillo"},
	{Name: "Extension Manager (GNOME)", FlatID: "com.mattjakeman.ExtensionManager"},
}
