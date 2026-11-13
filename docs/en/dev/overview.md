🌐 [Português](../../pt-br/dev/overview.md) | **English** | 🏠 [Index](../index.md)

---

# Overview

The first screen of Perci82 Dev: a summary of the Docker containers and repositories, its version, its updates and its uninstall.

![Overview](../../img/dev-overview.png)
<!-- TODO: capture a screenshot of Perci82 Dev's Overview -->

## The window

Perci82 Dev's sections are tabs at the top of the window: **Overview**, **Environment**, **Docker**, **Repositories**, **AI Contexts**, **Skills & MCPs** and **Settings**. Each section has a side list with its screens; in **Repositories**, the Git accounts are grouped under **Accounts**, one item per account. Screens that run actions have their own tabs plus **Run**, which shows the running action: its progress bar and the commands' output.

Only one action runs at a time, and the window doesn't close while one runs. Actions that need administrator rights never run in Perci82 Dev and Perci82 System at once: while one app runs such an action, the other refuses with a message naming that app.

Perci82 Dev runs once per session: opening it again brings the running window to the front.

## Cards

| Card | Shows |
|---|---|
| **Docker containers** | How many of the registered containers are running (e.g. "2 of 3 running"); **None** without containers, **Docker unreachable** when Docker can't be queried |
| **Repositories** | The number of Git accounts and of repositories found in their folders; **No account** without accounts |
| **Version** | Perci82 Dev's version, with an **up to date** or **update available** badge after each check |

## Updates

The app checks for a newer release when the screen opens (again at most every 10 minutes); **Check** checks at once. It looks only at Perci82 Dev's own releases (tags `dev-vX.Y.Z`) on GitHub.

When a newer release exists, **Update** appears. It downloads the new binary, checks its SHA-256 against the release's `checksums.txt`, replaces the installed binary and shows the progress in the **Run** tab. Restart the app afterwards.

A build from source reports the version `dev` and can't update itself.

## Uninstall

**Uninstall Perci82 Dev** opens a confirmation window with the backups of Docker, repositories and AI contexts, and the options to remove the Docker containers and Perci82 Dev's settings. See [Uninstalling](uninstallation.md).

## System tray

With **Show Perci82 Dev in the system tray** on ([Settings](settings.md#system-tray)), Perci82 Dev puts an icon in the system tray. Its menu has:

- **Open Application** — shows the main window.
- **Containers** — opens a compact window with **Docker → Manage Containers** only, and an **Open Application** button.
- **Quit** — quits Perci82 Dev (refused while an action runs).

Closing the main window then hides it in the tray instead of quitting, and Perci82 Dev starts hidden in the tray when you log in.
