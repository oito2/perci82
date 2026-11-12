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

// Package i18n holds the languages the Perci82 applications speak: the
// supported locales, the language choice stored in the config ("system"
// or a locale), the locale detected from the environment, and the
// embedded catalog of the messages the Go side shows to the user.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

//go:generate go run ./genoito2 -l10n ../theme/parity/l10n.json -out ../gui/web/i18n/oito2

//go:embed catalog/*.json
var catalogFiles embed.FS

const (
	// PtBR is Brazilian Portuguese.
	PtBR = "pt-BR"
	// En is English.
	En = "en"
	// SystemChoice is the language choice that follows the environment
	// (LC_ALL, LC_MESSAGES, LANG).
	SystemChoice = "system"
)

// locales lists the supported locales, in the order the language
// selector shows them.
var locales = []string{PtBR, En}

// Locales returns the supported locales.
func Locales() []string {
	return append([]string(nil), locales...)
}

// Option is one choice of the language selector.
type Option struct {
	// ID is the value stored in the config: "system" or a locale.
	ID string `json:"id"`
	// Label is the language's own name; empty for "system", whose label
	// the frontend translates.
	Label string `json:"label"`
}

// Options returns the language choices: "system", then each locale under
// its own name.
func Options() []Option {
	return []Option{
		{ID: SystemChoice},
		{ID: PtBR, Label: "Português (Brasil)"},
		{ID: En, Label: "English"},
	}
}

// Normalize returns locale when it is a supported locale, "" otherwise.
func Normalize(locale string) string {
	for _, l := range locales {
		if locale == l {
			return l
		}
	}
	return ""
}

// ChoiceOrSystem returns choice when it is a supported locale, "system"
// otherwise (unset or unknown values included).
func ChoiceOrSystem(choice string) string {
	if l := Normalize(choice); l != "" {
		return l
	}
	return SystemChoice
}

// Detect returns the locale of the environment read through getenv: the
// first non-empty of LC_ALL, LC_MESSAGES and LANG; a value starting with
// "pt" is Brazilian Portuguese, anything else (unset included) English.
func Detect(getenv func(string) string) string {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := getenv(name)
		if v == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(v), "pt") {
			return PtBR
		}
		return En
	}
	return En
}

// Resolve returns the locale a language choice stands for: the choice
// itself when it is a supported locale, otherwise the environment's
// locale (Detect over os.Getenv).
func Resolve(choice string) string {
	if l := Normalize(choice); l != "" {
		return l
	}
	return Detect(os.Getenv)
}

// placeholder matches a {name} placeholder of a catalog message.
var placeholder = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

// T returns the message key of the locale's catalog, with each {name}
// placeholder replaced by args[name] (formatted with %v). A key missing
// from the locale falls back to English, then to the key itself; a
// placeholder without an argument is kept as is.
func T(locale, key string, args map[string]any) string {
	cats := catalogs()
	msg, ok := cats[Resolve(locale)][key]
	if !ok {
		msg, ok = cats[En][key]
	}
	if !ok {
		return key
	}
	if len(args) == 0 {
		return msg
	}
	return placeholder.ReplaceAllStringFunc(msg, func(m string) string {
		if v, found := args[m[1:len(m)-1]]; found {
			return fmt.Sprint(v)
		}
		return m
	})
}

// Placeholders returns the names of the {name} placeholders of msg, in
// order of appearance.
func Placeholders(msg string) []string {
	var names []string
	for _, m := range placeholder.FindAllStringSubmatch(msg, -1) {
		names = append(names, m[1])
	}
	return names
}

// Catalog returns a copy of the embedded catalog of locale (nil for an
// unsupported one).
func Catalog(locale string) map[string]string {
	src, ok := catalogs()[locale]
	if !ok {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// catalogs parses the embedded catalogs once; they are part of the
// binary, so a failure is a programmer error.
var catalogs = sync.OnceValue(func() map[string]map[string]string {
	out := make(map[string]map[string]string, len(locales))
	for _, l := range locales {
		data, err := catalogFiles.ReadFile("catalog/" + l + ".json")
		if err != nil {
			panic(fmt.Sprintf("i18n: read catalog %s: %v", l, err))
		}
		var msgs map[string]string
		if err := json.Unmarshal(data, &msgs); err != nil {
			panic(fmt.Sprintf("i18n: parse catalog %s: %v", l, err))
		}
		out[l] = msgs
	}
	return out
})
