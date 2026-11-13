🌐 [Português](../../pt-br/dev/repositories.md) | **English** | 🏠 [Index](../index.md)

---

# Repositories

Git accounts and their repositories. Each account owns a folder: every repository inside it, in any subfolder, uses the account's commit name, e-mail and SSH key — in the terminal too. The side list has **Set Up Account**, **Manage Repositories** and, under **Accounts**, one item per registered account.

![Repositories](../../img/dev-repositories.png)
<!-- TODO: capture a screenshot of an account in Repositories -->

## How an account works

An account is a personal GitHub or GitLab account, on the public service or on a server of your own. Saving it:

- sets up its SSH key (a new ed25519 key at `~/.ssh/perci_<id>`, or an existing one);
- writes `~/.perci/git/<id>.gitconfig` with `user.name`, `user.email` and a `core.sshCommand` that uses only the account's key and Perci82 Dev's own `known_hosts` file;
- adds an `includeIf` rule to the global Git configuration, so that file applies to every repository under the account's folder:

```ini
[includeIf "gitdir:/home/you/dev/github/"]
	path = /home/you/.perci/git/personal.gitconfig
```

- trusts the published host keys of `github.com` and `gitlab.com` (port 22); any other server needs its key confirmed in the account details;
- stores the API token, when given, in the system keyring.

`<id>` comes from the account name: lowercase letters and digits joined by `-`.

## Set Up Account

The **Accounts** tab lists the accounts with **Account**, **Provider**, **Folder** and **SSH key** state. Click an account's name to open its details; the row's icons **Edit** and **Remove** it. **New account** opens the form.

### Account form

| Field | Notes |
|---|---|
| **Account name** | Shows in the Repositories menu. Up to 40 characters, unique |
| **Provider** | GitHub or GitLab |
| **Host** | Empty for the public service (`github.com`, `gitlab.com`); fill in for a server of your own |
| **SSH port** | Only for a server of your own; empty means 22 |
| **User** | The owner of the repositories on the provider |
| **Name in commits** / **E-mail in commits** | The commit identity inside the account's folder |
| **Account folder** | Chosen with **Choose folder**. Must exist and can't contain nor be inside another account's folder |
| **SSH key** | **Generate a new key (ed25519)**, with an optional passphrase; **Use an existing key** (the `.pub` must be next to it); or, when editing, **Keep the current key** |
| **API token (optional)** | Kept in the keyring, never in `config.yaml`. When editing, **Remove the saved token** deletes it |

**Open the SSH keys page** and **Open the tokens page** open the provider's pages for the typed host. The token is only used to list the account's repositories in **New Repository**: without one, only the public repositories show. The form describes the token to create:

- GitHub: a fine-grained token with Repository access = All repositories and only **Metadata: Read-only**.
- GitLab: a personal access token with the `read_api` scope only.

The token is stored with `secret-tool`, which must be installed.

### Account details

The details show the account's settings and its **Public key** with a state badge:

| State | Meaning |
|---|---|
| **ready** | The key has no passphrase |
| **unlocked** | The key has a passphrase and is loaded in the session's SSH agent |
| **locked** | The key has a passphrase and is not loaded in the agent |
| **no SSH agent** | The key has a passphrase and no SSH agent runs in this session |
| **not found** | The key file or its `.pub` is missing |

- **Copy** copies the public key; **Open SSH keys on GitHub/GitLab** opens the page where you add it (as an authentication key).
- **Unlock key** (only when **locked**): type the passphrase and click **Unlock** to load the key into the SSH agent for this session.
- **Server key** (only when the server's key is not confirmed yet): **Check the server key** asks the server for its keys (`ssh-keyscan`) and shows their fingerprints; **Trust these keys** adds them to `~/.perci/git/known_hosts`. Check the fingerprints with the server's administrator first.
- **Test connection** connects over SSH and passes when the server greets the account's **User**. It stays off until the server key is confirmed and the key is usable.

**Remove** forgets the account: its `includeIf` rule, its gitconfig file and its token go away. The repositories in its folder and its SSH key stay on disk; the repositories fall back to the default identity.

### Default identity

The **Default identity** tab sets the **Name** and **E-mail** of commits made outside every account folder (`git config --global user.name` / `user.email`). **Apply the default identity** writes them and, when no `credential.helper` is configured yet, sets one (libsecret when available, otherwise `cache`).

## Account screens

Each item under **Accounts** opens that account's repositories. The header shows the provider, user and folder, with **Reload** and **Set up account** (opens the account's details).

