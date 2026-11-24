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

const AGENTS = [
  { id: "claude", label: "Claude Code", selected: true },
  { id: "codex", label: "ChatGPT CLI (Codex)", selected: true },
];
const ITEMS = [
  { id: "Go", label: "Go", description: "Go.", base: "", status: "applied" },
  { id: "Linux Bash", label: "Linux Bash", description: "Bash.", base: "", status: "" },
];
const ROWS = [
  { name: "Perci", path: "/ws/perci", category: "Trabalho", state: "applied", divergent: false, models: ["Go"], agents: ["claude"] },
  { name: "Velho", path: "/ws/velho", category: "", state: "outdated", divergent: true, models: [], agents: ["codex"] },
  { name: "Mexido", path: "/ws/mexido", category: "Trabalho", state: "edited", divergent: false, models: ["Go"], agents: ["claude", "codex"] },
  { name: "Longe", path: "/ws/longe", category: "", state: "pending", divergent: false, models: ["Go"], agents: ["claude"] },
];
const CATEGORIES = [{ name: "Pessoal", uses: 0 }, { name: "Trabalho", uses: 2 }];

async function screen(fixtures = {}) {
  const loaded = loadScripts(scriptOrder(), {
    fixtures: { GetAIContextAgents: AGENTS, GetAIContextItems: ITEMS, GetAIContexts: ROWS, GetAICategories: CATEGORIES, ...fixtures },
  });
  await tick();
  loaded.ctx.loadAicRows();
  loaded.ctx.loadAicCategories();
  await settle();
  const rows = () => loaded.element("aic-rows").children;
  const actions = (tr) => tr.children[3].children[0].children; // [Atualizar, Alterar, +, Remover]
  const click = (el) => (el.listeners.click || []).forEach((fn) => fn({ preventDefault() {} }));
  const confirm = () => loaded.element("btn-confirm-ok").listeners.click.forEach((fn) => fn({}));
  // aicConfirm answers the AI Contexts confirmation, with the backup box
  // checked or not.
  const aicConfirm = (backup) => {
    loaded.element("aic-confirm-backup").checked = !!backup;
    loaded.element("btn-aic-confirm-ok").listeners.click.forEach((fn) => fn({}));
  };
  const called = (name) => loaded.calls.filter((c) => c.name === name);
  const resetRun = () => evalIn(loaded.ctx, "running = false; activeRun = null");
  return { ...loaded, rows, actions, click, confirm, aicConfirm, called, resetRun };
}

test("IA: Contextos: the list shows each folder with its name, folder, state and actions", async () => {
  const { rows, actions, element } = await screen();
  assert.equal(rows().length, 4);
  const cells = (tr) => tr.children.slice(0, 2).map((td) => td.textContent);
  assert.deepEqual(cells(rows()[0]), ["Perci", "/ws/perci"]);
  assert.deepEqual(cells(rows()[1]), ["Velho", "/ws/velho"]);
  assert.equal(rows()[0].children.length, 4, "Nome | Pasta | Estado | Ações");
  const badges = (tr) => tr.children[2].children.map((b) => b.textContent);
  assert.deepEqual(badges(rows()[0]), ["em dia"]);
  assert.deepEqual(badges(rows()[1]), ["desatualizado", "seleção diferente"]);
  assert.deepEqual(badges(rows()[2]), ["editado"]);
  assert.deepEqual(badges(rows()[3]), ["pendente"]);
  assert.deepEqual(actions(rows()[3]).map((b) => b.disabled === true), [true, true, false, false], "a pending folder can only be viewed and removed");
  assert.deepEqual(actions(rows()[0]).map((b) => b.disabled === true), [false, false, false, false]);
  assert.equal(element("btn-aic-update-all").disabled, false);
});

