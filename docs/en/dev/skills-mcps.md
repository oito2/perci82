🌐 [Português](../../pt-br/dev/skills-mcps.md) | **English** | 🏠 [Index](../index.md)

---

# Skills & MCPs

Shortcuts to give coding agents third-party skills and MCP servers. The side list has two screens: **SKILLs** and **MCPs**.

![Skills & MCPs](../../img/dev-skills-mcps.png)
<!-- TODO: capture a screenshot of Skills & MCPs → MCPs -->

## Installation type

Both screens start with **Installation type**:

- **Global** — installs globally.
- **Local** — installs only for one project: click **Select Folder** to choose its folder.

Each list shows what is installed in the chosen scope. The installed entries are saved in `~/.perci/config.yaml`, per scope: `agent_skills` for SKILLs and `mcp_servers` for MCPs.

## SKILLs

Skills published on [skills.sh](https://www.skills.sh/), installed with the `skills` CLI pinned at version 1.7.0 and run through `npx` (Node.js is required). Each skill is installed from its repository at a commit Perci82 Dev pins, for Claude Code, OpenCode, Antigravity and Codex at once:

```bash
npx -y skills@1.7.0 add <repository>#<commit> --skill <name> --agent claude-code --agent opencode --agent antigravity --agent codex --yes [--global]
```

The CLI's telemetry is turned off (`DISABLE_TELEMETRY=1`). In the Local scope the command runs in the chosen folder.

> SKILLs are third-party content: Perci pins the version, but does not review the content, and a SKILL runs with the agent's permissions. Read the code before using it.

| Button | What it does |
|---|---|
| **Install** | Installs the skill. When the run for every agent fails, each agent is tried on its own |
| **Update** | Installs it again at the pinned commit |
| **Remove** | Removes it from the four agents. For an entry that installs a whole repository, only the skills installed from that repository are removed |

Badges next to the name:

- **partial: …** — some agent did not accept the skill; the badge names the ones that did.
- **out of the catalog** — the entry left Perci's catalog; it is listed only while installed, and can only be removed.

### Catalog

| SKILL | Repository |
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
| DaisyUI | `saadeghi/daisyui` (every skill in the repository) |
| ShaCN UI | `shadcn-ui/ui` |
| Frontend Design | `anthropics/skills` |
| Svelte: Code Writer (official) | `sveltejs/ai-tools` |
| Svelte: Best Practices (official) | `sveltejs/ai-tools` |
| React: Best Practices (Vercel) | `vercel-labs/agent-skills` |
| Vite | `antfu/skills` |
| Vitest | `antfu/skills` |
| Web: Accessibility, SEO and Performance | `addyosmani/web-quality-skills` (every skill in the repository) |
| Web: Testing with Playwright | `anthropics/skills` |
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
| MariaDB: Query Optimization (official) | `MariaDB/skills` |
| MariaDB: Features (official) | `MariaDB/skills` |
| SQL: Code Review | `github/awesome-copilot` |
| MCP Server Dev | `anthropics/skills` |
| MCP Server Dev in Go | `github/awesome-copilot` |
| MCP: Security Audit | `github/awesome-copilot` |
| Dart: Official | `dart-lang/skills` (every skill in the repository) |
| Flutter: Official | `flutter/agent-plugins` (every skill in the repository) |

All repositories are on GitHub (`https://github.com/<repository>`).

## MCPs

MCP servers registered with Claude Code, Codex and Antigravity, each through its own mechanism:

| Agent | How | Global | Local |
|---|---|---|---|
| Claude Code | `claude mcp add` / `claude mcp remove` | `--scope user` | `--scope local`, run in the chosen folder |
| Codex | `codex mcp add` / `codex mcp remove` | Yes | Skipped |
| Antigravity | Edits `mcp_config.json`, keeping the other entries | `~/.gemini/config/mcp_config.json` | `<folder>/.agents/mcp_config.json` |

Codex is only registered when the **Installation type** is Global: its per-project scope is not confirmed as released yet. A server with the same name that Perci did not register is kept, never replaced, and **Remove** leaves it alone.

| Button | What it does |
|---|---|
| **Install** | Registers the server with every agent that applies; fails only when none accepts it |
| **Update** | Removes and registers it again with the same parameter (Dart + Flutter: see below) |
| **Remove** | Unregisters it; an agent that no longer has it is only a warning |

A row shows **partial: …** when an agent that applies to the scope is missing, and **out of the catalog** for an entry that can only be removed.

### Catalog

| MCP | Needs | Parameter |
|---|---|---|
| Dart + Flutter | The `dart-flutter` plugin from the `flutter/agent-plugins` marketplace, for Claude Code and Codex; Antigravity has it natively | — |
| Filesystem MCP (@modelcontextprotocol/server-filesystem) | `npx` (`@modelcontextprotocol/server-filesystem@1.0.0`) | **Allowed folder** |
| Context7 | `npx` (`@upstash/context7-mcp@4.4.1`); queries are sent to context7.com | — |
| Go: gopls MCP | `gopls` on the `PATH` (`go install golang.org/x/tools/gopls@v0.23.0`) | — |
| Chrome DevTools MCP | `npx` (`chrome-devtools-mcp@1.10.1`, isolated profile, usage statistics off) and Google Chrome | — |
| Svelte MCP | `npx` (`@sveltejs/mcp@0.1.26`) | — |
| DBHub (MySQL, MariaDB, SQLite...) | `npx` (`@bytebase/dbhub@1.5.0`) | **dbhub.toml file** |
| Build82 | Its release binary (see below) | — |
| Sync82 | Its release binary (see below) | — |
| SQLite | Out of the catalog (the server was archived and allows writes): can only be removed; use DBHub | — |
| Godot Studio | Manual installation (see below) | — |

A server with a parameter needs it before **Install** or **Update**: click **Select** to choose the folder or file (an absolute path). Without its launcher (`npx`, `gopls`), the server is not registered and the message says what to install. A missing Google Chrome is only a warning.

For DBHub, the databases are configured in the `dbhub.toml` (`[[sources]]` with the `dsn`); for read-only access, add `[[tools]]` with `name = "execute_sql"` and `readonly = true`. The password stays in that file: keep it out of git.

**Dart + Flutter** is a plugin: **Update** uses `claude plugin update` in Claude Code, and removes and adds the plugin again in Codex, which has no plugin update.

### Build82 and Sync82

| MCP | Provides | Repository | Binary |
|---|---|---|---|
| Build82 | Context of a Moodle installation and of the plugins in development | `oito2/mcp-build82` | `~/.local/bin/build82` |
| Sync82 | Persistent project memory shared between agents (local SQLite vault, `~/.sync82` by default) | `oito2/mcp-sync82` | `~/.local/bin/sync82` |

- **Install** downloads the `<name>_linux_<arch>` asset of the repository's latest stable release from GitHub, checks it against the release's `checksums.txt` (SHA-256), installs it and registers its path, with no arguments, with the agents; when no agent accepts it and no other scope uses it, the binary is removed again. The release and the binary's SHA-256 are saved under `mcp_binaries`, and the list shows **Installed version: …**.
- **Update** installs the latest release when it is newer than the installed one, then registers the server again.
- **Remove** unregisters the server and deletes the binary only when no other scope still registers it.
- A file at the binary's path that Perci did not install (its SHA-256 doesn't match the record) is never replaced or deleted; install fails until it is removed.

After installing Build82, ask the agent to set the Moodle folder through its `init_moodle_context` tool.

### Godot Studio

Godot Studio has no automated installation; **See instructions** opens its repository (`hi-godot/godot-ai`). Download a published release from GitHub Releases and extract it into `<project>/addons/godot_ai/` (do not overwrite an old tree). Open Godot from the project folder and, in the Godot AI dock, click **Configure**: the connection command only exists while the editor runs.
