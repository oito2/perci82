# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Each app is versioned on its own, following [Semantic Versioning](https://semver.org/spec/v2.0.0.html): Perci82 System releases are tagged `system-vX.Y.Z` and Perci82 Dev releases `dev-vX.Y.Z`.

## [Unreleased]

### Added

- **Perci82 System** (`perci82-system`) and **Perci82 Dev** (`perci82-dev`): two desktop apps split from Perci 1.9.0. System has the Linux screens (post-install, system update, fonts, file templates, Flatpak apps); Dev has the development screens (Desenvolvimento, Docker, Repositórios, Dev Tools) and the system tray.
- New icons for both apps, under their application IDs `io.github.oito2.perci82.system` and `io.github.oito2.perci82.dev`.
- Perci82 System's Visão Geral shows the detected system, the desktop environment and the version (with an "up to date" or "update available" badge after each check); Desinstalar opens a confirmation window with the "Também remover configurações" option.
- Installers per app, `install-system.sh` and `install-dev.sh`: they install the app's newest release with its application menu entry and offer to remove the old Perci (`prci`), keeping `~/.perci`.
- Configurações → Idioma in each app chooses between "Definido pelo sistema" (Portuguese when the system language is Portuguese, English otherwise), Português (Brasil) and English, saved per app (`system_language` in Perci82 System, `language` in Perci82 Dev). The switch applies at once, without a restart, including the tray menu of Perci82 Dev. Perci82 System is fully translated, including the progress messages of its actions; technical error details are shown in English. Perci82 Dev's screens are translated as they are migrated.

### Changed

- The tray of Perci82 Dev opens only the Contêineres window; the Atualizar Sistema shortcut was removed.
- The login autostart entry of Perci82 Dev is `~/.config/autostart/io.github.oito2.perci82.dev.desktop`.
- Releases are per app, tagged `system-vX.Y.Z` or `dev-vX.Y.Z`.
- Self-update looks only at the app's own releases, and uninstall removes only the app's own binary, menu entry and icons; "Também remover configurações" removes only that app's settings, and `~/.perci` only when no other Perci82 app nor the old Perci is installed and nothing else is left in it.
- Both apps share `~/.perci/config.yaml` safely: each app writes only its own settings and leaves the others, including settings it doesn't know, exactly as they were. Settings saved by Perci 1.9 and its export files keep working.
- Perci82 System saves its theme apart from Perci82 Dev's (`system_gui_theme`), so the two apps can use different themes.
- Actions that need administrator rights never run in both apps at once: while one app runs such an action, the other refuses with a message naming that app.
- Building from source requires Go 1.27.4 or later.
- New window layout in both apps: sections as tabs at the top (Perci82 System: Visão Geral, Pós-instalação, Atualizar, Fontes, Templates, Flatpaks, Configurações; Perci82 Dev: Visão Geral, Ambiente, Docker, Repositórios, Contextos de IA, Skills & MCPs, Configurações), a side list for the screens of a Perci82 Dev section (the Git accounts are grouped under "Contas" in Repositórios), and the screen's own tabs below its title. Section, screen and tab names follow the chosen language. The window is at least 1180 × 640 px.
- AI contexts no longer keep `vendor/` out of the agents' reach (`.claude/settings.json`, `.aiexclude`, `.geminiignore`): applying or updating a context removes that rule from projects that have it.
- Themes come from the oito2 design system: Configurações → Tema offers "Definido pelo sistema", which follows the desktop's light or dark preference, accent color and high contrast (and changes with them while the app is open), and the 13 oito2 presets (Everforest, Rosé Pine, Nord, Dracula/Alucard, Catppuccin). A daisyUI theme saved by Perci 1.9 shows as "Definido pelo sistema" until another theme is chosen.

### Removed

- The sidebar mascot's app icon choice (Ícone do Aplicativo): each app has a single icon.
- The daisyUI themes of Perci 1.9 (light, dark, cupcake, synthwave, retro, valentine, halloween, garden).
- The sidebar, its breadcrumb and the mascot logo, replaced by the section tabs; the sidebar logo choice (Logo da Sidebar) in Configurações went first, and a logo saved by Perci 1.9 is kept in `config.yaml` but no longer shown.
