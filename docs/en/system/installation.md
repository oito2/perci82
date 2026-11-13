🌐 [Português](../../pt-br/system/installation.md) | **English** | 🏠 [Index](../index.md)

---

# Installing Perci82 System

Perci82 System is released for **Linux on amd64** only: it is built with Wails v3, which uses cgo with GTK 4 and WebKitGTK 6.0, so each platform needs a native build.

## Requirements

| OS | Status | Distributions | Runtime dependencies |
|---|---|---|---|
| Linux (amd64) | Supported | Distributions with `apt` (Debian, Ubuntu, Linux Mint, ZorinOS, Pop!_OS) or `dnf` (Fedora). [Post-install](post-install.md) has its own list of supported systems | `apt`: `libgtk-4-1 libwebkitgtk-6.0-4` · `dnf`: `gtk4 webkitgtk6.0` |
| Windows | Not supported | — | — |
| macOS | Not supported | — | — |

The GTK 4 and WebKitGTK 6.0 libraries are normally already present on a desktop Linux. Actions that change the system ask for your password through the desktop's authentication dialog (`pkexec`/PolicyKit).

## Install script (recommended)

```bash
curl --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 10 --max-time 600 -fsSL https://raw.githubusercontent.com/oito2/perci82/main/install-system.sh | bash
```

The script needs `curl`, `jq`, `sha256sum`, `tar` and `sudo`. It:

1. Finds the newest Perci82 System release on GitHub (tags `system-vX.Y.Z`; drafts and prereleases are skipped).
2. Downloads `perci82-system-linux-amd64` and `checksums.txt`, and checks the binary's SHA-256. When the GitHub CLI (`gh`) is installed and logged in, it also checks that `checksums.txt` was attested by this repository's release workflow, which vouches for the binary and the menu entry archive through their checksums.
3. Installs the binary as `/usr/local/bin/perci82-system` (with `sudo` when that folder isn't writable).
4. Extracts the menu entry archive `perci82-system-menu.tar.gz` (checked against `checksums.txt` too) into `/usr`: only the application menu entry and the icons, whatever else the archive holds. A failure here doesn't undo the install; run `perci82-system` from a terminal instead.
5. When the old Perci (`prci`) is installed, lists its files and asks whether to remove them. Its settings in `~/.perci` are kept: Perci82 System reads them.

Open **Perci82 System** from the application menu, or run `perci82-system`.

## Manual install

Download `perci82-system-linux-amd64`, `perci82-system-menu.tar.gz` and `checksums.txt` from the newest `system-v*` release on the [releases page](https://github.com/oito2/perci82/releases), then:

```bash
sha256sum --ignore-missing -c checksums.txt
sudo install -m 755 -o root -g root perci82-system-linux-amd64 /usr/local/bin/perci82-system
sudo tar -xzf perci82-system-menu.tar.gz -C /usr --no-same-owner --no-same-permissions --no-overwrite-dir
sudo gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor
```

Every release file has a build provenance attestation, which the GitHub CLI checks:

```bash
gh attestation verify perci82-system-linux-amd64 --repo oito2/perci82
```

## Build from source

Requirements: Go 1.27.4 or later and the GTK 4 and WebKitGTK 6.0 development packages.

```bash
# Debian/Ubuntu
sudo apt install -y build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
# Fedora
sudo dnf install -y gcc-c++ pkgconf-pkg-config gtk4-devel webkitgtk6.0-devel
```

```bash
git clone https://github.com/oito2/perci82.git
cd perci82
make build-system   # writes ./perci82-system
./perci82-system
```

A build from source reports the version `dev`, and its self-update is turned off. To build the release files, as the release workflow does:

```bash
make release-system VERSION=v1.0.0
```

It writes `dist/perci82-system-linux-amd64`, `dist/perci82-system-menu.tar.gz` (the menu entry and icons, under `share/`) and `dist/checksums.txt`.

## Updating

**Overview → Updates → Check** looks for a newer Perci82 System release; **Update** appears only when there is one. The update downloads the new binary, checks its SHA-256 against the release's `checksums.txt`, replaces the installed binary and asks you to restart the app. A slow download is never cut; it stops only when no data arrives for a minute. The checksum comes from the same GitHub release as the binary, so the update trusts that release; it does not check the build attestation. To check it yourself, use the command in [Manual install](#manual-install) before updating. See [Overview](overview.md).

Running the install script again also installs the newest release.

## Next steps

- [Overview](overview.md) — the first screen.
- [Uninstalling](uninstallation.md).
