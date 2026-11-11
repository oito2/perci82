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
	"context"
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
)

// foundation is the part of foundation.json these tests replay.
type foundation struct {
	Contrast struct {
		Ratio []struct {
			A, B  string
			Ratio float64
		} `json:"ratio"`
		ForegroundFor []struct {
			Background, Foreground string
		} `json:"foregroundFor"`
		Ensure []struct {
			Color       string
			Backgrounds []string
			Target      float64
			Result      string
		} `json:"ensure"`
	} `json:"contrast"`
	DesktopEnvironments []struct {
		XdgCurrentDesktop *string `json:"xdgCurrentDesktop"`
		DesktopSession    *string `json:"desktopSession"`
		Result            string  `json:"result"`
	} `json:"desktopEnvironments"`
	SystemSettings []struct {
		OS                 string  `json:"os"`
		IsWeb              bool    `json:"isWeb"`
		DesktopEnvironment *string `json:"desktopEnvironment"`
		Result             struct {
			AccentColor  *string `json:"accentColor"`
			ColorScheme  *string `json:"colorScheme"`
			HighContrast bool    `json:"highContrast"`
		} `json:"result"`
	} `json:"systemSettings"`
}

func loadFoundation(t *testing.T) foundation {
	t.Helper()
	var f foundation
	if err := readJSON("parity/foundation.json", &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Contrast.Ratio) == 0 || len(f.DesktopEnvironments) == 0 || len(f.SystemSettings) == 0 {
		t.Fatal("foundation.json has no cases")
	}
	return f
}

func hexes(t *testing.T, values []string) []rgb {
	t.Helper()
	out := make([]rgb, len(values))
	for i, v := range values {
		out[i] = mustHex(v)
	}
	return out
}

// The contrast math gives oito2_ui's results for every case of
// foundation.json.
func TestContrast_FoundationCases(t *testing.T) {
	f := loadFoundation(t)
	for _, c := range f.Contrast.Ratio {
		if got := contrastRatio(mustHex(c.A), mustHex(c.B)); math.Abs(got-c.Ratio) > 1e-9 {
			t.Errorf("ratio(%s, %s) = %v, want %v", c.A, c.B, got, c.Ratio)
		}
	}
	for _, c := range f.Contrast.ForegroundFor {
		if got := foregroundFor(mustHex(c.Background)).hex(); got != c.Foreground {
			t.Errorf("foregroundFor(%s) = %s, want %s", c.Background, got, c.Foreground)
		}
	}
	for _, c := range f.Contrast.Ensure {
		if got := ensureContrast(mustHex(c.Color), hexes(t, c.Backgrounds), c.Target).hex(); got != c.Result {
			t.Errorf("ensure(%s, %v, %v) = %s, want %s", c.Color, c.Backgrounds, c.Target, got, c.Result)
		}
	}
}

// Desktop detection gives oito2_ui's result for every case of
// foundation.json.
func TestDetectDesktop_FoundationCases(t *testing.T) {
	deref := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	for _, c := range loadFoundation(t).DesktopEnvironments {
		if got := DetectDesktop(deref(c.XdgCurrentDesktop), deref(c.DesktopSession)); got != c.Result {
			t.Errorf("DetectDesktop(%q, %q) = %q, want %q", deref(c.XdgCurrentDesktop), deref(c.DesktopSession), got, c.Result)
		}
	}
}

