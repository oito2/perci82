# Contributing to Perci82

🌐 **Language:** English · [Português](docs/pt-br/contribuindo.md)

Thanks for considering a contribution. This document covers how to build and check the project, what a pull request needs, and how releases are made.

By participating in this project you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Before you start

For anything beyond a small fix (a new screen, a new item in a catalogue, a new dependency), open an issue first to discuss the approach. The project keeps its dependencies few and justified: see the `require` block in [`go.mod`](go.mod). Wails v3 is upgraded by hand, as its own change.

## Development setup

Requirements:

- Linux on amd64.
- Go 1.27.4 or later.
- The GTK 4 and WebKitGTK 6.0 development packages:

  ```bash
  # Debian/Ubuntu
  sudo apt install -y build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
  # Fedora
  sudo dnf install -y gcc-c++ pkgconf-pkg-config gtk4-devel webkitgtk6.0-devel
  ```

- Node.js, for the frontend tests.
- The `wails3` CLI of the Wails version in `go.mod`, to regenerate and check the bindings: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28`.
- ShellCheck and shfmt, for `make lint-shell`.

```bash
git clone https://github.com/oito2/perci82.git
cd perci82
make build          # writes ./perci82-system and ./perci82-dev
```

### Checks

Run them before opening a pull request; the CI runs the same ones.

```bash
make lint             # gofmt, go vet, golangci-lint
make test             # go test -race ./...
make test-frontend    # node --test, plain Node
make check-generated  # generated CSS and catalogs are up to date
make check-bindings   # frontend bindings match the Go services
make lint-shell       # ShellCheck and shfmt on install.sh and scripts/build-pages.sh
```

`make generate` and `make generate-bindings` rewrite what `check-generated` and `check-bindings` compare. `make install-nfpm` installs the pinned nfpm, and `make release-<app> VERSION=vX.Y.Z` builds that app's `.deb` and `.rpm` into `dist/`.

## Project layout

| Path | What it holds |
|---|---|
| `cmd/perci82-system`, `cmd/perci82-dev` | Each app's entry point and frontend (`frontend/dist`, with its generated bindings) |
| `internal/` | Everything else: GUI services (`internal/gui`), Docker stack (`internal/appstack`), AI contexts (`internal/manager/ai`), system actions, configuration, i18n catalogs… |
| `packaging/` | Icons, `.desktop` entries, nfpm configurations and the repository's public signing key |
| `install.sh` | The installer: adds the package repository and installs the apps |
| `scripts/build-pages.sh` | Builds the signed apt/dnf repository published on GitHub Pages |
| `tools/` | `imagetools` (pinned tool versions of the PHP images), `parity-sync` (oito2_ui fixtures), `installtest` (tests of `install.sh`) |
| `docs/en`, `docs/pt-br` | The documentation, in English and Portuguese |

## Making a change

- Keep the change scoped to what was discussed; large unsolicited refactors are likely to be declined.
- Code comments are in English and describe what the code does.
- A user-visible change updates the documentation in the same change, in **both** languages: `docs/en/` and `docs/pt-br/`, and `README.md` with `docs/pt-br/leiame.md`.
- A user-visible change adds an entry under `## [Unreleased]` in [`CHANGELOG.md`](CHANGELOG.md) (Keep a Changelog); released sections are never edited.
- Write a concise commit summary that says why the change was made.

## Pull requests

- Describe what changed and why, and link the issue it addresses.
- The `Lint and test` check must pass; the default branch keeps a linear history (squash or rebase).
- [Dependabot](.github/dependabot.yml) opens weekly pull requests for the GitHub Actions and the Go modules. It proposes a version only 14 days after its release (security updates are not delayed), and its commits start with `ci` or `deps`. Tools pinned by hand follow the same 14-day rule; the weekly **Image tools** workflow reports newer versions of the tools pinned in the PHP images.

## Reporting bugs and requesting features

Open a [GitHub issue](https://github.com/oito2/perci82/issues) with the app and its version (Overview → Version), your distribution and what happened.

Report security vulnerabilities privately, never in a public issue: use **Report a vulnerability** in the repository's [Security tab](https://github.com/oito2/perci82/security/advisories/new).

## Releases

Each app is released on its own, from `main`:

1. Add the `## [System X.Y.Z] - date` or `## [Dev X.Y.Z] - date` section to `CHANGELOG.md` (moving the entries out of `[Unreleased]`) and its link line.
2. Push the commit and wait for the CI.
3. Push the tag `system-vX.Y.Z` or `dev-vX.Y.Z`.

The [release workflow](.github/workflows/release.yml) then:

1. Runs the same checks as the CI.
2. Checks that the tagged commit is on `main`, and builds the `.deb`, the `.rpm` and `checksums.txt`.
3. Waits for the owner's approval in the `release` environment.
4. Attests the build provenance of the files and creates the GitHub release with them in one `gh release create` call. The release notes are the install instructions plus the CHANGELOG section.
5. Rebuilds the package repository at `https://oito2.github.io/perci82/` ([`pages.yml`](.github/workflows/pages.yml)), signed with the repository key. When both apps are tagged on the same commit, only the last release run publishes the site.

Protections:

- **Release tags:** only repository admins can create, move or delete `system-v*` and `dev-v*` tags (ruleset `release tags`).
- **Immutable releases:** a published release's files and its tag can't change. A bad release is fixed with a new patch version.
- **Repository signing key:** apt and dnf verify only GPG signatures, so the key that signs the package repository is kept as `REPO_SIGNING_KEY` and `REPO_SIGNING_PASSPHRASE` in the `github-pages` environment, which deploys only from `main`, `system-v*` and `dev-v*`. It is the project's only secret. Everything else is signed with OIDC (the build attestations).

## License

By contributing, you agree that your contributions are licensed under the [GPL-3.0](LICENSE).
