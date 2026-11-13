# Documentação do Perci82

🌐 [English](../en/index.md) | **Português** | 🏠 [Voltar ao Leiame](leiame.md)

O Perci82 são dois aplicativos desktop para Linux: o **Perci82 System** administra o sistema operacional — pós-instalação, atualização do sistema e dos aplicativos, fontes, templates de arquivos e aplicativos Flatpak — e o **Perci82 Dev** prepara o ambiente de desenvolvimento. Os dois leem e gravam a mesma pasta `~/.perci`. Estas páginas tratam dos dois aplicativos.

## 🚀 Primeiros passos

- [Instalar o Perci82 System](system/installation.md) — requisitos, script de instalação, instalação manual, compilação a partir do código-fonte e atualizações.
- [Desinstalar o Perci82 System](system/uninstallation.md) — o que a instalação coloca no sistema e como remover.
- [Instalar o Perci82 Dev](dev/installation.md) — requisitos, script de instalação, instalação manual, compilação a partir do código-fonte e atualizações.
- [Desinstalar o Perci82 Dev](dev/uninstallation.md) — o que a instalação e as telas do aplicativo deixam no sistema, os backups e como remover.

## 🖥️ Telas do Perci82 System

- [Visão Geral](system/overview.md) — o sistema detectado, a versão, as atualizações e a desinstalação.
- [Pós-instalação](system/post-install.md) — configuração recomendada para um sistema recém-instalado, por distribuição suportada.
- [Atualizar SO & Apps](system/update.md) — pacotes, Snap e Flatpak atualizados de uma vez.
- [Gerenciar Fontes](system/fonts.md) — fontes JetBrains Mono, Carlito, Caladea e Noto.
- [Gerenciar Templates de Arquivos](system/templates.md) — arquivos vazios para o menu Novo Documento do gerenciador de arquivos.
- [Flatpaks](system/flatpaks.md) — aplicativos desktop do Flathub.
- [Configurações](system/settings.md) — idioma, tema, escopo do Flatpak e onde ficam salvos.

## 🧰 Telas do Perci82 Dev

- [Visão Geral](dev/overview.md) — a janela, o resumo do Docker e dos repositórios, as atualizações, a desinstalação e a bandeja do sistema.
- [Ambiente](dev/environment.md) — pré-requisitos, linguagens e SDKs, aplicativos de IA, terminais, IDEs, Android Studio e Antigravity IDE.
- [Docker](dev/docker.md) — o stack com Nginx e MariaDB, os contêineres de apps, o comando de cada app, as pastas de desenvolvimento e exportar/importar.
- [Repositórios](dev/repositories.md) — contas Git com chave SSH, identidade e token próprios, e os repositórios delas.
- [Contextos de IA](dev/ai-contexts.md) — regras para os agentes de programação em cada projeto, anotações de projeto e o padrão PHP Perci82.
- [Skills & MCPs](dev/skills-mcps.md) — skills e servidores MCP para os agentes de programação, incluindo o Build82 e o Sync82.
- [Configurações](dev/settings.md) — idioma, tema, workspace, anotações de projeto, bandeja do sistema e onde ficam salvos.

## 🛠️ Solução de problemas

- [Perci82 System](system/troubleshooting.md) — bibliotecas ausentes, mensagens do instalador, autenticação e verificação de atualizações.
- [Perci82 Dev](dev/troubleshooting.md) — bibliotecas ausentes, a instância em execução, o ícone da bandeja, o acesso ao Docker e o chaveiro.
