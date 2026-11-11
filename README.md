<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/header-escuro.svg">
  <img alt="Perci82" src=".github/header-claro.svg" width="100%">
</picture>

# Perci82

Two Linux desktop apps: **Perci82 System** manages the OS, **Perci82 Dev** sets up your development environment.

![Go](https://img.shields.io/badge/go-1.26%2B-00ADD8?logo=go) ![Platform](https://img.shields.io/badge/platform-linux%2Famd64-informational) ![License](https://img.shields.io/badge/license-GPL--3.0-blue) [![CI](https://github.com/oito2/perci82/actions/workflows/ci.yml/badge.svg)](https://github.com/oito2/perci82/actions/workflows/ci.yml)

🌐 **Language:** English · [Português](docs/pt-br/leiame.md)

> **Status:** in development. There is no release yet; the apps can only be built from source.

## Table of Contents

- [Overview](#overview)
- [Prerequisites & Quick Installation](#prerequisites--quick-installation)
- [License](#license)

## Overview

Perci82 is a monorepo with two independent desktop apps built with Go and [Wails v3](https://v3.wails.io). Both read and write the same `~/.perci` folder.

| App | Binary | What it does |
|---|---|---|
| **Perci82 System** | `perci82-system` | Linux administration: post-install setup, system and app updates (packages, Snap, Flatpak), fonts, file templates and Flatpak apps |
| **Perci82 Dev** | `perci82-dev` | Development environment: prerequisites, languages and SDKs, AI apps, terminals, IDEs, Docker containers, Git repositories and accounts, AI contexts, skills and MCP servers; optional system tray |

## Prerequisites & Quick Installation

Requirements:

- Linux on amd64.
- Go 1.26.3 or later.
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
```

## License

GPL-3.0 — see [LICENSE](LICENSE).