When the account can't reach its server yet — key not found, locked, no SSH agent, or server key not confirmed — a warning says so and the network buttons stay off.

### Repository list

The list holds every Git repository found under the account's folder (hidden folders, `node_modules` and `vendor` are skipped). Opening the screen doesn't reach the network.

| Column | Shows |
|---|---|
| **Repository** | The path relative to the account folder and the `origin` URL |
| **Branch** | The current branch, or **(detached HEAD)** |
| **Last commit** | How long ago; the date and subject on hover |
| **Changes** | State badges and when the repository was last checked |

| Badge | Meaning |
|---|---|
| **up to date** | Nothing to push or pull since the last check |
| **↓ n** / **↑ n** | Commits to pull (as of the last check) / commits not pushed |
| **n modified** / **n new** | Changed tracked files / files outside Git |
| **no remote** | No `origin` remote |
| **HTTPS URL** | `origin` uses HTTPS; with SSH the account's key applies |
| **no upstream** | The current branch doesn't track a remote branch |
| **different identity** | The repository sets its own `user.name` or `user.email` |
| **read error** | The repository couldn't be read |

Repositories whose `origin` belongs to another owner or host don't show; a collapsed note lists them.

### Toolbar and row actions

| Button | What it does |
|---|---|
| **Check for Changes** | `git fetch --prune origin` in every repository with a remote, without changing the files |
| **Pull Changes** | Fetches and moves the current branch forward (fast-forward only). Repositories with uncommitted changes, without an upstream, on a detached HEAD or diverged from the remote are skipped, with the reason in the **Run** tab |
| **New Repository** | Clones a repository or starts a new one inside the account's folder |
| **Fix all URLs** | Only when a repository uses HTTPS: replaces each HTTPS `origin` of the account with the matching SSH URL, after checking the key reaches it |
| **Clone pending** | Only when there are pending repositories (see below) |

Each row also has **Check**, **Pull** and a **+** button that opens the repository's dialog. Long actions run in the **Run** tab.

### New Repository

**How** picks one of three modes:

- **Pick from the account's list** — **Load the account's repositories** lists the account's repositories on the provider (private ones too, with a token); those already cloned are marked and can't be picked.
- **Clone by URL or name** — a name (`perci`) or a URL of one of the account's repositories; an HTTPS URL is cloned over SSH. URLs of another owner or host are refused.
- **Start a local repository** — `git init`; with **Name on the provider (optional)**, the `origin` remote is set too. Create the empty repository on the site before the first push.

**Repository folder** is relative to the account's folder (for example `labs/perci`), must be new or empty and can't be inside another repository.

### Repository dialog

The **+** button opens three tabs:

