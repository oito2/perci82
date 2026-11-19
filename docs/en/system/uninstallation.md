🌐 [Português](../../pt-br/system/uninstallation.md) | **English** | 🏠 [Index](../index.md)

---

# Uninstalling Perci82 System

## What installing puts on the system

The package `perci82-system` holds the binary, the menu entry, the icons and the license; the package manager removes them. The rest stays until removed as below.

| Item | Path |
|---|---|
| Binary | `/usr/bin/perci82-system` |
| Application menu entry | `/usr/share/applications/io.github.oito2.perci82.system.desktop` |
| Icons | `/usr/share/icons/hicolor/<size>x<size>/apps/io.github.oito2.perci82.system.png` (16, 22, 24, 32, 48, 64, 128, 256 and 512), `/usr/share/icons/hicolor/scalable/apps/io.github.oito2.perci82.system.svg`, `/usr/share/icons/hicolor/symbolic/apps/io.github.oito2.perci82.system-symbolic.svg` |
| License | `/usr/share/doc/perci82-system/copyright` (`.deb`) or `/usr/share/licenses/perci82-system/LICENSE` (`.rpm`) |
| Package repository | Shared with Perci82 Dev. `apt`: `/etc/apt/sources.list.d/perci82.sources` and `/usr/share/keyrings/perci82-archive-keyring.asc`; `dnf`: `/etc/yum.repos.d/perci82.repo`, `/etc/pki/rpm-gpg/RPM-GPG-KEY-perci82` and the key imported into rpm (`gpg-pubkey-3c2a545f`) |
| Settings | `~/.perci/config.yaml` — shared with Perci82 Dev; Perci82 System's keys are listed in [Settings](settings.md#where-the-settings-are-saved) |

Perci82 System has no autostart entry, tray icon or service.

## From the app

1. Open **Overview** and click **Uninstall Perci82 System** in the **Uninstall** section.
2. In the confirmation window, check **Also remove the Perci82 System settings** if you want its settings gone too (language, theme, Flatpak scope and the post-install records). Perci82 Dev's settings are kept.
3. Click **Uninstall**. The **Run** tab shows the progress.
4. When it finishes, the **Cleanup finished** window shows the command that removes the package, with **Copy**. Copy it, click **Close the app** and run the command in a terminal:

   ```bash
   sudo apt remove perci82-system   # apt
   sudo dnf remove perci82-system   # dnf
   ```

With the settings option, Perci82 System's keys are removed from `~/.perci/config.yaml`; the `~/.perci` folder itself goes only when neither Perci82 Dev nor the old Perci is installed and nothing else is left in it.

## By hand

```bash
sudo apt remove perci82-system   # or: sudo dnf remove perci82-system
```

To remove the settings, delete the keys `postinstall_applied`, `postinstall_machine_id`, `flatpak_scope`, `system_gui_theme` and `system_language` from `~/.perci/config.yaml`. Delete the whole `~/.perci` folder only when Perci82 Dev isn't installed: it keeps its own settings and data there.

## Removing the package repository

Once neither Perci82 app is installed, remove the repository and its key:

```bash
# apt
sudo rm -f /etc/apt/sources.list.d/perci82.sources /usr/share/keyrings/perci82-archive-keyring.asc
sudo apt update
# dnf
sudo rm -f /etc/yum.repos.d/perci82.repo /etc/pki/rpm-gpg/RPM-GPG-KEY-perci82
sudo rpm -e gpg-pubkey-3c2a545f
```

## What the app's actions leave behind

Uninstalling removes the app, not what its screens installed. Undo those first if you want them gone:

| Screen | Leaves | How to remove |
|---|---|---|
| [Fonts](fonts.md) | Packages, and `~/.local/share/fonts/perci/` for JetBrains Mono | Uncheck the fonts and click **Run** |
| [Templates](templates.md) | Files in your templates folder | Uncheck the templates and click **Run** |
| [Flatpaks](flatpaks.md) | Flatpak applications | Uncheck them and click **Run** |
| [Post-install](post-install.md) | Packages, repositories, `/etc/sysctl.d/99-perci.conf`, `/swapfile` and its `/etc/fstab` line, the `fstrim.timer` service | By hand, with your distribution's tools — post-install has no undo |
