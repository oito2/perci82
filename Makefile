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

.PHONY: build build-system build-dev run-system run-dev test test-frontend lint clean release release-system release-dev generate-bindings generate check-generated parity-sync

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

clean:
	rm -f perci82-system perci82-dev
	rm -rf $(DIST)

# Only linux/amd64 is built — Wails uses cgo (GTK4/WebKitGTK 6.0 via C
# bindings), so a plain "GOARCH=arm64 go build" doesn't cross-compile.
#
# release-<app> writes dist/<app>-linux-amd64, dist/<app>-menu.tar.gz (the
# application menu entry: share/applications/<id>.desktop and the app's
# hicolor icons) and dist/checksums.txt covering both. The archive is built
# reproducibly (sorted entries, fixed owner/mtime, gzip -n) so the same tag
# yields the same hash.
MENU_STAGE := $(DIST)/menu

release: release-system release-dev

release-system:
	$(MAKE) release-app APP=perci82-system APP_ID=io.github.oito2.perci82.system

release-dev:
	$(MAKE) release-app APP=perci82-dev APP_ID=io.github.oito2.perci82.dev

release-app:
	@test -n "$(APP)" -a -n "$(APP_ID)" || { echo "APP and APP_ID are required" >&2; exit 1; }
	@echo "Building $(APP) $(VERSION)..."
	@mkdir -p $(DIST)
	go build -tags $(TAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(APP)-linux-amd64 ./cmd/$(APP)
	rm -rf $(MENU_STAGE)
	mkdir -p $(MENU_STAGE)/share/applications $(MENU_STAGE)/share/icons
	cp packaging/$(APP_ID).desktop $(MENU_STAGE)/share/applications/$(APP_ID).desktop
	cd packaging && find hicolor -name '$(APP_ID).*' -o -name '$(APP_ID)-symbolic.*' | while read -r f; do \
		install -D -m 644 "$$f" "../$(MENU_STAGE)/share/icons/$$f" || exit 1; \
	done
	chmod -R u=rwX,go=rX $(MENU_STAGE)
	tar --sort=name --owner=0 --group=0 --numeric-owner --mtime='@0' \
		-C $(MENU_STAGE) -cf - share | gzip -n > $(DIST)/$(APP)-menu.tar.gz
	rm -rf $(MENU_STAGE)
	cd $(DIST) && sha256sum $(APP)-linux-amd64 $(APP)-menu.tar.gz > checksums.txt
	@echo "Binaries in $(DIST):"
	@ls -lh $(DIST)/

# generate-bindings regenerates each application's frontend/dist/bindings/
# (JS generated from the exported methods of the services its main.go
# registers) via the `wails3` CLI. Not part of build/release: the generated
# files are versioned. Requires the `wails3` CLI installed once:
# `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28`.
generate-bindings:
	for app in $(APPS); do \
		(cd cmd/$$app && wails3 generate bindings -names -b -d frontend/dist/bindings ./...) || exit 1; \
	done

# generate rewrites the files generated from the oito2_ui parity fixtures
# (internal/gui/web/css/oito2.css); check-generated fails when they are
# out of date, without writing anything.
generate:
	go generate ./internal/theme

check-generated:
	go run ./internal/theme/gencss -out internal/gui/web/css/oito2.css -check

# parity-sync copies the oito2_ui fixtures Perci82 uses from a checkout of
# oito2_ui (default references/oito2_ui; OITO2_UI=<dir> picks another) and
# records its commit; run `make generate` afterwards.
OITO2_UI ?= references/oito2_ui
parity-sync:
	go run ./tools/parity-sync --from $(OITO2_UI)
