🌐 [English](../../en/dev/installation.md) | **Português** | 🏠 [Índice](../index.md)

---

# Instalar o Perci82 Dev

O Perci82 Dev é publicado só para **Linux em amd64**: ele é feito com o Wails v3, que usa cgo com GTK 4 e WebKitGTK 6.0, então cada plataforma precisa de uma compilação nativa.

## Requisitos

| SO | Situação | Distribuições | Dependências de execução |
|---|---|---|---|
| Linux (amd64) | Suportado | Distribuições com `apt` (Debian 13+, Ubuntu 24.04+, Linux Mint 22+, LMDE 7+, ZorinOS 18+, Pop!_OS 24.04+) ou `dnf` (Fedora) | `apt`: `libgtk-4-1 libwebkitgtk-6.0-4` · `dnf`: `gtk4 webkitgtk6.0` |
| Windows | Não suportado | — | — |
| macOS | Não suportado | — | — |

As bibliotecas GTK 4 e WebKitGTK 6.0 normalmente já estão presentes num Linux desktop. As ações que alteram o sistema pedem a sua senha pela janela de autenticação do ambiente gráfico (`pkexec`/PolicyKit).

As ferramentas que as telas usam — Git, Docker, Node.js, `secret-tool` e outras — não são necessárias para abrir o aplicativo: instale-as em [Ambiente → Pré-requisitos](environment.md).

## Script de instalação (recomendado)

```bash
curl --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 10 --max-time 600 -fsSL https://raw.githubusercontent.com/oito2/perci82/main/install-dev.sh | bash
```

O script precisa de `curl`, `jq`, `sha256sum`, `tar` e `sudo`. Ele:

1. Procura a versão mais recente do Perci82 Dev no GitHub (tags `dev-vX.Y.Z`; rascunhos e pré-lançamentos são ignorados).
2. Baixa `perci82-dev-linux-amd64` e `checksums.txt` e confere o SHA-256 do binário. Com o GitHub CLI (`gh`) instalado e logado, confere também a attestation de build.
3. Instala o binário como `/usr/local/bin/perci82-dev` (com `sudo` quando essa pasta não é gravável).
4. Extrai em `/usr` o arquivo da entrada no menu, `perci82-dev-menu.tar.gz` (também conferido pelo `checksums.txt`): a entrada no menu de aplicativos e os ícones. Uma falha aqui não desfaz a instalação; nesse caso, rode `perci82-dev` num terminal.
5. Quando o Perci antigo (`prci`) está instalado, lista os arquivos dele e pergunta se deve removê-los. As configurações dele em `~/.perci` são mantidas: o Perci82 Dev as lê.

Abra o **Perci82 Dev** pelo menu de aplicativos ou rode `perci82-dev`.

## Instalação manual

Baixe `perci82-dev-linux-amd64`, `perci82-dev-menu.tar.gz` e `checksums.txt` da versão `dev-v*` mais recente na [página de releases](https://github.com/oito2/perci82/releases) e depois:

```bash
sha256sum --ignore-missing -c checksums.txt
sudo install -m 755 -o root -g root perci82-dev-linux-amd64 /usr/local/bin/perci82-dev
sudo tar -xzf perci82-dev-menu.tar.gz -C /usr --no-same-owner --no-same-permissions --no-overwrite-dir
sudo gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor
```

Todo arquivo da release tem attestation de procedência de build, que o GitHub CLI confere:

```bash
gh attestation verify perci82-dev-linux-amd64 --repo oito2/perci82
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
make build-dev   # gera ./perci82-dev
./perci82-dev
```

Uma compilação a partir do código-fonte informa a versão `dev`, e a autoatualização dela fica desligada. Para gerar os arquivos da release, como faz o workflow de release:

```bash
make release-dev VERSION=v1.0.0
```

Isso gera `dist/perci82-dev-linux-amd64`, `dist/perci82-dev-menu.tar.gz` (a entrada no menu e os ícones, em `share/`) e `dist/checksums.txt`.

## Atualizar

**Visão Geral → Atualização → Verificar** procura uma versão mais nova do Perci82 Dev; **Atualizar** só aparece quando existe uma. A atualização baixa o novo binário, confere o SHA-256 dele pelo `checksums.txt` da release, substitui o binário instalado e pede que você reinicie o aplicativo. Veja a [Visão Geral](overview.md).

Rodar o script de instalação de novo também instala a versão mais recente.

## Próximos passos

- [Visão Geral](overview.md) — a primeira tela, a janela e a bandeja do sistema.
- [Configurações](settings.md) — escolha o workspace antes de criar contêineres Docker.
- [Desinstalar](uninstallation.md).
