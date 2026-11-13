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

package gitaccount

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/manager/repofiles"
	"github.com/oito2/perci82/internal/ui"
)

// TransferVersion is the newest version of the repositories export file
// format this build reads. Version 2 added the contact e-mails; a file
// without them is still written as version 1, which older builds read.
const TransferVersion = 2

// TransferFile is the repositories export file: accounts with their
// repositories. It never holds an SSH key or an API token.
type TransferFile struct {
	Version  int               `yaml:"version" json:"version"`
	Accounts []TransferAccount `yaml:"accounts" json:"accounts"`
}

// TransferAccount is one account of a TransferFile.
type TransferAccount struct {
	Name     string `yaml:"name" json:"name"`
	Provider string `yaml:"provider" json:"provider"`
	Host     string `yaml:"host" json:"host"`
	SSHPort  int    `yaml:"ssh_port,omitempty" json:"sshPort"`
	Username string `yaml:"username" json:"username"`
	GitName  string `yaml:"git_name" json:"gitName"`
	GitEmail string `yaml:"git_email" json:"gitEmail"`
	Folder   string `yaml:"folder" json:"folder"`
	// ContactEmail is the account's contact e-mail for generated files.
	ContactEmail string         `yaml:"contact_email,omitempty" json:"contactEmail"`
	Repos        []TransferRepo `yaml:"repos,omitempty" json:"repos"`
}

// TransferRepo is one repository of a TransferAccount: its folder
// relative to the account folder and its SSH URL.
type TransferRepo struct {
	Path string `yaml:"path" json:"path"`
	URL  string `yaml:"url" json:"url"`
	// ContactEmail is the repository's own contact e-mail, overriding
	// the account's.
	ContactEmail string `yaml:"contact_email,omitempty" json:"contactEmail"`
}

// TransferPick names one account of a TransferFile (by name) and the
// paths of its repositories to export or import.
type TransferPick struct {
	Account string   `json:"account"`
	Repos   []string `json:"repos"`
}

// account returns t as a normalized account, without key or pending
// repositories.
func (t TransferAccount) account() config.GitAccount {
	acc := Normalize(config.GitAccount{
		Name: t.Name, Provider: t.Provider, Host: t.Host, SSHPort: t.SSHPort,
		Username: t.Username, GitName: t.GitName, GitEmail: t.GitEmail, Folder: t.Folder,
	})
	if acc.SSHPort == 22 {
		acc.SSHPort = 0
	}
	return acc
}

// ErrNothingToExport is returned when an export would hold no account.
var ErrNothingToExport = i18n.NewError("git.nothingToExport")

// Exportable lists what accounts can export: each account's fields and
// its repositories with an origin of the account (as SSH URL) plus its
// pending ones. Repositories without a remote can't be cloned elsewhere
// and are returned in skipped, as "<account>: <path>".
func Exportable(ctx context.Context, exe *executor.Executor, accounts []config.GitAccount) (TransferFile, []string, error) {
	file := TransferFile{Version: TransferVersion, Accounts: []TransferAccount{}}
	skipped := []string{}
	for _, acc := range accounts {
		t := TransferAccount{
			Name: acc.Name, Provider: acc.Provider, Host: acc.Host, SSHPort: acc.SSHPort,
			Username: acc.Username, GitName: acc.GitName, GitEmail: acc.GitEmail, Folder: acc.Folder,
			ContactEmail: acc.ContactEmail, Repos: []TransferRepo{},
		}
		listed, err := ListRepos(ctx, exe, acc)
		if err != nil {
			return TransferFile{}, nil, fmt.Errorf("%s: %w", i18n.Tr("git.accountLabel", i18n.Args{"name": acc.Name}), err)
		}
		seen := map[string]bool{}
		for _, st := range listed.Repos {
			r, ok := ParseRemote(st.Remote)
			if !ok || st.Path == "." {
				skipped = append(skipped, acc.Name+": "+st.Path)
				continue
			}
			path := filepath.ToSlash(st.Path)
			seen[path] = true
			_, own := ContactEmail(acc, path)
			t.Repos = append(t.Repos, TransferRepo{Path: path, URL: SSHURL(acc, r.Owner, r.Repo), ContactEmail: own})
		}
		for _, p := range acc.Pending {
			if path := filepath.ToSlash(p.Path); !seen[path] {
				seen[path] = true
				_, own := ContactEmail(acc, path)
				t.Repos = append(t.Repos, TransferRepo{Path: path, URL: p.URL, ContactEmail: own})
			}
		}
		file.Accounts = append(file.Accounts, t)
	}
	file.Version = file.formatVersion()
	return file, skipped, nil
}

