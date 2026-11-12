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

// Command genoito2 writes the common frontend catalogs generated from the
// oito2_ui l10n.json fixture: one <locale>.json per supported locale in
// the folder -out, with the "material" and "getters" messages under the
// key "oito2.<name>". With -check it writes nothing and fails when a file
// differs from the generated one.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/oito2/perci82/internal/i18n"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

// run parses args, then writes or checks the catalogs, returning the exit
// code.
func run(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("genoito2", flag.ContinueOnError)
	flags.SetOutput(stderr)
	l10nPath := flags.String("l10n", "", "oito2_ui l10n.json fixture to read")
	out := flags.String("out", "", "folder of the catalogs to write")
	check := flags.Bool("check", false, "fail when a catalog in -out differs from the generated one")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *l10nPath == "" || *out == "" {
		fmt.Fprintln(stderr, "Error: -l10n and -out are required")
		return 2
	}
	data, err := os.ReadFile(*l10nPath)
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	files, err := generate(data)
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	stale := false
	for _, locale := range i18n.Locales() {
		path := filepath.Join(*out, locale+".json")
		if *check {
			current, err := os.ReadFile(path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				fmt.Fprintln(stderr, "Error:", err)
				return 1
			}
			if !bytes.Equal(current, files[locale]) {
				fmt.Fprintf(stderr, "%s is out of date: run `go generate ./internal/i18n`\n", path)
				stale = true
			}
			continue
		}
		if err := os.WriteFile(path, files[locale], 0o644); err != nil {
			fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
	}
	if stale {
		return 1
	}
	return 0
}

// l10nLocale is the part of one locale of l10n.json the catalogs use.
type l10nLocale struct {
	Material map[string]string `json:"material"`
	Getters  map[string]string `json:"getters"`
}

// generate returns the catalog file of each supported locale built from
// the l10n.json contents data. A locale missing from data, or a name
// present in both "material" and "getters", is an error.
func generate(data []byte) (map[string][]byte, error) {
	var fixture map[string]l10nLocale
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("parse l10n.json: %w", err)
	}
	files := make(map[string][]byte)
	for _, locale := range i18n.Locales() {
		src, ok := fixture[locale]
		if !ok {
			return nil, fmt.Errorf("l10n.json has no locale %s", locale)
		}
		msgs := make(map[string]string, len(src.Material)+len(src.Getters))
		for name, msg := range src.Material {
			msgs["oito2."+name] = msg
		}
		for name, msg := range src.Getters {
			if _, dup := msgs["oito2."+name]; dup {
				return nil, fmt.Errorf("l10n.json %s: %q is both a material and a getters message", locale, name)
			}
			msgs["oito2."+name] = msg
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "\t")
		if err := enc.Encode(msgs); err != nil {
			return nil, fmt.Errorf("encode %s catalog: %w", locale, err)
		}
		files[locale] = buf.Bytes()
	}
	return files, nil
}
