# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Each app is versioned on its own, following [Semantic Versioning](https://semver.org/spec/v2.0.0.html): Perci82 System releases are tagged `system-vX.Y.Z` and Perci82 Dev releases `dev-vX.Y.Z`.

## [Unreleased]

## [Dev 1.0.0] - 2026-11-13

First release of Perci82 Dev.

### Added

- Documentation of every Perci82 Dev screen, of installing and of uninstalling, in English (`docs/en/dev/`) and Portuguese (`docs/pt-br/dev/`).
- **Perci82 Dev** (`perci82-dev`), split from Perci 1.9.0: the development screens (Ambiente, Docker, Repositórios, Contextos de IA, Skills & MCPs) and the system tray.
- A new icon, under the application ID `io.github.oito2.perci82.dev`.
- The installer `install-dev.sh`: it installs Perci82 Dev's newest release with its application menu entry and offers to remove the old Perci (`prci`), keeping `~/.perci`.
- Configurações → Idioma chooses between "Definido pelo sistema" (Portuguese when the system language is Portuguese, English otherwise), Português (Brasil) and English, saved as `language`. The switch applies at once, without a restart, including the tray menu. Every screen is translated, including the progress messages of the actions; technical error details are shown in English.
- Docker → Pastas de Desenvolvimento, which replaces Dev Tools → Plugins Moodle, mounts a development folder into any app container: as a plugin in a Moodle app (as before), or at a folder relative to the project root in PHP, Genérico, Node and PHP + Node apps (inside `api/` or `app/` for PHP + Node). Nginx is restarted only for the apps it serves. The mounts stay recorded under `moodle_plugins`, so the settings and Docker export files of Perci 1.9 keep working.
- Each app container gets one command named after its folder in `~/.local/bin` (for example `loja composer install`, `mdle-401 phpcs --standard=moodle local/x`, `site npm test`): a subcommand picks the tool (composer, phpunit, phpcs, phpcbf, and npm in Node and PHP + Node apps), anything else runs through `php` (`node` in Node apps), the working folder inside the container follows the current folder, and host paths into the project are rewritten. It replaces the `php-<app>`, `composer-<app>`, `phpcs-<app>`, `phpcbf-<app>`, `phpunit-<app>` and `npm-<app>` commands when the container is created, recreated or edited; Gerenciar Contêineres offers to switch an app without recreating it. When another program already has the app's name, the command is not created and the old ones are kept.
- Criar Contêiner and Editar have a checklist of the command's tools (Composer, PHPUnit, PHP_CodeSniffer and, for Moodle, moodle-cs), saved as `disabled_tools`; a tool turned off answers that it isn't enabled. The PHP image adds the Moodle coding standard (moodle-cs 3.7.0, with PHPCSExtra 1.5.1 and PHPCSUtils 1.2.3, checked by SHA-256), which phpcs and phpcbf know in the Moodle apps with moodle-cs on.
- Configurações → Anotações de projeto sets a folder for project notes, saved as `project_notes_path`. With it, applying an AI context gives the project a notes folder `<folder>/<project>` (PROJECT.md, notes/, plans/, references/) outside the repository: the project rules move there from `.instructions/PROJECT.md` (kept as `PROJECT.md.perci-bak`; rules written in AGENTS.md move too), and the agent files point at it and let Claude Code, Gemini, Codex and OpenCode read and write it (`.claude/settings.json`, `.gemini/settings.json`, `.codex/config.toml`, `opencode.json`). Without the setting, nothing changes.
- The "PHP Perci82" standard (in the Moodle plugin and PHP + Slim presets) writes `.instructions/PHP-PERCI82.md` from the settings of the Docker app that holds the project, as its project root or a mounted development folder: its command and tools, versions, URL, folders, database (without the password), mounts and logs. It is rewritten when the app changes and skipped when no app holds the project. It also lets the agents run the app's command without asking (`.claude/settings.json`, `.gemini/settings.json`, `.codex/rules/perci.rules`, `opencode.json`).
- Skills & MCPs → MCPs offers Build82 (Moodle context) and Sync82 (project memory): installing one downloads the binary of its latest release from GitHub (`oito2/mcp-build82`, `oito2/mcp-sync82`), checks it against the release's `checksums.txt` (SHA-256), installs it as `~/.local/bin/build82` or `~/.local/bin/sync82` and registers it with Claude Code, Codex and Antigravity. Atualizar installs a newer release when there is one; Remover deletes the binary once no scope registers it. The installed release is shown in the list and saved under `mcp_binaries`. A binary or a registration with the same name that Perci did not create is kept, never replaced. Build82's Moodle folder is set afterwards through its `init_moodle_context` tool.
- The "Open in <terminal>" entries that Terminais adds to the Nautilus, Nemo and Dolphin context menus follow the chosen language.

### Changed

