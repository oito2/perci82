# PHP in the Perci82 Dev Container

How to run this project's PHP code, tools and tests inside its Perci82 Dev Docker container. Perci82 Dev writes this file from the app's settings and rewrites it when they change: never edit it.

---

## The App

| Item | Value |
|---|---|
| App | `{{APP}}` ({{APP_TYPE}}) |
| URL | {{APP_URL}} |
| PHP | {{PHP_VERSION}} |
{{MOODLE_ROWS}}| Project root | {{ROOTS}} |

---

## The `{{APP}}` Command

Run PHP and its tools only through `{{APP}}` (in `~/.local/bin`), never with a PHP installed on the host. It runs inside the app's container, as the user that owns the project files.

| Command | Runs |
|---|---|
{{TOOL_ROWS}}

- The first argument picks the tool; anything else goes to `{{FALLBACK}}`: `{{APP}} script.php`, `{{APP}} -r 'echo 1;'`. A file named like a tool needs `./`: `{{APP}} ./composer`.
- Inside the project, the command runs in the matching folder of the container; outside it, at the project root (it says so on stderr).
- Host paths under the project root are rewritten to container paths; relative paths work as they are.
- A tool that answers "not enabled in this app" was turned off in the app's settings: ask the user to enable it in Perci82 Dev (Docker → Manage Containers → Edit). Never install it another way.
{{TOOL_NOTES}}
---

## Database

{{DATABASE}}

---

## Mounted Development Folders

{{MOUNTS}}

---

## Logs and Restarts

{{LOGS}}
- Restart, recreate or change the app only through Perci82 Dev (Docker → Manage Containers). Never run `docker` commands that change containers, and never touch the Nginx or MariaDB containers.
