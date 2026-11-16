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

package ai

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
)

// CustomContext is a context the user wrote: ID names its file and never
// changes, Name is how the screens show it, AppliesTo the optional
// sentence AGENTS.md lists it with, Content its text.
type CustomContext struct {
	ID        string `json:"id" yaml:"id"`
	Name      string `json:"name" yaml:"name"`
	AppliesTo string `json:"appliesTo" yaml:"applies_to,omitempty"`
	Content   string `json:"content" yaml:"content"`
}

// customPrefix starts the standard name (Model.Name) of a custom context,
// followed by its ID: the name config.yaml and export files record.
const customPrefix = "custom:"

// Limits of a custom context.
const (
	maxCustomIDLen      = 40
	maxCustomContentLen = 256 << 10
	maxCustomAppliesLen = 300
)

// customIDRe matches a custom context ID: lowercase letters and digits in
// groups joined by single hyphens.
var customIDRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// customMarker starts the first line Perci adds to a custom context it
// writes into a project, followed by the SHA-256 of the rest and " -->".
const customMarker = "<!-- Custom context written by Perci82 Dev. sha256="

// ErrCustomNotFound is returned for an ID no custom context has.
var ErrCustomNotFound = i18n.NewError("aic.custom.notFound")

// customDir is the folder holding one file per custom context,
// ~/.perci/user-contexts.
func customDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".perci", "user-contexts"), nil
}

// IsCustom reports whether the standard name names a custom context.
func IsCustom(name string) bool {
	return strings.HasPrefix(name, customPrefix)
}

// CustomModelName is the standard name of the custom context id.
func CustomModelName(id string) string {
	return customPrefix + id
}

// signCustom returns content with the signed first line.
func signCustom(content string) string {
	sum := sha256.Sum256([]byte(content))
	return customMarker + hex.EncodeToString(sum[:]) + " -->\n" + content
}

// customSignedUnedited reports whether data is a custom context as Perci
// wrote it, any version of it: its rest still matches its signature.
func customSignedUnedited(data string) bool {
	first, body, ok := strings.Cut(data, "\n")
	if !ok || !strings.HasPrefix(first, customMarker) || !strings.HasSuffix(first, " -->") {
		return false
	}
	want := strings.TrimSuffix(strings.TrimPrefix(first, customMarker), " -->")
	sum := sha256.Sum256([]byte(body))
	return want == hex.EncodeToString(sum[:])
}

// model is c as a standard of the catalog.
func (c CustomContext) model() Model {
	applies := c.AppliesTo
	if applies == "" {
		applies = "Custom context \"" + c.Name + "\""
	}
	return Model{
		Name:        CustomModelName(c.ID),
		Instruction: "user-contexts/CUSTOM-" + strings.ToUpper(c.ID) + ".md",
		AppliesTo:   applies,
		Description: c.AppliesTo,
		Custom:      true,
		label:       c.Name,
		content:     c.Content,
	}
}

// customModels is every readable custom context as a standard, ordered
// by name; a file that cannot be read is left out.
func customModels() []Model {
	list, _ := CustomContexts()
	out := make([]Model, len(list))
	for i, c := range list {
		out[i] = c.model()
	}
	return out
}

// customHeader is the YAML front matter of a custom context file.
type customHeader struct {
	Name      string `yaml:"name"`
	AppliesTo string `yaml:"applies_to,omitempty"`
}

// CustomContexts lists the custom contexts, ordered by name (ignoring
// case). Files whose name is not a valid ID, links, and files that are too
// large or malformed are skipped.
func CustomContexts() ([]CustomContext, error) {
	dir, err := customDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []CustomContext{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []CustomContext{}
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".md")
		if !ok || !validCustomID(id) || !e.Type().IsRegular() {
			continue
		}
		c, err := readCustom(filepath.Join(dir, e.Name()), id)
		if err != nil {
			continue
		}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b CustomContext) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out, nil
}

// FindCustomContext returns the custom context id.
func FindCustomContext(id string) (CustomContext, error) {
	if !validCustomID(id) {
		return CustomContext{}, ErrCustomNotFound
	}
	dir, err := customDir()
	if err != nil {
		return CustomContext{}, err
	}
	path := filepath.Join(dir, id+".md")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return CustomContext{}, ErrCustomNotFound
	}
	return readCustom(path, id)
}

// validCustomID reports whether id is a well-formed custom context ID.
func validCustomID(id string) bool {
	return len(id) <= maxCustomIDLen && customIDRe.MatchString(id)
}

