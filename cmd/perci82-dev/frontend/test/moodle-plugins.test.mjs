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

import { test } from "node:test";
import assert from "node:assert/strict";
import { evalIn, loadScripts, scriptOrder } from "./harness.mjs";

const tick = () => new Promise((r) => setTimeout(r, 0));

function load(check) {
  const loaded = loadScripts(scriptOrder(), {
    fixtures: {
      InspectDevFolder: check,
      GetDevFolders: {
        apps: [{ folder: "mdle", label: "Moodle (mdle)", type: "moodle", root: "localhost/html/mdle" }, { folder: "loja", label: "Loja (loja)", type: "php", root: "localhost/html/loja" }, { folder: "painel", label: "painel", type: "php_node", root: "localhost/html/painel" }],
        rows: [
          { app: "mdle", appLabel: "Moodle (mdle)", component: "local_a", devPath: "/ws/a", devLabel: "a", relPath: "local/a", destLabel: "mdle/local/a", status: "applied", declaredAs: "" },
          { app: "mdle", appLabel: "Moodle (mdle)", component: "local_b", devPath: "/ws/b", devLabel: "b", relPath: "local/b", destLabel: "mdle/local/b", status: "pending", declaredAs: "local_c" },
        ],
      },
    },
  });
  for (const id of ["mp-type", "mp-name", "mp-app"]) {
    loaded.element(id).value = "";
    loaded.element(id).options = [];
  }
  evalIn(loaded.ctx, `mpDev = "/ws/dev";`);
  return loaded;
}

test("Plugins Moodle: type and name are locked to version.php and Criar is enabled", async () => {
  const { ctx, element, calls } = load({ locked: true, type: "format", name: "grid", ok: true, destLabel: "mdle/course/format/grid" });
  element("mp-name").value = "typed";
  ctx.checkMpForm();
  await tick();
  assert.equal(element("btn-mp-create").disabled, false);
  assert.equal(element("mp-type").disabled, true);
  assert.equal(element("mp-name").disabled, true);
  assert.equal(element("mp-name").value, "grid");
  assert.equal(element("mp-status-text").textContent, "Será montado em: mdle/course/format/grid");
  assert.ok(calls.some((c) => c.name === "InspectDevFolder" && c.args[0] === "/ws/dev" && c.args[3] === "typed"));
});

test("Plugins Moodle: an empty folder keeps the fields free and a refusal disables Criar", async () => {
  const { ctx, element } = load({ locked: false, type: "local", name: "x", ok: false, message: "Já existe conteúdo em mdle/local/x" });
  ctx.checkMpForm();
  await tick();
  assert.equal(element("btn-mp-create").disabled, true);
  assert.equal(element("mp-type").disabled, false);
  assert.equal(element("mp-name").disabled, false);
  assert.equal(element("mp-status-text").textContent, "Já existe conteúdo em mdle/local/x");
});

test("Plugins Moodle: one row per mount with its status and the version.php warning", async () => {
  const { ctx, element } = load({ ok: false });
  ctx.loadMpRows();
  await tick();
  const rows = element("mp-rows").children;
  assert.equal(rows.length, 2);
  assert.equal(rows[0].children[3].children[0].textContent, "Aplicado");
  assert.equal(rows[1].children[3].children[0].textContent, "Reinício pendente");
  assert.equal(rows[1].children[0].children[0].textContent, "version.php declara local_c");
});

// trackClasses gives el a working classList (the harness's is a no-op).
function trackClasses(el) {
  const set = new Set();
  el.classList = {
    add: (c) => set.add(c),
    remove: (...cs) => cs.forEach((c) => set.delete(c)),
    toggle: (c, force) => ((force === undefined ? !set.has(c) : force) ? set.add(c) : set.delete(c)),
    contains: (c) => set.has(c),
  };
}

test("Pastas de Desenvolvimento: a non-Moodle app takes a destination and sends it", async () => {
  const { ctx, element, calls } = load({ ok: true, destLabel: "loja/modules/x", locked: false });
  trackClasses(element("mp-moodle-fields"));
  trackClasses(element("mp-dest-field"));
  ctx.renderMoodlePluginsScreen();
  await tick();
  evalIn(ctx, 'mpDev = "/ws/dev";');
  element("mp-app").value = "loja";
  element("mp-app").listeners.change.forEach((fn) => fn());
  assert.equal(element("mp-moodle-fields").classList.contains("hidden"), true);
  assert.equal(element("mp-dest-field").classList.contains("hidden"), false);
  assert.equal(element("mp-dest-hint").textContent, "Pasta relativa a localhost/html/loja. Precisa estar vazia ou não existir.");
  element("mp-dest").value = " modules/x ";
  ctx.checkMpForm();
  await tick();
  const last = calls.filter((c) => c.name === "InspectDevFolder").pop();
  assert.deepEqual([last.args[1], last.args[4]], ["loja", "modules/x"]);
  assert.equal(element("btn-mp-create").disabled, false);

  element("mp-app").value = "painel";
  element("mp-app").listeners.change.forEach((fn) => fn());
  assert.match(element("mp-dest-hint").textContent, /dentro de api\/ \(PHP\) ou de app\/ \(Node\)/);

  element("mp-app").value = "mdle";
  element("mp-app").listeners.change.forEach((fn) => fn());
  assert.equal(element("mp-moodle-fields").classList.contains("hidden"), false);
  assert.equal(element("mp-dest-field").classList.contains("hidden"), true);
});

test("Pastas de Desenvolvimento: a folder mounted outside Moodle is named by its destination", async () => {
  const { ctx, element } = load({ ok: false });
  ctx.showMpRows({ rows: [{ app: "loja", appLabel: "Loja (loja)", component: "", devPath: "/ws/p", devLabel: "p", relPath: "modules/p", destLabel: "loja/modules/p", status: "stopped", declaredAs: "" }] });
  const row = element("mp-rows").children[0];
  assert.equal(row.children[0].textContent, "loja/modules/p");
  assert.equal(row.children[3].children[0].textContent, "Contêiner parado");
});

test("Plugins Moodle: mounts with old systemd units show a warning whose button updates them", async () => {
  const { ctx, element, calls } = load({ ok: true });
  ctx.showMpRows({ rows: [], outdatedUnits: 2 });
  assert.match(element("mp-units-text").textContent, /^2 pasta\(s\) montada\(s\)/);
  (element("btn-mp-units").listeners.click || []).forEach((fn) => fn({}));
  await tick();
  assert.ok(calls.some((c) => c.name === "UpdateDevFolderUnits"));
});
