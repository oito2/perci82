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
const settle = async () => { for (let i = 0; i < 5; i++) await tick(); };

const ITEMS = [
  { id: "Go", label: "Go", description: "Go.", base: "", custom: false, status: "" },
  { id: "Go: Wails v3", label: "Go: Wails v3", description: "Wails.", base: "Go", custom: false, status: "" },
  { id: "custom:team", label: "Team", description: "Always", base: "", custom: true, status: "" },
];
const CUSTOM = [{ id: "team", name: "Team", appliesTo: "Always", content: "Rule.\n" }];

async function screen(fixtures = {}) {
  const loaded = loadScripts(scriptOrder(), {
    fixtures: { GetAIContextItems: ITEMS, GetCustomContexts: CUSTOM, ...fixtures },
  });
  await settle();
  const click = (el) => (el.listeners.click || []).forEach((fn) => fn({ preventDefault() {} }));
  const called = (name) => loaded.calls.filter((c) => c.name === name);
  return { ...loaded, click, called };
}

test("Gerenciar Contextos: the Perci82 contexts are listed with Visualizar only", async () => {
  const { ctx, element, click, called } = await screen({ GetAIContextStandardText: "# Go\n" });
  evalIn(ctx, "renderAicLibraryScreen()");
  await settle();
  const rows = element("aic-lib-rows").children;
  assert.deepEqual(rows.map((tr) => tr.children[0].textContent), ["Go", "Go: Wails v3"], "custom contexts stay out");
  const buttons = rows[0].children[2].children;
  assert.deepEqual(buttons.map((b) => b.textContent), ["Visualizar"]);
  click(buttons[0]);
  await settle();
  assert.deepEqual(called("GetAIContextStandardText")[0].args, ["Go"]);
  assert.equal(element("aic-view-content").textContent, "# Go\n");
});

test("Gerenciar Contextos: custom contexts get Visualizar, Editar and Apagar", async () => {
  const { ctx, element, click, called } = await screen();
  evalIn(ctx, "renderAicLibraryScreen()");
  await settle();
  const rows = element("aic-custom-rows").children;
  assert.equal(rows.length, 1);
  assert.deepEqual(rows[0].children.slice(0, 2).map((td) => td.textContent), ["Team", "Always"]);
  const [view, edit, del] = rows[0].children[2].children[0].children;
  assert.deepEqual([view, edit, del].map((b) => b.textContent), ["Visualizar", "Editar", "Apagar"]);

  click(view);
  assert.equal(element("aic-view-content").textContent, "Rule.\n");

  click(edit);
  assert.equal(element("aic-custom-name").value, "Team");
  element("aic-custom-content").value = "Rule 2.";
  click(element("btn-aic-custom-save"));
  await settle();
  assert.deepEqual(JSON.parse(JSON.stringify(called("SaveCustomContext")[0].args[0])),
    { id: "team", name: "Team", appliesTo: "Always", content: "Rule 2." });

  click(del);
  element("btn-confirm-ok").listeners.click.forEach((fn) => fn({}));
  await settle();
  assert.deepEqual(called("DeleteCustomContext")[0].args, ["team"]);
});

test("Gerenciar Contextos: Adicionar opens an empty form and needs a name and content", async () => {
  const { element, click, called } = await screen();
  element("aic-custom-name").value = "old";
  click(element("btn-aic-custom-add"));
  assert.equal(element("aic-custom-name").value, "");
  click(element("btn-aic-custom-save"));
  await settle();
  assert.equal(called("SaveCustomContext").length, 0);
  element("aic-custom-name").value = "Novo";
  element("aic-custom-content").value = "texto";
  click(element("btn-aic-custom-save"));
  await settle();
  assert.equal(called("SaveCustomContext")[0].args[0].id, "");
});

test("Aplicar contexto: custom contexts go to their own section and a project type leaves them alone", async () => {
  const presets = [{ name: "Wails", label: "Wails", models: ["Go", "Go: Wails v3"], hint: "" }];
  const { ctx, element } = await screen({
    GetAIContextPresets: presets,
    GetAIContextRecord: { registered: false, hasContext: false, name: "", models: [], agents: [] },
    GetAIContextAgents: [{ id: "claude", label: "Claude Code", selected: true }],
  });
  evalIn(ctx, 'aicFolder = "/ws/p"; renderAicItems()');
  await settle();
  const ids = (ul) => ul.children.map((li) => li.children[0].id);
  assert.deepEqual(ids(element("aic-items")), ["aic-check-Go", "aic-check-Go: Wails v3"]);
  assert.deepEqual(ids(element("aic-custom-items")), ["aic-check-custom:team"]);

  const custom = element("aic-custom-items").children[0].children[0];
  custom.checked = true;
  element("aic-preset").value = "Wails";
  element("aic-preset").onchange();
  assert.equal(custom.checked, true, "the project type kept the custom context");
  assert.equal(element("aic-items").children[1].children[0].checked, true);
});