// readCustom reads the custom context file at path: a YAML front matter
// between "---" lines, then the content.
func readCustom(path, id string) (CustomContext, error) {
	f, err := os.Open(path)
	if err != nil {
		return CustomContext{}, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxCustomContentLen+4096+1))
	if err != nil {
		return CustomContext{}, err
	}
	if len(data) > maxCustomContentLen+4096 {
		return CustomContext{}, fmt.Errorf("%s: file too large", path)
	}
	rest, ok := bytes.CutPrefix(data, []byte("---\n"))
	if !ok {
		return CustomContext{}, fmt.Errorf("%s: missing header", path)
	}
	head, body, ok := bytes.Cut(rest, []byte("\n---\n"))
	if !ok {
		return CustomContext{}, fmt.Errorf("%s: missing header end", path)
	}
	var h customHeader
	if err := yaml.Unmarshal(head, &h); err != nil {
		return CustomContext{}, fmt.Errorf("%s: %w", path, err)
	}
	c := CustomContext{ID: id, Name: h.Name, AppliesTo: h.AppliesTo, Content: string(body)}
	if err := c.check(); err != nil {
		return CustomContext{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// normalize trims the name and the sentence (folded onto one line) and
// gives the content Unix line ends and a final line end.
func (c CustomContext) normalize() CustomContext {
	c.Name = strings.TrimSpace(c.Name)
	c.AppliesTo = strings.Join(strings.Fields(c.AppliesTo), " ")
	c.Content = strings.ReplaceAll(c.Content, "\r\n", "\n")
	if c.Content != "" && !strings.HasSuffix(c.Content, "\n") {
		c.Content += "\n"
	}
	return c
}

// check validates an already normalized c: a set name of at most
// maxContextNameLen characters, a sentence of at most
// maxCustomAppliesLen characters, a non-blank UTF-8 content of at most
// maxCustomContentLen bytes.
func (c CustomContext) check() error {
	switch {
	case c.Name == "":
		return i18n.NewError("aic.custom.noName")
	case utf8.RuneCountInString(c.Name) > maxContextNameLen || strings.ContainsAny(c.Name, "\n\r"):
		return i18n.NewError("aic.custom.nameTooLong")
	case utf8.RuneCountInString(c.AppliesTo) > maxCustomAppliesLen:
		return i18n.NewError("aic.custom.appliesTooLong")
	case strings.TrimSpace(c.Content) == "":
		return i18n.NewError("aic.custom.noContent")
	case len(c.Content) > maxCustomContentLen:
		return i18n.NewError("aic.custom.contentTooLong")
	case !utf8.ValidString(c.Content):
		return i18n.NewError("aic.custom.notText")
	}
	return nil
}

// newCustomID derives an ID from name, unique among taken: its ASCII
// letters and digits in lowercase, other runs of characters as one
// hyphen, then "-2", "-3"... when taken.
func newCustomID(name string, taken []CustomContext) string {
	var b strings.Builder
	hyphen := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			hyphen = false
		case b.Len() > 0 && !hyphen:
			b.WriteByte('-')
			hyphen = true
		}
	}
	base := strings.TrimRight(b.String(), "-")
	if len(base) > maxCustomIDLen-4 {
		base = strings.TrimRight(base[:maxCustomIDLen-4], "-")
	}
	if base == "" {
		base = "context"
	}
	used := func(id string) bool {
		return slices.ContainsFunc(taken, func(c CustomContext) bool { return c.ID == id })
	}
	id := base
	for n := 2; used(id); n++ {
		id = base + "-" + strconv.Itoa(n)
	}
	return id
}

// SaveCustomContext creates c (ID empty: a new ID is derived from its
// name) or replaces the custom context c.ID, and returns it as saved. The
// name must not be used by another custom context (ignoring case).
func SaveCustomContext(c CustomContext) (CustomContext, error) {
	c = c.normalize()
	if err := c.check(); err != nil {
		return CustomContext{}, err
	}
	list, err := CustomContexts()
	if err != nil {
		return CustomContext{}, err
	}
	if c.ID == "" {
		c.ID = newCustomID(c.Name, list)
	} else if !slices.ContainsFunc(list, func(o CustomContext) bool { return o.ID == c.ID }) {
		return CustomContext{}, ErrCustomNotFound
	}
	if slices.ContainsFunc(list, func(o CustomContext) bool { return o.ID != c.ID && strings.EqualFold(o.Name, c.Name) }) {
		return CustomContext{}, errors.New(i18n.Tr("aic.custom.nameTaken", i18n.Args{"name": c.Name}))
	}
	if err := writeCustom(c); err != nil {
		return CustomContext{}, err
	}
	return c, nil
}

// writeCustom writes c to its file, atomically, readable only by the
// user, creating the folder (0700) when missing.
func writeCustom(c CustomContext) error {
	dir, err := customDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	head, err := yaml.Marshal(customHeader{Name: c.Name, AppliesTo: c.AppliesTo})
	if err != nil {
		return fmt.Errorf("generate yaml: %w", err)
	}
	data := "---\n" + string(head) + "---\n" + c.Content
	path := filepath.Join(dir, c.ID+".md")
	if err := fsutil.WriteFileAtomic(path, []byte(data), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// CustomContextUsers lists the names of the registered contexts that use
// the custom context id.
func CustomContextUsers(id string) ([]string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return customUsers(cfg.AIContexts, id), nil
}

// customUsers lists the names of contexts using the custom context id.
func customUsers(contexts []config.AIContext, id string) []string {
	var names []string
	for _, c := range contexts {
		if slices.Contains(c.Models, CustomModelName(id)) {
			names = append(names, c.Name)
		}
	}
	return names
}

// DeleteCustomContext removes the custom context id; it is refused while
// a registered context uses it.
func DeleteCustomContext(id string) error {
	if _, err := FindCustomContext(id); err != nil {
		return err
	}
	users, err := CustomContextUsers(id)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return errors.New(i18n.Tr("aic.custom.inUse", i18n.Args{"names": strings.Join(users, ", ")}))
	}
	dir, err := customDir()
	if err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(dir, id+".md")); err != nil {
		return fmt.Errorf("remove custom context %s: %w", id, err)
	}
	return nil
}

// StandardText returns the text of the catalog standard name as its
// template holds it, placeholders included; custom contexts are not
// served here.
func StandardText(name string) (string, error) {
	for _, m := range models {
		if m.Name == name {
			return readTpl(m.Instruction)
		}
	}
	return "", errors.New(i18n.Tr("aic.unknownStandard", i18n.Args{"name": name}))
}
