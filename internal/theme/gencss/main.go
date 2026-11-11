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

// Command gencss writes the oito2.css stylesheet generated from the
// parity fixtures (theme.GenerateCSS) to the file -out. With -check it
// writes nothing and fails when the file differs from the generated one.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/oito2/perci82/internal/theme"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

// run parses args, then writes or checks the stylesheet, returning the
// exit code.
func run(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("gencss", flag.ContinueOnError)
	flags.SetOutput(stderr)
	out := flags.String("out", "", "stylesheet to write")
	check := flags.Bool("check", false, "fail when -out differs from the generated stylesheet")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *out == "" {
		fmt.Fprintln(stderr, "Error: -out is required")
		return 2
	}
	css := theme.GenerateCSS()
	if *check {
		current, err := os.ReadFile(*out)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if string(current) != css {
			fmt.Fprintf(stderr, "%s is out of date: run `go generate ./internal/theme`\n", *out)
			return 1
		}
		return 0
	}
	if err := os.WriteFile(*out, []byte(css), 0o644); err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	return 0
}
