🌐 [English](../../en/dev/skills-mcps.md) | **Português** | 🏠 [Índice](../index.md)

---

# Skills & MCPs

Atalhos para dar aos agentes de programação skills e servidores MCP de terceiros. A lista lateral tem duas telas: **SKILLs** e **MCPs**.

![Skills & MCPs](../../img/dev-skills-mcps.png)
<!-- TODO: capture a screenshot of Skills & MCPs → MCPs -->

## Tipo de Instalação

As duas telas começam com **Tipo de Instalação**:

- **Global** — instala globalmente.
- **Local** — instala só para um projeto: clique em **Selecionar Pasta** para escolher a pasta dele.

Cada lista mostra o que está instalado no escopo escolhido. Os itens instalados ficam salvos em `~/.perci/config.yaml`, por escopo: `agent_skills` para SKILLs e `mcp_servers` para MCPs.

## SKILLs

Skills publicadas no [skills.sh](https://www.skills.sh/), instaladas com a CLI `skills` fixada na versão 1.7.0 e executada pelo `npx` (requer o Node.js). Cada skill é instalada a partir do repositório dela, num commit fixado pelo Perci82 Dev, para Claude Code, OpenCode, Antigravity e Codex de uma vez:

```bash
npx -y skills@1.7.0 add <repositório>#<commit> --skill <nome> --agent claude-code --agent opencode --agent antigravity --agent codex --yes [--global]
```

A telemetria da CLI fica desligada (`DISABLE_TELEMETRY=1`). No escopo Local, o comando roda na pasta escolhida.

> As SKILLs são conteúdo de terceiros: o Perci fixa a versão, mas não revisa o conteúdo, e a SKILL executa com as permissões do agente. Leia o código antes de usar.

| Botão | O que faz |
|---|---|
| **Instalar** | Instala a skill. Quando a execução para todos os agentes falha, cada agente é tentado separadamente |
| **Atualizar** | Instala de novo no commit fixado |
| **Remover** | Remove dos quatro agentes. Num item que instala um repositório inteiro, só as skills instaladas a partir desse repositório são removidas |

Selos ao lado do nome:

- **parcial: …** — algum agente não aceitou a skill; o selo cita os que aceitaram.
- **fora do catálogo** — o item saiu do catálogo do Perci; só aparece enquanto instalado, e só pode ser removido.

### Catálogo

| SKILL | Repositório |
|---|---|
| Godot: UI | `gamedev-skills/awesome-gamedev-agent-skills` |
| Godot: Master | `thedivergentai/Game-Design-Agentic-Skills` |
| Godot: Mobile (Android/iOS) | `thedivergentai/Game-Design-Agentic-Skills` |
| Godot: Animations | `gamedev-skills/awesome-gamedev-agent-skills` |
| Godot: Shaders | `gamedev-skills/awesome-gamedev-agent-skills` |
| Godot: GDScript | `gamedev-skills/awesome-gamedev-agent-skills` |
| Godot: Export & Builds | `gamedev-skills/awesome-gamedev-agent-skills` |
| Godot: Patterns | `wshobson/agents` |
| Moodle: Theme v5+ | `nicolasflores9/skills` |
| Moodle: External API | `sickn33/agentic-awesome-skills` |
| DaisyUI | `saadeghi/daisyui` (todas as skills do repositório) |
| ShaCN UI | `shadcn-ui/ui` |
| Frontend Design | `anthropics/skills` |
| Svelte: Code Writer (oficial) | `sveltejs/ai-tools` |
| Svelte: Best Practices (oficial) | `sveltejs/ai-tools` |
| React: Best Practices (Vercel) | `vercel-labs/agent-skills` |
| Vite | `antfu/skills` |
| Vitest | `antfu/skills` |
| Web: Acessibilidade, SEO e Performance | `addyosmani/web-quality-skills` (todas as skills do repositório) |
| Web: Testes com Playwright | `anthropics/skills` |
| SuperPowers | `obra/superpowers` |
| Code Review | `mattpocock/skills` |
| Security Review (Sentry) | `getsentry/skills` |
| GitHub Actions: Hardening | `github/awesome-copilot` |
| Go: Code Style | `samber/cc-skills-golang` |
| Go: Error Handling | `samber/cc-skills-golang` |
| Go: Testing | `samber/cc-skills-golang` |
| Go: Security | `samber/cc-skills-golang` |
| Go: Lint | `samber/cc-skills-golang` |
| Go: Continuous Integration | `samber/cc-skills-golang` |
| Go: Golang Pro | `jeffallan/claude-skills` |
| Go: Wails | `spf13/go-skills` |
| Rust: Best Practices | `apollographql/skills` |
| PHP: Pro 8.3+ | `jeffallan/claude-skills` |
| MariaDB: Query Optimization (oficial) | `MariaDB/skills` |
| MariaDB: Features (oficial) | `MariaDB/skills` |
| SQL: Code Review | `github/awesome-copilot` |
| MCP Server Dev | `anthropics/skills` |
| MCP Server Dev in Go | `github/awesome-copilot` |
| MCP: Security Audit | `github/awesome-copilot` |
| Dart: Oficial | `dart-lang/skills` (todas as skills do repositório) |
| Flutter: Oficial | `flutter/agent-plugins` (todas as skills do repositório) |

Todos os repositórios estão no GitHub (`https://github.com/<repositório>`).

## MCPs

Servidores MCP registrados no Claude Code, no Codex e no Antigravity, cada um pelo seu próprio mecanismo:

| Agente | Como | Global | Local |
|---|---|---|---|
| Claude Code | `claude mcp add` / `claude mcp remove` | `--scope user` | `--scope local`, executado na pasta escolhida |
| Codex | `codex mcp add` / `codex mcp remove` | Sim | Pulado |
| Antigravity | Edita o `mcp_config.json`, mantendo as outras entradas | `~/.gemini/config/mcp_config.json` | `<pasta>/.agents/mcp_config.json` |

O Codex só é registrado quando o **Tipo de Instalação** é Global: o escopo por projeto dele não está confirmado como já lançado. Um servidor com o mesmo nome que não foi registrado pelo Perci é mantido, nunca substituído, e **Remover** não mexe nele; o mesmo vale para uma entrada cujo comando ou argumentos foram alterados (uma opção ou chave de API a mais, outro pacote). No arquivo do Antigravity, as chaves que o Perci não grava na própria entrada (como `env`) são mantidas quando ela é atualizada.

| Botão | O que faz |
|---|---|
| **Instalar** | Registra o servidor em todos os agentes que se aplicam; só falha quando nenhum aceita |
| **Atualizar** | Remove e registra de novo com o mesmo parâmetro (Dart + Flutter: veja abaixo) |
| **Remover** | Tira o registro; um agente que já não o tem gera só um aviso |

Uma linha mostra **parcial: …** quando falta um agente que se aplica ao escopo, e **fora do catálogo** num item que só pode ser removido.

### Catálogo

| MCP | Precisa de | Parâmetro |
|---|---|---|
| Dart + Flutter | O plugin `dart-flutter` do marketplace `flutter/agent-plugins`, para Claude Code e Codex; o Antigravity já o tem nativamente | — |
| Filesystem MCP (@modelcontextprotocol/server-filesystem) | `npx` (`@modelcontextprotocol/server-filesystem@1.0.0`) | **Pasta permitida** |
| Context7 | `npx` (`@upstash/context7-mcp@4.4.1`); as consultas são enviadas para context7.com | — |
| Go: gopls MCP | `gopls` no `PATH` (`go install golang.org/x/tools/gopls@v0.23.0`) | — |
| Chrome DevTools MCP | `npx` (`chrome-devtools-mcp@1.10.1`, perfil isolado, estatísticas de uso desligadas) e o Google Chrome | — |
| Svelte MCP | `npx` (`@sveltejs/mcp@0.1.26`) | — |
| DBHub (MySQL, MariaDB, SQLite...) | `npx` (`@bytebase/dbhub@1.5.0`) | **Arquivo dbhub.toml** |
| Build82 | O binário da release (veja abaixo) | — |
| Sync82 | O binário da release (veja abaixo) | — |
| SQLite | Fora do catálogo (o servidor foi arquivado e permite escrita): só pode ser removido; use o DBHub | — |
| Godot Studio | Instalação manual (veja abaixo) | — |

Um servidor com parâmetro precisa dele antes de **Instalar** ou **Atualizar**: clique em **Selecionar** para escolher a pasta ou o arquivo (um caminho absoluto). Sem o lançador (`npx`, `gopls`), o servidor não é registrado e a mensagem diz o que instalar. A falta do Google Chrome gera só um aviso.

No DBHub, os bancos são configurados no `dbhub.toml` (`[[sources]]` com o `dsn`); para somente leitura, adicione `[[tools]]` com `name = "execute_sql"` e `readonly = true`. A senha fica nesse arquivo: mantenha-o fora do git.

**Dart + Flutter** é um plugin: **Atualizar** usa `claude plugin update` no Claude Code, e remove e adiciona o plugin de novo no Codex, que não tem atualização de plugin.

### Build82 e Sync82

| MCP | Oferece | Repositório | Binário |
|---|---|---|---|
| Build82 | Contexto de uma instalação do Moodle e dos plugins em desenvolvimento | `oito2/mcp-build82` | `~/.local/bin/build82` |
| Sync82 | Memória de projeto persistente, compartilhada entre os agentes (cofre SQLite local, por padrão em `~/.sync82`) | `oito2/mcp-sync82` | `~/.local/bin/sync82` |

- **Instalar** baixa do GitHub o arquivo `<nome>_linux_<arquitetura>` da última release estável do repositório, confere com o `checksums.txt` da release (SHA-256), instala e registra o caminho dele, sem argumentos, nos agentes; quando nenhum agente aceita e nenhum outro escopo o usa, o binário é removido de novo. A release e o SHA-256 do binário ficam salvos em `mcp_binaries`, e a lista mostra **Versão instalada: …**.
- **Atualizar** instala a última release quando ela é mais nova que a instalada e registra o servidor de novo.
- **Remover** tira o registro e apaga o binário só quando nenhum outro escopo ainda o registra.
- Um arquivo no caminho do binário que não foi instalado pelo Perci (o SHA-256 não confere com o registro) nunca é substituído nem apagado; a instalação falha até ele ser removido.

Depois de instalar o Build82, peça ao agente para indicar a pasta do Moodle pela ferramenta `init_moodle_context`.

### Godot Studio

O Godot Studio não tem instalação automatizada; **Ver instruções** abre o repositório dele (`hi-godot/godot-ai`). Baixe uma versão publicada em GitHub Releases e extraia em `<projeto>/addons/godot_ai/` (não sobrescreva uma árvore antiga). Abra o Godot a partir da pasta do projeto e, no dock do Godot AI, clique em **Configure**: o comando de conexão só existe com o editor rodando.
