# Copyright (C) 2026  oito2
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

APPS     := perci82-system perci82-dev
VERSION  ?= dev
MODULE   := github.com/oito2/perci82
LDFLAGS  := -X $(MODULE)/internal/version.Version=$(VERSION)
DIST     := dist
# The production tag turns off Wails' debug mode (webview developer tools).
TAGS     := production

.PHONY: build build-system build-dev run-system run-dev test test-frontend lint lint-shell clean release release-system release-dev release-app install-nfpm generate-bindings check-bindings generate check-generated parity-sync

# Builds both applications with the production tag only: GTK4 + WebKitGTK
# 6.0 is the default stack. -trimpath keeps the builder's local paths out of
# the binaries.
build: build-system build-dev

build-system:
	go build -tags $(TAGS) -trimpath -ldflags "$(LDFLAGS)" -o perci82-system ./cmd/perci82-system

build-dev:
	go build -tags $(TAGS) -trimpath -ldflags "$(LDFLAGS)" -o perci82-dev ./cmd/perci82-dev

run-system: build-system
	./perci82-system

run-dev: build-dev
	./perci82-dev

test:
	go test -race ./...

# Frontend smoke test: syntax of every script, then each application's
# node:test suite (plain Node, no npm dependencies).
FRONTEND_JS := $(wildcard internal/gui/web/js/*.js internal/gui/web/js/screens/*.js \
	cmd/*/frontend/dist/js/*.js cmd/*/frontend/dist/js/screens/*.js)

test-frontend:
	for f in $(FRONTEND_JS); do node --check "$$f" || exit 1; done
	node --test cmd/perci82-system/frontend/test/*.test.mjs
	node --test cmd/perci82-dev/frontend/test/*.test.mjs

lint:
	gofmt -l . | (! grep .)
	go vet ./...
	golangci-lint run

# lint-shell checks the installer and scripts/build-pages.sh with ShellCheck
# and their formatting with shfmt (4-space indentation, indented case items);
# both tools must be installed.
SHELL_SCRIPTS := install.sh scripts/build-pages.sh
lint-shell:
	shellcheck --shell=bash $(SHELL_SCRIPTS)
	shfmt -d -i 4 -ci $(SHELL_SCRIPTS)

clean:
	rm -f perci82-system perci82-dev
	rm -rf $(DIST)

# Only linux/amd64 is built — Wails uses cgo (GTK4/WebKitGTK 6.0 via C
# bindings), so a plain "GOARCH=arm64 go build" doesn't cross-compile.
#
# release-<app> builds the app into dist/build/<app> and packages it with
# nfpm (packaging/nfpm/<app>.yaml) as dist/<app>_<X.Y.Z>-1_amd64.deb and
# dist/<app>-<X.Y.Z>-1.x86_64.rpm, then adds both to dist/checksums.txt,
# keeping the lines of the other application's packages (release builds both
# into one checksums.txt). VERSION must be vX.Y.Z. The packages' file times
# come from SOURCE_DATE_EPOCH, the time of the current commit, so the same
# commit yields the same package contents.
NFPM_VERSION := v2.48.0
NFPM         ?= nfpm
SOURCE_DATE_EPOCH ?= $(shell git log -1 --format=%ct 2>/dev/null || echo 0)

release:
	rm -f $(DIST)/checksums.txt
	$(MAKE) release-system
	$(MAKE) release-dev

release-system:
	$(MAKE) release-app APP=perci82-system

release-dev:
	$(MAKE) release-app APP=perci82-dev

release-app:
	@test -n "$(APP)" || { echo "APP is required" >&2; exit 1; }
	@echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "VERSION must be vX.Y.Z (got '$(VERSION)')" >&2; exit 1; }
	@echo "Building $(APP) $(VERSION)..."
	@mkdir -p $(DIST)/build
	go build -tags $(TAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/build/$(APP) ./cmd/$(APP)
	rm -f $(DIST)/$(APP)_*.deb $(DIST)/$(APP)-[0-9]*.rpm
	for fmt in deb rpm; do \
		VERSION=$(VERSION) PKG_BINARY=$(DIST)/build/$(APP) SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH) \
			$(NFPM) package --config packaging/nfpm/$(APP).yaml --packager $$fmt --target $(DIST)/ || exit 1; \
	done
	cd $(DIST) && { if [ -f checksums.txt ]; then grep -v -e '  $(APP)_[^ ]*\.deb$$' -e '  $(APP)-[0-9][^ ]*\.rpm$$' checksums.txt || true; fi; \
		sha256sum $(APP)_*.deb $(APP)-[0-9]*.rpm; } > checksums.txt.new && mv checksums.txt.new checksums.txt
	@echo "Packages in $(DIST):"
	@ls -lh $(DIST)/

# install-nfpm installs the pinned nfpm CLI that release-app uses.
install-nfpm:
	go install github.com/goreleaser/nfpm/v2/cmd/nfpm@$(NFPM_VERSION)

# generate-bindings regenerates each application's frontend/dist/bindings/
# (JS generated from the exported methods of the services its main.go
# registers) via the `wails3` CLI. Not part of build/release: the generated
# files are versioned. Requires the `wails3` CLI installed once:
# `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28`.
generate-bindings:
	for app in $(APPS); do \
		(cd cmd/$$app && wails3 generate bindings -names -b -d frontend/dist/bindings ./...) || exit 1; \
	done

# check-bindings regenerates each application's bindings into
# $(DIST)/bindings-check/<app> and fails when they differ from the
# versioned ones (a service method changed without make generate-bindings).
# Requires the wails3 CLI, which empties its output folder before writing.
check-bindings:
	for app in $(APPS); do \
		out="$(abspath $(DIST))/bindings-check/$$app"; \
		mkdir -p "$$out" && \
		(cd cmd/$$app && wails3 generate bindings -names -b -d "$$out" ./... >/dev/null) && \
		diff -r "$$out" cmd/$$app/frontend/dist/bindings || { echo "cmd/$$app: bindings out of date, run make generate-bindings" >&2; exit 1; }; \
	done

# generate rewrites the generated files: those built from the oito2_ui
# parity fixtures (internal/gui/web/css/oito2.css and the common message
# catalogs in internal/gui/web/i18n/oito2/); check-generated fails when
# they are out of date, without writing anything.
generate:
	go generate ./internal/theme ./internal/i18n

check-generated:
	go run ./internal/theme/gencss -out internal/gui/web/css/oito2.css -check
	go run ./internal/i18n/genoito2 -l10n internal/theme/parity/l10n.json -out internal/gui/web/i18n/oito2 -check

# parity-sync copies the oito2_ui fixtures Perci82 uses from a checkout of
# oito2_ui (default references/oito2_ui; OITO2_UI=<dir> picks another) and
# records its commit; run `make generate` afterwards.
OITO2_UI ?= references/oito2_ui
parity-sync:
	go run ./tools/parity-sync --from $(OITO2_UI)
