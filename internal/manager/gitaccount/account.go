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

// Package gitaccount manages Git hosting accounts: each account owns a
// folder, and every repository under it uses the account's commit identity
// and SSH key through an includeIf rule in the global git configuration.
package gitaccount

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/i18n"
)

// Supported providers.
const (
	ProviderGitHub = "github"
	ProviderGitLab = "gitlab"
)

// DefaultHost returns the public host of provider, or "" for an unknown
// provider.
func DefaultHost(provider string) string {
	switch provider {
	case ProviderGitHub:
		return "github.com"
	case ProviderGitLab:
		return "gitlab.com"
	}
	return ""
}

// KeysURL is the page of acc's provider where SSH keys are registered.
func KeysURL(acc config.GitAccount) string {
	switch acc.Provider {
	case ProviderGitHub:
		return "https://" + acc.Host + "/settings/keys"
	case ProviderGitLab:
		return "https://" + acc.Host + "/-/user_settings/ssh_keys"
	}
	return ""
}

var (
	slugSeparators = regexp.MustCompile(`[^a-z0-9]+`)
	hostPattern    = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)
	// usernamePattern accepts GitHub and GitLab user names: letters,
	// digits, ".", "_" and "-", starting and ending with a letter or digit.
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)
	emailPattern    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

// maxNameLength bounds the display name shown in the menu.
const maxNameLength = 40

// Slug derives an account ID from name: lowercase ASCII letters and digits
// separated by single "-".
func Slug(name string) string {
	return strings.Trim(slugSeparators.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// Normalize trims every text field, lowercases the host, fills in the
// provider's default host when Host is empty, derives ID from Name when
// ID is empty and cleans the folder and key paths.
func Normalize(acc config.GitAccount) config.GitAccount {
	acc.Name = strings.TrimSpace(acc.Name)
	acc.Provider = strings.TrimSpace(acc.Provider)
	acc.Host = strings.ToLower(strings.TrimSpace(acc.Host))
	if acc.Host == "" {
		acc.Host = DefaultHost(acc.Provider)
	}
	acc.Username = strings.TrimSpace(acc.Username)
	acc.GitName = strings.TrimSpace(acc.GitName)
	acc.GitEmail = strings.TrimSpace(acc.GitEmail)
	if acc.ID == "" {
		acc.ID = Slug(acc.Name)
	}
	if acc.Folder != "" {
		acc.Folder = filepath.Clean(acc.Folder)
	}
	if acc.SSHKey != "" {
		acc.SSHKey = filepath.Clean(acc.SSHKey)
	}
	return acc
}

// Validate checks a normalized account. others holds every other
// registered account (not acc itself): the name, ID and folder must not
// clash with any of them. The SSH key is checked by CheckKey.
func Validate(acc config.GitAccount, others []config.GitAccount) error {
	if err := validateFields(acc); err != nil {
		return err
	}
	if err := checkFolder(acc.Folder); err != nil {
		return err
	}
	return checkClash(acc, others)
}

// validateFields checks every field of a normalized account except the
// folder and the SSH key.
func validateFields(acc config.GitAccount) error {
	if acc.Name == "" {
		return i18n.NewError("git.noName")
	}
	if len([]rune(acc.Name)) > maxNameLength {
		return errors.New(i18n.Tr("git.nameTooLong", i18n.Args{"max": maxNameLength}))
	}
	if acc.ID == "" || acc.ID != Slug(acc.ID) {
		return errors.New(i18n.Tr("git.nameNoAlnum", i18n.Args{"name": acc.Name}))
	}
	if DefaultHost(acc.Provider) == "" {
		return errors.New(i18n.Tr("git.badProvider", i18n.Args{"provider": acc.Provider}))
	}
	if !hostPattern.MatchString(acc.Host) {
		return errors.New(i18n.Tr("git.badHost", i18n.Args{"host": acc.Host}))
	}
	if acc.SSHPort < 0 || acc.SSHPort > 65535 {
		return errors.New(i18n.Tr("git.badPort", i18n.Args{"port": acc.SSHPort}))
	}
	if !usernamePattern.MatchString(acc.Username) {
		return errors.New(i18n.Tr("git.badUser", i18n.Args{"user": acc.Username}))
	}
	if acc.GitName == "" || strings.ContainsAny(acc.GitName, "\n\r") {
		return i18n.NewError("git.noGitName")
	}
	if !emailPattern.MatchString(acc.GitEmail) {
		return errors.New(i18n.Tr("git.badEmail", i18n.Args{"email": acc.GitEmail}))
	}
	return nil
}

// checkClash fails when acc's name, ID or folder clashes with one of
// others.
func checkClash(acc config.GitAccount, others []config.GitAccount) error {
	for _, o := range others {
		if o.ID == acc.ID || strings.EqualFold(o.Name, acc.Name) {
			return errors.New(i18n.Tr("git.nameTaken", i18n.Args{"name": o.Name}))
		}
		if foldersOverlap(o.Folder, acc.Folder) {
			return errors.New(i18n.Tr("git.folderOverlap", i18n.Args{"folder": acc.Folder, "name": o.Name, "other": o.Folder}))
		}
	}
	return nil
}

// checkFolder requires an absolute path to an existing directory.
func checkFolder(folder string) error {
	if folder == "" || !filepath.IsAbs(folder) {
		return i18n.NewError("git.noFolder")
	}
	if err := checkGlobFree(folder); err != nil {
		return err
	}
	info, err := os.Stat(folder)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.Tr("git.folderLabel", nil), err)
	}
	if !info.IsDir() {
		return errors.New(i18n.Tr("git.notAFolder", i18n.Args{"path": folder}))
	}
	return nil
}

// foldersOverlap reports whether a and b are the same folder or one
// contains the other, comparing both the paths as given and with symlinks
// resolved.
func foldersOverlap(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if within(a, b) || within(b, a) {
		return true
	}
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	if errA != nil || errB != nil {
		return false
	}
	return within(ra, rb) || within(rb, ra)
}

// within reports whether path is dir itself or lies below it.
// checkGlobFree refuses a folder with a character git reads as a pattern
// in an includeIf gitdir: rule (* ? [ \): the rule would miss the
// account's repositories, or match other folders.
func checkGlobFree(folder string) error {
	if strings.ContainsAny(folder, `*?[\`) {
		return errors.New(i18n.Tr("git.folderGlob", i18n.Args{"path": folder}))
	}
	return nil
}

func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
