🌐 [English](../../en/dev/environment.md) | **Português** | 🏠 [Índice](../index.md)

---

# Ambiente

Instala, atualiza e remove as ferramentas de desenvolvimento: pré-requisitos, linguagens, aplicativos de IA, terminais e editores.

![Environment](../../img/dev-environment.png)
<!-- TODO: capture a screenshot of Environment -->

Cada tela da lista lateral tem uma aba **Itens**, onde você escolhe o que fazer, e uma aba **Execução**, que mostra a saída da ação em andamento. Só uma ação roda por vez.

Os pacotes vêm do `apt` (família Debian/Ubuntu) ou do `dnf` (Fedora). Instalar e remover pacotes pede a sua senha; o que vai para a sua pasta pessoal não pede. Várias ferramentas acrescentam linhas ao arquivo de inicialização do seu shell (`~/.bashrc`, `~/.zshrc` ou `~/.config/fish/config.fish`, conforme o `$SHELL`) — abra um terminal novo para usá-las. As ferramentas instaladas em `~/.local/bin` também acrescentam `export PATH="$HOME/.local/bin:$PATH"` (`fish_add_path $HOME/.local/bin` no fish) quando essa pasta ainda não está no `PATH`.

## Pré-requisitos

Cada linha tem seus próprios botões **Instalar**, **Atualizar** e **Desinstalar**. Um item que depende de outro mostra **Requer: …** e o seu **Instalar** fica desativado até esse item ser instalado. **Desinstalar** pede confirmação, com um aviso quando a remoção leva mais do que o próprio item. Instalar e atualizar precisam do Ubuntu 24.04, Debian 13, Linux Mint 22, LMDE 7, Zorin OS 18, Pop!_OS 24.04 ou de uma versão mais recente: as versões anteriores não têm alguns dos pacotes (como o `libfuse2t64`).

