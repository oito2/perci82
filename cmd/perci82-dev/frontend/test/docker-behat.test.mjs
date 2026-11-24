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
import { loadScripts, scriptOrder } from "./harness.mjs";

const tick = () => new Promise((r) => setTimeout(r, 0));

const SETUP = {
  host: "mdle-behat.test",
  wwwroot: "http://mdle-behat.test",
  configPhp: "// Behat.\n$CFG->behat_wwwroot  = 'http://mdle-behat.test';\n",
  initCommand: "mdle admin/tool/behat/cli/init.php",
  runCommands: ["mdle vendor/bin/behat --config /var/www/data/behat/behatrun/behat/behat.yml --tags='@local_x&&~@javascript'"],
  selenium: false,
};

const CATALOG = { moodleVersions: ["5.1+"], phpVersions: ["8.3"], nodeVersions: ["24"], mariadbReady: true, tools: {} };

function buttons(el) {
  const all = [];
  (function walk(e) { (e.children || []).forEach((c) => { all.push(c); walk(c); }); })(el);
  return all.filter((e) => e.getAttribute && e.getAttribute("aria-label"));
}

test("Criar/Editar: only a Moodle app offers the Behat test site, saved in the request", () => {
  const { ctx, element } = loadScripts(scriptOrder());
  assert.match(ctx.buildContainerFieldsHTML("dc", "moodle", CATALOG, { moodleVersion: "5.1+" }), /id="dc-behat" type="checkbox" class="checkbox mt-0.5" \/>/);
  assert.match(ctx.buildContainerFieldsHTML("edit", "moodle", CATALOG, { moodleVersion: "5.1+", behat: true }), /id="edit-behat" type="checkbox" class="checkbox mt-0.5" checked/);
  assert.doesNotMatch(ctx.buildContainerFieldsHTML("dc", "php", CATALOG, {}), /-behat/);
  assert.match(ctx.buildContainerFieldsHTML("dc", "moodle", CATALOG, { moodleVersion: "5.1+" }), /<label class="label[^"]*o2-wrap"><input id="dc-behat"/, "the Behat hint wraps");

  element("t2-behat").checked = true;
  assert.equal(ctx.readContainerFieldsValues("t2", "moodle", CATALOG).behat, true);
  assert.equal(ctx.readContainerFieldsValues("t2", "php", CATALOG).behat, undefined);
});

test("Criar: Selenium has no parameters and the screen renders with it already created", async () => {
  const { ctx, element } = loadScripts(scriptOrder(), {
    fixtures: { GetDockerCreateCatalog: { ...CATALOG, nginxExists: true, mariadbExists: true, seleniumExists: true, mkcertReady: true } },
  });
  const selenium = ctx.buildContainerFieldsHTML("dc", "selenium", CATALOG, {});
  assert.match(selenium, /Selenium não tem parâmetros/);
  assert.match(selenium, /class="alert alert-info alert-soft[^"]*dc-note"/, "an alert spanning the form");
  assert.match(selenium, /<code>http:\/\/localhost:7900<\/code>/);
  assert.match(selenium, /<code>secret<\/code>/);
  assert.match(ctx.buildContainerFieldsHTML("dc", "nginx", CATALOG, {}), /dc-note"[\s\S]*Nginx não tem parâmetros/);
  assert.deepEqual({ ...ctx.readContainerFieldsValues("dc", "selenium", CATALOG) }, { kind: "selenium" });
  element("dc-kind").value = "nginx";
  element("dc-apptype").value = "moodle";
  ctx.renderDockerCreateScreen();
  await tick();
  assert.match(element("dc-fields").innerHTML, /id="dc-behat"/);
});

test("Gerenciar: a Moodle app with Behat opens its configuration; Selenium updates its image", async () => {
  const { ctx, element, calls } = loadScripts(scriptOrder(), { fixtures: { GetBehatSetup: SETUP } });
  const withBehat = ctx.buildDockerRow({ family: "app", type: "moodle", name: "Moodle", folder: "mdle", status: "running", canEdit: true, behat: true });
  const without = ctx.buildDockerRow({ family: "app", type: "moodle", name: "Outro", folder: "outro", status: "running", canEdit: true });
  assert.equal(buttons(without).filter((b) => b.getAttribute("aria-label") === "Configuração do Behat").length, 0);

  const btn = buttons(withBehat).find((b) => b.getAttribute("aria-label") === "Configuração do Behat");
  assert.ok(btn, "no Behat button");
  btn.listeners.click.forEach((fn) => fn());
  await tick();
  assert.deepEqual(calls.filter((c) => c.name === "GetBehatSetup").map((c) => c.args), [["mdle"]]);
  assert.match(element("behat-modal-intro").textContent, /http:\/\/mdle-behat\.test/);
  assert.equal(element("behat-modal-config").textContent, SETUP.configPhp);
  assert.equal(element("behat-modal-commands").textContent, SETUP.initCommand + "\n" + SETUP.runCommands[0]);
  assert.match(element("behat-modal-note").textContent, /deixam de fora os cenários @javascript/);

  assert.match(ctx.updateImageMessage({ family: "selenium", name: "Selenium" }), /imagem do Selenium/);
});

test("Exportar/Importar: Selenium is listed and selected like the other stack containers", () => {
  const { ctx, element } = loadScripts(scriptOrder());
  const list = element("transfer-export-list");
  ctx.renderTransferList(list, { selenium: true, apps: [], plugins: [] }, "export", () => {});
  const box = list.querySelectorAll("input[data-key]").find((b) => b.dataset.key === "selenium");
  assert.ok(box, "no Selenium row");
  box.checked = true;
  const sel = ctx.transferSelection(list);
  assert.equal(sel.selenium, true);
  assert.equal(ctx.selectionCount(sel), 1);
  assert.match(ctx.importConfirmMessage({ commands: [] }, sel), /Selenium/);
});

test("icon buttons have no background and colour only the icon, red for a removal", () => {
  const { ctx } = loadScripts(scriptOrder());
  const edit = ctx.iconButton("pencil", "Editar");
  assert.equal(edit.className, "btn btn-xs o2-icon-btn");
  assert.equal(ctx.iconButton("trash2", "Remover").className, "btn btn-xs o2-icon-btn o2-icon-danger");
  assert.doesNotMatch(edit.className, /btn-ghost/);
});
