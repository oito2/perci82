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

package i18n

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"nothing set", nil, En},
		{"LANG pt_BR", map[string]string{"LANG": "pt_BR.UTF-8"}, PtBR},
		{"LANG pt_PT", map[string]string{"LANG": "pt_PT.UTF-8"}, PtBR},
		{"LANG en_US", map[string]string{"LANG": "en_US.UTF-8"}, En},
		{"LANG de_DE", map[string]string{"LANG": "de_DE.UTF-8"}, En},
		{"LANG C", map[string]string{"LANG": "C.UTF-8"}, En},
		{"LC_MESSAGES over LANG", map[string]string{"LANG": "en_US.UTF-8", "LC_MESSAGES": "pt_BR.UTF-8"}, PtBR},
		{"LC_ALL over LC_MESSAGES", map[string]string{"LC_MESSAGES": "pt_BR.UTF-8", "LC_ALL": "en_US.UTF-8"}, En},
		{"empty LC_ALL is skipped", map[string]string{"LC_ALL": "", "LANG": "pt_BR.UTF-8"}, PtBR},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect(func(k string) string { return tc.env[k] })
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestChoiceAndResolve(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "pt_BR.UTF-8")
	cases := []struct {
		choice, wantChoice, wantLocale string
	}{
		{"", SystemChoice, PtBR},
		{SystemChoice, SystemChoice, PtBR},
		{"en", "en", En},
		{"pt-BR", "pt-BR", PtBR},
		{"pt_BR", SystemChoice, PtBR},
		{"../../etc/passwd", SystemChoice, PtBR},
	}
	for _, tc := range cases {
		if got := ChoiceOrSystem(tc.choice); got != tc.wantChoice {
			t.Errorf("ChoiceOrSystem(%q) got %q, want %q", tc.choice, got, tc.wantChoice)
		}
		if got := Resolve(tc.choice); got != tc.wantLocale {
			t.Errorf("Resolve(%q) got %q, want %q", tc.choice, got, tc.wantLocale)
		}
	}
	t.Setenv("LANG", "en_US.UTF-8")
	if got := Resolve(SystemChoice); got != En {
		t.Errorf("Resolve(system) with LANG=en_US got %q, want en", got)
	}
}

func TestOptions(t *testing.T) {
	opts := Options()
	ids := make([]string, len(opts))
	for i, o := range opts {
		ids[i] = o.ID
	}
	if want := append([]string{SystemChoice}, Locales()...); !slices.Equal(ids, want) {
		t.Errorf("option IDs got %v, want %v", ids, want)
	}
	if opts[0].Label != "" {
		t.Errorf("system option label got %q, want empty", opts[0].Label)
	}
}

func TestT(t *testing.T) {
	catalogs() // parsed before the swap below
	orig := catalogs
	t.Cleanup(func() { catalogs = orig })
	catalogs = func() map[string]map[string]string {
		return map[string]map[string]string{
			PtBR: {"greet": "Olá, {name}! {name}?", "only.pt": "só"},
			En:   {"greet": "Hello, {name}!", "only.en": "english"},
		}
	}
	cases := []struct {
		name, locale, key string
		args              map[string]any
		want              string
	}{
		{"placeholders filled", PtBR, "greet", map[string]any{"name": "Ana"}, "Olá, Ana! Ana?"},
		{"no args keeps placeholders", En, "greet", nil, "Hello, {name}!"},
		{"missing arg kept", En, "greet", map[string]any{"other": 1}, "Hello, {name}!"},
		{"non-string arg", En, "greet", map[string]any{"name": 3}, "Hello, 3!"},
		{"falls back to English", PtBR, "only.en", nil, "english"},
		{"unknown key", En, "nope", nil, "nope"},
		{"key of another locale only", En, "only.pt", nil, "only.pt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := T(tc.locale, tc.key, tc.args); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// keyUse matches a literal message key passed to T.
var keyUse = regexp.MustCompile(`i18n\.T\([^,()]+,\s*"([^"]+)"`)

// navKey matches a literal key of the section catalog (internal/gui),
// whose fields are translated later.
var navKey = regexp.MustCompile(`"(nav\.[A-Za-z0-9.]+)"`)

// stringLiteral matches any double-quoted Go string literal.
var stringLiteral = regexp.MustCompile(`"((?:[^"\\\n]|\\.)*)"`)

// keyUses returns, over the module's non-test Go files, the keys that
// must exist (passed to i18n.T, or section catalog keys) and every string
// literal (a catalog key appearing in none is unused).
func keyUses(t *testing.T) (required, literals map[string]bool) {
	t.Helper()
	required, literals = map[string]bool{}, map[string]bool{}
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "vendor" || d.Name() == "references" || strings.HasPrefix(d.Name(), ".")) && path != root {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, re := range []*regexp.Regexp{keyUse, navKey} {
			for _, m := range re.FindAllStringSubmatch(string(src), -1) {
				required[m[1]] = true
			}
		}
		for _, m := range stringLiteral.FindAllStringSubmatch(string(src), -1) {
			literals[m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return required, literals
}

// The embedded catalogs: same keys and placeholders in every locale, every
// key used by the code exists, and no key is left unused.
func TestCatalogs(t *testing.T) {
	base := Catalog(Locales()[0])
	if len(base) == 0 {
		t.Fatal("empty catalog")
	}
	keys := func(m map[string]string) []string {
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	for _, locale := range Locales()[1:] {
		cat := Catalog(locale)
		if !slices.Equal(keys(cat), keys(base)) {
			t.Errorf("%s keys got %v, want %v", locale, keys(cat), keys(base))
		}
		for k, msg := range base {
			got, want := Placeholders(cat[k]), Placeholders(msg)
			sort.Strings(got)
			sort.Strings(want)
			if !slices.Equal(got, want) {
				t.Errorf("%s %s placeholders got %v, want %v", locale, k, got, want)
			}
		}
	}

	required, literals := keyUses(t)
	if len(required) == 0 {
		t.Fatal("no message key found in the code")
	}
	for k := range required {
		if _, ok := base[k]; !ok {
			t.Errorf("key %q is used but missing from the catalogs", k)
		}
	}
	for k := range base {
		if !literals[k] {
			t.Errorf("key %q is in the catalogs but never used", k)
		}
	}
}

func TestCatalogUnsupportedLocale(t *testing.T) {
	if got := Catalog("fr"); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}
