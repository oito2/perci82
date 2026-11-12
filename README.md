<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/header-escuro.svg">
  <img alt="Perci82" src=".github/header-claro.svg" width="100%">
</picture>

# Perci82

Two Linux desktop apps: **Perci82 System** manages the OS, **Perci82 Dev** sets up your development environment.

![Go](https://img.shields.io/badge/go-1.27%2B-00ADD8?logo=go) ![Platform](https://img.shields.io/badge/platform-linux%2Famd64-informational) ![License](https://img.shields.io/badge/license-GPL--3.0-blue) [![CI](https://github.com/oito2/perci82/actions/workflows/ci.yml/badge.svg)](https://github.com/oito2/perci82/actions/workflows/ci.yml)

🌐 **Language:** English · [Português](docs/pt-br/leiame.md)

> **Status:** Perci82 System is released (tags `system-vX.Y.Z`). Perci82 Dev is in development and can only be built from source.

## Table of Contents

- [Overview](#overview)
- [Prerequisites & Quick Installation](#prerequisites--quick-installation)
- [Automated Installation](#automated-installation)
- [Update & Maintenance](#update--maintenance)
- [Documentation](#documentation)
- [License](#license)

## Overview

Perci82 is a monorepo with two independent desktop apps built with Go and [Wails v3](https://v3.wails.io). Both read and write the same `~/.perci` folder, and each is released, updated and uninstalled on its own.

| App | Binary | What it does |
|---|---|---|
| **Perci82 System** | `perci82-system` | Linux administration: post-install setup, system and app updates (packages, Snap, Flatpak), fonts, file templates and Flatpak apps |
| **Perci82 Dev** | `perci82-dev` | Development environment: prerequisites, languages and SDKs, AI apps, terminals, IDEs, Docker containers, Git repositories and accounts, AI contexts, skills and MCP servers; optional system tray |

Perci82 System's main features:

- **Post-install** — the recommended setup of Ubuntu, Ubuntu Budgie, Linux Mint, Zorin OS, Fedora and Pop!_OS, item by item.
- **Update OS & Apps** — distribution packages, Snap and Flatpak updated in one run, with a single password prompt.
- **Fonts, file templates and Flatpaks** — checklists that install what you check and remove what you uncheck.
- **Self-update and uninstall** from the Overview screen; Portuguese and English interface; oito2 themes.

![Perci82 System's Overview](docs/img/system-overview.png)
<!-- TODO: capture a screenshot of Perci82 System's Overview -->

## Prerequisites & Quick Installation

| OS | Status | Distributions | Runtime dependencies |
|---|---|---|---|
| Linux (amd64) | Supported | Distributions with `apt` (Debian, Ubuntu, Linux Mint, Zorin OS, Pop!_OS) or `dnf` (Fedora) | `apt`: `libgtk-4-1 libwebkitgtk-6.0-4` · `dnf`: `gtk4 webkitgtk6.0` |
| Windows | Not supported | — | — |
| macOS | Not supported | — | — |

Release binaries run on Linux amd64 with the GTK 4 and WebKitGTK 6.0 runtime libraries, normally already present on a desktop Linux. To install a release, see [Automated Installation](#automated-installation); the manual install is in [Installing Perci82 System](docs/en/system/installation.md#manual-install).

### Build from source

Requirements:

- Linux on amd64.
- Go 1.27.4 or later.
- GTK 4 and WebKitGTK 6.0 development packages, for example on Debian/Ubuntu:

  ```bash
  sudo apt install -y build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
  ```

- Node.js, only to run the frontend tests.

Build and run:

```bash
git clone https://github.com/oito2/perci82.git
cd perci82
make build          # writes ./perci82-system and ./perci82-dev
./perci82-system
./perci82-dev
```

Run the checks:

```bash
make lint
make test
make test-frontend
make check-generated
```

The colors, measures, presets and common interface texts (OK, Cancel, Back…) come from [oito2_ui](https://github.com/oito2/oito2_ui), copied as fixtures into `internal/theme/parity/`. To update them, check out oito2_ui into `references/oito2_ui` (or pass `OITO2_UI=<dir>`) and run:

```bash
make parity-sync
make generate
```

## Automated Installation

Each app has its own installer, which installs the app's newest release. Perci82 System:

```bash
curl --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 10 --max-time 600 -fsSL https://raw.githubusercontent.com/oito2/perci82/main/install-system.sh | bash
```

The installers need `curl`, `jq`, `sha256sum`, `tar` and `sudo`. They check the binary's SHA-256 (and its build attestation when the GitHub CLI is logged in), install it as `/usr/local/bin/perci82-system` or `/usr/local/bin/perci82-dev` and add the application menu entry. When the old Perci (`prci`) is installed, they ask whether to remove it; its settings in `~/.perci` are kept, since both apps read them. Perci82 Dev's installer, `install-dev.sh`, works the same way from Perci82 Dev's first release on.

## Update & Maintenance

- **Update:** Overview → **Check** looks for a newer release of the app; **Update** downloads it, checks its SHA-256 and replaces the binary. Running the installer again also installs the newest release. Builds from source report the version `dev` and don't update themselves.
- **Uninstall:** Overview → **Uninstall** removes the app's binary, menu entry and icons. "Also remove the settings" removes only that app's settings (and, in Perci82 Dev, its Docker stack files and Git account files in `~/.perci`); `~/.perci` itself goes only when no other Perci82 app nor the old Perci is installed and nothing else is left in it. See [Uninstalling Perci82 System](docs/en/system/uninstallation.md).

## Documentation

- [Documentation index](docs/en/index.md)
- [Installing Perci82 System](docs/en/system/installation.md) and [uninstalling it](docs/en/system/uninstallation.md)
- Perci82 System's screens: [Overview](docs/en/system/overview.md), [Post-install](docs/en/system/post-install.md), [Update OS & Apps](docs/en/system/update.md), [Fonts](docs/en/system/fonts.md), [Templates](docs/en/system/templates.md), [Flatpaks](docs/en/system/flatpaks.md), [Settings](docs/en/system/settings.md)
- [Troubleshooting](docs/en/system/troubleshooting.md)

## License

GPL-3.0 — see [LICENSE](LICENSE).
