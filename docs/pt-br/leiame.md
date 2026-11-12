<picture>
  <source media="(prefers-color-scheme: dark)" srcset="../../.github/header-escuro.svg">
  <img alt="Perci82" src="../../.github/header-claro.svg" width="100%">
</picture>

# Perci82

Dois aplicativos desktop para Linux: o **Perci82 System** administra o sistema operacional e o **Perci82 Dev** prepara o seu ambiente de desenvolvimento.

![Go](https://img.shields.io/badge/go-1.27%2B-00ADD8?logo=go) ![Plataforma](https://img.shields.io/badge/plataforma-linux%2Famd64-informational) ![Licença](https://img.shields.io/badge/licen%C3%A7a-GPL--3.0-blue) [![CI](https://github.com/oito2/perci82/actions/workflows/ci.yml/badge.svg)](https://github.com/oito2/perci82/actions/workflows/ci.yml)

🌐 **Idioma:** [English](../../README.md) · Português

> **Situação:** em desenvolvimento. Ainda não há release; os aplicativos só podem ser compilados a partir do código-fonte.

## Sumário

- [Visão geral](#visão-geral)
- [Pré-requisitos e instalação rápida](#pré-requisitos-e-instalação-rápida)
- [Licença](#licença)

## Visão geral

O Perci82 é um monorepo com dois aplicativos desktop independentes, feitos em Go com o [Wails v3](https://v3.wails.io). Os dois leem e gravam a mesma pasta `~/.perci`.

| Aplicativo | Binário | O que faz |
|---|---|---|
| **Perci82 System** | `perci82-system` | Administração do Linux: pós-instalação, atualização do sistema e dos aplicativos (pacotes, Snap, Flatpak), fontes, templates de arquivos e aplicativos Flatpak |
| **Perci82 Dev** | `perci82-dev` | Ambiente de desenvolvimento: pré-requisitos, linguagens e SDKs, aplicativos de IA, terminais, IDEs, contêineres Docker, repositórios e contas Git, contextos de IA, skills e servidores MCP; bandeja do sistema opcional |

## Pré-requisitos e instalação rápida

### Instalar uma release

Cada aplicativo é instalado separadamente, a partir da sua release mais recente (tags `system-v*` e `dev-v*`), em Linux amd64 com as bibliotecas de execução do GTK 4 e do WebKitGTK 6.0 (`libgtk-4-1` e `libwebkitgtk-6.0-4` no Debian/Ubuntu). Os instaladores precisam de `curl`, `jq`, `sha256sum`, `tar` e `sudo`.

```bash
# Perci82 System
curl --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 10 --max-time 600 -fsSL https://raw.githubusercontent.com/oito2/perci82/main/install-system.sh | bash
# Perci82 Dev
curl --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 10 --max-time 600 -fsSL https://raw.githubusercontent.com/oito2/perci82/main/install-dev.sh | bash
```

O instalador confere o SHA-256 do binário (e a attestation de build, quando o GitHub CLI está logado), instala em `/usr/local/bin/perci82-system` ou `/usr/local/bin/perci82-dev` e cria a entrada no menu de aplicativos. Quando o Perci antigo (`prci`) está instalado, ele pergunta se deve removê-lo; as configurações em `~/.perci` são mantidas, porque os dois aplicativos as usam.

A Visão Geral procura e aplica as atualizações das releases do próprio aplicativo, e **Desinstalar** remove o binário, a entrada no menu e os ícones do aplicativo. "Também remover configurações" remove só as configurações daquele aplicativo (e, no Perci82 Dev, os arquivos da stack Docker e das contas Git em `~/.perci`); a pasta `~/.perci` só é apagada quando nenhum outro aplicativo Perci82 nem o Perci antigo está instalado e não sobrou mais nada nela.

### Compilar a partir do código

Requisitos:

- Linux em amd64.
- Go 1.27.4 ou mais recente.
- Pacotes de desenvolvimento do GTK 4 e do WebKitGTK 6.0, por exemplo no Debian/Ubuntu:

  ```bash
  sudo apt install -y build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
  ```

- Node.js, só para rodar os testes do frontend.

Compilar e executar:

```bash
git clone https://github.com/oito2/perci82.git
cd perci82
make build          # gera ./perci82-system e ./perci82-dev
./perci82-system
./perci82-dev
```

Rodar as verificações:

```bash
make lint
make test
make test-frontend
make check-generated
```

As cores, medidas, presets e textos comuns da interface (OK, Cancelar, Voltar…) vêm do [oito2_ui](https://github.com/oito2/oito2_ui), copiados como fixtures em `internal/theme/parity/`. Para atualizá-los, clone o oito2_ui em `references/oito2_ui` (ou passe `OITO2_UI=<pasta>`) e rode:

```bash
make parity-sync
make generate
```

## Licença

GPL-3.0 — veja [LICENSE](../../LICENSE).
