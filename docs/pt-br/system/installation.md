🌐 [English](../../en/system/installation.md) | **Português** | 🏠 [Índice](../index.md)

---

# Instalar o Perci82 System

O Perci82 System é publicado só para **Linux em amd64**, como pacotes `.deb` e `.rpm`: ele é feito com o Wails v3, que usa cgo com GTK 4 e WebKitGTK 6.0, então cada plataforma precisa de uma compilação nativa.

## Requisitos

| SO | Situação | Distribuições | Dependências de execução |
|---|---|---|---|
| Linux (amd64) | Suportado | Debian 13+, Ubuntu 24.04+, Linux Mint 22+, LMDE 7+, Zorin OS 18+, Pop!_OS 24.04+ (`apt`) ou Fedora 40+ (`dnf`). A [Pós-instalação](post-install.md) tem a sua própria lista de sistemas suportados | `apt`: `libgtk-4-1 libwebkitgtk-6.0-4` · `dnf`: `gtk4 webkitgtk6.0` |
| Windows | Não suportado | — | — |
| macOS | Não suportado | — | — |

O pacote depende das bibliotecas de execução do GTK 4 e do WebKitGTK 6.0, e o gerenciador de pacotes as instala quando faltam; versões mais antigas das distribuições não têm o WebKitGTK 6.0. As ações que alteram o sistema pedem a sua senha pela janela de autenticação do ambiente gráfico (`pkexec`/PolicyKit).

## Script de instalação (recomendado)

```bash
curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/oito2/perci82/main/install.sh | bash -s -- system
```

Sem `-s -- system`, o script pergunta quais aplicativos instalar: Perci82 System, Perci82 Dev ou os dois. Ele também aceita `system`, `dev` ou `all` como argumentos. Precisa de `curl`, `gpg` e `sudo`, e:

1. Confere se a máquina é amd64 com `apt` ou `dnf` e, no `apt`, se a versão da distribuição tem o WebKitGTK 6.0.
2. Baixa a chave de assinatura do repositório, `https://oito2.github.io/perci82/perci82.asc`, e confere a impressão digital dela: `A1B1 AC75 B535 0344 B6C8  043C FF9E 1B0F 3C2A 545F`.
3. Adiciona o repositório de pacotes do Perci82, o mesmo para os dois aplicativos:
   - `apt`: a chave como `/usr/share/keyrings/perci82-archive-keyring.asc` e a fonte `/etc/apt/sources.list.d/perci82.sources`;
   - `dnf`: a chave como `/etc/pki/rpm-gpg/RPM-GPG-KEY-perci82` (também importada no rpm) e o repositório `/etc/yum.repos.d/perci82.repo`.
4. Instala o pacote `perci82-system` com `apt-get install` ou `dnf install`. Um aplicativo já instalado é atualizado quando existe uma versão mais nova.
5. Quando o Perci antigo (`prci`) está instalado, lista os arquivos dele e pergunta se deve removê-los. As configurações dele em `~/.perci` são mantidas: o Perci82 System as lê.

Abra o **Perci82 System** pelo menu de aplicativos ou rode `perci82-system`.

## Adicionar o repositório à mão

No `apt` (Debian, Ubuntu, Linux Mint, Zorin OS, Pop!_OS):

```bash
sudo curl -fsSLo /usr/share/keyrings/perci82-archive-keyring.asc https://oito2.github.io/perci82/perci82.asc
gpg --show-keys /usr/share/keyrings/perci82-archive-keyring.asc   # deve mostrar A1B1AC75B5350344B6C8043CFF9E1B0F3C2A545F
sudo curl -fsSLo /etc/apt/sources.list.d/perci82.sources https://oito2.github.io/perci82/perci82.sources
sudo apt update
sudo apt install perci82-system
```

No `dnf` (Fedora):

```bash
sudo curl -fsSLo /etc/pki/rpm-gpg/RPM-GPG-KEY-perci82 https://oito2.github.io/perci82/perci82.asc
gpg --show-keys /etc/pki/rpm-gpg/RPM-GPG-KEY-perci82   # deve mostrar A1B1AC75B5350344B6C8043CFF9E1B0F3C2A545F
sudo curl -fsSLo /etc/yum.repos.d/perci82.repo https://oito2.github.io/perci82/perci82.repo
sudo dnf install perci82-system
```

O primeiro `dnf install` pede para importar a chave; confira se ele mostra a mesma impressão digital.

## Instalar um pacote baixado

Baixe o pacote e o `checksums.txt` da versão `system-v*` mais recente na [página de releases](https://github.com/oito2/perci82/releases) e depois, com a versão da release no lugar de `X.Y.Z`:

```bash
sha256sum --ignore-missing -c checksums.txt
sudo apt install ./perci82-system_X.Y.Z-1_amd64.deb    # Debian, Ubuntu, Linux Mint
sudo dnf install ./perci82-system-X.Y.Z-1.x86_64.rpm   # Fedora
```

Um pacote instalado assim não é atualizado pelo `apt upgrade` nem pelo `dnf upgrade`: para isso, adicione o repositório. Todo arquivo da release tem attestation de procedência de build, que o GitHub CLI confere:

```bash
gh attestation verify perci82-system_X.Y.Z-1_amd64.deb --repo oito2/perci82
```

## Compilar a partir do código-fonte

Requisitos: Go 1.27.4 ou mais recente e os pacotes de desenvolvimento do GTK 4 e do WebKitGTK 6.0.

```bash
# Debian/Ubuntu
sudo apt install -y build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
# Fedora
sudo dnf install -y gcc-c++ pkgconf-pkg-config gtk4-devel webkitgtk6.0-devel
```

```bash
git clone https://github.com/oito2/perci82.git
cd perci82
make build-system   # gera ./perci82-system
./perci82-system
```

Uma compilação a partir do código-fonte informa a versão `dev`. Para gerar os pacotes da release, como faz o workflow de release (o nfpm vai para `$(go env GOPATH)/bin`, que precisa estar no `PATH`):

```bash
make install-nfpm   # uma vez: o nfpm, a ferramenta de empacotamento
make release-system VERSION=v1.2.0
```

Isso gera `dist/perci82-system_1.2.0-1_amd64.deb`, `dist/perci82-system-1.2.0-1.x86_64.rpm` e `dist/checksums.txt`, e o binário que eles contêm em `dist/build/perci82-system`.

## Atualizar

O gerenciador de pacotes atualiza o Perci82 System junto com o resto do sistema: `sudo apt update && sudo apt upgrade` ou `sudo dnf upgrade`. Quando existe uma versão mais nova, **Visão Geral → Atualização** também mostra o comando que instala só ela, com **Copiar**:

```bash
sudo apt update && sudo apt install --only-upgrade perci82-system   # apt
sudo dnf upgrade --refresh perci82-system                             # dnf
```

Veja a [Visão Geral](overview.md).

## Próximos passos

- [Visão Geral](overview.md) — a primeira tela.
- [Desinstalar](uninstallation.md).
