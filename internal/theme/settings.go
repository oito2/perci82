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

package theme

import (
	"math"
	"strings"
)

// Desktop environments detected from the session variables.
const (
	DesktopGNOME    = "gnome"
	DesktopKDE      = "kde"
	DesktopXfce     = "xfce"
	DesktopBudgie   = "budgie"
	DesktopCOSMIC   = "cosmic"
	DesktopCinnamon = "cinnamon"
	DesktopMATE     = "mate"
	DesktopLXQt     = "lxqt"
	DesktopPantheon = "pantheon"
	DesktopUnity    = "unity"
	DesktopUnknown  = "unknown"
)

// desktopNames maps the lowercase names found in XDG_CURRENT_DESKTOP and
// DESKTOP_SESSION to a desktop environment.
var desktopNames = map[string]string{
	"gnome": DesktopGNOME, "gnome-classic": DesktopGNOME, "gnome-xorg": DesktopGNOME, "gnome-wayland": DesktopGNOME,
	"kde": DesktopKDE, "plasma": DesktopKDE, "plasmawayland": DesktopKDE, "plasmax11": DesktopKDE,
	"xfce": DesktopXfce, "xfce4": DesktopXfce, "xubuntu": DesktopXfce,
	"budgie": DesktopBudgie, "budgie-desktop": DesktopBudgie,
	"cosmic":     DesktopCOSMIC,
	"x-cinnamon": DesktopCinnamon, "cinnamon": DesktopCinnamon, "cinnamon2d": DesktopCinnamon,
	"mate": DesktopMATE,
	"lxqt": DesktopLXQt, "lubuntu": DesktopLXQt,
	"pantheon": DesktopPantheon,
	"unity":    DesktopUnity, "unity7": DesktopUnity,
}

// DetectDesktop returns the desktop environment named by
// XDG_CURRENT_DESKTOP (its first known entry) or, failing that, by
// DESKTOP_SESSION; DesktopUnknown when neither names one.
func DetectDesktop(xdgCurrentDesktop, desktopSession string) string {
	for entry := range strings.SplitSeq(xdgCurrentDesktop, ":") {
		if de, ok := desktopNames[strings.ToLower(strings.TrimSpace(entry))]; ok {
			return de
		}
	}
	if de, ok := desktopNames[strings.ToLower(strings.TrimSpace(desktopSession))]; ok {
		return de
	}
	return DesktopUnknown
}

// gnomeFamily reports whether de keeps its appearance in GNOME's
// org.gnome.desktop.interface settings.
func gnomeFamily(de string) bool {
	switch de {
	case DesktopGNOME, DesktopBudgie, DesktopPantheon, DesktopUnity, DesktopUnknown:
		return true
	}
	return false
}

// SystemSettings are the appearance preferences read from the desktop.
type SystemSettings struct {
	// ColorScheme is "light", "dark" or "" (no preference).
	ColorScheme string
	// Accent is the accent color as "#RRGGBB", "" when the desktop has
	// none.
	Accent string
	// HighContrast reports whether the desktop asks for high contrast.
	HighContrast bool
}

// SystemScheme is what "Definido pelo sistema" resolves to.
type SystemScheme struct {
	// Scheme is the base scheme's CSS name ("dark-teal", "light-blue-hc"…).
	Scheme string `json:"scheme"`
	// HighContrast reports whether the desktop asks for high contrast;
	// a preset is then shown in its high contrast variant.
	HighContrast bool `json:"highContrast"`
}

// defaultAccent is the accent used when the desktop has none.
const defaultAccent = "blue"

// Resolve returns the base scheme of s: its brightness (light without a
// preference), the oito2 accent closest to its accent color (blue without
// one) and its contrast.
func (s SystemSettings) Resolve() SystemScheme {
	brightness := "light"
	if s.ColorScheme == "dark" {
		brightness = "dark"
	}
	accent := defaultAccent
	if c, err := parseHex(s.Accent); err == nil {
		accent = nearestAccent(c)
	}
	return SystemScheme{Scheme: BaseSchemeName(brightness, accent, s.HighContrast), HighContrast: s.HighContrast}
}

// nearestAccent returns the name of the oito2 accent closest to c, by
// distance in RGB.
func nearestAccent(c rgb) string {
	best, bestDist := defaultAccent, math.Inf(1)
	for name, hex := range DesignTokens().Accents {
		a := mustHex(hex)
		d := (a.r-c.r)*(a.r-c.r) + (a.g-c.g)*(a.g-c.g) + (a.b-c.b)*(a.b-c.b)
		if d < bestDist || (d == bestDist && name < best) {
			best, bestDist = name, d
		}
	}
	return best
}
