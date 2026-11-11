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

// Command parity-sync copies the parity fixtures oito2_ui exports
// (tools/parity/fixtures/*.json) that Perci82 uses into
// internal/theme/parity, byte for byte, and records where they came from
// in internal/theme/parity/source.json.
//
//	go run ./tools/parity-sync [--from <dir>]          copy the fixtures
//	go run ./tools/parity-sync [--from <dir>] --check  copy nothing; fail when a fixture differs
//
// <dir> is a checkout of oito2_ui (default: references/oito2_ui). It runs
// from the repository root.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// usedFixtures are the fixtures Perci82 copies and compares, without the
// ".json" extension.
var usedFixtures = []string{"tokens", "schemes", "foundation", "l10n"}

// ignoredFixtures are fixtures oito2_ui exports that Perci82 doesn't use.
var ignoredFixtures = []string{"format", "syntax", "qr", "manifest"}

const (
	sourceFolder = "tools/parity/fixtures"
	targetFolder = "internal/theme/parity"
	sourceRecord = "source.json"
	repository   = "oito2/oito2_ui"
)

// errStale is returned by --check when a fixture differs from oito2_ui.
var errStale = errors.New("parity fixtures are out of date: run `go run ./tools/parity-sync`")

func main() {
	os.Exit(run(os.Args[1:], ".", os.Stdout, os.Stderr))
}

// run parses args and syncs the fixtures of root, returning the exit code.
func run(args []string, root string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("parity-sync", flag.ContinueOnError)
	flags.SetOutput(stderr)
	from := flags.String("from", filepath.Join("references", "oito2_ui"), "checkout of oito2_ui")
	check := flags.Bool("check", false, "compare only; fail when a fixture differs")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if err := syncFixtures(*from, root, *check, stdout, stderr); err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	return 0
}

// syncFixtures compares every used fixture of the oito2_ui checkout from
// with its copy under root. With check it only reports the differing ones
// and returns errStale; otherwise it overwrites them and writes
// source.json with the checkout's commit and its manifest.json. A fixture
// oito2_ui exports that Perci82 neither uses nor ignores is reported as a
// warning on stderr.
func syncFixtures(from, root string, check bool, stdout, stderr io.Writer) error {
	sourceDir := filepath.Join(from, sourceFolder)
	targetDir := filepath.Join(root, targetFolder)

	for _, name := range unknownFixtures(sourceDir) {
		fmt.Fprintf(stderr, "warning: oito2_ui exports %s.json, which Perci82 doesn't use yet\n", name)
	}

	var changed []string
	for _, name := range usedFixtures {
		src, err := os.ReadFile(filepath.Join(sourceDir, name+".json"))
		if err != nil {
			return fmt.Errorf("read fixture: %w", err)
		}
		dst := filepath.Join(targetDir, name+".json")
		current, err := os.ReadFile(dst)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read copy: %w", err)
		}
		if bytes.Equal(src, current) {
			continue
		}
		changed = append(changed, filepath.ToSlash(filepath.Join(targetFolder, name+".json")))
		if check {
			continue
		}
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", targetFolder, err)
		}
		if err := os.WriteFile(dst, src, 0o644); err != nil {
			return fmt.Errorf("write copy: %w", err)
		}
	}

	if check {
		for _, path := range changed {
			fmt.Fprintf(stderr, "%s differs from oito2_ui\n", path)
		}
		if len(changed) > 0 {
			return errStale
		}
		fmt.Fprintln(stdout, "parity fixtures are up to date")
		return nil
	}

	if err := writeSourceRecord(from, targetDir); err != nil {
		return err
	}
	for _, path := range changed {
		fmt.Fprintf(stdout, "updated %s\n", path)
	}
	return nil
}

// unknownFixtures returns the names of the fixtures in dir that are
// neither used nor ignored, sorted.
func unknownFixtures(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || e.IsDir() || slices.Contains(usedFixtures, name) || slices.Contains(ignoredFixtures, name) {
			continue
		}
		names = append(names, name)
	}
	return names
}

// sourceInfo is the content of source.json.
type sourceInfo struct {
	Repository string          `json:"repository"`
	Commit     string          `json:"commit"`
	Manifest   json.RawMessage `json:"manifest"`
}

// writeSourceRecord writes source.json into targetDir: the repository, the
// commit checked out in from (empty when it isn't a git checkout) and the
// content of its manifest.json, tab-indented with a final newline.
func writeSourceRecord(from, targetDir string) error {
	manifest, err := os.ReadFile(filepath.Join(from, sourceFolder, "manifest.json"))
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, manifest); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	info := sourceInfo{Repository: repository, Commit: headCommit(from), Manifest: compact.Bytes()}
	data, err := json.MarshalIndent(info, "", "\t")
	if err != nil {
		return fmt.Errorf("encode %s: %w", sourceRecord, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(targetDir, sourceRecord), data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", sourceRecord, err)
	}
	return nil
}

// headCommit returns the commit checked out in dir, empty when dir is not
// a git checkout.
func headCommit(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