// Filter returns the part of f that picks names, in f's order. A name
// that is not in f is an error.
func (f TransferFile) Filter(picks []TransferPick) (TransferFile, error) {
	out := TransferFile{Version: f.Version, Accounts: []TransferAccount{}}
	for _, pick := range picks {
		i := f.accountIndex(pick.Account)
		if i < 0 {
			return TransferFile{}, errors.New(i18n.Tr("git.accountNotInFile", i18n.Args{"name": pick.Account}))
		}
		for _, path := range pick.Repos {
			if f.Accounts[i].repoIndex(path) < 0 {
				return TransferFile{}, errors.New(i18n.Tr("git.repoNotInAccount", i18n.Args{"path": path, "name": pick.Account}))
			}
		}
	}
	for _, t := range f.Accounts {
		var pick *TransferPick
		for i := range picks {
			if picks[i].Account == t.Name {
				pick = &picks[i]
				break
			}
		}
		if pick == nil {
			continue
		}
		part := t
		part.Repos = []TransferRepo{}
		for _, r := range t.Repos {
			for _, path := range pick.Repos {
				if path == r.Path {
					part.Repos = append(part.Repos, r)
					break
				}
			}
		}
		out.Accounts = append(out.Accounts, part)
	}
	return out, nil
}

// accountIndex returns the index of the account named name, or -1.
func (f TransferFile) accountIndex(name string) int {
	for i, t := range f.Accounts {
		if t.Name == name {
			return i
		}
	}
	return -1
}

// repoIndex returns the index of the repository at path, or -1.
func (t TransferAccount) repoIndex(path string) int {
	for i, r := range t.Repos {
		if r.Path == path {
			return i
		}
	}
	return -1
}

// WriteTransfer writes f to path as YAML, atomically, readable only by the
// user.
func WriteTransfer(path string, f TransferFile) error {
	if len(f.Accounts) == 0 {
		return ErrNothingToExport
	}
	f.Version = f.formatVersion()
	data, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("generate yaml: %w", err)
	}
	if err := fsutil.WriteFileAtomic(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// formatVersion returns the oldest file format version that holds f: 2
// when it has a contact e-mail, otherwise 1.
func (f TransferFile) formatVersion() int {
	for _, t := range f.Accounts {
		if t.ContactEmail != "" {
			return 2
		}
		for _, r := range t.Repos {
			if r.ContactEmail != "" {
				return 2
			}
		}
	}
	return 1
}

// LoadTransfer reads and validates the export file at path. Unknown keys
// are refused, so a file of another kind (such as the Docker export) is
// not mistaken for one.
func LoadTransfer(path string) (TransferFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TransferFile{}, fmt.Errorf("read %s: %w", path, err)
	}
	var f TransferFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		return TransferFile{}, fmt.Errorf("%s: %w", i18n.Tr("git.notTransferFile", i18n.Args{"path": path}), err)
	}
	if err := f.Validate(); err != nil {
		return TransferFile{}, fmt.Errorf("%s: %w", i18n.Tr("git.invalidFile", nil), err)
	}
	return f, nil
}

