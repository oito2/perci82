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

// Perci82 System's page: every script loads, its screens resolve to their
// containers, and its own parts of Home and the Linux checklists work.

import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { commonDir, distDir, evalIn, loadScripts, scriptOrder } from "./harness.mjs";

const tick = () => new Promise((r) => setTimeout(r, 0));

function load(fixtures = {}) {
  return loadScripts(scriptOrder(), { fixtures });
}

test("all classic scripts load together", () => {
  const order = scriptOrder();
  assert.ok(order.includes("system-core.js") && order.includes("app.js"), "index.html script list: " + order.join(", "));
  const { ctx } = load();
  for (const fn of ["escHTML", "latestOnly", "beginRun", "renderPostinstallScreen", "updateExecutarButton", "renderMultiselectScreen"]) {
    assert.equal(typeof ctx[fn], "function", fn + " is defined");
  }
  // Perci82 Dev's scripts are not part of this page.
  for (const name of ["renderDockerListScreen", "createScopedTable", "activateCompactMode", "renderFileAppScreen"]) {
    assert.equal(evalIn(ctx, "typeof " + name), "undefined", name);
  }
});

test("screenFor resolves each System screen to its container", () => {
  const { ctx, element } = load();
  const el = (id) => ctx.screenFor({ actionId: id }).el;
  assert.equal(el("dashboard"), element("dashboard-container"));
  assert.equal(el("self-config"), element("selfconfig-container"));
  assert.equal(el("postinstall"), element("postinstall-container"));
  assert.equal(el("system-update"), element("simple-container"));
  for (const id of ["fonts", "templates", "flatpak-apps"]) assert.equal(el(id), element("multiselect-container"), id);
  for (const id of ["docker-manage", "terminals", "prereqs", "androidstudio"]) assert.equal(ctx.screenFor({ actionId: id }), null, id);
});

test("doneHandlerFor: postinstall re-renders, system-update re-evaluates Executar", () => {
  const { ctx } = load();
  assert.equal(typeof ctx.doneHandlerFor("postinstall"), "function");
  ctx.__selectedItem = { actionId: "system-update", title: "Atualizar Sistema", desc: "" };
  ctx.doneHandlerFor("system-update")({ buttons: [], actionId: "system-update" });
  const none = ctx.doneHandlerFor("unknown-screen");
  ctx.__selectedItem = { actionId: "fonts", title: "Fontes", desc: "" };
  assert.notEqual(ctx.doneHandlerFor("fonts"), none);
});

test("checklist: Executar runs the screen that is showing", async () => {
  const { ctx, element, calls } = load({
    GetFontsInfo: [{ id: "f1", label: "F1", installed: true }],
    GetTemplatesInfo: [{ id: "t1", label: "T1" }],
  });
  ctx.renderMultiselectScreen({ actionId: "fonts", title: "Fontes" });
  ctx.renderMultiselectScreen({ actionId: "templates", title: "Templates" });
  await tick();
  element("btn-executar-multiselect").onclick();
  assert.equal(calls.filter((c) => c.name === "RunFonts").length, 0);
  assert.equal(calls.filter((c) => c.name === "RunTemplates").length, 1);
});

test("getPostinstallProfile asks the backend once", async () => {
  const { ctx, calls } = load({ GetPostinstallProfile: { supported: true, label: "Ubuntu" } });
  const before = calls.filter((c) => c.name === "GetPostinstallProfile").length;
  const [a, b] = await Promise.all([ctx.getPostinstallProfile(), ctx.getPostinstallProfile()]);
  assert.equal(a.label, "Ubuntu");
  assert.equal(a, b);
  const after = calls.filter((c) => c.name === "GetPostinstallProfile").length;
  assert.ok(after - before <= 1, `called ${after - before} times`);
});

test("Visão Geral shows the detected system; Configurações the Flatpak scope", async () => {
  const { ctx, element } = load({
    GetPostinstallProfile: { supported: true, label: "Fedora 43" },
    GetFlatpakScope: "user",
  });
  ctx.renderDashboardScreen();
  ctx.renderSelfConfigScreen();
  await tick();
  assert.equal(element("dashboard-system-label").textContent, "Fedora 43");
  assert.equal(element("sc-flatpak").value, "user");
});

test("Desinstalar confirms, then asks for the uninstall with the config choice", () => {
  const { ctx, element, calls } = load();
  element("su2-remove-config").checked = true;
  element("btn-su2-uninstall").listeners.click[0]();
  assert.match(element("confirm-modal-message").textContent, /Perci82 System/);
  element("btn-confirm-ok").listeners.click[0]();
  const call = calls.find((c) => c.name === "RunSelfUninstall");
  assert.deepEqual(call && call.args, [true]);
});

test("every icon the page uses is in the vendored lucide set", () => {
  const icons = fs.readFileSync(path.join(commonDir, "vendor", "lucide", "icons.js"), "utf8");
  const used = new Set();
  for (const root of [distDir, commonDir]) {
    for (const dir of [path.join(root, "js"), path.join(root, "js", "screens")]) {
      for (const f of fs.readdirSync(dir).filter((n) => n.endsWith(".js"))) {
        const src = fs.readFileSync(path.join(dir, f), "utf8");
        for (const m of src.matchAll(/(?:iconButton|LUCIDE\.svg)\("([A-Za-z0-9]+)"/g)) used.add(m[1]);
        const cat = /const CATEGORY_ICONS = \{([^}]*)\}/.exec(src);
        if (cat) for (const m of cat[1].matchAll(/:\s*"([A-Za-z0-9]+)"/g)) used.add(m[1]);
      }
    }
  }
  const missing = [...used].filter((name) => !new RegExp("^\\s+" + name + ":\\s*'", "m").test(icons));
  assert.deepEqual(missing, []);
});
