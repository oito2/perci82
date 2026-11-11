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

// Package theme holds the oito2_ui design data the Perci82 windows use:
// the parity fixtures copied from oito2_ui (parity/), the color schemes
// and presets built from them, the CSS generated from them, and the
// system appearance settings ("Definido pelo sistema").
package theme

import (
	"embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
)

//go:generate go run ./gencss -out ../gui/web/css/oito2.css

//go:embed parity/tokens.json parity/schemes.json parity/foundation.json
var parity embed.FS

// SystemChoice is the theme choice that follows the system's appearance
// settings.
const SystemChoice = "system"

// Tokens are the design measures and fixed colors of tokens.json.
type Tokens struct {
	Radius     map[string]float64           `json:"radius"`
	Spacing    map[string]float64           `json:"spacing"`
	Sizes      map[string]float64           `json:"sizes"`
	Accents    map[string]string            `json:"accents"`
	Typography Typography                   `json:"typography"`
	Neutrals   map[string]map[string]string `json:"neutrals"`
}

// Typography is the type scale of tokens.json.
type Typography struct {
	FontFamilyFallback []string             `json:"fontFamilyFallback"`
	EmphasisWeight     int                  `json:"emphasisWeight"`
	Scale              map[string]TextStyle `json:"scale"`
}

// TextStyle is one entry of the type scale.
type TextStyle struct {
	FontSize      float64 `json:"fontSize"`
	LineHeight    float64 `json:"lineHeight"`
	FontWeight    int     `json:"fontWeight"`
	LetterSpacing float64 `json:"letterSpacing"`
}

// ColorTheme is one resolved color scheme of schemes.json.
type ColorTheme struct {
	Brightness   string            `json:"brightness"`
	HighContrast bool              `json:"highContrast"`
	Background   string            `json:"background"`
	ColorScheme  map[string]string `json:"colorScheme"`
	Semantic     map[string]string `json:"semantic"`
	Chrome       map[string]string `json:"chrome"`
}

// Scheme is a named color scheme: a base scheme (light or dark with an
// accent) or a preset, each in normal and high contrast.
type Scheme struct {
	// Name is the CSS name, the value of data-oito2-scheme: "light-blue",
	// "dark-teal-hc", "everforest-light", "nord-dark-hc"…
	Name  string
	Theme ColorTheme
}

// Preset is a named theme of oito2_ui_themes.
type Preset struct {
	// ID is the preset's oito2_ui name ("everforestLight"), stored in
	// config.yaml.
	ID string `json:"id"`
	// Label is the name shown to the user.
	Label string `json:"label"`
	// Scheme is the CSS scheme name in normal contrast; the high contrast
	// one adds "-hc".
	Scheme string `json:"scheme"`
	// Dark reports whether the preset is dark.
	Dark bool `json:"dark"`
}

type schemesFile struct {
	Base []struct {
		Brightness   string     `json:"brightness"`
		Accent       string     `json:"accent"`
		HighContrast bool       `json:"highContrast"`
		Theme        ColorTheme `json:"theme"`
	} `json:"base"`
	Presets []struct {
		Preset       string     `json:"preset"`
		Label        string     `json:"label"`
		IsDark       bool       `json:"isDark"`
		HighContrast bool       `json:"highContrast"`
		Theme        ColorTheme `json:"theme"`
	} `json:"presets"`
}

// data is the design data parsed from the embedded fixtures.
type data struct {
	tokens  Tokens
	schemes []Scheme
	presets []Preset
}

// load parses the embedded fixtures once.
var load = sync.OnceValues(func() (*data, error) {
	var d data
	if err := readJSON("parity/tokens.json", &d.tokens); err != nil {
		return nil, err
	}
	var s schemesFile
	if err := readJSON("parity/schemes.json", &s); err != nil {
		return nil, err
	}
	for _, b := range s.Base {
		d.schemes = append(d.schemes, Scheme{Name: BaseSchemeName(b.Brightness, b.Accent, b.HighContrast), Theme: b.Theme})
	}
	for _, p := range s.Presets {
		name := kebab(p.Preset)
		d.schemes = append(d.schemes, Scheme{Name: withContrast(name, p.HighContrast), Theme: p.Theme})
		if !slices.ContainsFunc(d.presets, func(x Preset) bool { return x.ID == p.Preset }) {
			d.presets = append(d.presets, Preset{ID: p.Preset, Label: p.Label, Scheme: name, Dark: p.IsDark})
		}
	}
	return &d, nil
})

// readJSON decodes the embedded fixture at path into v.
func readJSON(path string, v any) error {
	raw, err := parity.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// mustLoad returns the parsed fixtures; they are embedded, so a failure is
// a build defect.
func mustLoad() *data {
	d, err := load()
	if err != nil {
		panic(err)
	}
	return d
}

// DesignTokens returns the design tokens.
func DesignTokens() Tokens {
	return mustLoad().tokens
}

// Schemes returns every color scheme: the base schemes, then the presets.
func Schemes() []Scheme {
	return slices.Clone(mustLoad().schemes)
}

// Presets returns the named presets in oito2_ui's order.
func Presets() []Preset {
	return slices.Clone(mustLoad().presets)
}

// ValidChoice reports whether choice is SystemChoice or a preset ID.
func ValidChoice(choice string) bool {
	return choice == SystemChoice || slices.ContainsFunc(mustLoad().presets, func(p Preset) bool { return p.ID == choice })
}

// ChoiceOrSystem returns choice when it is valid, SystemChoice otherwise
// (an unset value, or a daisyUI theme name saved by Perci 1.9).
func ChoiceOrSystem(choice string) string {
	if ValidChoice(choice) {
		return choice
	}
	return SystemChoice
}

// BaseSchemeName returns the CSS name of the base scheme of brightness
// ("light" or "dark") and accent, with "-hc" for high contrast.
func BaseSchemeName(brightness, accent string, highContrast bool) string {
	return withContrast(brightness+"-"+accent, highContrast)
}

// withContrast appends "-hc" to name for high contrast.
func withContrast(name string, highContrast bool) string {
	if highContrast {
		return name + "-hc"
	}
	return name
}

// kebab converts camelCase to kebab-case.
func kebab(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('-')
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}