// Validate checks everything an import would apply: the format version,
// every account's fields (its folder only needs to be absolute), unique
// names, users and folders, and every repository's path and URL. Nothing
// in the file is trusted to be well-formed.
func (f TransferFile) Validate() error {
	switch {
	case f.Version == 0:
		return i18n.NewError("git.noVersion")
	case f.Version > TransferVersion:
		return errors.New(i18n.Tr("git.newerVersion", i18n.Args{"version": f.Version, "supported": TransferVersion}))
	}
	var accounts []config.GitAccount
	for _, t := range f.Accounts {
		acc := t.account()
		if err := validateFields(acc); err != nil {
			return fmt.Errorf("%s: %w", i18n.Tr("git.accountLabel", i18n.Args{"name": t.Name}), err)
		}
		if !filepath.IsAbs(acc.Folder) {
			return errors.New(i18n.Tr("git.folderNotAbs", i18n.Args{"name": t.Name}))
		}
		if err := checkClash(acc, accounts); err != nil {
			return fmt.Errorf("%s: %w", i18n.Tr("git.accountLabel", i18n.Args{"name": t.Name}), err)
		}
		if err := checkOptionalEmail(t.ContactEmail); err != nil {
			return fmt.Errorf("%s: %w", i18n.Tr("git.accountLabel", i18n.Args{"name": t.Name}), err)
		}
		for _, o := range accounts {
			if sameUser(o, acc) {
				return errors.New(i18n.Tr("git.userTwice", i18n.Args{"name": t.Name, "user": acc.Username, "host": acc.Host}))
			}
		}
		accounts = append(accounts, acc)

		paths := map[string]bool{}
		for _, r := range t.Repos {
			if err := checkRepoPath(r.Path); err != nil {
				return fmt.Errorf("%s: %w", i18n.Tr("git.accountLabel", i18n.Args{"name": t.Name}), err)
			}
			if paths[r.Path] {
				return errors.New(i18n.Tr("git.repoTwice", i18n.Args{"name": t.Name, "path": r.Path}))
			}
			paths[r.Path] = true
			if ClassifyRemote(acc, r.URL) != RemoteOK {
				return errors.New(i18n.Tr("git.notSSHURL", i18n.Args{"name": t.Name, "url": RedactURL(r.URL), "user": acc.Username, "host": acc.Host}))
			}
			if err := checkOptionalEmail(r.ContactEmail); err != nil {
				return fmt.Errorf("%s: %w", i18n.Tr("git.accountRepoLabel", i18n.Args{"name": t.Name, "path": r.Path}), err)
			}
		}
	}
	return nil
}

// checkRepoPath requires a clean relative path below the account folder,
// written with "/".
func checkRepoPath(path string) error {
	if path == "" || path == "." || !filepath.IsLocal(filepath.FromSlash(path)) || filepath.ToSlash(filepath.Clean(path)) != path {
		return errors.New(i18n.Tr("git.badRepoPath", i18n.Args{"path": path}))
	}
	return nil
}

// checkOptionalEmail accepts an empty e-mail or a valid contact address
// written without surrounding spaces.
func checkOptionalEmail(email string) error {
	if email == "" {
		return nil
	}
	checked, err := repofiles.CheckContactEmail(email)
	if err == nil && checked != email {
		err = errors.New(i18n.Tr("git.contactSpaces", i18n.Args{"email": email}))
	}
	return err
}

// sameUser reports whether a and b are the same user on the same server.
func sameUser(a, b config.GitAccount) bool {
	return a.Provider == b.Provider && a.Host == b.Host && strings.EqualFold(a.Username, b.Username)
}

// Repository states of an import (ImportRepo.State).
const (
	ImportNew      = "new"      // becomes a pending repository
	ImportPending  = "pending"  // already pending on this machine
	ImportPresent  = "present"  // already cloned in its folder
	ImportConflict = "conflict" // its folder holds something else; skipped
)

// ImportAccount is one account of an export file checked against this
// machine. ExistingID is the registered account it merges into (same user
// on the same server); Folder is where its repositories go; Reason says
// why it can't be imported (empty when it can).
type ImportAccount struct {
	TransferAccount
	ExistingID   string       `json:"existingId"`
	ExistingName string       `json:"existingName"`
	Reason       string       `json:"reason"`
	States       []ImportRepo `json:"states"`
}