test("IA: Contextos: Atualizar todos previews the outdated and edited folders and updates the ones that change", async () => {
  const previews = [
    { name: "Velho", path: "/ws/velho", pending: false, changes: [{ path: "AGENTS.md", action: "update" }] },
    { name: "Mexido", path: "/ws/mexido", pending: false, changes: [] },
  ];
  const { element, click, aicConfirm, called } = await screen({ PreviewAIContextUpdate: previews });
  click(element("btn-aic-update-all"));
  await settle();
  assert.deepEqual(JSON.parse(JSON.stringify(called("PreviewAIContextUpdate")[0].args[0])), ["/ws/velho", "/ws/mexido"]);
  assert.equal(element("aic-confirm-backup").checked, false, "the backup box starts unchecked");
  const msg = element("aic-confirm-message").textContent;
  assert.match(msg, /Velho — \/ws\/velho\n {2}• atualizar: AGENTS\.md/);
  assert.match(msg, /Mexido — \/ws\/mexido \(tem arquivos editados à mão\)\n {2}• nada a mudar/);
  aicConfirm(true);
  assert.deepEqual(JSON.parse(JSON.stringify(called("UpdateAIContexts")[0].args)), [["/ws/velho"], true]);
});

test("IA: Contextos: Atualizar with nothing to change runs nothing; with all folders up to date the button is off", async () => {
  const { rows, actions, click, element, called } = await screen({
    PreviewAIContextUpdate: [{ name: "Perci", path: "/ws/perci", pending: false, changes: [] }],
  });
  click(actions(rows()[0])[0]);
  await settle();
  assert.match(element("error-modal-message").textContent, /já estão/);
  assert.equal(called("UpdateAIContexts").length, 0);

  const upToDate = await screen({ GetAIContexts: [ROWS[0], ROWS[3]] });
  assert.equal(upToDate.element("btn-aic-update-all").disabled, true);
  const empty = await screen({ GetAIContexts: [] });
  assert.equal(empty.rows().length, 0);
  assert.equal(empty.element("btn-aic-export").disabled, true);
});

test("IA: Contextos: Remover previews the files and removes after the confirmation", async () => {
  const { rows, actions, click, confirm, element, called } = await screen({
    PreviewAIContextRemove: [{ path: "AGENTS.md", action: "remove" }],
  });
  click(actions(rows()[1])[3]);
  await settle();
  assert.match(element("confirm-modal-message").textContent, /• remover: AGENTS\.md[\s\S]*PROJECT\.md são mantidos[\s\S]*Remover "Velho" da lista\?/);
  confirm();
  assert.equal(called("RemoveAIContext")[0].args[0], "/ws/velho");
});

test("IA: Contextos: Alterar opens the folder on Configurar Projeto with its recorded name, agents and standards", async () => {
  const record = { registered: true, hasContext: true, name: "Mexido", models: ["Linux Bash"], agents: ["codex"] };
  const { rows, actions, click, element, ctx } = await screen({ GetAIContextRecord: record });
  click(actions(rows()[2])[1]);
  await settle();
  assert.equal(evalIn(ctx, "aicFolder"), "/ws/mexido");
  assert.equal(element("aic-name").value, "Mexido");
  const agentBoxes = element("aic-agents").children.map((l) => [l.children[0].dataset.agent, l.children[0].checked]);
  assert.deepEqual(agentBoxes, [["claude", false], ["codex", true]]);
  assert.deepEqual(element("aic-items").children.map((li) => li.children[0].checked), [false, true], "the record wins over the files");
});

test("IA: Contextos: a folder with an unregistered context offers Registrar, which needs a free name", async () => {
  const record = { registered: false, hasContext: true, name: "", models: ["Go"], agents: ["claude"] };
  const { element, ctx, click, called } = await screen({ GetAIContextRecord: record });
  evalIn(ctx, 'aicFolder = "/ws/antigo"; renderAicItems();');
  await settle();
  assert.equal(element("aic-name").value, "");
  assert.match(element("aic-unregistered-desc").textContent, /padrões: Go; agentes: Claude Code/);

  click(element("btn-aic-register"));
  assert.match(element("error-modal-message").textContent, /Informe o nome/);
  element("aic-name").value = "perci";
  click(element("btn-aic-register"));
  assert.match(element("error-modal-message").textContent, /já é usado pela pasta \/ws\/perci/);
  assert.equal(called("RegisterAIContext").length, 0);

  element("aic-name").value = " Antigo ";
  click(element("btn-aic-register"));
  await settle();
  assert.deepEqual(called("RegisterAIContext")[0].args, ["/ws/antigo", "Antigo", ""]);
});

