🌐 [Português](../../pt-br/system/installation.md) | **English** | 🏠 [Index](../index.md)

---

# Installing Perci82 System

Perci82 System is released for **Linux on amd64** only, as `.deb` and `.rpm` packages: it is built with Wails v3, which uses cgo with GTK 4 and WebKitGTK 6.0, so each platform needs a native build.

## Requirements

| OS | Status | Distributions | Runtime dependencies |
|---|---|---|---|
| Linux (amd64) | Supported | Debian 13+, Ubuntu 24.04+, Linux Mint 22+, LMDE 7+, Zorin OS 18+, Pop!_OS 24.04+ (`apt`) or Fedora 40+ (`dnf`). [Post-install](post-install.md) has its own list of supported systems | `apt`: `libgtk-4-1 libwebkitgtk-6.0-4` · `dnf`: `gtk4 webkitgtk6.0` |
| Windows | Not supported | — | — |
| macOS | Not supported | — | — |

The package depends on the GTK 4 and WebKitGTK 6.0 runtime libraries, and the package manager installs them when they are missing; older releases don't have WebKitGTK 6.0. Actions that change the system ask for your password through the desktop's authentication dialog (`pkexec`/PolicyKit).

## Install script (recommended)

```bash
curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/oito2/perci82/main/install.sh | bash -s -- system
```

Without `-s -- system`, the script asks which apps to install: Perci82 System, Perci82 Dev or both. It also takes `system`, `dev` or `all` as arguments. It needs `curl`, `gpg` and `sudo`, and:

1. Checks that the machine is amd64 with `apt` or `dnf` and, on `apt`, that the release has WebKitGTK 6.0.
2. Downloads the repository's signing key, `https://oito2.github.io/perci82/perci82.asc`, and checks its fingerprint: `A1B1 AC75 B535 0344 B6C8  043C FF9E 1B0F 3C2A 545F`.
3. Adds the Perci82 package repository, shared by both apps:
   - `apt`: the key as `/usr/share/keyrings/perci82-archive-keyring.asc` and the source `/etc/apt/sources.list.d/perci82.sources`;
   - `dnf`: the key as `/etc/pki/rpm-gpg/RPM-GPG-KEY-perci82` (also imported into rpm) and the repository `/etc/yum.repos.d/perci82.repo`.
4. Installs the package `perci82-system` with `apt-get install` or `dnf install`. An app that is already installed is updated when a newer version exists.
5. When the old Perci (`prci`) is installed, lists its files and asks whether to remove them. Its settings in `~/.perci` are kept: Perci82 System reads them.

Open **Perci82 System** from the application menu, or run `perci82-system`.

## Add the repository by hand

On `apt` (Debian, Ubuntu, Linux Mint, Zorin OS, Pop!_OS):

```bash
sudo curl -fsSLo /usr/share/keyrings/perci82-archive-keyring.asc https://oito2.github.io/perci82/perci82.asc
gpg --show-keys /usr/share/keyrings/perci82-archive-keyring.asc   # must show A1B1AC75B5350344B6C8043CFF9E1B0F3C2A545F
sudo curl -fsSLo /etc/apt/sources.list.d/perci82.sources https://oito2.github.io/perci82/perci82.sources
sudo apt update
sudo apt install perci82-system
```

On `dnf` (Fedora):

```bash
sudo curl -fsSLo /etc/pki/rpm-gpg/RPM-GPG-KEY-perci82 https://oito2.github.io/perci82/perci82.asc
gpg --show-keys /etc/pki/rpm-gpg/RPM-GPG-KEY-perci82   # must show A1B1AC75B5350344B6C8043CFF9E1B0F3C2A545F
sudo curl -fsSLo /etc/yum.repos.d/perci82.repo https://oito2.github.io/perci82/perci82.repo
sudo dnf install perci82-system
```

The first `dnf install` asks to import the key; check that it shows the same fingerprint.

## Install a downloaded package

Download the package and `checksums.txt` from the newest `system-v*` release on the [releases page](https://github.com/oito2/perci82/releases), then, with the release's version in place of `X.Y.Z`:

```bash
sha256sum --ignore-missing -c checksums.txt
sudo apt install ./perci82-system_X.Y.Z-1_amd64.deb    # Debian, Ubuntu, Linux Mint
sudo dnf install ./perci82-system-X.Y.Z-1.x86_64.rpm   # Fedora
```

A package installed this way isn't updated by `apt upgrade` or `dnf upgrade`: add the repository for that. Every release file has a build provenance attestation, which the GitHub CLI checks:

```bash
gh attestation verify perci82-system_X.Y.Z-1_amd64.deb --repo oito2/perci82
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

A build from source reports the version `dev`. To build the release packages, as the release workflow does (nfpm goes to `$(go env GOPATH)/bin`, which must be in `PATH`):

```bash
make install-nfpm   # once: nfpm, the packaging tool
make release-system VERSION=v1.2.0
```

It writes `dist/perci82-system_1.2.0-1_amd64.deb`, `dist/perci82-system-1.2.0-1.x86_64.rpm` and `dist/checksums.txt`, and the binary they hold in `dist/build/perci82-system`.

## Updating

The package manager updates Perci82 System with the rest of the system: `sudo apt update && sudo apt upgrade` or `sudo dnf upgrade`. When a newer release exists, **Overview → Updates** also shows the command that installs just that one, with **Copy**:

```bash
sudo apt update && sudo apt install --only-upgrade perci82-system   # apt
sudo dnf upgrade --refresh perci82-system                             # dnf
```

See [Overview](overview.md).

## Next steps

- [Overview](overview.md) — the first screen.
- [Uninstalling](uninstallation.md).