// ImportRepo is one repository of an ImportAccount, with its ImportNew,
// ImportPending, ImportPresent or ImportConflict state and, for a
// conflict, the reason.
type ImportRepo struct {
	Path   string `json:"path"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}

// PlanImport checks every account of f against accounts, the ones
// registered here. folders replaces, by account name, the folder of a
// new account; an account that merges into a registered one keeps that
// account's folder.
func PlanImport(ctx context.Context, exe *executor.Executor, f TransferFile, accounts []config.GitAccount, folders map[string]string) []ImportAccount {
	plans := make([]ImportAccount, 0, len(f.Accounts))
	for _, t := range f.Accounts {
		p := ImportAccount{TransferAccount: t, States: []ImportRepo{}}
		acc := t.account()
		var existing *config.GitAccount
		for i := range accounts {
			if sameUser(accounts[i], acc) {
				existing = &accounts[i]
				break
			}
		}
		if existing != nil {
			p.ExistingID, p.ExistingName = existing.ID, existing.Name
			acc = *existing
		} else {
			fromFile := true
			if folder := folders[t.Name]; folder != "" {
				acc.Folder = filepath.Clean(folder)
				fromFile = false
			}
			if err := checkNewAccount(acc, accounts); err != nil {
				p.Reason = err.Error()
			} else if fromFile && folderTooWide(acc.Folder) {
				p.Reason = i18n.Tr("git.folderTooWide", i18n.Args{"path": acc.Folder})
			}
		}
		p.Folder = acc.Folder
		for _, r := range t.Repos {
			p.States = append(p.States, repoImportState(ctx, exe, acc, r))
		}
		plans = append(plans, p)
	}
	return plans
}

// checkNewAccount checks a new account of an import: its folder must be
// absolute and, when it exists, a folder; nothing may clash with the
// registered accounts.
func checkNewAccount(acc config.GitAccount, accounts []config.GitAccount) error {
	if !filepath.IsAbs(acc.Folder) {
		return i18n.NewError("git.mustBeAbs")
	}
	if err := checkGlobFree(acc.Folder); err != nil {
		return err
	}
	if info, err := os.Stat(acc.Folder); err == nil && !info.IsDir() {
		return errors.New(i18n.Tr("git.notAFolder", i18n.Args{"path": acc.Folder}))
	}
	return checkClash(acc, accounts)
}

// folderTooWide reports whether folder, as written in an import file, is
// outside the home folder or is the home folder itself: its includeIf rule
// would give the account's identity and key to repositories that are not
// the account's. A folder chosen in the file picker is not checked.
func folderTooWide(folder string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return true
	}
	home = filepath.Clean(home)
	return folder == home || !within(folder, home)
}

// repoImportState tells what importing r into acc would do.
func repoImportState(ctx context.Context, exe *executor.Executor, acc config.GitAccount, r TransferRepo) ImportRepo {
	st := ImportRepo{Path: r.Path, State: ImportNew}
	for _, p := range acc.Pending {
		if p.Path == r.Path {
			st.State = ImportPending
			return st
		}
	}
	dest := filepath.Join(acc.Folder, filepath.FromSlash(r.Path))
	if _, err := os.Lstat(filepath.Join(dest, ".git")); err == nil {
		if sameRepo(ctx, exe, dest, r.URL) {
			st.State = ImportPresent
		} else {
			st.State, st.Reason = ImportConflict, i18n.Tr("git.otherRepo", nil)
		}
		return st
	}
	if _, err := DestDir(acc, r.Path); err != nil {
		st.State, st.Reason = ImportConflict, err.Error()
	}
	return st
}

// sameRepo reports whether the origin of the repository at dir is the
// repository url names, whatever the URL form.
func sameRepo(ctx context.Context, exe *executor.Executor, dir, url string) bool {
	local, err := localConfig(ctx, exe, dir)
	if err != nil {
		return false
	}
	a, okA := ParseRemote(local["remote.origin.url"])
	b, okB := ParseRemote(url)
	return okA && okB && a.Host == b.Host &&
		strings.EqualFold(a.Owner, b.Owner) && strings.EqualFold(a.Repo, b.Repo)
}

// Import applies the accounts and repositories of f that picks names:
// an account of a user already registered here keeps its data and only
// gains the new repositories as pending; a new account gets its folder,
// an SSH key (~/.ssh/perci_<id>, reused when it already exists,
// otherwise generated without passphrase), the built-in server keys, its
// gitconfig and includeIf rule, and its repositories as pending. folders
// replaces, by account name, the folder of a new account. Repositories
// are not cloned here (ClonePending). Accounts or repositories that
// can't be imported are skipped with a warning, and then Import returns
// executor.ErrCompletedWithWarnings.
func Import(ctx context.Context, exe *executor.Executor, stdout io.Writer, f TransferFile, picks []TransferPick, folders map[string]string) error {
	if err := f.Validate(); err != nil {
		return fmt.Errorf("%s: %w", i18n.Tr("git.invalidFile", nil), err)
	}
	part, err := f.Filter(picks)
	if err != nil {
		return err
	}
	if len(part.Accounts) == 0 {
		return i18n.NewError("git.nothingToImport")
	}

	skipped := 0
	for _, t := range part.Accounts {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		plan := PlanImport(ctx, exe, TransferFile{Version: part.Version, Accounts: []TransferAccount{t}}, cfg.GitAccounts, folders)[0]
		if plan.Reason != "" {
			ui.Warning(stdout, i18n.Tr("git.run.notImported", i18n.Args{"name": t.Name, "reason": plan.Reason}))
			skipped++
			continue
		}
		// The account the repositories go to: their URLs are written in its
		// SSH form.
		target := t.account()
		for _, a := range cfg.GitAccounts {
			if a.ID == plan.ExistingID {
				target = a
			}
		}
		var pending []config.PendingRepo
		emails := map[string]string{}
		for i, st := range plan.States {
			if st.State != ImportConflict && t.Repos[i].ContactEmail != "" {
				emails[st.Path] = t.Repos[i].ContactEmail
			}
			switch st.State {
			case ImportNew:
				pending = append(pending, config.PendingRepo{Path: st.Path, URL: canonicalRemote(target, t.Repos[i].URL)})
			case ImportPending:
				ui.Info(stdout, i18n.Tr("git.run.alreadyPending", i18n.Args{"name": t.Name, "path": st.Path}))
			case ImportPresent:
				ui.Info(stdout, i18n.Tr("git.run.alreadyCloned", i18n.Args{"name": t.Name, "path": st.Path}))
			case ImportConflict:
				ui.Warning(stdout, i18n.Tr("git.run.repoSkipped", i18n.Args{"name": t.Name, "path": st.Path, "reason": st.Reason}))
				skipped++
			}
		}

		if plan.ExistingID != "" {
			if err := addPending(plan.ExistingID, pending); err != nil {
				return err
			}
			if err := mergeContactEmails(plan.ExistingID, t.ContactEmail, emails); err != nil {
				return err
			}
			ui.Success(stdout, i18n.Tr("git.run.pendingAdded", i18n.Args{"name": t.Name, "existing": plan.ExistingName, "count": len(pending)}))
			continue
		}

		acc := t.account()
		acc.Folder = plan.Folder
		acc.Pending = pending
		acc.ContactEmail = t.ContactEmail
		for _, st := range plan.States {
			if email := emails[st.Path]; email != "" {
				acc.RepoSettings = append(acc.RepoSettings, config.RepoSettings{Path: st.Path, ContactEmail: email})
			}
		}
		if err := createAccount(ctx, exe, stdout, acc, cfg.GitAccounts); err != nil {
			return fmt.Errorf("%s: %w", i18n.Tr("git.accountLabel", i18n.Args{"name": t.Name}), err)
		}
	}

	ui.Info(stdout, i18n.Tr("git.run.importNext", nil))
	if skipped > 0 {
		return executor.ErrCompletedWithWarnings
	}
	ui.Success(stdout, i18n.Tr("git.run.imported", nil))
	return nil
}

// createAccount registers acc, a new account of an import: creates its
// folder, sets up its SSH key, trusts the built-in server keys, writes
// its gitconfig and includeIf rule and saves it. A key generated here is
// removed again when a later step fails.
func createAccount(ctx context.Context, exe *executor.Executor, stdout io.Writer, acc config.GitAccount, others []config.GitAccount) error {
	if err := os.MkdirAll(acc.Folder, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", acc.Folder, err)
	}
	if err := Validate(acc, others); err != nil {
		return err
	}
	path, err := DefaultKeyPath(acc.ID)
	if err != nil {
		return err
	}
	generated := false
	if CheckKey(path) == nil {
		ui.Info(stdout, i18n.Tr("git.run.existingKey", i18n.Args{"path": path}))
	} else {
		if err := GenerateKey(ctx, exe, Askpass{}, path, acc.GitEmail, ""); err != nil {
			return err
		}
		generated = true
	}
	acc.SSHKey, acc.KeyGenerated = path, generated

	err = TrustPinnedHostKeys(acc)
	if err == nil {
		err = Apply(ctx, exe, stdout, nil, acc)
		if err == nil {
			err = config.Update(func(c *config.Config) error {
				c.GitAccounts = append(c.GitAccounts, acc)
				return nil
			})
			if err != nil {
				UndoApply(ctx, exe, nil, acc)
			}
		}
	}
	if err != nil {
		if generated {
			_ = os.Remove(path)
			_ = os.Remove(path + ".pub")
		}
		return err
	}

	ui.Success(stdout, i18n.Tr("git.run.accountCreated", i18n.Args{"name": acc.Name, "folder": acc.Folder, "count": len(acc.Pending)}))
	if pub, err := PublicKey(path); err == nil {
		ui.Info(stdout, i18n.Tr("git.run.publicKey", i18n.Args{"name": acc.Name, "url": KeysURL(acc)}))
		_, _ = fmt.Fprintln(stdout, pub)
	}
	if !IsPinnedHost(acc) {
		ui.Warning(stdout, i18n.Tr("git.run.confirmHost", i18n.Args{"host": acc.Host}))
	}
	return nil
}

// addPending appends repos to the pending repositories of the account id.
func addPending(id string, repos []config.PendingRepo) error {
	if len(repos) == 0 {
		return nil
	}
	return config.Update(func(c *config.Config) error {
		for i := range c.GitAccounts {
			if c.GitAccounts[i].ID == id {
				c.GitAccounts[i].Pending = append(c.GitAccounts[i].Pending, repos...)
				return nil
			}
		}
		return errors.New(i18n.Tr("git.accountNotFound", i18n.Args{"id": id}))
	})
}

// mergeContactEmails gives the account id the contact e-mails of an
// import without replacing any it already has: account becomes the
// account's address when it has none, and each repository (by path) of
// repos gets its address when it has no override.
func mergeContactEmails(id, account string, repos map[string]string) error {
	if account == "" && len(repos) == 0 {
		return nil
	}
	return config.Update(func(c *config.Config) error {
		for i := range c.GitAccounts {
			acc := &c.GitAccounts[i]
			if acc.ID != id {
				continue
			}
			if acc.ContactEmail == "" {
				acc.ContactEmail = account
			}
			paths := make([]string, 0, len(repos))
			for path := range repos {
				paths = append(paths, path)
			}
			slices.Sort(paths)
			for _, path := range paths {
				if _, own := ContactEmail(*acc, path); own == "" {
					acc.RepoSettings = setRepoEmail(acc.RepoSettings, path, repos[path])
				}
			}
			return nil
		}
		return errors.New(i18n.Tr("git.accountNotFound", i18n.Args{"id": id}))
	})
}

// DiscardPending removes the pending repository at path from the account
// id.
func DiscardPending(id, path string) error {
	return config.Update(func(c *config.Config) error {
		for i := range c.GitAccounts {
			if c.GitAccounts[i].ID != id {
				continue
			}
			pending := c.GitAccounts[i].Pending
			for j, p := range pending {
				if p.Path == path {
					c.GitAccounts[i].Pending = append(pending[:j:j], pending[j+1:]...)
					return nil
				}
			}
			return errors.New(i18n.Tr("git.notPending", i18n.Args{"path": path}))
		}
		return errors.New(i18n.Tr("git.accountNotFound", i18n.Args{"id": id}))
	})
}

// ClonePending clones every pending repository of acc into its folder, one
// at a time, and removes from the pending list the ones cloned (or found
// already cloned). It fails when any of them could not be cloned.
func ClonePending(ctx context.Context, exe *executor.Executor, stdout io.Writer, acc config.GitAccount) error {
	if len(acc.Pending) == 0 {
		return i18n.NewError("git.noPending")
	}
	var done []string
	failed := 0
	for i, p := range acc.Pending {
		ui.Step(stdout, i+1, len(acc.Pending), p.Path)
		dest := filepath.Join(acc.Folder, filepath.FromSlash(p.Path))
		if _, err := os.Lstat(filepath.Join(dest, ".git")); err == nil && sameRepo(ctx, exe, dest, p.URL) {
			ui.Info(stdout, i18n.Tr("git.run.cloneDone", i18n.Args{"path": p.Path}))
			done = append(done, p.Path)
			continue
		}
		err := checkRepoPath(p.Path)
		if err == nil && ClassifyRemote(acc, p.URL) != RemoteOK {
			err = errors.New(i18n.Tr("git.notAccountSSH", i18n.Args{"url": RedactURL(p.URL)}))
		}
		if err == nil {
			dest, err = DestDir(acc, p.Path)
		}
		if err == nil {
			err = Clone(ctx, exe, stdout, acc, canonicalRemote(acc, p.URL), dest)
		}
		if err != nil {
			ui.Err(stdout, "✗ "+p.Path+": "+err.Error())
			failed++
			continue
		}
		done = append(done, p.Path)
	}

	if len(done) > 0 {
		if err := config.Update(func(c *config.Config) error {
			for i := range c.GitAccounts {
				if c.GitAccounts[i].ID != acc.ID {
					continue
				}
				kept := []config.PendingRepo{}
				for _, p := range c.GitAccounts[i].Pending {
					if !slices.Contains(done, p.Path) {
						kept = append(kept, p)
					}
				}
				c.GitAccounts[i].Pending = kept
			}
			return nil
		}); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
	}
	if failed > 0 {
		return errors.New(i18n.Tr("git.cloneFailed", i18n.Args{"count": failed}))
	}
	return nil
}