test("IA: Contextos: Aplicar sends the name; renaming without file changes asks to update only the record", async () => {
  const record = { registered: true, hasContext: true, name: "Perci", models: ["Go"], agents: ["claude"] };
  const { element, ctx, called, confirm } = await screen({ GetAIContextRecord: record, PreviewAIContext: [] });
  evalIn(ctx, 'aicFolder = "/ws/perci"; renderAicItems();');
  await settle();
  element("aic-name").value = "Mexido";
  element("btn-aic-apply").onclick();
  assert.match(element("error-modal-message").textContent, /já é usado/);
  assert.equal(called("PreviewAIContext").length, 0);

  element("aic-name").value = "Perci 2";
  element("btn-aic-apply").onclick();
  await settle();
  assert.match(element("confirm-modal-message").textContent, /só o registro de "Perci 2"/);
  confirm();
  const apply = called("ApplyAIContext")[0];
  assert.deepEqual(JSON.parse(JSON.stringify(apply.args)), ["/ws/perci", "Perci 2", "", ["Go"], ["claude"], false], "nothing replaced: no backup");
});

test("IA: Contextos: Exportar saves the checked folders", async () => {
  const { element, click, called } = await screen({ PickAIContextExportPath: "/tmp/ctx.yaml" });
  const boxes = element("aic-export-list").children.map((row) => row.children[0]);
  assert.equal(boxes.length, 4);
  boxes[1].checked = false;
  boxes[1].listeners.change.forEach((fn) => fn());
  click(element("btn-aic-export"));
  await settle();
  const exp = called("ExportAIContexts")[0];
  assert.deepEqual(JSON.parse(JSON.stringify(exp.args)), ["/tmp/ctx.yaml", ["/ws/perci", "/ws/mexido", "/ws/longe"]]);
  assert.match(element("error-modal-message").textContent, /3 pasta\(s\) exportada/);

  boxes.forEach((b) => { b.checked = false; b.listeners.change.forEach((fn) => fn()); });
  assert.equal(element("btn-aic-export").disabled, true);
});

test("IA: Contextos: aicImportNotes blocks empty, taken and repeated names", async () => {
  const { ctx } = await screen();
  const notes = (picks, taken) => JSON.parse(JSON.stringify(ctx.aicImportNotes(picks, taken)));
  assert.deepEqual(notes([{ source: "A", name: "A" }, { source: "B", name: "B" }], ["C"]), []);
  assert.equal(notes([{ source: "A", name: "" }], []).length, 1);
  assert.match(notes([{ source: "A", name: "perci" }], ["Perci"])[0], /já é usado/);
  assert.match(notes([{ source: "A", name: "X" }, { source: "B", name: "x" }], [])[0], /mais de um/);
});