// With nothing read from the desktop, the settings are oito2_ui's Linux
// fallback: no accent, no color scheme, normal contrast — resolved to the
// light scheme with the blue accent.
func TestCompose_NothingReadIsTheLinuxFallback(t *testing.T) {
	checked := 0
	for _, c := range loadFoundation(t).SystemSettings {
		if c.OS != "linux" || c.IsWeb {
			continue
		}
		de := DesktopUnknown
		if c.DesktopEnvironment != nil {
			de = *c.DesktopEnvironment
		}
		got := compose(context.Background(), portalValues{}, de, func(context.Context, string, string) string { return "" })
		want := SystemSettings{HighContrast: c.Result.HighContrast}
		if c.Result.AccentColor != nil {
			want.Accent = *c.Result.AccentColor
		}
		if c.Result.ColorScheme != nil {
			want.ColorScheme = *c.Result.ColorScheme
		}
		if got != want {
			t.Errorf("desktop %q: settings = %+v, want %+v", de, got, want)
		}
		if r := got.Resolve(); r.Scheme != "light-blue" || r.HighContrast {
			t.Errorf("desktop %q: resolved = %+v, want light-blue", de, r)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no Linux case in foundation.json")
	}
}

func u32(v uint32) *uint32 { return &v }

// The portal's answers come first; GNOME's settings fill in what it
// doesn't answer, only on a GNOME-family desktop.
func TestCompose(t *testing.T) {
	gnome := map[string]string{
		gnomeInterface + "/color-scheme": "'prefer-dark'\n",
		gnomeInterface + "/accent-color": "'teal'\n",
		gnomeA11y + "/high-contrast":     "true\n",
	}
	get := func(_ context.Context, schema, key string) string { return gnome[schema+"/"+key] }
	teal := DesignTokens().Accents["teal"]

	cases := []struct {
		name   string
		portal portalValues
		de     string
		want   SystemSettings
	}{
		{"portal only", portalValues{colorScheme: u32(2), accent: &[3]float64{1, 0, 0}, highContrast: u32(0)}, DesktopKDE,
			SystemSettings{ColorScheme: "light", Accent: "#FF0000"}},
		{"portal wins over GNOME", portalValues{colorScheme: u32(2), accent: &[3]float64{0, 0, 1}, highContrast: u32(1)}, DesktopGNOME,
			SystemSettings{ColorScheme: "light", Accent: "#0000FF", HighContrast: true}},
		{"portal without preference", portalValues{colorScheme: u32(0), accent: &[3]float64{-1, -1, -1}}, DesktopGNOME,
			SystemSettings{HighContrast: true}},
		{"no portal on GNOME", portalValues{}, DesktopGNOME, SystemSettings{ColorScheme: "dark", Accent: teal, HighContrast: true}},
		{"no portal on KDE", portalValues{}, DesktopKDE, SystemSettings{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := compose(context.Background(), c.portal, c.de, get); got != c.want {
				t.Errorf("compose = %+v, want %+v", got, c.want)
			}
		})
	}
}

// Every oito2 accent resolves to itself; any other color to the closest.
func TestResolve(t *testing.T) {
	for name, hex := range DesignTokens().Accents {
		if got := (SystemSettings{ColorScheme: "dark", Accent: hex}).Resolve(); got.Scheme != "dark-"+name {
			t.Errorf("accent %s resolved to %q", name, got.Scheme)
		}
	}
	got := SystemSettings{Accent: "#3080E0", HighContrast: true}.Resolve()
	if got.Scheme != "light-blue-hc" || !got.HighContrast {
		t.Errorf("near-blue high contrast = %+v", got)
	}
}

// schemes.json gives the 36 base schemes, the 13 presets in both
// contrasts, and unique names.
func TestSchemesAndPresets(t *testing.T) {
	schemes := Schemes()
	if len(schemes) != 62 {
		t.Errorf("%d schemes, want 62", len(schemes))
	}
	seen := map[string]bool{}
	for _, s := range schemes {
		if seen[s.Name] {
			t.Errorf("duplicate scheme %q", s.Name)
		}
		seen[s.Name] = true
	}
	presets := Presets()
	if len(presets) != 13 {
		t.Fatalf("%d presets, want 13", len(presets))
	}
	for _, p := range presets {
		if !seen[p.Scheme] || !seen[p.Scheme+"-hc"] || p.Label == "" {
			t.Errorf("preset %+v lacks a scheme or a label", p)
		}
	}
	if presets[0].ID != "everforestLight" || presets[0].Scheme != "everforest-light" {
		t.Errorf("first preset = %+v", presets[0])
	}
}

// A preset ID or "system" is kept; anything else — such as a daisyUI theme
// saved by Perci 1.9 — becomes "system".
func TestChoiceOrSystem(t *testing.T) {
	for in, want := range map[string]string{
		"system": "system", "nordDark": "nordDark", "catppuccinMocha": "catppuccinMocha",
		"": "system", "dark": "system", "garden": "system", "nord-dark": "system",
	} {
		if got := ChoiceOrSystem(in); got != want {
			t.Errorf("ChoiceOrSystem(%q) = %q, want %q", in, got, want)
		}
	}
}

// The committed stylesheet is the generated one, and it has a rule for
// every scheme and the daisyUI mapping.
func TestGeneratedCSSIsCurrent(t *testing.T) {
	committed, err := os.ReadFile("../gui/web/css/oito2.css")
	if err != nil {
		t.Fatal(err)
	}
	css := GenerateCSS()
	if string(committed) != css {
		t.Fatal("internal/gui/web/css/oito2.css is out of date: run `go generate ./internal/theme`")
	}
	for _, s := range Schemes() {
		if !strings.Contains(css, ":root[data-oito2-scheme='"+s.Name+"'] {") {
			t.Errorf("no rule for %s", s.Name)
		}
	}
	for _, m := range daisyMapping {
		if !strings.Contains(css, "\t"+m[0]+": ") {
			t.Errorf("daisyUI variable %s not mapped", m[0])
		}
	}
}

// Every semantic color gets a text color that reaches at least 4.5:1, and
// every variable the daisyUI mapping refers to is set by every scheme.
func TestGeneratedCSS_SchemesDefineMappedVariables(t *testing.T) {
	css := GenerateCSS()
	rules := strings.Split(css, ":root[data-oito2-scheme='")[1:]
	for _, rule := range rules {
		name, body, _ := strings.Cut(rule, "'] {")
		body, _, _ = strings.Cut(body, "}")
		for _, m := range daisyMapping {
			ref, ok := strings.CutPrefix(m[1], "var(")
			if !ok || !strings.Contains(ref, "-color-") && !strings.Contains(ref, "-chrome-") {
				continue
			}
			ref = strings.TrimSuffix(ref, ")")
			if !strings.Contains(body, "\t"+ref+": ") {
				t.Errorf("scheme %s doesn't set %s", name, ref)
			}
		}
	}
	for _, s := range Schemes() {
		for _, k := range []string{"info", "success", "warning"} {
			bg := mustHex(s.Theme.Semantic[k])
			if r := contrastRatio(foregroundFor(bg), bg); r < 4.5 {
				t.Errorf("%s: text on %s reaches only %.2f:1", s.Name, k, r)
			}
		}
	}
}

// The embedded fixtures are the ones parity-sync copied.
func TestParityFixturesRecorded(t *testing.T) {
	raw, err := os.ReadFile("parity/source.json")
	if err != nil {
		t.Fatal(err)
	}
	var src struct {
		Repository, Commit string
		Manifest           map[string]any
	}
	if err := json.Unmarshal(raw, &src); err != nil {
		t.Fatal(err)
	}
	if src.Repository != "oito2/oito2_ui" || len(src.Commit) != 40 || src.Manifest["flutter"] == nil {
		t.Errorf("source.json = %+v", src)
	}
}
