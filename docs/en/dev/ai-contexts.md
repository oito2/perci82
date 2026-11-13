🌐 [Português](../../pt-br/dev/ai-contexts.md) | **English** | 🏠 [Index](../index.md)

---

# AI Contexts

An AI context is the set of rule files Perci82 Dev writes into a project folder for coding agents: the shared rules in `AGENTS.md`, one standard per technology under `.instructions/`, and each agent's own files. The side list has three screens: **Set Up Project**, **Manage Contexts** and **Export/Import**.

![AI Contexts](../../img/dev-ai-contexts.png)
<!-- TODO: capture a screenshot of AI Contexts → Set Up Project -->

## What a context writes

| File | Written for | Holds |
|---|---|---|
| `AGENTS.md` | Every agent | The shared rules, the list of active standards (each with the tasks it covers) and a closing comment naming the agents |
| `.instructions/<STANDARD>.md` | Every agent | One file per checked standard |
| `.instructions/PROJECT.md` | Every agent | The project's own rules; created once and never rewritten (see [Project notes](#project-notes)) |
| `CLAUDE.md` | Claude Code | Imports `AGENTS.md` and `.instructions/PROJECT.md` |
| `.claude/rules/perci/*.md` | Claude Code | The standards that cover file types, loaded only when Claude Code works on a matching file |
| `.claude/settings.json` | Claude Code | Read rules that keep secrets and dependency folders out of reach, plus the [agent grants](#agent-grants) |
| `GEMINI.md` | Gemini | Imports `AGENTS.md` and `.instructions/PROJECT.md` |
| `.aiexclude` / `.geminiignore` | Gemini | Files kept out of Gemini Code Assist and Gemini CLI |
| `.gemini/settings.json` | Gemini | The agent grants |
| `.codex/config.toml` / `.codex/rules/perci.rules` | ChatGPT CLI (Codex) | The agent grants |
| `opencode.json` | OpenCode | An `instructions` entry that loads the project rules, plus the agent grants |

Codex and Antigravity read `AGENTS.md` as it is. The kept-out files are credentials (`.env`, `*.pem`, `*.key`, `config.php`, `wp-config.php`, `auth.json`…), dependency and data folders (`node_modules/`, `moodledata/`, `.git/`…), IDE folders and test/profiling output; `dist/`, `build/`, `out/`, `env/` and `.next/` stay readable when git tracks files in them.

A replaced file keeps a copy with the `.perci-bak` suffix. A `CLAUDE.md` or `GEMINI.md` that is a link to `AGENTS.md` is kept. Perci82 Dev does not change `opencode.jsonc`; it tells you which entry to add by hand.

## Set Up Project

1. Click **Choose folder** and pick the project folder (**Change folder** picks another one).
2. Enter a **Name**: up to 80 characters, unique among the registered folders (case is ignored).
3. Check the **AI agents**: at least one.
4. Choose a **Project type** or check the standards one by one.
5. Click **Apply**. A confirmation lists each file that will be created, updated or removed; replaced files keep a `.perci-bak` copy.

Applying writes the files of the checked agents and standards, removes the files Perci wrote for unchecked ones (only while they are still unedited), and registers the folder in **Manage Contexts**. When no file changes, only the record is updated.

### AI agents

| Agent | ID in `config.yaml` |
|---|---|
| Claude Code | `claude` |
| Gemini (CLI + Code Assist) | `gemini` |
| ChatGPT CLI (Codex) | `codex` |
| OpenCode | `opencode` |
| Antigravity | `antigravity` |

### Project types

A project type checks a set of standards at once; the checkboxes stay editable, and changing one sets the type back to **Custom**. For a folder with no standard yet, the type its files point to is chosen automatically ("Suggested by the folder's files.").

| Project type | Standards checked | Suggested when |
|---|---|---|
| MCP server in Go | Go, Go: MCP server, Project documentation, Documentation: MCP server | `go.mod` requires `github.com/modelcontextprotocol/go-sdk` |
| Go + Wails app | Go, Go: Wails v3, Project documentation, Documentation: Desktop app | `go.mod` requires `github.com/wailsapp/wails/v3` |
| Moodle plugin | Moodle, PHP Perci82, MySQL/MariaDB, Project documentation, Documentation: Moodle plugin | `version.php` sets `$plugin->component` |
| Rust + Tauri v2 app | Rust + Tauri v2, Project documentation, Documentation: Desktop app | The folder has `src-tauri/` |
| Website | Website, Project documentation, Documentation: Web app | — |
| PHP + Slim system | PHP + Slim 4, PHP Perci82, MySQL/MariaDB, Project documentation, Documentation: Web app | `composer.json` requires `slim/slim` |
| Godot game (Android) | Godot 4 (GDScript), Project documentation, Documentation: Game | The folder has `project.godot` |

Rust + Tauri v2 app, Website and PHP + Slim system show a hint of what to add by hand: the frontend complement, Landing Page Design, or the data layer complement.

### Standards

A complement is listed under its base standard; checking it checks the base too, and Apply adds a missing base by itself.

| Standard | File | Covers |
|---|---|---|
| Linux Bash | `BASH.md` | Bash/shell scripts, command-line tools and installers |
| Go | `GO.md` | Go code: layout, errors, concurrency, tests, lint and releases |
| Go: MCP server | `GO-MCP.md` | MCP server with the official Go SDK (complements Go) |
| Go: Wails v3 | `WAILS.md` | Desktop app with Wails v3 (complements Go) |
| Rust + Tauri v2 | `TAURI.md` | Rust + Tauri v2 app: commands, permissions, CSP, build and updater |
| Tauri: Vanilla TS + Vite | `TAURI-VANILLA.md` | Plain TypeScript frontend with Vite (complements Tauri) |
| Tauri: SvelteKit | `TAURI-SVELTEKIT.md` | SvelteKit frontend, static (complements Tauri) |
| Tauri: React + Vite | `TAURI-REACT.md` | React + TypeScript frontend with Vite (complements Tauri) |
| PHP + Slim 4 | `PHP-SLIM.md` | PHP systems with Slim 4: layout, routes, middleware, DI, security and tests |
| Slim: PDO | `SLIM-PDO.md` | Data access with PDO (complements Slim) |
| Slim: Doctrine DBAL | `SLIM-DBAL.md` | Data access with Doctrine DBAL (complements Slim) |
| Slim: Eloquent | `SLIM-ELOQUENT.md` | Data access with Eloquent (complements Slim) |
| Moodle | `MOODLE.md` | Moodle plugins: file layout, APIs and architecture |
| PHP Perci82 | `PHP-PERCI82.md` | Running PHP, its tools, tests and the database in the project's Docker container (see [below](#php-perci82)) |
| Dart + Flutter | `FLUTTER.md` | Dart/Flutter apps: architecture, state, tests and publishing |
| Godot 4 (GDScript) | `GODOT.md` | Godot 4 games with GDScript, published on Android |
| MySQL/MariaDB | `MYSQL-MARIADB.md` | Schema, migrations, query tuning and configuration |
| TypeScript/Node | `TYPESCRIPT-NODE.md` | TypeScript and Node.js backends, CLIs and libraries |
| Docker/Compose | `DOCKER.md` | Dockerfiles and Compose: images, build, security and running |
| CI (GitHub Actions) | `GITHUB-ACTIONS.md` | Workflows: triggers, permissions, security, cache and releases |
| Website | `WEBSITE.md` | Websites: structure, accessibility, performance, SEO and hosting |
| Landing Page Design | `LANDING-PAGE.md` | Landing page design and building |
| Project documentation | `PROJECT-DOCUMENTATION.md` | README, docs/, CONTRIBUTING, license, GitHub files and releases |
| Documentation: MCP server | `PROJECT-DOCUMENTATION-MCP.md` | Documentation of an MCP server (complements Project documentation) |
| Documentation: Moodle plugin | `PROJECT-DOCUMENTATION-MOODLE.md` | Documentation of a Moodle plugin |
| Documentation: Desktop app | `PROJECT-DOCUMENTATION-DESKTOP.md` | Documentation of a desktop app |
| Documentation: Mobile app | `PROJECT-DOCUMENTATION-MOBILE.md` | Documentation of a mobile app |
| Documentation: Web app | `PROJECT-DOCUMENTATION-WEB.md` | Documentation of a web app |
| Documentation: Game | `PROJECT-DOCUMENTATION-GAME.md` | Documentation of a game |

`MOODLE.md` takes the Moodle version from the folder's `version.php` (or `public/version.php`); without one it says "unknown".

Each standard already in the folder shows a badge:

| Badge | Meaning |
|---|---|
| **applied** | The same as what this version of Perci82 Dev generates |
| **outdated** | Generated by an earlier version of Perci: Apply updates it |
| **edited** | Edited by hand: Apply replaces it, keeping a `.perci-bak` |

### Registering an existing context

A folder whose files already hold a context generated by Perci, but that is not in **Manage Contexts**, shows **Context not registered** with the standards and agents found. Enter the name and click **Register**: the folder joins the list and no file changes.

## Manage Contexts

The table lists the registered folders with **Name**, **Folder**, **Standards**, **Agents** and **State**.

| State | Meaning |
|---|---|
| **up to date** | The files are what this version of Perci82 Dev generates for the registered context |
| **outdated** | Generated by an earlier version of Perci, or with files missing |
| **edited** | `AGENTS.md` has rules written by hand, or a registered standard was edited |
| **pending** | The folder doesn't exist on this machine; once it is created (for example, by cloning the repository), it shows as outdated |

A **different selection** badge means the folder's files hold other standards or agents than the registered ones.

| Action | What it does |
|---|---|
| **Update** | Regenerates the registered standards and agents, after a confirmation listing the changes; edited files are replaced, keeping a `.perci-bak` |
| **Update all** | The same for every outdated or edited folder; a missing folder is skipped |
| **Change** | Opens the folder in **Set Up Project** with its registered name, agents and standards |
| **Remove** | Removes the Perci files still unedited and drops the folder from the list; edited files and `.instructions/PROJECT.md` stay. A missing folder is only dropped from the list |

**Update** and **Change** are off for a pending folder.

## Export/Import

**Export** saves the checked folders, with their names, standards and agents, to a YAML file (`perci-contextos-ia.yaml` by default) readable only by you. A folder inside the workspace (`workspace_path`, or `~/workspace` when it is not set) is saved with its path relative to the workspace; any other keeps its absolute path.

**Import** reads such a file (**Choose file**) and lists each folder:

| Label | Meaning |
|---|---|
| **gets the context** | The folder exists here: its context is generated and it is registered |
| **pending: the folder doesn't exist** | It is registered as pending in **Manage Contexts** |
| **already registered here** | Kept as it is |
| **unavailable** | It can't be imported; the reason is shown |

Relative paths go into this machine's workspace. Each folder gets a name on this machine (**Name on this machine**), unique here and in the import, and **Change folder** picks another folder for it. Standards and agents this version doesn't know are left out and listed.

## Project notes

With **Settings → Project notes** set (saved as `project_notes_path`; see [Settings](settings.md)), each project gets a notes folder outside its repository, `<project notes folder>/<project folder name>/`:

```text
<project notes folder>/<project>/
├── PROJECT.md      # the project's rules, in place of .instructions/PROJECT.md
├── notes/          # notes and analyses
├── plans/          # implementation plans
└── references/     # reference material
```

The folder is created on **Apply** or **Update**. `AGENTS.md` gets a "Project Notes" section, and `CLAUDE.md`, `GEMINI.md` and `opencode.json` load the notes `PROJECT.md`. The project's rules move there:

- Rules in `.instructions/PROJECT.md` join the notes `PROJECT.md` (under a "Moved from …" heading when the notes file already exists), and the old file is renamed `.instructions/PROJECT.md.perci-bak`; an untouched template is deleted instead.
- Rules written by hand in `AGENTS.md` move too.

Without the setting, nothing changes: each project keeps its rules in `.instructions/PROJECT.md`, and hand-written rules found in `AGENTS.md` move there. After changing the notes folder, use **Update** in **Manage Contexts**.

## Agent grants

A context lets each checked agent reach the project notes folder (read and write) and run the [PHP Perci82](#php-perci82) app command without asking. Each grant goes into the agent's per-project settings:

| Agent | File | Notes folder | App command |
|---|---|---|---|
| Claude Code | `.claude/settings.json` | `permissions.additionalDirectories` and an `Edit(…)` rule | `Bash(<app> *)` in `permissions.allow` |
| Gemini | `.gemini/settings.json` | `context.includeDirectories` | `run_shell_command(<app>)` in `tools.allowed` |
| ChatGPT CLI (Codex) | `.codex/config.toml`, `.codex/rules/perci.rules` | `writable_roots` of `[sandbox_workspace_write]` | A `prefix_rule` with `decision="allow"` |
| OpenCode | `opencode.json` | `external_directory` and `edit` permissions | A `shell` permission |

Antigravity has no per-project setting for this. Entries you wrote yourself stay; Perci's entries that no longer apply are taken out. A settings file that isn't valid JSON is left as it is, with a warning. The `.codex/` files are Perci's own: one without its `# Generated by Perci82 Dev` first line is not changed.

## PHP Perci82

The **PHP Perci82** standard (in the Moodle plugin and PHP + Slim system types) writes `.instructions/PHP-PERCI82.md` from the settings of the [Docker](docker.md) app that holds the project, as its project root or as a mounted development folder (Node apps don't count). It describes:

- The app: type, versions, URL and the host and container folders.
- The app's command (for example `loja composer install`) and its enabled tools: Composer, PHPUnit, PHP_CodeSniffer, npm in PHP + Node apps, and the `moodle` standard and CLI scripts in Moodle apps.
- The database: host, port, database and user, when the app has database access. The password is never written: the file says it is only in the app's environment (`DB_PASS`).
- The mounted development folders, logs and restarts.

The file's first line carries a SHA-256 signature of the rest. A signed copy written for earlier app settings shows as **outdated**, and **Update** rewrites it; a copy changed by hand fails the signature and shows as **edited**. When no Docker app holds the folder, the standard is skipped (the confirmation shows "not generated (no Docker app uses this folder)").
