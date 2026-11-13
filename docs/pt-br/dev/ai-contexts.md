🌐 [English](../../en/dev/ai-contexts.md) | **Português** | 🏠 [Índice](../index.md)

---

# Contextos de IA

Um contexto de IA é o conjunto de arquivos de regras que o Perci82 Dev grava na pasta de um projeto para os agentes de programação: as regras comuns no `AGENTS.md`, um padrão por tecnologia em `.instructions/` e os arquivos próprios de cada agente. A lista lateral tem três telas: **Configurar Projeto**, **Gerenciar Contextos** e **Exportar/Importar**.

![Contextos de IA](../../img/dev-ai-contexts.png)
<!-- TODO: capture a screenshot of AI Contexts → Set Up Project -->

## O que um contexto grava

| Arquivo | Gravado para | Contém |
|---|---|---|
| `AGENTS.md` | Todos os agentes | As regras comuns, a lista de padrões ativos (cada um com as tarefas que cobre) e um comentário final com os agentes |
| `.instructions/<PADRÃO>.md` | Todos os agentes | Um arquivo por padrão marcado |
| `.instructions/PROJECT.md` | Todos os agentes | As regras do próprio projeto; criado uma vez e nunca reescrito (veja [Anotações de projeto](#anotações-de-projeto)) |
| `CLAUDE.md` | Claude Code | Importa o `AGENTS.md` e o `.instructions/PROJECT.md` |
| `.claude/rules/perci/*.md` | Claude Code | Os padrões que cobrem tipos de arquivo, carregados só quando o Claude Code trabalha num arquivo correspondente |
| `.claude/settings.json` | Claude Code | Regras de leitura que deixam segredos e pastas de dependências fora de alcance, mais os [acessos dos agentes](#acessos-dos-agentes) |
| `GEMINI.md` | Gemini | Importa o `AGENTS.md` e o `.instructions/PROJECT.md` |
| `.aiexclude` / `.geminiignore` | Gemini | Arquivos mantidos fora do Gemini Code Assist e do Gemini CLI |
| `.gemini/settings.json` | Gemini | Os acessos do agente |
| `.codex/config.toml` / `.codex/rules/perci.rules` | ChatGPT CLI (Codex) | Os acessos do agente |
| `opencode.json` | OpenCode | Uma entrada em `instructions` que carrega as regras do projeto, mais os acessos do agente |

O Codex e o Antigravity leem o `AGENTS.md` como está. Os arquivos mantidos fora são credenciais (`.env`, `*.pem`, `*.key`, `config.php`, `wp-config.php`, `auth.json`…), pastas de dependências e de dados (`node_modules/`, `moodledata/`, `.git/`…), pastas de IDE e saídas de teste e de profiling; `dist/`, `build/`, `out/`, `env/` e `.next/` continuam legíveis quando o git rastreia arquivos nelas.

Um arquivo substituído fica com uma cópia com o sufixo `.perci-bak`. Um `CLAUDE.md` ou `GEMINI.md` que é um link para o `AGENTS.md` é mantido. O Perci82 Dev não altera o `opencode.jsonc`; ele informa qual entrada adicionar manualmente.

## Configurar Projeto

1. Clique em **Selecionar pasta** e escolha a pasta do projeto (**Trocar pasta** escolhe outra).
2. Informe o **Nome**: até 80 caracteres, único entre as pastas registradas (sem diferenciar maiúsculas).
3. Marque os **Agentes de IA**: ao menos um.
4. Escolha um **Tipo de projeto** ou marque os padrões um a um.
5. Clique em **Aplicar**. Uma confirmação lista cada arquivo que será criado, atualizado ou removido; arquivos substituídos ficam com uma cópia `.perci-bak`.

Aplicar grava os arquivos dos agentes e padrões marcados, remove os que o Perci gravou para os desmarcados (só enquanto não foram editados) e registra a pasta em **Gerenciar Contextos**. Quando nenhum arquivo muda, só o registro é atualizado.

### Agentes de IA

| Agente | ID no `config.yaml` |
|---|---|
| Claude Code | `claude` |
| Gemini (CLI + Code Assist) | `gemini` |
| ChatGPT CLI (Codex) | `codex` |
| OpenCode | `opencode` |
| Antigravity | `antigravity` |

### Tipos de projeto

Um tipo de projeto marca um conjunto de padrões de uma vez; as caixas continuam editáveis, e mudar uma delas volta o tipo para **Personalizado**. Para uma pasta ainda sem padrão, o tipo indicado pelos arquivos dela já vem escolhido ("Sugerido pelos arquivos da pasta.").

| Tipo de projeto | Padrões marcados | Sugerido quando |
|---|---|---|
| Servidor MCP em Go | Go, Go: Servidor MCP, Documentação de projeto, Documentação: Servidor MCP | O `go.mod` requer `github.com/modelcontextprotocol/go-sdk` |
| Aplicativo Go + Wails | Go, Go: Wails v3, Documentação de projeto, Documentação: Aplicação Desktop | O `go.mod` requer `github.com/wailsapp/wails/v3` |
| Plugin Moodle | Moodle, PHP Perci82, MySQL/MariaDB, Documentação de projeto, Documentação: Plugin Moodle | O `version.php` define `$plugin->component` |
| Aplicativo Rust + Tauri v2 | Rust + Tauri v2, Documentação de projeto, Documentação: Aplicação Desktop | A pasta tem `src-tauri/` |
| Website | Website, Documentação de projeto, Documentação: Aplicação Web | — |
| Sistema PHP + Slim | PHP + Slim 4, PHP Perci82, MySQL/MariaDB, Documentação de projeto, Documentação: Aplicação Web | O `composer.json` requer `slim/slim` |
| Jogo Godot (Android) | Godot 4 (GDScript), Documentação de projeto, Documentação: Jogo | A pasta tem `project.godot` |

Aplicativo Rust + Tauri v2, Website e Sistema PHP + Slim mostram uma dica do que marcar manualmente: o complemento do frontend, Landing Page Design ou o complemento da camada de dados.

### Padrões

Um complemento aparece abaixo do padrão base; marcá-lo marca também a base, e Aplicar inclui sozinho uma base que faltar.

| Padrão | Arquivo | Cobre |
|---|---|---|
| Linux Bash | `BASH.md` | Scripts Bash/shell, ferramentas de linha de comando e instaladores |
| Go | `GO.md` | Código Go: estrutura, erros, concorrência, testes, lint e releases |
| Go: Servidor MCP | `GO-MCP.md` | Servidor MCP com o SDK oficial em Go (complementa o Go) |
| Go: Wails v3 | `WAILS.md` | Aplicativo desktop com Wails v3 (complementa o Go) |
| Rust + Tauri v2 | `TAURI.md` | Aplicativo Rust + Tauri v2: comandos, permissões, CSP, build e atualizador |
| Tauri: Vanilla TS + Vite | `TAURI-VANILLA.md` | Frontend em TypeScript puro com Vite (complementa o Tauri) |
| Tauri: SvelteKit | `TAURI-SVELTEKIT.md` | Frontend em SvelteKit, estático (complementa o Tauri) |
| Tauri: React + Vite | `TAURI-REACT.md` | Frontend em React + TypeScript com Vite (complementa o Tauri) |
| PHP + Slim 4 | `PHP-SLIM.md` | Sistemas PHP com Slim 4: estrutura, rotas, middleware, injeção de dependências, segurança e testes |
| Slim: PDO | `SLIM-PDO.md` | Acesso a dados com PDO (complementa o Slim) |
| Slim: Doctrine DBAL | `SLIM-DBAL.md` | Acesso a dados com Doctrine DBAL (complementa o Slim) |
| Slim: Eloquent | `SLIM-ELOQUENT.md` | Acesso a dados com Eloquent (complementa o Slim) |
| Moodle | `MOODLE.md` | Plugins Moodle: estrutura de arquivos, APIs e arquitetura |
| PHP Perci82 | `PHP-PERCI82.md` | Como rodar PHP, as ferramentas, os testes e o banco no contêiner Docker do projeto (veja [abaixo](#php-perci82)) |
| Dart + Flutter | `FLUTTER.md` | Apps Dart/Flutter: arquitetura, estado, testes e publicação |
| Godot 4 (GDScript) | `GODOT.md` | Jogos em Godot 4 com GDScript, publicados no Android |
| MySQL/MariaDB | `MYSQL-MARIADB.md` | Esquema, migrations, otimização de consultas e configuração |
| TypeScript/Node | `TYPESCRIPT-NODE.md` | Backends, CLIs e bibliotecas em TypeScript e Node.js |
| Docker/Compose | `DOCKER.md` | Dockerfiles e Compose: imagens, build, segurança e execução |
| CI (GitHub Actions) | `GITHUB-ACTIONS.md` | Workflows: gatilhos, permissões, segurança, cache e releases |
| Website | `WEBSITE.md` | Sites: estrutura, acessibilidade, desempenho, SEO e hospedagem |
| Landing Page Design | `LANDING-PAGE.md` | Design e construção de landing pages |
| Documentação de projeto | `PROJECT-DOCUMENTATION.md` | README, docs/, CONTRIBUTING, licença, arquivos do GitHub e releases |
| Documentação: Servidor MCP | `PROJECT-DOCUMENTATION-MCP.md` | Documentação de um servidor MCP (complementa a Documentação de projeto) |
| Documentação: Plugin Moodle | `PROJECT-DOCUMENTATION-MOODLE.md` | Documentação de um plugin Moodle |
| Documentação: Aplicação Desktop | `PROJECT-DOCUMENTATION-DESKTOP.md` | Documentação de um aplicativo desktop |
| Documentação: Aplicação Mobile | `PROJECT-DOCUMENTATION-MOBILE.md` | Documentação de um aplicativo mobile |
| Documentação: Aplicação Web | `PROJECT-DOCUMENTATION-WEB.md` | Documentação de uma aplicação web |
| Documentação: Jogo | `PROJECT-DOCUMENTATION-GAME.md` | Documentação de um jogo |

O `MOODLE.md` pega a versão do Moodle do `version.php` da pasta (ou `public/version.php`); sem ele, diz "unknown".

Cada padrão que já está na pasta mostra um selo:

| Selo | Significado |
|---|---|
| **aplicado** | Igual ao que esta versão do Perci82 Dev gera |
| **desatualizado** | Gerado por uma versão anterior do Perci: Aplicar atualiza |
| **editado** | Editado à mão: Aplicar substitui, guardando um `.perci-bak` |

### Registrar um contexto existente

Uma pasta cujos arquivos já têm um contexto gerado pelo Perci, mas que não está em **Gerenciar Contextos**, mostra **Contexto não registrado** com os padrões e agentes encontrados. Informe o nome e clique em **Registrar**: a pasta entra na lista e nenhum arquivo é alterado.

## Gerenciar Contextos

A tabela lista as pastas registradas com **Nome**, **Pasta**, **Padrões**, **Agentes** e **Estado**.

| Estado | Significado |
|---|---|
| **em dia** | Os arquivos são o que esta versão do Perci82 Dev gera para o contexto registrado |
| **desatualizado** | Gerado por uma versão anterior do Perci ou com arquivos faltando |
| **editado** | O `AGENTS.md` tem regras escritas à mão, ou um padrão registrado foi editado |
| **pendente** | A pasta não existe nesta máquina; depois de criada (por exemplo, clonando o repositório), aparece como desatualizada |

Um selo **seleção diferente** indica que os arquivos da pasta têm outros padrões ou agentes que os registrados.

| Ação | O que faz |
|---|---|
| **Atualizar** | Regera os padrões e agentes registrados, depois de uma confirmação com as mudanças; arquivos editados são substituídos, guardando um `.perci-bak` |
| **Atualizar todos** | O mesmo para todas as pastas desatualizadas ou editadas; uma pasta que não existe é pulada |
| **Alterar** | Abre a pasta em **Configurar Projeto** com o nome, os agentes e os padrões registrados |
| **Remover** | Remove os arquivos do Perci ainda não editados e tira a pasta da lista; arquivos editados e o `.instructions/PROJECT.md` ficam. Uma pasta que não existe só sai da lista |

**Atualizar** e **Alterar** ficam desligados para uma pasta pendente.

## Exportar/Importar

**Exportar** grava as pastas marcadas, com nomes, padrões e agentes, num arquivo YAML (`perci-contextos-ia.yaml` por padrão) legível só por você. Uma pasta dentro do workspace (`workspace_path`, ou `~/workspace` quando não definido) é gravada com o caminho relativo a ele; as demais mantêm o caminho absoluto.

**Importar** lê esse arquivo (**Selecionar arquivo**) e lista cada pasta:

| Rótulo | Significado |
|---|---|
| **recebe o contexto** | A pasta existe aqui: o contexto é gerado e ela é registrada |
| **pendente: a pasta não existe** | Ela é registrada como pendente em **Gerenciar Contextos** |
| **já registrada aqui** | Mantida como está |
| **indisponível** | Não pode ser importada; o motivo é mostrado |

Caminhos relativos vão para o workspace desta máquina. Cada pasta recebe um nome nesta máquina (**Nome nesta máquina**), único aqui e na importação, e **Trocar pasta** escolhe outra pasta para ela. Padrões e agentes que esta versão não conhece ficam de fora e são listados.

## Anotações de projeto

Com **Configurações → Anotações de projeto** definida (salva como `project_notes_path`; veja [Configurações](settings.md)), cada projeto ganha uma pasta de anotações fora do repositório, `<pasta de anotações>/<nome da pasta do projeto>/`:

```text
<pasta de anotações>/<projeto>/
├── PROJECT.md      # as regras do projeto, no lugar do .instructions/PROJECT.md
├── notes/          # anotações e análises
├── plans/          # planos de implementação
└── references/     # material de referência
```

A pasta é criada ao **Aplicar** ou **Atualizar**. O `AGENTS.md` ganha uma seção "Project Notes", e o `CLAUDE.md`, o `GEMINI.md` e o `opencode.json` carregam o `PROJECT.md` das anotações. As regras do projeto passam para lá:

- As regras do `.instructions/PROJECT.md` entram no `PROJECT.md` das anotações (sob um título "Moved from …" quando ele já existe), e o arquivo antigo é renomeado para `.instructions/PROJECT.md.perci-bak`; um modelo intocado é apagado.
- Regras escritas à mão no `AGENTS.md` também passam para lá.

Sem a configuração, nada muda: cada projeto mantém as regras no `.instructions/PROJECT.md`, e regras escritas à mão no `AGENTS.md` passam para ele. Depois de mudar a pasta de anotações, use **Atualizar** em **Gerenciar Contextos**.

## Acessos dos agentes

Um contexto deixa cada agente marcado acessar a pasta de anotações do projeto (leitura e escrita) e rodar sem perguntar o comando do app do [PHP Perci82](#php-perci82). Cada acesso vai para as configurações por projeto do agente:

| Agente | Arquivo | Pasta de anotações | Comando do app |
|---|---|---|---|
| Claude Code | `.claude/settings.json` | `permissions.additionalDirectories` e uma regra `Edit(…)` | `Bash(<app> *)` em `permissions.allow` |
| Gemini | `.gemini/settings.json` | `context.includeDirectories` | `run_shell_command(<app>)` em `tools.allowed` |
| ChatGPT CLI (Codex) | `.codex/config.toml`, `.codex/rules/perci.rules` | `writable_roots` de `[sandbox_workspace_write]` | Uma `prefix_rule` com `decision="allow"` |
| OpenCode | `opencode.json` | Permissões `external_directory` e `edit` | Uma permissão `shell` |

O Antigravity não tem configuração por projeto para isso. Entradas escritas por você ficam; as do Perci que não valem mais são retiradas. Um arquivo de configurações que não é JSON válido fica como está, com um aviso. Os arquivos de `.codex/` são do próprio Perci: um sem a primeira linha `# Generated by Perci82 Dev` não é alterado.

## PHP Perci82

O padrão **PHP Perci82** (nos tipos Plugin Moodle e Sistema PHP + Slim) grava o `.instructions/PHP-PERCI82.md` a partir das configurações do app do [Docker](docker.md) que contém o projeto, como raiz do projeto ou como pasta de desenvolvimento montada (apps Node não contam). Ele descreve:

- O app: tipo, versões, URL e as pastas no host e no contêiner.
- O comando do app (por exemplo `loja composer install`) e as ferramentas ativas: Composer, PHPUnit, PHP_CodeSniffer, npm nos apps PHP + Node, e o padrão `moodle` e os scripts de CLI nos apps Moodle.
- O banco: host, porta, banco e usuário, quando o app tem acesso ao banco. A senha nunca é gravada: o arquivo diz que ela só está no ambiente do app (`DB_PASS`).
- As pastas de desenvolvimento montadas, os logs e os reinícios.

A primeira linha do arquivo traz uma assinatura SHA-256 do restante. Uma cópia assinada, gravada para configurações anteriores do app, aparece como **desatualizado**, e **Atualizar** a regrava; uma cópia alterada à mão não confere com a assinatura e aparece como **editado**. Quando nenhum app do Docker contém a pasta, o padrão é pulado (a confirmação mostra "não gerado (nenhum app do Docker usa esta pasta)").
