# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Each app is versioned on its own, following [Semantic Versioning](https://semver.org/spec/v2.0.0.html): Perci82 System releases are tagged `system-vX.Y.Z` and Perci82 Dev releases `dev-vX.Y.Z`.

## [Unreleased]

### Fixed

- The package repository at `https://oito2.github.io/perci82/` lacked Perci82 Dev 1.2.0: when both apps are tagged on the same commit, GitHub Pages kept the first release's site. Each publication now deploys with its own build version, and the site can be rebuilt by hand (Actions → Pages → Run workflow).
- `install.sh` stopped with "This release has no WebKitGTK 6.0" on releases that have it (Ubuntu 24.04 and its derivatives among them): its check failed whenever `apt-cache` kept writing after the line it looked for.

## [Dev 1.2.0] - 2026-11-19

### Added

- Perci82 Dev is published as `.deb` and `.rpm` packages for amd64 (Debian 13+, Ubuntu 24.04+, Linux Mint 22+, LMDE 7+, Zorin OS 18+, Pop!_OS 24.04+ and Fedora 40+), in a signed `apt`/`dnf` repository at `https://oito2.github.io/perci82/` (key `A1B1 AC75 B535 0344 B6C8  043C FF9E 1B0F 3C2A 545F`), so `apt upgrade` and `dnf upgrade` update it. Each release also attaches the packages, their `checksums.txt` and build attestations.
- Perci82 Dev: Overview → Updates shows, when a newer release exists and the app runs from its package, the command that installs it, with **Copy**.
- Perci82 Dev: after **Uninstall**, a window shows the command that removes the package, with **Copy** and **Close the app**.

### Changed

