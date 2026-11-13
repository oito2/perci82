🌐 [Português](../../pt-br/dev/settings.md) | **English** | 🏠 [Index](../index.md)

---

# Settings

Perci82 Dev's own preferences. Every change is saved and applied at once; there is no Save button.

![Settings](../../img/dev-settings.png)
<!-- TODO: capture a screenshot of Perci82 Dev's Settings -->

## General

**Language** chooses the language of the whole app, including the progress messages of its actions and the tray menu:

- **System default** — Portuguese when the system language (`LC_ALL`, `LC_MESSAGES` or `LANG`) is Portuguese, English otherwise.
- **Português (Brasil)**.
- **English**.

The switch applies at once, without a restart. Technical error details are shown in English. Perci82 System has its own language setting.

## Theme

- **System default** follows the desktop's light or dark preference, accent color and high contrast, and changes with them while the app is open.
- The 13 oito2 presets: Everforest Light, Everforest Dark, Rosé Pine, Rosé Pine Moon, Rosé Pine Dawn, Nord Dark, Nord Light, Dracula, Alucard, Catppuccin Latte, Catppuccin Frappé, Catppuccin Macchiato and Catppuccin Mocha.

Perci82 Dev keeps its theme apart from Perci82 System's.

## Workspace path

The folder that holds the [Docker](docker.md) apps' files (`<workspace>/localhost/`) and that the [AI Contexts](ai-contexts.md) export writes paths relative to. Without a saved path, Perci82 Dev uses `~/workspace`.

**Select Folder** opens the folder chooser (where you can also create a folder); **Create Workspace** saves the chosen folder. Only a folder chosen in that window is accepted. Choose the workspace before creating Docker containers: their folders are created under it.

## Project notes

A folder outside the repositories with one subfolder per project that has an AI context: `PROJECT.md` (the project's rules), `notes/`, `plans/` and `references/`. The coding agents get access to it when a context is applied or updated. See [AI Contexts](ai-contexts.md).

**Choose folder** saves the folder at once (only a folder chosen in this dialog, not one chosen for another setting); **Turn off** clears it, and each project keeps its rules in its own `.instructions/PROJECT.md`. After changing this folder, use **Update** in **AI Contexts → Manage Contexts**.

## System tray

**Show Perci82 Dev in the system tray** turns the tray icon on or off at once (it is off by default). When on:

- closing the main window hides it in the tray instead of quitting; use **Quit** in the icon's menu to quit for good;
- Perci82 Dev starts hidden in the tray with the graphical session, through `~/.config/autostart/io.github.oito2.perci82.dev.desktop` (removed when the tray is turned off).

The icon needs a desktop with StatusNotifierItem support: KDE, XFCE and COSMIC show it natively; GNOME shows it only with the "AppIndicator and KStatusNotifierItem Support" extension. The tray menu is described in [Overview](overview.md#system-tray).

## Where the settings are saved

In `~/.perci/config.yaml`, shared with Perci82 System. Each app writes only its own keys and keeps the others as they are, including keys it doesn't know. A `config.yaml` that is a symbolic link (a dotfiles manager's) stays one: its target is written.

| Key | Holds |
|---|---|
| `language` | The language: `system`, `pt-BR` or `en` |
| `gui_theme` | The theme |
| `workspace_path` | The workspace folder |
| `project_notes_path` | The project notes folder |
| `tray_enabled` | Whether the system tray is on |
| `docker` | The Docker stack and its apps |
| `moodle_plugins` | The development folders mounted into the apps |
| `git_accounts` | The Git accounts and their repository settings |
| `ai_contexts` | The folders with an AI context |
| `agent_skills` | The skills installed, per scope |
| `mcp_servers` | The MCP servers registered, per scope |
| `mcp_binaries` | The MCP server binaries Perci82 Dev installed, with their version and SHA-256 |

Perci82 Dev also keeps the Docker stack files in `~/.perci/appstack/` and the Git accounts' gitconfig files in `~/.perci/git/`. Settings saved by Perci 1.9 keep working.
