🌐 [English](../../en/dev/settings.md) | **Português** | 🏠 [Índice](../index.md)

---

# Configurações

As preferências do próprio Perci82 Dev. Toda mudança é salva e aplicada na hora; não há botão Salvar.

![Configurações](../../img/dev-settings.png)
<!-- TODO: capture a screenshot of Perci82 Dev's Settings -->

## Geral

**Idioma** escolhe o idioma do aplicativo inteiro, inclusive das mensagens de progresso das ações e do menu da bandeja:

- **Definido pelo sistema** — português quando o idioma do sistema (`LC_ALL`, `LC_MESSAGES` ou `LANG`) é português; inglês nos demais casos.
- **Português (Brasil)**.
- **English**.

A troca vale na hora, sem reiniciar. Detalhes técnicos de erros aparecem em inglês. O Perci82 System tem a sua própria configuração de idioma.

## Tema

- **Definido pelo sistema** segue a preferência de claro ou escuro, a cor de destaque e o alto contraste do ambiente gráfico, e muda junto com eles enquanto o aplicativo está aberto.
- Os 13 presets do oito2: Everforest Light, Everforest Dark, Rosé Pine, Rosé Pine Moon, Rosé Pine Dawn, Nord Dark, Nord Light, Dracula, Alucard, Catppuccin Latte, Catppuccin Frappé, Catppuccin Macchiato e Catppuccin Mocha.

O Perci82 Dev guarda o tema separado do tema do Perci82 System.

## Caminho do Workspace

A pasta que guarda os arquivos dos apps do [Docker](docker.md) (`<workspace>/localhost/`) e em relação à qual a exportação de [Contextos de IA](ai-contexts.md) grava os caminhos. Sem um caminho salvo, o Perci82 Dev usa `~/workspace`.

**Selecionar Pasta** abre o seletor de pastas (onde também dá para criar uma pasta); **Criar Workspace** salva a pasta escolhida. Só é aceita uma pasta escolhida nessa janela. Escolha o workspace antes de criar contêineres Docker: as pastas deles são criadas dentro dele.

## Anotações de projeto

Uma pasta fora dos repositórios com uma subpasta por projeto que tem contexto de IA: `PROJECT.md` (as regras do projeto), `notes/`, `plans/` e `references/`. Os agentes de programação recebem acesso a ela quando um contexto é aplicado ou atualizado. Veja [Contextos de IA](ai-contexts.md).

**Selecionar pasta** salva a pasta na hora (só uma pasta escolhida nessa janela, não uma escolhida para outra configuração); **Desativar** a remove, e cada projeto volta a guardar as regras no próprio `.instructions/PROJECT.md`. Depois de mudar esta pasta, use **Atualizar** em **Contextos de IA → Gerenciar Contextos**.

## Bandeja do sistema

**Mostrar o Perci82 Dev na bandeja do sistema** liga ou desliga o ícone da bandeja na hora (vem desligado). Quando ligado:

- fechar a janela principal a esconde na bandeja em vez de encerrar o aplicativo; use **Sair** no menu do ícone para encerrar de vez;
- o Perci82 Dev abre oculto na bandeja junto com a sessão gráfica, pelo `~/.config/autostart/io.github.oito2.perci82.dev.desktop` (removido quando a bandeja é desligada).

O ícone precisa de um ambiente gráfico com suporte a StatusNotifierItem: KDE, XFCE e COSMIC o mostram nativamente; o GNOME só o mostra com a extensão "AppIndicator and KStatusNotifierItem Support". O menu da bandeja está descrito na [Visão Geral](overview.md#bandeja-do-sistema).

## Onde as configurações ficam salvas

No `~/.perci/config.yaml`, compartilhado com o Perci82 System. Cada aplicativo grava só as próprias chaves e mantém as outras como estão, inclusive chaves que ele não conhece. Um `config.yaml` que é um link simbólico (de um gerenciador de dotfiles) continua sendo link: o alvo dele é gravado.

| Chave | Guarda |
|---|---|
| `language` | O idioma: `system`, `pt-BR` ou `en` |
| `gui_theme` | O tema |
| `workspace_path` | A pasta do workspace |
| `project_notes_path` | A pasta de anotações de projeto |
| `tray_enabled` | Se a bandeja do sistema está ligada |
| `docker` | O stack Docker e os apps dele |
| `moodle_plugins` | As pastas de desenvolvimento montadas nos apps |
| `git_accounts` | As contas Git e as configurações dos repositórios delas |
| `ai_contexts` | As pastas com contexto de IA |
| `agent_skills` | As skills instaladas, por escopo |
| `mcp_servers` | Os servidores MCP registrados, por escopo |
| `mcp_binaries` | Os binários de servidores MCP que o Perci82 Dev instalou, com versão e SHA-256 |

O Perci82 Dev também guarda os arquivos do stack Docker em `~/.perci/appstack/` e os arquivos gitconfig das contas Git em `~/.perci/git/`. As configurações salvas pelo Perci 1.9 continuam funcionando.
