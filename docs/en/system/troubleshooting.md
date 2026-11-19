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

## The installer says "This release has no WebKitGTK 6.0"

The distribution release is older than the supported ones (Debian 13, Ubuntu 24.04, Linux Mint 22 or newer): it has no `libwebkitgtk-6.0-4`, which Perci82 System needs. Upgrade the distribution.

## The installer says the repository key "does not have the expected fingerprint"

The key downloaded from `https://oito2.github.io/perci82/perci82.asc` isn't the one the script expects, so it stops before changing anything. Check the network (a proxy may be answering in its place) and run it again; if it keeps failing, report it in the repository's issues.

## An older version still opens after installing the package

Perci82 System 1.1 and older were installed as `/usr/local/bin/perci82-system`, which comes before the package's `/usr/bin/perci82-system` in `PATH`. Remove the old file:

```bash
sudo rm -f /usr/local/bin/perci82-system
```

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

Overview reads the newest version from the package repository site and, when it can't, from the GitHub API, which limits how often one address can ask (60 times an hour without a login; more when the GitHub CLI is logged in). Check the network connection and click **Check** again.
