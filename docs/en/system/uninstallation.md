🌐 [Português](../../pt-br/system/uninstallation.md) | **English** | 🏠 [Index](../index.md)

---

# Uninstalling Perci82 System

## What installing puts on the system

| Item | Path |
|---|---|
| Binary | `/usr/local/bin/perci82-system` |
| Application menu entry | `/usr/share/applications/io.github.oito2.perci82.system.desktop` |
| Icons | `/usr/share/icons/hicolor/<size>x<size>/apps/io.github.oito2.perci82.system.png` (16, 22, 24, 32, 48, 64, 128, 256 and 512), `/usr/share/icons/hicolor/scalable/apps/io.github.oito2.perci82.system.svg`, `/usr/share/icons/hicolor/symbolic/apps/io.github.oito2.perci82.system-symbolic.svg` |
| Settings | `~/.perci/config.yaml` — shared with Perci82 Dev; Perci82 System's keys are listed in [Settings](settings.md#where-the-settings-are-saved) |

Perci82 System has no autostart entry, tray icon or service.

## From the app

1. Open **Overview** and click **Uninstall Perci82 System** in the **Uninstall** section.
2. In the confirmation window, check **Also remove the Perci82 System settings** if you want its settings gone too (language, theme, Flatpak scope and the post-install records). Perci82 Dev's settings are kept.
3. Click **Uninstall** and type your password when asked. The **Execution** tab shows the progress.

The binary, the menu entry and the icons are removed. With the settings option, Perci82 System's keys are removed from `~/.perci/config.yaml`; the `~/.perci` folder itself goes only when neither Perci82 Dev nor the old Perci is installed and nothing else is left in it.

## By hand

```bash
sudo rm -f /usr/local/bin/perci82-system \
  /usr/share/applications/io.github.oito2.perci82.system.desktop \
  /usr/share/icons/hicolor/*/apps/io.github.oito2.perci82.system.png \
  /usr/share/icons/hicolor/scalable/apps/io.github.oito2.perci82.system.svg \
  /usr/share/icons/hicolor/symbolic/apps/io.github.oito2.perci82.system-symbolic.svg
sudo gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor
```

To remove the settings, delete the keys `postinstall_applied`, `postinstall_machine_id`, `flatpak_scope`, `system_gui_theme` and `system_language` from `~/.perci/config.yaml`. Delete the whole `~/.perci` folder only when Perci82 Dev isn't installed: it keeps its own settings and data there.

## What the app's actions leave behind

Uninstalling removes the app, not what its screens installed. Undo those first if you want them gone:

| Screen | Leaves | How to remove |
|---|---|---|
| [Fonts](fonts.md) | Packages, and `~/.local/share/fonts/perci/` for JetBrains Mono | Uncheck the fonts and click **Run** |
| [Templates](templates.md) | Files in your templates folder | Uncheck the templates and click **Run** |
| [Flatpaks](flatpaks.md) | Flatpak applications | Uncheck them and click **Run** |
| [Post-install](post-install.md) | Packages, repositories, `/etc/sysctl.d/99-perci.conf`, `/swapfile` and its `/etc/fstab` line, the `fstrim.timer` service | By hand, with your distribution's tools — post-install has no undo |
