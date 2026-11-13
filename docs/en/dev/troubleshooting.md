🌐 [Português](../../pt-br/dev/troubleshooting.md) | **English** | 🏠 [Index](../index.md)

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

## The installer says "No Perci82 Dev release has been published yet."

The script found no `dev-vX.Y.Z` release on GitHub (drafts and prereleases don't count). Build from source instead — see [Installing](installation.md#build-from-source).

## The installer kept the old Perci: "No terminal to ask on"

The script asks on the terminal before removing the old Perci; with no terminal to ask on, it keeps it. Run the script again from a terminal, or remove the files it listed by hand.

## Opening Perci82 Dev only brings back an old window

Perci82 Dev runs once per session, so opening it again focuses the window that is already running — including one started hidden in the tray at login. After installing or building a new version, quit the running one with **Quit** in the tray menu (or close its window when the tray is off), then open it again.

## The tray icon doesn't show

The icon needs a desktop with StatusNotifierItem support. KDE, XFCE and COSMIC show it natively; on GNOME, install and turn on the "AppIndicator and KStatusNotifierItem Support" extension.

## "Another action of Perci82 System is running — wait for it to finish."

Actions that need administrator rights never run in both apps at once. Wait for the action in Perci82 System to finish, then try again.

## The window doesn't close

The window stays open while an action runs. Follow it in the **Run** tab and close the window after it finishes.

## An action fails right after the password dialog

The authentication was cancelled or the password was wrong. Run the action again. The app uses the desktop's authentication dialog (`pkexec`), so a PolicyKit authentication agent must be running, as it is in the usual desktop sessions.

## Overview shows "Docker unreachable"

Docker isn't installed, isn't running, or your user can't reach it. Install Docker in [Environment → Prerequisites](environment.md): it adds your user to the `docker` group, which only takes effect after you log out and back in.

## "secret-tool not found — install Libsecret Tools in Environment → Prerequisites"

A Git account's API token is stored in the system keyring through `secret-tool`. Install **Libsecret Tools** in [Environment → Prerequisites](environment.md) and save the account again.

## "Path not picked in the file chooser"

Folders and files that an action writes to must be chosen in the app's own file chooser, in the same session. Choose the folder again with the screen's button instead of typing it.

## "Couldn't check for updates"

Overview reaches the GitHub API to look for a new release. Check the network connection and click **Check** again; the GitHub API also limits how often one address can ask.
