🌐 [Português](../../pt-br/dev/uninstallation.md) | **English** | 🏠 [Index](../index.md)

---

# Uninstalling Perci82 Dev

## What installing puts on the system

The package `perci82-dev` holds the binary, the menu entry, the icons and the license; the package manager removes them. The rest stays until removed as below.

| Item | Path |
|---|---|
| Binary | `/usr/bin/perci82-dev` |
| Application menu entry | `/usr/share/applications/io.github.oito2.perci82.dev.desktop` |
| Icons | `/usr/share/icons/hicolor/<size>x<size>/apps/io.github.oito2.perci82.dev.png` (16, 22, 24, 32, 48, 64, 128, 256 and 512), `/usr/share/icons/hicolor/scalable/apps/io.github.oito2.perci82.dev.svg`, `/usr/share/icons/hicolor/symbolic/apps/io.github.oito2.perci82.dev-symbolic.svg` |
| License | `/usr/share/doc/perci82-dev/copyright` (`.deb`) or `/usr/share/licenses/perci82-dev/LICENSE` (`.rpm`) |
| Package repository | Shared with Perci82 System. `apt`: `/etc/apt/sources.list.d/perci82.sources` and `/usr/share/keyrings/perci82-archive-keyring.asc`; `dnf`: `/etc/yum.repos.d/perci82.repo`, `/etc/pki/rpm-gpg/RPM-GPG-KEY-perci82` and the key imported into rpm (`gpg-pubkey-3c2a545f`) |
| Login autostart | `~/.config/autostart/io.github.oito2.perci82.dev.desktop` — only while the system tray is on ([Settings](settings.md#system-tray)) |
| Settings | `~/.perci/config.yaml` — shared with Perci82 System; Perci82 Dev's keys are listed in [Settings](settings.md#where-the-settings-are-saved) |
| Data | `~/.perci/appstack/` (Docker stack files) and `~/.perci/git/` (the Git accounts' gitconfig files) |

## From the app

1. Open **Overview** and click **Uninstall Perci82 Dev** in the **Uninstall** section.
2. In the confirmation window, check what you want (every option starts unchecked):
   - **Backups before removing** — **Docker (containers and development folders)**, **Repositories (Git accounts)** and **AI contexts (registered folders)**. Each backup is saved to its own file, in the format that the matching Export/Import screen reads. The save windows open one after the other once you confirm; if you cancel one, nothing is removed. A failed backup also stops the uninstall before anything is removed.
   - **Also remove** — **The Docker containers (Nginx, MariaDB, Selenium and apps)**: the containers and each app's command in `~/.local/bin` are removed; the data on disk under the workspace is kept.
   - **Also remove** — **Perci82 Dev's settings**: its keys in `~/.perci/config.yaml`, the folder `~/.perci/git` and, only when the Docker containers are removed too (they still use it), `~/.perci/appstack` — except `~/.perci/appstack/backups`, the MariaDB backups, which are always kept —, the `includeIf` rules of the Git accounts in `~/.gitconfig` and the accounts' tokens in the keyring.
3. Click **Uninstall** and type your password if asked. The **Run** tab shows the progress.
4. When it finishes, the **Cleanup finished** window shows the command that removes the package, with **Copy**. Copy it, click **Close the app** and run the command in a terminal:

   ```bash
   sudo apt remove perci82-dev   # apt
   sudo dnf remove perci82-dev   # dnf
   ```

The login autostart is removed. The `~/.perci` folder itself goes only when neither Perci82 System nor the old Perci is installed and nothing else is left in it. SSH keys, repositories, the custom AI contexts (`~/.perci/user-contexts`) and Perci82 System's settings are always kept.

## By hand

```bash
sudo apt remove perci82-dev   # or: sudo dnf remove perci82-dev
rm -f ~/.config/autostart/io.github.oito2.perci82.dev.desktop
```

To remove the settings, delete the keys `workspace_path`, `docker`, `git_accounts`, `agent_skills`, `mcp_servers`, `mcp_binaries`, `moodle_plugins`, `ai_contexts`, `ai_context_categories`, `tray_enabled`, `gui_theme`, `language` and `project_notes_path` from `~/.perci/config.yaml`, and the folders `~/.perci/appstack` and `~/.perci/git`. Delete the whole `~/.perci` folder only when Perci82 System isn't installed: it keeps its own settings there.

## Removing the package repository

Once neither Perci82 app is installed, remove the repository and its key:

```bash
# apt
sudo rm -f /etc/apt/sources.list.d/perci82.sources /usr/share/keyrings/perci82-archive-keyring.asc
sudo apt update
# dnf
sudo rm -f /etc/yum.repos.d/perci82.repo /etc/pki/rpm-gpg/RPM-GPG-KEY-perci82
sudo rpm -e gpg-pubkey-3c2a545f
```

## What the app's actions leave behind

Uninstalling removes the app, not what its screens installed or created. Undo those first if you want them gone:

| Screen | Leaves | How to remove |
|---|---|---|
| [Environment](environment.md) | Packages, SDKs, IDEs, terminals and AI apps | Remove them on their screens before uninstalling |
| [Docker](docker.md) | Containers (unless removed by the uninstall), the `perci-php*` images, the data under `<workspace>/localhost/`, the mkcert certificate authority | **Manage Containers** removes containers; images and data with `docker image rm` and by hand |
| [Docker → Development Folders](docker.md) | Mount units in `/etc/systemd/system/` | Remove each folder on that screen before uninstalling |
| [Repositories](repositories.md) | SSH keys and cloned repositories | By hand; the uninstall never removes them |
| [AI Contexts](ai-contexts.md) | The context files in each project and the project notes folder | By hand; the uninstall never touches them |
| [Skills & MCPs](skills-mcps.md) | Installed skills, MCP registrations in the agents, and `~/.local/bin/build82` / `~/.local/bin/sync82` | **Remove** on each row before uninstalling — after the settings are removed, Perci82 Dev no longer knows which of these files it installed |
