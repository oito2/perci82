# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Each app is versioned on its own, following [Semantic Versioning](https://semver.org/spec/v2.0.0.html): Perci82 System releases are tagged `system-vX.Y.Z` and Perci82 Dev releases `dev-vX.Y.Z`.

## [Unreleased]

### Added

- **Perci82 System** (`perci82-system`) and **Perci82 Dev** (`perci82-dev`): two desktop apps split from Perci 1.9.0. System has the Linux screens (post-install, system update, fonts, file templates, Flatpak apps); Dev has the development screens (Desenvolvimento, Docker, Repositórios, Dev Tools) and the system tray.
- New icons for both apps, under their application IDs `io.github.oito2.perci82.system` and `io.github.oito2.perci82.dev`.

### Changed

- The tray of Perci82 Dev opens only the Contêineres window; the Atualizar Sistema shortcut was removed.
- The login autostart entry of Perci82 Dev is `~/.config/autostart/io.github.oito2.perci82.dev.desktop`.
- Releases are per app, tagged `system-vX.Y.Z` or `dev-vX.Y.Z`.
- Both apps share `~/.perci/config.yaml` safely: each app writes only its own settings and leaves the others, including settings it doesn't know, exactly as they were. Settings saved by Perci 1.9 and its export files keep working.
- Perci82 System saves its theme apart from Perci82 Dev's (`system_gui_theme`), so the two apps can use different themes.
- Actions that need administrator rights never run in both apps at once: while one app runs such an action, the other refuses with a message naming that app.
- Building from source requires Go 1.27.4 or later.

### Removed

- The sidebar mascot's app icon choice (Ícone do Aplicativo): each app has a single icon.
- The sidebar logo choice (Logo da Sidebar) in Configurações; a logo chosen in Perci 1.9 is still shown.
- Self-update and uninstall from the apps are not available yet; their buttons report that.
