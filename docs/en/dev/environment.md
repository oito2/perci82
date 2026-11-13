🌐 [Português](../../pt-br/dev/environment.md) | **English** | 🏠 [Index](../index.md)

---

# Environment

Installs, updates and removes the development tools: prerequisites, languages, AI apps, terminals and editors.

![Environment](../../img/dev-environment.png)
<!-- TODO: capture a screenshot of Environment -->

Each screen of the side list has an **Items** tab, where you choose what to do, and a **Run** tab, which shows the output of the running action. Only one action runs at a time.

Packages come from `apt` (Debian/Ubuntu family) or `dnf` (Fedora). Package installs and removals ask for your password; what goes to your home folder doesn't. Several tools add lines to the startup file of your shell (`~/.bashrc`, `~/.zshrc` or `~/.config/fish/config.fish`, according to `$SHELL`) — open a new terminal to use them. Tools installed in `~/.local/bin` also add `export PATH="$HOME/.local/bin:$PATH"` (`fish_add_path $HOME/.local/bin` in fish) when that folder isn't in `PATH` yet.

## Prerequisites

Each row has its own **Install**, **Update** and **Uninstall** buttons. An item that depends on another shows **Requires: …** and its **Install** stays off until that item is installed. **Uninstall** asks for confirmation, with a warning when the removal takes more than the item itself. Installing and updating need Ubuntu 24.04, Debian 13, Linux Mint 22, LMDE 7, Zorin OS 18, Pop!_OS 24.04 or a later release: older releases lack some of the packages (such as `libfuse2t64`).