- **Settings** — the **Contact e-mail** used in `CODE_OF_CONDUCT.md` and `SUPPORT.md`: the account's default and an optional override for this repository only. **Fix identity** (when the repository sets its own identity): **Use the account's identity** removes `user.name` and `user.email` from the repository's own config; commits already made don't change.
- **Remote and SSH** — shows **Already uses SSH**; or **Move to SSH** with **Switch to SSH** for an HTTPS `origin`; or **Set remote** for a repository without `origin` (the name on the provider; the repository doesn't need to exist yet).
- **Files** — community files, compared with what Perci82 Dev generates now:

| File | Notes |
|---|---|
| `.gitignore` | Patterns by project type detected at the root (Go, Flutter, Dart, Moodle, PHP, Node.js, Python, Ruby, Rust, Java, Shell), plus editors and OS; generic patterns when none is detected. **Update** only adds the missing patterns |
| `LICENSE` | GNU GPL v3.0 (default), GNU AGPL v3.0, GNU LGPL v3.0, MIT, Apache 2.0, Mozilla Public License 2.0 or BSD 3-Clause; MIT and BSD 3-Clause also take **Copyright holder** and **Year** |
| `CODE_OF_CONDUCT.md` | With a Portuguese mirror at `docs/pt-br/codigo-de-conduta.md`; needs the contact e-mail |
| `SUPPORT.md` | With a Portuguese mirror at `docs/pt-br/suporte.md`; needs the contact e-mail |
| Issue templates | `.github/ISSUE_TEMPLATE/` on GitHub, `.gitlab/issue_templates/` on GitLab |
| Pull/merge request template | `.github/PULL_REQUEST_TEMPLATE.md` on GitHub, `.gitlab/merge_request_templates/Default.md` on GitLab |
| `.github/FUNDING.yml` | **coming soon** |

Each row offers **Create**, **Update** or **Remove** by its state. **Update** replaces the file and keeps the previous version as `<file>.perci-bak`, with its permissions. A file or folder that is a link to outside the repository is never written through: that file is skipped and the reason shown.

### Pending repositories

Repositories imported in **Manage Repositories** and not cloned yet show under **Pending**. Add the account's public key on the server and use **Test connection** first. **Clone pending** tests the connection and clones them into their folders; **Discard** takes one off the list without deleting anything.

## Manage Repositories

Moves accounts and repositories to another machine through a YAML file. This file is separate from Docker's export.

### Export

The **Export** tab lists each account with its repositories to check. The file keeps each account's name, provider, host, SSH port, user, commit identity, folder and contact e-mails, and for each repository its folder and SSH URL. SSH keys, tokens and the repositories' files are not exported. Repositories without a remote are left out (there is nowhere to clone them from); pending repositories are included.

**Export…** asks where to save (default name `perci-repositorios.yaml`); the file is readable only by you. It is written as format version 2 when it holds a contact e-mail, otherwise as version 1.

```yaml
version: 2
accounts:
  - name: Personal
    provider: github
    host: github.com
    username: your-user
    git_name: Your Name
    git_email: you@example.com
    folder: /home/you/dev/github
    contact_email: you@example.com
    repos:
      - path: labs/project
        url: git@github.com:your-user/project.git
```

### Import

In the **Import** tab, **Choose file** reads an export file and lists what it would do:

- **new account** — gets its folder (created when missing; **Change folder** picks another; a folder in the file outside your home folder, or the home folder itself, makes the account unavailable until you pick one), an SSH key without a passphrase at `~/.ssh/perci_<id>` (an existing key there is reused), the built-in server keys and its `includeIf` rule.
- **already set up — merges with …** — same user on the same server: the account keeps its data and only gains the repositories.
- Repositories already pending or already cloned are marked; one whose folder holds something else is skipped, with the reason.

**Import** adds the checked repositories as pending on each account's screen; nothing is cloned yet.

## What is stored where

| What | Where |
|---|---|
| Accounts, pending repositories and contact e-mails | `~/.perci/config.yaml`, key `git_accounts` |
| Each account's commit identity and SSH command | `~/.perci/git/<id>.gitconfig` |
| The `includeIf` rules and the default identity | Global Git configuration (`~/.gitconfig`) |
| Confirmed server keys | `~/.perci/git/known_hosts` |
| Generated SSH keys | `~/.ssh/perci_<id>` and `~/.ssh/perci_<id>.pub` |
| API tokens | System keyring (`secret-tool`, service `perci-git`) |

Uninstalling Perci82 Dev can first save the accounts and repositories to an export file; when it also removes the settings, it forgets the accounts the same way as **Remove**. See [Uninstallation](uninstallation.md).