| Item | Origem | Observações |
|---|---|---|
| Curl, Git, Lsof, CMake, Clang, GNOME Keyring, Flatpak, AppStream | Pacote de mesmo nome (`apt`/`dnf`) | |
| Flatpak Builder | Pacote `flatpak-builder` | Requer Flatpak |
| Ninja | Pacote `ninja-build` | |
| Build Essential | Pacote `build-essential` (`apt`) ou `gcc`, `gcc-c++`, `make` (`dnf`) | |
| OpenSSL | `openssl` mais `libssl-dev` (`apt`) ou `openssl-devel` (`dnf`) | Desinstalar mantém o `openssl` |
| GTK3 Dev Headers | `libgtk-3-dev` / `gtk3-devel` | |
| GTK4 Dev Headers | `libgtk-4-dev` / `gtk4-devel` | |
| WebKitGTK 6.0 Dev Headers | `libwebkitgtk-6.0-dev` / `webkitgtk6.0-devel` | |
| WebKitGTK 4.1 Dev Headers | `libwebkit2gtk-4.1-dev` / `webkit2gtk4.1-devel` | |
| Libxdo Dev | `libxdo-dev` / `libxdo-devel` | |
| Wails3 CLI | `go install` do `wails3` v3.0.0-beta.28 na pasta de binários do Go | Requer o Go ([Linguagens e SDKs](#linguagens-e-sdks)); sem senha |
| Libsecret Tools | `libsecret-1-0`, `libsecret-1-dev`, `libsecret-tools` (`apt`) ou `libsecret`, `libsecret-devel`, `git-credential-libsecret` (`dnf`) | Desinstalar mantém o `libsecret-1-0` |
| Librsvg Tools | `librsvg2-bin` + `librsvg2-dev` (`apt`) ou `librsvg2-tools` + `librsvg2-devel` (`dnf`) | |
| GitHub CLI (gh) | No `apt`, adiciona o repositório assinado do GitHub (`/etc/apt/sources.list.d/github-cli.list`) e instala o `gh`; no `dnf`, pacote `gh` | Requer Curl; desinstalar também remove o repositório |
| Docker Engine | `docker.io`, `docker-compose-v2` (`docker-compose` no Debian), `docker-buildx` (`apt`) ou `moby-engine`, `docker-compose`, `docker-buildx` (`dnf`) | Ativa o serviço `docker` e adiciona você ao grupo `docker` (acesso equivalente ao de root); depois, encerre a sessão e entre de novo |
| Node.js | A versão LTS pelo nvm v0.40.8 (script oficial) em `~/.nvm` | Requer Curl; sem senha. Atualizar passa para a LTS mais recente, levando os pacotes npm globais. Desinstalar apaga o `~/.nvm` inteiro (todas as versões e pacotes globais, inclusive o Codex CLI) e as linhas do nvm nos arquivos de inicialização do shell |
| Flatpak Builder Tools | Pacotes `pipx`, `python3-aiohttp`, `python3-tomlkit`; clona o `flatpak-builder-tools` em `~/tools/flatpak-builder-tools` num commit fixo e instala o `flatpak-node-generator` com o `pipx` | Requer Git e Flatpak Builder. Desinstalar mantém os três pacotes |
| Libfuse2 | `libfuse2t64` (`apt`) ou `fuse-libs` (`dnf`) | |
| AppImageTool | Release 1.9.1 baixada do GitHub (SHA-256 conferido com o digest da release) para `/usr/local/bin/appimagetool` | Requer Curl |
| Mkcert | `mkcert` mais `libnss3-tools` (`apt`) ou `nss-tools` (`dnf`) | |

**Atualizar** atualiza os pacotes (`apt-get install --only-upgrade` ou `dnf upgrade`) ou executa o instalador de novo. Antes de remover pacotes, o Perci82 Dev simula a remoção: se outros pacotes fossem junto, nada é removido e a execução os lista.

## Linguagens e SDKs

Mesmos botões e regras de [Pré-requisitos](#pré-requisitos).

| SDK | Requer | Instalar | Desinstalar |
|---|---|---|---|
| Go SDK | Curl | Versão estável mais recente baixada de `dl.google.com/go` (SHA-256 conferido com o `go.dev`) para `/usr/local/go` (senha); acrescenta `/usr/local/go/bin` ao `PATH` | Remove `/usr/local/go` e a sua linha no `PATH` |
| Rust (rustup) | Curl, Build Essential | `rustup-init` 1.29.1 de `static.rust-lang.org` (SHA-256 conferido); toolchain stable com o perfil padrão mais o `rust-analyzer`, em `~/.cargo` e `~/.rustup`; acrescenta `~/.cargo/bin` ao `PATH` | `rustup self uninstall` (remove `~/.cargo`, inclusive os programas do `cargo install`, e `~/.rustup`) e a linha no `PATH` |
| Flutter + Dart SDK | Git | Veja abaixo | Remove `~/development/flutter` e a sua linha no `PATH`; o Android SDK e o wrapper do Chrome continuam |

**Atualizar** instala a versão mais recente do Go (nada acontece se ela já estiver em dia), executa `rustup update` ou executa `flutter upgrade`.

Instalação do Flutter:

- Clona a branch `stable` do `flutter/flutter` em `~/development/flutter` e acrescenta o seu `bin` ao `PATH`.
- Instala os pacotes de compilação Linux do Flutter (senha); se não houver Java 17 ou mais recente, também o JDK 21 (`openjdk-21-jdk`, ou `temurin-21-jdk` do repositório da Adoptium no Fedora).
- Baixa o Android cmdline-tools (SHA-1 conferido com o manifesto do repositório do Google) para `~/Android/Sdk/cmdline-tools/latest`, acrescenta `ANDROID_HOME`, `ANDROID_SDK_ROOT` e as pastas do SDK ao arquivo de inicialização do shell, **aceita as licenças do Android SDK em seu nome** (termos em https://developer.android.com/studio/terms) e instala o `platform-tools`.
- Quando o Chrome está instalado só como o Flatpak `com.google.Chrome`, cria o wrapper `~/.local/bin/google-chrome` e define `CHROME_EXECUTABLE`.
- Termina com `flutter doctor`.

## IA

Uma lista de seleção: marque os aplicativos que quer, desmarque os que quer remover e clique em **Executar**.

| Aplicativo | Instalar | Desinstalar |
|---|---|---|
| Claude Code CLI | Script oficial `https://claude.ai/install.sh` | Remove o comando `claude` e, quando ele aponta para lá, `~/.local/share/claude` |
| Antigravity CLI | Script oficial `https://antigravity.google/cli/install.sh` | Remove o comando `agy` |
| Codex CLI | `npm install -g @openai/codex@0.162.1` (precisa do [Node.js](#pré-requisitos)) | `npm uninstall -g @openai/codex` |
| OpenCode CLI | Script oficial `https://opencode.ai/install` | Remove o comando `opencode`, `~/.opencode/bin` e as linhas do OpenCode no `PATH` dos arquivos de inicialização do shell |

Nenhum deles pede a sua senha.

## Terminais

Uma lista de seleção, como a de [IA](#ia).

| Terminal | Instalar | Desinstalar |
|---|---|---|
| Kitty | Instalador oficial em `~/.local/kitty.app`, com `kitty` e `kitten` ligados em `~/.local/bin` | Remove `~/.local/kitty.app` e os dois links |
| Alacritty | Pacote `alacritty` (senha) | Remove o pacote (senha) |
| Black Box | Flatpak `com.raggesilver.BlackBox` do Flathub, no escopo definido por `flatpak_scope` em `~/.perci/config.yaml` (sistema por padrão) | Desinstala o Flatpak |
| GNOME Console | Pacote `gnome-console` (senha) | Remove o pacote (senha) |

Depois de cada execução, cada terminal instalado ganha a opção **Abrir no <terminal>** no menu de contexto de pastas do Nautilus (submenu Scripts, `~/.local/share/nautilus/scripts/`), do Nemo (`~/.local/share/nemo/scripts/`) e do Dolphin (`~/.local/share/kio/servicemenus/`); os terminais removidos perdem a sua.

Dois botões abaixo da lista cuidam do prompt Starship, independentemente dos itens marcados:

- **Aplicar Starship** instala o Starship pelo script oficial em `~/.local/bin`, cria `~/.config/starship.toml` a partir do preset `gruvbox-rainbow` quando esse arquivo não existe e acrescenta a sua linha de inicialização ao `~/.bashrc`, ao `~/.zshrc` e ao `~/.config/fish/config.fish` (só nos arquivos que existem).
- **Remover Starship** (pede confirmação) remove o binário e essas linhas de inicialização, e preserva a configuração como `~/.config/starship.toml.perci-bak`.

## IDEs

Mesmos botões e regras de [Pré-requisitos](#pré-requisitos).

| IDE | Instalar | Desinstalar |
|---|---|---|
| Zed Editor | Script oficial `https://zed.dev/install.sh` em `~/.local` (requer Curl; sem senha) | Remove `~/.local/zed.app`, `~/.local/bin/zed` e o atalho do menu; `~/.config/zed` continua |
| VS Code | Adiciona o repositório assinado da Microsoft e instala o `code` (senha) | Remove o pacote `code`; o repositório, `~/.config/Code` e `~/.vscode` continuam |
| VSCodium | Adiciona o repositório assinado do VSCodium e instala o `codium` (senha) | Remove o pacote `codium`; o repositório, `~/.config/VSCodium` e `~/.vscode-oss` continuam |

**Atualizar** executa a instalação de novo.

## Android Studio

Instala a partir de um `.tar.gz` que você mesmo baixa em https://developer.android.com/studio?hl=pt-br. Clique em **Selecionar arquivo**, escolha o arquivo e clique em **Instalar**; **Desinstalar** não precisa de arquivo. Não há **Atualizar**: o Android Studio se atualiza sozinho.

- O arquivo precisa ser um pacote do Android Studio (todas as entradas dentro de `android-studio/`).
- Ele é extraído em `/opt/android-studio`, que pertence ao seu usuário; a senha é pedida para preparar essa pasta. Uma instalação anterior é mantida como `/opt/android-studio.old` até a extração dar certo.
- O assistente de configuração abre na própria janela. Para ter um atalho no menu, escolha **Create Desktop Entry** no menu da engrenagem (ou de três pontos).
- **Desinstalar** (senha) remove `/opt/android-studio` e `~/.local/share/applications/jetbrains-studio.desktop`.

## Antigravity IDE

Instala a partir do `.tar.gz` do Antigravity IDE (Standalone), baixado em https://antigravity.google/download#antigravity-ide. Clique em **Selecionar arquivo** e depois em **Instalar**, ou em **Atualizar** para um pacote mais novo; **Desinstalar** não precisa de arquivo.

- **Instalar** recusa quando `/opt/antigravity` já existe: use **Atualizar**.
- O programa vai para `/opt/antigravity`, que pertence ao root (senha). O seu `chrome-sandbox` tem o SHA-256 conferido depois da cópia e recebe a permissão setuid (`4755`). Uma atualização só substitui a versão anterior depois que a nova está no lugar.
- Cria o atalho de menu `~/.local/share/applications/antigravity.desktop` e o ícone `~/.local/share/icons/hicolor/512x512/apps/antigravity.png`.
- **Desinstalar** (senha) remove `/opt/antigravity`, o atalho de menu e o ícone.