- One install script, `install.sh`, installs Perci82 Dev, Perci82 System or both: it asks which ones, or takes `system`, `dev` or `all` as arguments. It adds the package repository (checking the key's fingerprint) and installs the packages; `install-system.sh` and `install-dev.sh` are gone.
- Perci82 Dev is installed as `/usr/bin/perci82-dev`; an older `/usr/local/bin/perci82-dev` comes first in `PATH` and must be removed (`sudo rm -f /usr/local/bin/perci82-dev`).
- Perci82 Dev: the update check reads the newest version from the repository site (`latest/dev.json`) and falls back to the GitHub API, authenticated with the GitHub CLI's token when there is one.
- Perci82 Dev: **Uninstall** removes the app's settings and data but no longer its binary, menu entry or icons, which belong to the package, and no longer closes the window on its own.

### Removed

- Perci82 Dev: the built-in self-update (**Update** button). The package manager updates the app.
- The standalone binary (`perci82-dev-linux-amd64`) and the menu entry archive (`perci82-dev-menu.tar.gz`) are no longer published.

### Fixed

- The installer and the update check failed with HTTP 403 when GitHub's limit for requests without login (60 an hour per address) was reached, common on shared networks: the installer no longer uses the GitHub API, and the update check asks it only when the repository site can't answer.

## [System 1.2.0] - 2026-11-19

### Added

- Perci82 System is published as `.deb` and `.rpm` packages for amd64 (Debian 13+, Ubuntu 24.04+, Linux Mint 22+, LMDE 7+, Zorin OS 18+, Pop!_OS 24.04+ and Fedora 40+), in a signed `apt`/`dnf` repository at `https://oito2.github.io/perci82/` (key `A1B1 AC75 B535 0344 B6C8  043C FF9E 1B0F 3C2A 545F`), so `apt upgrade` and `dnf upgrade` update it. Each release also attaches the packages, their `checksums.txt` and build attestations.
- Perci82 System: Overview → Updates shows, when a newer release exists and the app runs from its package, the command that installs it, with **Copy**.
- Perci82 System: after **Uninstall**, a window shows the command that removes the package, with **Copy** and **Close the app**.

### Changed

- One install script, `install.sh`, installs Perci82 System, Perci82 Dev or both: it asks which ones, or takes `system`, `dev` or `all` as arguments. It adds the package repository (checking the key's fingerprint) and installs the packages; `install-system.sh` and `install-dev.sh` are gone.
- Perci82 System is installed as `/usr/bin/perci82-system`; an older `/usr/local/bin/perci82-system` comes first in `PATH` and must be removed (`sudo rm -f /usr/local/bin/perci82-system`).
- Perci82 System: the update check reads the newest version from the repository site (`latest/system.json`) and falls back to the GitHub API, authenticated with the GitHub CLI's token when there is one.
- Perci82 System: **Uninstall** removes the app's settings and data but no longer its binary, menu entry or icons, which belong to the package, and no longer closes the window on its own.

### Removed

- Perci82 System: the built-in self-update (**Update** button). The package manager updates the app.
- The standalone binary (`perci82-system-linux-amd64`) and the menu entry archive (`perci82-system-menu.tar.gz`) are no longer published.

### Fixed

- The installer and the update check failed with HTTP 403 when GitHub's limit for requests without login (60 an hour per address) was reached, common on shared networks: the installer no longer uses the GitHub API, and the update check asks it only when the repository site can't answer.

## [Dev 1.1.0] - 2026-11-18

### Added

- Perci82 Dev: Moodle apps have a **Behat test site** option (off by default). When on, Nginx also serves the app's folder at `http://<folder>-behat.test` over HTTP, without the `/behat/` and `/fixtures/` blocks of the app's own site, and the Nginx container answers to that name on the Docker network, so Behat in the app container reaches it. Turning the option on or off reconnects Nginx to the network without recreating it. The name stays outside `.localhost`, which PHP's curl and Chrome resolve to the loopback address.
- Perci82 Dev: **Behat configuration** in Docker → Manage Containers shows the lines to add to the app's `config.php` (Perci82 Dev never edits it) and the commands that initialize the test site and run the scenarios of the app's mounted plugins, each with **Copy**.
- Perci82 Dev: an optional shared **Selenium** container (`selenium/standalone-chrome:4.52.0-20261101`, 2 GB `/dev/shm`, two sessions, noVNC at `http://localhost:7900`) runs Chrome for the `@javascript` Behat scenarios of every Moodle app. It is created, managed, updated, exported and imported like Nginx and MariaDB, removed by the uninstall with the other containers, and doesn't start again on its own after the computer restarts. `selenium` is now a reserved app folder name.
- Perci82 Dev: the PHP Perci82 AI context of Moodle apps gains a "Moodle Tests" section — PHPUnit, and the Behat site, `config.php` lines and commands, following the app's Behat option and the Selenium container. Projects with the context applied show as outdated until updated.

### Changed

- Perci82 Dev: the Moodle AI context explains that Behat can't use a `*.localhost` site name inside containers, that the Behat site must not block `/behat/` and `/fixtures/`, and that `@javascript` scenarios need a WebDriver browser set in `$CFG->behat_profiles`.

## [Dev 1.0.2] - 2026-11-17

### Fixed

- Perci82 Dev: the PHP images lacked the `en_AU.UTF-8` locale, so Moodle's PHPUnit refused to initialize ("Required locale 'en_AU.UTF-8' is not installed"). The images now generate `en_AU.UTF-8` and `pt_BR.UTF-8`; existing containers get them with Docker → Gerenciar Contêineres → Recriar, which rebuilds the image.

## [System 1.0.1] - 2026-11-16

### Fixed

- Perci82 System: the symbolic app icon is drawn with filled shapes, so desktops that recolor symbolic icons show the gear instead of a solid shape.

## [Dev 1.0.1] - 2026-11-16

### Fixed

- Perci82 Dev: the system tray icon did nothing when clicked, and turning the tray on with the app already open showed an icon without a menu; turning it off now removes the icon at once. The tray now uses its own StatusNotifierItem implementation.
- Perci82 Dev: the tray icon was always white, invisible on light panels. It is now the symbolic app icon, drawn by the panel in its own text color, with the white or graphite pictogram as fallback following the desktop's color scheme. The symbolic app icon is drawn with filled shapes, so desktops can recolor it.

## [Dev 1.0.0] - 2026-11-16

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
- Contextos de IA → Gerenciar Contextos: the "Contextos do Perci82" tab lists the standards Perci82 Dev ships, each with Visualizar (read only), and the "Contextos personalizados" tab creates, views, edits and deletes your own contexts (name, an optional "Quando usar" sentence and the content), saved in `~/.perci/user-contexts/`. They show in Aplicar Contexto under "Contextos Personalizados" and are written as `.instructions/CUSTOM-<ID>.md`, listed in `AGENTS.md` and loaded by Claude Code in every session. Editing one marks the projects that use it as outdated; deleting one is refused while a registered folder uses it. The AI contexts export file carries the custom contexts its folders use, and importing it creates the missing ones.

### Changed

- The tray opens only the Contêineres window; the Atualizar Sistema shortcut was removed.
- Visão Geral shows the Docker containers running, the Git accounts and their repositories, and the version with an update badge. Desinstalar opens a confirmation window with the options: the Docker, repositories and AI contexts backups (each saved to its own file), removing the Docker containers, and "Também remover configurações".
- The login autostart entry is `~/.config/autostart/io.github.oito2.perci82.dev.desktop`.
- Releases are tagged `dev-vX.Y.Z`. Self-update looks only at Perci82 Dev's releases, and uninstall removes only its binary, menu entry and icons; "Também remover configurações" removes only its settings, its Docker stack files and its Git account files, and `~/.perci` only when neither Perci82 System nor the old Perci is installed and nothing else is left in it.
- Perci82 Dev shares `~/.perci/config.yaml` safely with Perci82 System: each app writes only its own settings and leaves the others, including settings it doesn't know, exactly as they were. Settings saved by Perci 1.9 and its export files keep working.
- Actions that need administrator rights never run in both apps at once: while one app runs such an action, the other refuses with a message naming that app.
- Building from source requires Go 1.27.4 or later.
- New window layout: sections as tabs at the top (Visão Geral, Ambiente, Docker, Repositórios, Contextos de IA, Skills & MCPs, Configurações), a side list for the screens of a section (the Git accounts are grouped under "Contas" in Repositórios; Contextos de IA has Aplicar Contexto, Gerenciar Contextos and Exportar/Importar), and the screen's own tabs below its title. The window is at least 1180 × 640 px.
- The Svelte MCP entry is named "Svelte MCP"; its description says it is the official one.
- The Moodle and PHP + Slim standards no longer describe the `php-<app>` wrappers or the Docker setup; that is in PHP Perci82.
- AI contexts no longer keep `vendor/` out of the agents' reach (`.claude/settings.json`, `.aiexclude`, `.geminiignore`): applying or updating a context removes that rule from projects that have it.
- Themes come from the oito2 design system: Configurações → Tema offers "Definido pelo sistema", which follows the desktop's light or dark preference, accent color and high contrast (and changes with them while the app is open), and the 13 oito2 presets (Everforest, Rosé Pine, Nord, Dracula/Alucard, Catppuccin). A daisyUI theme saved by Perci 1.9 shows as "Definido pelo sistema" until another theme is chosen.
- Contextos de IA → Aplicar Contexto (formerly Configurar Projeto) holds the registered folders in its "Contextos aplicados" tab (formerly Gerenciar Contextos), with more space between the form's fields; Exportar/Importar has an Exportar and an Importar tab; the project type that matches no preset reads "Customizado".
- Repositórios → Gerenciar Repositórios is now Repositórios → Exportar/Importar, as in Docker.
- Docker → Pastas de Desenvolvimento → Pastas montadas: the button that rewrites outdated mount units reads "Corrigir".
- The export files of Docker, Repositórios and Contextos de IA, and the uninstall backups, get the `.yaml` extension when the name typed in the save window has no `.yaml` or `.yml` extension.
- Focused text fields and selects no longer show a thick outline around them; their border takes the theme's accent color instead.
- Ambiente installs and updates prerequisites and Flutter's packages only on Ubuntu 24.04, Debian 13, Linux Mint 22, LMDE 7, Zorin OS 18, Pop!_OS 24.04 or later: older releases lack packages such as `libfuse2t64` and `openjdk-21-jdk`, and get a message naming the release needed.

### Removed

- The sidebar mascot's app icon choice (Ícone do Aplicativo): the app has a single icon.
- The daisyUI themes of Perci 1.9 (light, dark, cupcake, synthwave, retro, valentine, halloween, garden).
- The sidebar, its breadcrumb and the mascot logo, replaced by the section tabs; the sidebar logo choice (Logo da Sidebar) in Configurações went first, and a logo saved by Perci 1.9 is kept in `config.yaml` but no longer shown.

### Fixed

- The PHP images pick the PHPUnit release by the image's full PHP version: 9.6 for PHP 7.4 and 8.0 (they got 12.5 and 10.5, which don't run there), 10.5 for 8.1, 11.5 for 8.2 and 8.3, and 12.5 for 8.4.
- Ambiente → Docker Engine installs on Debian, where Compose v2 is the `docker-compose` package (`docker-compose-v2` exists only on Ubuntu and its derivatives).
- Contextos de IA no longer fails to preview, update or remove a project whose Claude Code rule for PHP Perci82 is still there after the standard was unchecked or its Docker app was removed.
- Contextos aplicados → Alterar opens the Aplicar contexto tab on the row's folder (it opened without a folder).
- The "Tipo de projeto" select no longer shows its first option as the raw key `aic.custom` when the screen loads before the translations.
- "Também remover configurações" removes `~/.perci/appstack` only together with the Docker containers, which still use it, and always keeps the MariaDB backups in `~/.perci/appstack/backups`.
- Two projects in different places with the same folder name no longer share a notes folder: the second gets `<name>-2` (`-3`…), saved in its context's record as `notes`, so it never moves.
- Ambiente finds the tools whose installers only change the shell startup files when Perci82 Dev was opened from the desktop menu: Go (`/usr/local/go/bin`, `~/go/bin`), Rust (`~/.cargo/bin`) and OpenCode (`~/.opencode/bin`). The Wails3 CLI no longer reads "Requires Go" after Go is installed, and OpenCode shows as installed and can be removed.
- Uninstalling Libsecret Tools on Fedora keeps the `libsecret` package, which GNOME Shell, gvfs and KWallet need: the removal no longer stops.
- A Flutter clone that fails or is cancelled no longer leaves a `~/development/flutter` that blocks the next install.
- Listing an account's repositories tells a token without permission (HTTP 403) apart from the rate limit.
- An account folder with `*`, `?`, `[` or `\` in its path is refused: Git reads them as a pattern in the `includeIf` rule, which then missed the account's repositories.
- When saving an account fails, its `includeIf` rule and gitconfig file are put back as they were.
- An app folder must start with a letter or digit and have at least 2 characters, as Docker requires for a container name: such names were accepted and failed only after the image was built.
- Editing an app never creates a second container for a folder that isn't registered, and switching an app's command validates its entry first.
- In PHP + Node apps, the start command and the worker run with `HOME=/tmp`, like the app's command: npm and npx no longer fail on `/root`.
- Docker → Importar applies exactly the file the tab listed: a file changed after it was chosen is refused.
- Self-update has no fixed 5-minute limit: a slow download is never cut (it stops only when no data arrives for a minute), and the password dialog of the final step is never interrupted.
- The Execução tab shows only its own action: a window no longer takes the output or the result of an action another window (the tray's) started.
- Docker → Gerenciar: Iniciar, Parar and Reiniciar run one at a time, so a second click no longer shows an error.
- Contextos de IA → Exportar/Importar's save dialog and file name follow the language; the Node.js-missing message of npm installs, the keyring label of the account tokens and the repository files' result message are translated.
- Uninstalling VS Code or VSCodium also removes the repository and key their install added, in the same password prompt; uninstalling Android Studio or Antigravity IDE asks first, like every other uninstall.
- Testing an account's connection names a private key others can read as the cause (`chmod 600`), instead of telling to register the key on the provider.
- The account's repository list says when it shows only the first 1000.
- A `config.yaml` that is a symbolic link (dotfiles managers) stays one when the settings are saved.
- Opening the window or ending an action never waits more than 5 seconds for the shell that refreshes `PATH` (a slow `nvm.sh`).

### Security

- Repositórios → Arquivos never writes through a file or folder that is a link to outside the repository (a cloned repository could point `SUPPORT.md` at `~/.bashrc`); that file is skipped with the reason, and `.perci-bak` copies keep the original's permissions.
- Listing an account's repositories refuses a redirect to another host or from https to http, so the API token never leaves the account's server.
- Importing Git accounts refuses a new account whose folder in the file is outside the home folder or the home folder itself, until a folder is picked; importing Docker settings never mounts a development folder outside the home folder, the home folder itself or one inside a hidden folder of it; importing AI contexts refuses folders that are the home folder, above it, a hidden folder of it or led out of the workspace by a link, until a folder is picked.
- The agents' settings files an AI context merges into (`.claude/settings.json`, `.gemini/settings.json`, `opencode.json`) keep their permissions and get no `.perci-bak` copy, since they may hold tokens.
- An MCP entry whose command or arguments were changed (an API key, an extra option) is no longer taken for Perci's: Instalar, Atualizar and Remover leave it alone, and Antigravity's own entries keep the keys Perci does not write, such as `env`.
- `install-dev.sh` checks, when the GitHub CLI is logged in, that `checksums.txt` was attested by the release workflow (covering the binary and the menu entry archive), and extracts only menu entries and icons from the archive.
- Installing or updating the Antigravity IDE refuses an extracted app folder or a `chrome-sandbox` replaced by a symbolic link while the password dialog is open, so the setuid bit can only land on the verified copy in `/opt/antigravity`.
- The systemd units of Docker → Pastas de Desenvolvimento refuse to mount when a folder on the way to the development folder or to the destination has become a symbolic link, and a destination reached through a link is refused when the mount is created. Mounts made earlier show a warning on the Pastas montadas tab with Atualizar unidades, which rewrites their units.
- The SSH command of the Git accounts accepts only the server keys in Perci82 Dev's `known_hosts` (`StrictHostKeyChecking=yes`), whatever the user's `~/.ssh/config` says; repository URLs from an import file reach git only in Perci's own SSH form; a key's passphrase goes to ssh-keygen and ssh-add through a 0600 file instead of an environment variable.
- In `opencode.json`, Perci's `allow` permissions go first, so the user's later `deny` or `ask` rules keep the last word (OpenCode applies the last matching rule).
- The nvm installer is run only after its SHA-256 matches.
- A folder chosen in a file dialog is accepted only for the setting it was chosen for: the workspace, the notes folder and the Filesystem MCP folder each need their own choice, and the notes folder must be a folder.

## [System 1.0.0] - 2026-11-16

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
- Focused text fields and selects no longer show a thick outline around them; their border takes the theme's accent color instead.
- Pós-instalação's Aplicar buttons sit further from the window's right edge, so the scrollbar no longer covers them.
- Pós-instalação offers the Linux Mint profiles from Linux Mint 22 and the Zorin OS profiles from Zorin OS 18 on; older releases (and LMDE) are shown as not supported, since they lack packages the profiles install, such as `libfuse2t64`.

### Removed

- The app icon choice (Ícone do Aplicativo): the app has a single icon.
- The daisyUI themes of Perci 1.9 (light, dark, cupcake, synthwave, retro, valentine, halloween, garden).
- The sidebar, its breadcrumb and the mascot logo, replaced by the section tabs, and the sidebar logo choice (Logo da Sidebar).

### Fixed

- Gerenciar Fontes installs Noto from `fonts-noto-core`, the package that holds "Noto Sans": removing the `fonts-noto` metapackage left the font installed and checked.
- Self-update has no fixed 5-minute limit: a slow download is never cut (it stops only when no data arrives for a minute), and the password dialog of the final step is never interrupted.
- The Execução tab shows only the action this window started.
- `make release` writes one `checksums.txt` covering both applications.
- Pós-instalação's "Enabling the … component" message is translated.
- A `config.yaml` that is a symbolic link (dotfiles managers) stays one when the settings are saved.
- Perci82 System no longer refreshes `PATH` from a login shell at start and after each action: only Perci82 Dev installs tools that need it.

### Security

- `install-system.sh` checks, when the GitHub CLI is logged in, that `checksums.txt` was attested by the release workflow (covering the binary and the menu entry archive), and extracts only menu entries and icons from the archive.

[Unreleased]: https://github.com/oito2/perci82/compare/dev-v1.2.0...HEAD
[Dev 1.2.0]: https://github.com/oito2/perci82/releases/tag/dev-v1.2.0
[System 1.2.0]: https://github.com/oito2/perci82/releases/tag/system-v1.2.0
[Dev 1.1.0]: https://github.com/oito2/perci82/releases/tag/dev-v1.1.0
[Dev 1.0.2]: https://github.com/oito2/perci82/releases/tag/dev-v1.0.2
[System 1.0.1]: https://github.com/oito2/perci82/releases/tag/system-v1.0.1
[Dev 1.0.1]: https://github.com/oito2/perci82/releases/tag/dev-v1.0.1
[Dev 1.0.0]: https://github.com/oito2/perci82/releases/tag/dev-v1.0.0
[System 1.0.0]: https://github.com/oito2/perci82/releases/tag/system-v1.0.0