- The tray opens only the Contêineres window; the Atualizar Sistema shortcut was removed.
- Visão Geral shows the Docker containers running, the Git accounts and their repositories, and the version with an update badge. Desinstalar opens a confirmation window with the options: the Docker, repositories and AI contexts backups (each saved to its own file), removing the Docker containers, and "Também remover configurações".
- The login autostart entry is `~/.config/autostart/io.github.oito2.perci82.dev.desktop`.
- Releases are tagged `dev-vX.Y.Z`. Self-update looks only at Perci82 Dev's releases, and uninstall removes only its binary, menu entry and icons; "Também remover configurações" removes only its settings, its Docker stack files and its Git account files, and `~/.perci` only when neither Perci82 System nor the old Perci is installed and nothing else is left in it.
- Perci82 Dev shares `~/.perci/config.yaml` safely with Perci82 System: each app writes only its own settings and leaves the others, including settings it doesn't know, exactly as they were. Settings saved by Perci 1.9 and its export files keep working.
- Actions that need administrator rights never run in both apps at once: while one app runs such an action, the other refuses with a message naming that app.
- Building from source requires Go 1.27.4 or later.
- New window layout: sections as tabs at the top (Visão Geral, Ambiente, Docker, Repositórios, Contextos de IA, Skills & MCPs, Configurações), a side list for the screens of a section (the Git accounts are grouped under "Contas" in Repositórios; Contextos de IA has Configurar Projeto, Gerenciar Contextos and Exportar/Importar), and the screen's own tabs below its title. The window is at least 1180 × 640 px.
- The Svelte MCP entry is named "Svelte MCP"; its description says it is the official one.
- The Moodle and PHP + Slim standards no longer describe the `php-<app>` wrappers or the Docker setup; that is in PHP Perci82.
- AI contexts no longer keep `vendor/` out of the agents' reach (`.claude/settings.json`, `.aiexclude`, `.geminiignore`): applying or updating a context removes that rule from projects that have it.
- Themes come from the oito2 design system: Configurações → Tema offers "Definido pelo sistema", which follows the desktop's light or dark preference, accent color and high contrast (and changes with them while the app is open), and the 13 oito2 presets (Everforest, Rosé Pine, Nord, Dracula/Alucard, Catppuccin). A daisyUI theme saved by Perci 1.9 shows as "Definido pelo sistema" until another theme is chosen.

### Removed

- The sidebar mascot's app icon choice (Ícone do Aplicativo): the app has a single icon.
- The daisyUI themes of Perci 1.9 (light, dark, cupcake, synthwave, retro, valentine, halloween, garden).
- The sidebar, its breadcrumb and the mascot logo, replaced by the section tabs; the sidebar logo choice (Logo da Sidebar) in Configurações went first, and a logo saved by Perci 1.9 is kept in `config.yaml` but no longer shown.

## [System 1.0.0] - 2026-11-12

First release of Perci82 System, split from Perci 1.9.0.

### Added

- **Perci82 System** (`perci82-system`): the Linux administration screens of Perci 1.9.0 — Pós-instalação, Atualizar (Atualizar SO & Apps), Fontes, Templates and Flatpaks — plus Visão Geral and Configurações.
- A new icon, under the application ID `io.github.oito2.perci82.system`.
- Visão Geral shows the detected system, the desktop environment and the version, with an "up to date" or "update available" badge after each check; **Atualizar** appears only when a newer release exists. **Desinstalar** opens a confirmation window with the "Também remover as configurações do Perci82 System" option.
- The installer `install-system.sh`: it installs Perci82 System's newest release with its application menu entry and offers to remove the old Perci (`prci`), keeping `~/.perci`.
- Configurações → Idioma chooses between "Definido pelo sistema" (Portuguese when the system language is Portuguese, English otherwise), Português (Brasil) and English, saved as `system_language`. The switch applies at once, without a restart. The app is fully translated, including the progress messages of its actions; technical error details are shown in English.
- Documentation of every screen, of installing and of uninstalling, in English (`docs/en/`) and Portuguese (`docs/pt-br/`).

### Changed

- Releases are tagged `system-vX.Y.Z`. Self-update looks only at Perci82 System's releases, and uninstall removes only its binary, menu entry and icons; "Também remover as configurações" removes only its settings (language, theme, Flatpak scope and the post-install records), and `~/.perci` only when neither Perci82 Dev nor the old Perci is installed and nothing else is left in it.
- Perci82 System shares `~/.perci/config.yaml` safely with Perci82 Dev: each app writes only its own settings and leaves the others, including settings it doesn't know, exactly as they were. Settings saved by Perci 1.9 keep working.
- Perci82 System saves its theme apart from Perci82 Dev's (`system_gui_theme`), so the two apps can use different themes.
- Actions that need administrator rights never run in both apps at once: while one app runs such an action, the other refuses with a message naming that app.
- Building from source requires Go 1.27.4 or later.
- New window layout: sections as tabs at the top (Visão Geral, Pós-instalação, Atualizar, Fontes, Templates, Flatpaks, Configurações) and the screen's own tabs (Itens, Execução) below its title. Section, screen and tab names follow the chosen language. The window is at least 1180 × 640 px.
- Flatpak scope options read "Sistema (todos os usuários)" and "Usuário (só você)".
- Themes come from the oito2 design system: Configurações → Tema offers "Definido pelo sistema", which follows the desktop's light or dark preference, accent color and high contrast (and changes with them while the app is open), and the 13 oito2 presets (Everforest, Rosé Pine, Nord, Dracula/Alucard, Catppuccin).

### Removed

- The app icon choice (Ícone do Aplicativo): the app has a single icon.
- The daisyUI themes of Perci 1.9 (light, dark, cupcake, synthwave, retro, valentine, halloween, garden).
- The sidebar, its breadcrumb and the mascot logo, replaced by the section tabs, and the sidebar logo choice (Logo da Sidebar).

[Unreleased]: https://github.com/oito2/perci82/commits/main
[Dev 1.0.0]: https://github.com/oito2/perci82/releases/tag/dev-v1.0.0
[System 1.0.0]: https://github.com/oito2/perci82/releases/tag/system-v1.0.0
