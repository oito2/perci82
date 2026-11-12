// Copyright (C) 2026  oito2
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// The i18n tests every application runs against its own page: catalog
// consistency (same keys and placeholders in every locale, every used key
// present, no unused key) and i18n.js behavior (t, data-i18n*, live
// switch, the language selector).

import { test } from "node:test";
import assert from "node:assert/strict";
import path from "node:path";
import { FakeElement } from "./harness.mjs";
import { LOCALES, appDistDirs, layers, literals, placeholders, readCatalog, usedKeys } from "./i18n-catalog.mjs";

const tick = () => new Promise((r) => setTimeout(r, 0));

// registerI18nTests registers the tests for the application whose dist/
// folder and harness functions are given.
export function registerI18nTests({ distDir, evalIn, loadScripts, scriptOrder }) {
  const dirs = layers(distDir);
  const catalogsOf = (dir) => Object.fromEntries(LOCALES.map((l) => [l, readCatalog(path.join(dir, l + ".json"))]));

  test("i18n: every catalog layer has the same keys and placeholders in every locale", () => {
    for (const [name, dir] of Object.entries(dirs)) {
      const cats = catalogsOf(dir);
      const base = cats[LOCALES[0]];
      assert.ok(Object.keys(base).length > 0, name + " catalog is empty");
      for (const locale of LOCALES.slice(1)) {
        assert.deepEqual(Object.keys(cats[locale]).sort(), Object.keys(base).sort(), name + " " + locale + " keys");
        for (const key of Object.keys(base)) {
          assert.deepEqual(placeholders(cats[locale][key]), placeholders(base[key]), name + " " + locale + " " + key);
        }
      }
    }
  });

  test("i18n: every key the page uses exists in every locale", () => {
    const used = usedKeys(distDir);
    assert.ok(used.size > 0, "the page uses no message key");
    for (const locale of LOCALES) {
      const merged = Object.assign({}, ...Object.values(dirs).map((d) => readCatalog(path.join(d, locale + ".json"))));
      const missing = [...used].filter((k) => !(k in merged));
      assert.deepEqual(missing, [], locale + " is missing keys");
    }
  });

  test("i18n: no unused key in the application or shared catalogs", () => {
    const own = new Set([...usedKeys(distDir), ...literals(distDir)]);
    const unusedOwn = Object.keys(catalogsOf(dirs.app)[LOCALES[0]]).filter((k) => !own.has(k));
    assert.deepEqual(unusedOwn, [], "application catalog");
    const anyApp = new Set(appDistDirs().flatMap((d) => [...usedKeys(d), ...literals(d)]));
    const unusedShared = Object.keys(catalogsOf(dirs.shared)[LOCALES[0]]).filter((k) => !anyApp.has(k));
    assert.deepEqual(unusedShared, [], "shared catalog");
  });

  function load(fixtures = {}) {
    return loadScripts(scriptOrder(), {
      fixtures: Object.assign({ GetLanguage: { choice: "system", locale: "pt-BR" } }, fixtures),
    });
  }

  test("i18n: t fills placeholders and returns an unknown key as is", async () => {
    const { ctx } = load();
    await evalIn(ctx, "i18nReady");
    assert.equal(ctx.t("settings.theme"), "Tema");
    assert.equal(ctx.t("oito2.cancel"), "Cancelar");
    assert.equal(ctx.t("settings.themeSaveFailed", { error: "x" }), "Falha ao salvar o tema: x");
    assert.equal(ctx.t("settings.themeSaveFailed"), "Falha ao salvar o tema: {error}");
    assert.equal(ctx.t("no.such.key"), "no.such.key");
  });

  test("i18n: starts in the saved locale and switches live on language-changed", async () => {
    const { ctx, emit, query } = load({ GetLanguage: { choice: "en", locale: "en" } });
    const heading = new FakeElement("h3");
    heading.setAttribute("data-i18n", "settings.theme");
    const select = new FakeElement("select");
    select.setAttribute("data-i18n-title", "settings.language");
    query("[data-i18n]", [heading]);
    query("[data-i18n-title]", [select]);
    await evalIn(ctx, "i18nReady");
    assert.equal(heading.textContent, "Theme");
    assert.equal(select.getAttribute("title"), "Language");
    assert.equal(ctx.document.documentElement.getAttribute("lang"), "en");

    emit("language-changed", { choice: "pt-BR", locale: "pt-BR" });
    await tick();
    assert.equal(heading.textContent, "Tema");
    assert.equal(select.getAttribute("title"), "Idioma");
    assert.equal(ctx.document.documentElement.getAttribute("lang"), "pt-BR");
    assert.equal(evalIn(ctx, "i18nLocale"), "pt-BR");
  });

  test("i18n: the language selector saves the choice and switches the page", async () => {
    let notified = null;
    const { ctx, calls, element } = load({
      GetLanguageOptions: [{ id: "system", label: "" }, { id: "pt-BR", label: "Português (Brasil)" }, { id: "en", label: "English" }],
      SetLanguage: (choice) => ({ choice, locale: choice }),
    });
    await evalIn(ctx, "i18nReady");
    ctx.onLocaleChange((locale) => { notified = locale; });
    ctx.renderLanguageSelect();
    await tick();
    const select = element("cfg-language");
    assert.deepEqual(select.children.map((o) => o.textContent), ["Definido pelo sistema", "Português (Brasil)", "English"]);
    assert.equal(select.value, "system");

    select.value = "en";
    select.listeners.change.forEach((fn) => fn());
    await tick();
    assert.deepEqual(calls.filter((c) => c.name === "SetLanguage").map((c) => c.args), [["en"]]);
    assert.equal(evalIn(ctx, "i18nLocale"), "en");
    assert.equal(notified, "en");
  });
}
