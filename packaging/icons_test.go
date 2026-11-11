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

package packaging

import (
	"bytes"
	"image/png"
	"os"
	"strings"
	"testing"
)

func assertPNGSize(t *testing.T, name string, data []byte, size int) {
	t.Helper()
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("%s is not a PNG: %v", name, err)
	}
	if cfg.Width != size || cfg.Height != size {
		t.Errorf("%s is %dx%d, want %dx%d", name, cfg.Width, cfg.Height, size, size)
	}
}

func TestIconsExistForEveryAppAndSize(t *testing.T) {
	for _, app := range []string{AppSystem, AppDev} {
		for _, size := range IconSizes {
			data, err := Icon(app, size)
			if err != nil {
				t.Fatalf("Icon(%q, %d): %v", app, size, err)
			}
			assertPNGSize(t, IconRelPath(app, size), data, size)
		}
	}
}

func TestTrayIconsExistForEveryVariantAndSize(t *testing.T) {
	for _, variant := range []string{"claro", "escuro"} {
		for _, size := range TraySizes {
			data, err := TrayIcon(variant, size)
			if err != nil {
				t.Fatalf("TrayIcon(%q, %d): %v", variant, size, err)
			}
			assertPNGSize(t, variant, data, size)
		}
	}
}

func TestIconRelPath(t *testing.T) {
	if got, want := IconRelPath(AppDev, 48), "48x48/apps/io.github.oito2.perci82.dev.png"; got != want {
		t.Errorf("IconRelPath(dev, 48) = %q, want %q", got, want)
	}
}

func TestMustIconPanicsForMissingIcon(t *testing.T) {
	want, err := Icon(AppSystem, 512)
	if err != nil {
		t.Fatal(err)
	}
	if got := MustIcon(AppSystem, 512); !bytes.Equal(got, want) {
		t.Error("MustIcon returned different bytes than Icon")
	}
	defer func() {
		if recover() == nil {
			t.Error("MustIcon must panic for an icon that isn't embedded")
		}
	}()
	MustIcon("other", 512)
}

func TestDesktopEntries(t *testing.T) {
	for _, c := range []struct{ app, exec string }{
		{AppSystem, "/usr/local/bin/perci82-system"},
		{AppDev, "/usr/local/bin/perci82-dev"},
	} {
		name := IconName(c.app) + ".desktop"
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, want := range []string{
			"\nIcon=" + IconName(c.app) + "\n",
			"\nExec=" + c.exec + "\n",
			"\nStartupWMClass=" + IconName(c.app) + "\n",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s must contain %q:\n%s", name, strings.TrimSpace(want), text)
			}
		}
	}
}