test("IA: Contextos: Importar lists the file, edits names and folders, and imports the picks after a preview", async () => {
  const preview = {
    taken: ["Perci"],
    items: [
      { name: "Perci", path: "/ws/perci2", models: ["Go"], agents: ["claude"], unknown: [], state: "apply", reason: "" },
      { name: "Novo", path: "/ws/novo", models: [], agents: ["codex"], unknown: ["future"], state: "pending", reason: "" },
      { name: "Aqui", path: "/ws/perci", models: [], agents: ["codex"], unknown: [], state: "registered", reason: "pasta já registrada nesta máquina" },
    ],
  };
  const changes = [
    { name: "Perci B", path: "/ws/outra", pending: false, changes: [{ path: "AGENTS.md", action: "create" }] },
    { name: "Novo", path: "/ws/novo", pending: true, changes: [] },
  ];
  const { element, click, aicConfirm, called } = await screen({
    PickAIContextImportPath: "/tmp/in.yaml", PreviewAIContextImport: preview,
    PickAIContextFolder: "/ws/outra", PreviewAIContextImportChanges: changes,
  });
  click(element("btn-aic-import-pick"));
  await settle();
  const list = element("aic-import-list").children;
  assert.equal(list.length, 3);
  const box = (i) => list[i].children[0];
  const input = (i) => list[i].children[1].children[2].children[0];
  assert.deepEqual([0, 1, 2].map((i) => [box(i).checked, !!box(i).disabled]), [[true, false], [true, false], [false, true]]);
  assert.match(list[1].children[1].children[1].textContent, /ficam de fora: future/);
  assert.equal(element("btn-aic-import").disabled, true, "Perci is taken here");
  assert.match(element("aic-import-note-text").textContent, /"Perci" já é usado/);

  input(0).value = "Perci B";
  input(0).listeners.input.forEach((fn) => fn());
  assert.equal(element("btn-aic-import").disabled, false);

  // Trocar pasta reloads the preview with the picked folder; the typed
  // name survives the reload.
  click(list[0].children[1].children[2].children[1]);
  await settle();
  const last = called("PreviewAIContextImport").at(-1);
  assert.deepEqual(JSON.parse(JSON.stringify(last.args)), ["/tmp/in.yaml", { Perci: "/ws/outra" }]);
  assert.equal(input(0).value, "Perci B");

  click(element("btn-aic-import"));
  await settle();
  const picks = [{ source: "Perci", name: "Perci B", path: "/ws/outra" }, { source: "Novo", name: "Novo", path: "" }];
  assert.deepEqual(JSON.parse(JSON.stringify(called("PreviewAIContextImportChanges")[0].args)), ["/tmp/in.yaml", picks]);
  assert.match(element("aic-confirm-message").textContent, /Novo — \/ws\/novo\n {2}• fica pendente/);
  aicConfirm(false);
  assert.deepEqual(JSON.parse(JSON.stringify(called("ImportAIContexts")[0].args)), ["/tmp/in.yaml", picks, false]);
});

test("IA: Contextos: Alterar keeps the folder when the menu item renders Aplicar Contexto", async () => {
  const sections = [{ id: "ai", label: "AI", items: [
    { title: "Aplicar Contexto", actionId: "aicontext-define" },
    { title: "Gerenciar Contextos", actionId: "aicontext-library" },
  ] }];
  const record = { registered: true, hasContext: true, name: "Perci", models: ["Go"], agents: ["claude"] };
  const { rows, actions, click, ctx } = await screen({ GetSections: sections, GetAIContextRecord: record });
  evalIn(ctx, "loadMenu()");
  await settle();
  evalIn(ctx, "loadAicRows()");
  await settle();
  click(actions(rows()[0])[1]);
  await settle();
  assert.equal(evalIn(ctx, "aicFolder"), "/ws/perci");
});

test("IA: Contextos: Filtrar shows a category's folders, and Atualizar todos covers only those", async () => {
  const { rows, element, click, called } = await screen({ PreviewAIContextUpdate: [] });
  const options = element("aic-filter").children.map((o) => o.textContent);
  assert.deepEqual(options, ["Todas", "Sem categoria", "Pessoal", "Trabalho"]);
  const pick = (value) => { element("aic-filter").value = value; element("aic-filter").listeners.change.forEach((fn) => fn()); };

  pick("Trabalho");
  assert.deepEqual(rows().map((tr) => tr.children[0].textContent), ["Perci", "Mexido"]);
  click(element("btn-aic-update-all"));
  await settle();
  assert.deepEqual(JSON.parse(JSON.stringify(called("PreviewAIContextUpdate")[0].args[0])), ["/ws/mexido"]);

  pick(element("aic-filter").children[1].value); // Sem categoria
  assert.deepEqual(rows().map((tr) => tr.children[0].textContent), ["Velho", "Longe"]);

  pick("Pessoal");
  assert.equal(rows().length, 0);
  assert.equal(element("btn-aic-update-all").disabled, true);
});

