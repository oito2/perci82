🌐 [Português](../../pt-br/system/troubleshooting.md) | **English** | 🏠 [Index](../index.md)

---

# Troubleshooting

## The app doesn't start: `libwebkitgtk-6.0.so.4` or `libgtk-4.so.1` not found

The GTK 4 or WebKitGTK 6.0 runtime library is missing. Install it:

```bash
# Debian/Ubuntu
sudo apt install -y libgtk-4-1 libwebkitgtk-6.0-4
# Fedora
sudo dnf install -y gtk4 webkitgtk6.0
```

## The installer says "No Perci82 System release has been published yet."

The script found no `system-vX.Y.Z` release on GitHub (drafts and prereleases don't count). Build from source instead — see [Installing](installation.md#build-from-source).

## The installer kept the old Perci: "No terminal to ask on"

The script asks on the terminal before removing the old Perci; with no terminal to ask on, it keeps it. Run the script again from a terminal, or remove the files it listed by hand.

## "Another action of Perci82 Dev is running — wait for it to finish."

Actions that need administrator rights never run in both apps at once. Wait for the action in Perci82 Dev to finish, then try again.

## The window doesn't close

The window stays open while an action runs. Follow it in the **Run** tab and close the window after it finishes.

## Post-install says the system is not supported

Post-install has items only for the systems listed in [Post-install](post-install.md#supported-systems). The other screens work on any distribution with `apt` or `dnf`.

## An action fails right after the password dialog

The authentication was cancelled or the password was wrong. Run the action again. The app uses the desktop's authentication dialog (`pkexec`), so a PolicyKit authentication agent must be running, as it is in the usual desktop sessions.

## "Couldn't check for updates"

Overview reaches the GitHub API to look for a new release. Check the network connection and click **Check** again; the GitHub API also limits how often one address can ask.