| Item | Source | Notes |
|---|---|---|
| Curl, Git, Lsof, CMake, Clang, GNOME Keyring, Flatpak, AppStream | Package of the same name (`apt`/`dnf`) | |
| Flatpak Builder | Package `flatpak-builder` | Requires Flatpak |
| Ninja | Package `ninja-build` | |
| Build Essential | Package `build-essential` (`apt`) or `gcc`, `gcc-c++`, `make` (`dnf`) | |
| OpenSSL | `openssl` plus `libssl-dev` (`apt`) or `openssl-devel` (`dnf`) | Uninstall keeps `openssl` |
| GTK3 Dev Headers | `libgtk-3-dev` / `gtk3-devel` | |
| GTK4 Dev Headers | `libgtk-4-dev` / `gtk4-devel` | |
| WebKitGTK 6.0 Dev Headers | `libwebkitgtk-6.0-dev` / `webkitgtk6.0-devel` | |
| WebKitGTK 4.1 Dev Headers | `libwebkit2gtk-4.1-dev` / `webkit2gtk4.1-devel` | |
| Libxdo Dev | `libxdo-dev` / `libxdo-devel` | |
| Wails3 CLI | `go install` of `wails3` v3.0.0-beta.28 into Go's binary folder | Requires Go ([Languages & SDKs](#languages--sdks)); no password |
| Libsecret Tools | `libsecret-1-0`, `libsecret-1-dev`, `libsecret-tools` (`apt`) or `libsecret`, `libsecret-devel`, `git-credential-libsecret` (`dnf`) | Uninstall keeps the runtime library (`libsecret-1-0`, or `libsecret` on Fedora), which the desktop uses |
| Librsvg Tools | `librsvg2-bin` + `librsvg2-dev` (`apt`) or `librsvg2-tools` + `librsvg2-devel` (`dnf`) | |
| GitHub CLI (gh) | On `apt`, adds GitHub's signed repository (`/etc/apt/sources.list.d/github-cli.list`) and installs `gh`; on `dnf`, package `gh` | Requires Curl; uninstall also removes the repository |
| Docker Engine | `docker.io`, `docker-compose-v2` (`docker-compose` on Debian), `docker-buildx` (`apt`) or `moby-engine`, `docker-compose`, `docker-buildx` (`dnf`) | Enables the `docker` service and adds you to the `docker` group (root-equivalent access); log out and back in afterwards |
| Node.js | The LTS version through nvm v0.40.8 (official script, checked by SHA-256 before it runs) in `~/.nvm` | Requires Curl; no password. Update moves to the newest LTS, carrying over the global npm packages. Uninstall deletes the whole `~/.nvm` (every version and global package, Codex CLI included) and the nvm lines in the shell startup files |
| Flatpak Builder Tools | Packages `pipx`, `python3-aiohttp`, `python3-tomlkit`; clones `flatpak-builder-tools` into `~/tools/flatpak-builder-tools` at a fixed commit and installs `flatpak-node-generator` with `pipx` | Requires Git and Flatpak Builder. Uninstall keeps the three packages |
| Libfuse2 | `libfuse2t64` (`apt`) or `fuse-libs` (`dnf`) | |
| AppImageTool | Release 1.9.1 downloaded from GitHub (SHA-256 checked against the release's digest) into `/usr/local/bin/appimagetool` | Requires Curl |
| Mkcert | `mkcert` plus `libnss3-tools` (`apt`) or `nss-tools` (`dnf`) | |

**Update** upgrades the packages (`apt-get install --only-upgrade` or `dnf upgrade`) or reruns the installer. Before removing packages, Perci82 Dev simulates the removal: if other packages would go with them, nothing is removed and the run lists them.

## Languages & SDKs

Same buttons and rules as [Prerequisites](#prerequisites).

| SDK | Requires | Install | Uninstall |
|---|---|---|---|
| Go SDK | Curl | Newest stable release downloaded from `dl.google.com/go` (SHA-256 checked against `go.dev`) into `/usr/local/go` (password); adds `/usr/local/go/bin` to `PATH` | Removes `/usr/local/go` and its `PATH` line |
| Rust (rustup) | Curl, Build Essential | `rustup-init` 1.29.1 from `static.rust-lang.org` (SHA-256 checked); stable toolchain with the default profile plus `rust-analyzer`, in `~/.cargo` and `~/.rustup`; adds `~/.cargo/bin` to `PATH` | `rustup self uninstall` (removes `~/.cargo`, programs from `cargo install` included, and `~/.rustup`) and the `PATH` line |
| Flutter + Dart SDK | Git | See below | Removes `~/development/flutter` and its `PATH` line; the Android SDK and the Chrome wrapper stay |

**Update** installs the newest Go release (nothing happens when it is already current), runs `rustup update`, or runs `flutter upgrade`.

Installing Flutter:

- Clones the `stable` branch of `flutter/flutter` into `~/development/flutter` and adds its `bin` to `PATH`.
- Installs Flutter's Linux build packages (password); when there is no Java 17 or newer, also JDK 21 (`openjdk-21-jdk`, or `temurin-21-jdk` from Adoptium's repository on Fedora).
- Downloads the Android cmdline-tools (SHA-1 checked against Google's repository manifest) into `~/Android/Sdk/cmdline-tools/latest`, adds `ANDROID_HOME`, `ANDROID_SDK_ROOT` and the SDK folders to the shell startup file, **accepts the Android SDK licenses on your behalf** (terms at https://developer.android.com/studio/terms) and installs `platform-tools`.
- When Chrome is installed only as the Flatpak `com.google.Chrome`, creates the wrapper `~/.local/bin/google-chrome` and sets `CHROME_EXECUTABLE`.
- Ends with `flutter doctor`.

## AI

A checklist: check the apps you want, uncheck the ones to remove, then click **Run**.

| App | Install | Uninstall |
|---|---|---|
| Claude Code CLI | Official script `https://claude.ai/install.sh` | Removes the `claude` command and, when it points there, `~/.local/share/claude` |
| Antigravity CLI | Official script `https://antigravity.google/cli/install.sh` | Removes the `agy` command |
| Codex CLI | `npm install -g @openai/codex@0.162.1` (needs [Node.js](#prerequisites)) | `npm uninstall -g @openai/codex` |
| OpenCode CLI | Official script `https://opencode.ai/install` | Removes the `opencode` command, `~/.opencode/bin` and OpenCode's `PATH` lines in the shell startup files |

None of them asks for your password.

## Terminals

A checklist, like [AI](#ai).

| Terminal | Install | Uninstall |
|---|---|---|
| Kitty | Official installer into `~/.local/kitty.app`, with `kitty` and `kitten` linked in `~/.local/bin` | Removes `~/.local/kitty.app` and the two links |
| Alacritty | Package `alacritty` (password) | Removes the package (password) |
| Black Box | Flatpak `com.raggesilver.BlackBox` from Flathub, in the scope set by `flatpak_scope` in `~/.perci/config.yaml` (system by default) | Uninstalls the Flatpak |
| GNOME Console | Package `gnome-console` (password) | Removes the package (password) |

After each run, every installed terminal gets an **Open in <terminal>** entry in the folder context menu of Nautilus (Scripts submenu, `~/.local/share/nautilus/scripts/`), Nemo (`~/.local/share/nemo/scripts/`) and Dolphin (`~/.local/share/kio/servicemenus/`); removed terminals lose theirs.

Two buttons below the checklist manage the Starship prompt, regardless of the checked items:

- **Apply Starship** installs Starship with its official script into `~/.local/bin`, creates `~/.config/starship.toml` from the `gruvbox-rainbow` preset when that file doesn't exist, and adds its startup line to `~/.bashrc`, `~/.zshrc` and `~/.config/fish/config.fish` (only the files that exist).
- **Remove Starship** (asks for confirmation) removes the binary and those startup lines, and keeps the configuration as `~/.config/starship.toml.perci-bak`.

## IDEs

Same buttons and rules as [Prerequisites](#prerequisites).

| IDE | Install | Uninstall |
|---|---|---|
| Zed Editor | Official script `https://zed.dev/install.sh` into `~/.local` (requires Curl; no password) | Removes `~/.local/zed.app`, `~/.local/bin/zed` and the menu entry; `~/.config/zed` stays |
| VS Code | Adds Microsoft's signed repository and installs `code` (password) | Removes the `code` package and Microsoft's repository and key (one password prompt); `~/.config/Code` and `~/.vscode` stay |
| VSCodium | Adds VSCodium's signed repository and installs `codium` (password) | Removes the `codium` package and VSCodium's repository and key (one password prompt); `~/.config/VSCodium` and `~/.vscode-oss` stay |

**Update** runs the install again.

## Android Studio

Installs from a `.tar.gz` you download yourself from https://developer.android.com/studio. Click **Choose file**, pick the file, then **Install**; **Uninstall** doesn't need a file. There is no **Update**: Android Studio updates itself.

- The file must be an Android Studio archive (every entry under `android-studio/`).
- It is extracted into `/opt/android-studio`, owned by your user; the password is asked to prepare that folder. A previous install is kept as `/opt/android-studio.old` until the extraction succeeds.
- The setup wizard opens in its own window. To get a menu entry, choose **Create Desktop Entry** in its gear (or three-dot) menu.
- **Uninstall** (password, after a confirmation) removes `/opt/android-studio` and `~/.local/share/applications/jetbrains-studio.desktop`.

## Antigravity IDE

Installs from the `.tar.gz` of Antigravity IDE (Standalone), downloaded from https://antigravity.google/download#antigravity-ide. Click **Choose file**, then **Install**, or **Update** for a newer archive; **Uninstall** doesn't need a file.

- **Install** refuses when `/opt/antigravity` already exists: use **Update**.
- The program goes to `/opt/antigravity`, owned by root (password). Its `chrome-sandbox` is SHA-256 checked after copying and gets the setuid permission (`4755`). An update replaces the previous version only once the new one is in place.
- Creates the menu entry `~/.local/share/applications/antigravity.desktop` and the icon `~/.local/share/icons/hicolor/512x512/apps/antigravity.png`.
- **Uninstall** (password, after a confirmation) removes `/opt/antigravity`, the menu entry and the icon.