test("IA: Contextos: + shows the folder's category, standards and agents as badges", async () => {
  const { rows, actions, click, element } = await screen();
  click(actions(rows()[2])[2]);
  assert.equal(element("aic-details-title").textContent, "Mexido");
  assert.equal(element("aic-details-category").textContent, "Trabalho");
  const badges = (id) => element(id).children[0].children.map((b) => b.textContent);
  assert.deepEqual(badges("aic-details-models"), ["Go"]);
  assert.deepEqual(badges("aic-details-agents"), ["Claude Code", "ChatGPT CLI (Codex)"]);
  click(actions(rows()[1])[2]);
  assert.equal(element("aic-details-category").textContent, "Sem categoria");
});

test("IA: Contextos: Categorias adds, renames and deletes, telling how many projects lose it", async () => {
  const { element, click, confirm, called } = await screen();
  const catRows = element("aic-cat-rows").children;
  assert.deepEqual(catRows.map((tr) => [tr.children[0].textContent, tr.children[1].textContent]), [["Pessoal", "0"], ["Trabalho", "2"]]);

  element("aic-cat-new").value = "  ";
  click(element("btn-aic-cat-add"));
  assert.match(element("error-modal-message").textContent, /Informe o nome/);
  element("aic-cat-new").value = " Clientes ";
  click(element("btn-aic-cat-add"));
  await settle();
  assert.deepEqual(called("AddAICategory")[0].args, ["Clientes"]);

  const act = (i) => catRows[i].children[2].children[0].children; // [Renomear, Apagar]
  click(act(1)[0]);
  assert.equal(element("aic-cat-name").value, "Trabalho");
  element("aic-cat-name").value = "Empresa";
  click(element("btn-aic-cat-save"));
  await settle();
  assert.deepEqual(called("RenameAICategory")[0].args, ["Trabalho", "Empresa"]);

  click(act(1)[1]);
  assert.match(element("confirm-modal-message").textContent, /2 projeto\(s\) ficarão sem categoria/);
  confirm();
  await settle();
  assert.deepEqual(called("DeleteAICategory")[0].args, ["Trabalho"]);
});

test("IA: Contextos: the form shows the agents disabled until a folder is chosen, and sends the category", async () => {
  const record = { registered: true, hasContext: true, name: "Perci", category: "Trabalho", models: ["Go"], agents: ["claude"] };
  const { element, ctx, called, confirm } = await screen({ GetAIContextRecord: record, PreviewAIContext: [] });
  evalIn(ctx, 'aicFolder = ""; renderAicItems();');
  await settle();
  const boxes = () => element("aic-agents").children.map((l) => l.children[0]);
  assert.equal(boxes().length, 2);
  assert.ok(boxes().every((b) => b.disabled), "agents disabled without a folder");
  assert.equal(element("aic-category").disabled, true);
  assert.deepEqual(element("aic-category").children.map((o) => o.textContent), ["Sem categoria", "Pessoal", "Trabalho"]);

  evalIn(ctx, 'aicFolder = "/ws/perci"; renderAicItems();');
  await settle();
  assert.ok(boxes().every((b) => !b.disabled));
  assert.equal(element("aic-category").value, "Trabalho");
  element("aic-category").value = "Pessoal";
  element("btn-aic-apply").onclick();
  await settle();
  confirm();
  assert.equal(called("ApplyAIContext")[0].args[2], "Pessoal");
});

test("IA: Contextos: each Exportar row shows the name, folder, agents and contexts as badges", async () => {
  const { element } = await screen();
  const row = element("aic-export-list").children[2];
  const text = row.children[1];
  assert.equal(text.children[0].textContent, "Mexido");
  assert.equal(text.children[1].textContent, "/ws/mexido");
  const badgeLine = (i) => [text.children[i].children[0].textContent, text.children[i].children[1].children.map((b) => b.textContent)];
  assert.deepEqual(badgeLine(2), ["Agentes aplicados:", ["Claude Code", "ChatGPT CLI (Codex)"]]);
  assert.deepEqual(badgeLine(3), ["Contextos aplicados:", ["Go"]]);
});
