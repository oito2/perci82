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

// Run lifecycle (core.js/widgets.js), screen resolution (navigation.js) and
// the async races of the screens that share a container.

import { test } from "node:test";
import assert from "node:assert/strict";
import { FakeElement, evalIn, loadScripts, scriptOrder } from "./harness.mjs";

const click = (el) => el.listeners.click.forEach((fn) => fn({ preventDefault() {} }));
const tick = () => new Promise((r) => setTimeout(r, 0));
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// deferred returns a promise and the functions that settle it.
function deferred() {
  const d = {};
  d.promise = new Promise((resolve, reject) => { d.resolve = resolve; d.reject = reject; });
  return d;
}

// queued makes a fixture that hands out the given deferreds, one per call.
function queued(list) {
  let i = 0;
  return () => list[i++].promise;
}

function load(fixtures = {}) {
  return loadScripts(scriptOrder(), { fixtures });
}

const state = (ctx, expr) => evalIn(ctx, expr);

test("startExecution -> finishRun releases the buttons and reports the result", () => {
  const { ctx, element } = load();
  const b1 = new FakeElement("button");
  const b2 = new FakeElement("button");
  ctx.startExecution("Fazer", [b1, null, b2], 3);
  assert.equal(state(ctx, "running"), true);
  assert.ok(b1.disabled && b2.disabled);
  assert.equal(element("term-title").textContent, "Executando: Fazer...");
  assert.equal(element("tab-radio-terminal").checked, true);

  ctx.finishRun({ ok: true });
  assert.equal(state(ctx, "running"), false);
  assert.equal(state(ctx, "activeRun"), null);
  assert.ok(!b1.disabled && !b2.disabled);
  assert.equal(element("term-title").textContent, "Concluído.");
});

test("failRun ends only a run this window started", () => {
  const { ctx, element } = load();
  ctx.failRun(new Error("sem efeito"));
  assert.equal(element("term-title").textContent, "");
  ctx.startExecution("Fazer");
  ctx.failRun(new Error("recusada"));
  assert.equal(state(ctx, "running"), false);
  assert.equal(element("term-title").textContent, "Falhou.");
});

test("action-done and step events reach only the window that runs", () => {
  const { ctx, element, emit } = load();
  emit("action-done", { ok: true, error: "" }); // another window's run
  assert.equal(element("term-title").textContent, "");
  ctx.startExecution("Fazer");
  emit("step", { index: 1, total: 4, label: "passo" });
  assert.equal(element("term-title").textContent, "Executando: passo");
  assert.equal(element("term-progress").max, 4);
  emit("action-done", { ok: false, error: "x" });
  assert.equal(element("term-title").textContent, "Falhou.");
  assert.equal(state(ctx, "running"), false);
});

test("beginRun: one run at a time, refusals and sync errors end the run", async () => {
  const { ctx, element } = load();
  let calls = 0;
  const call = () => { calls++; return Promise.resolve(); };
  assert.equal(ctx.beginRun("A", call), true);
  assert.equal(ctx.beginRun("B", call), false);
  assert.equal(calls, 1);
  ctx.finishRun({ ok: true });

  assert.equal(ctx.beginRun("C", () => Promise.reject(new Error("outra ação já está em execução"))), true);
  await tick();
  assert.equal(state(ctx, "running"), false);
  assert.equal(element("term-title").textContent, "Falhou.");

  assert.equal(ctx.beginRun("D", () => { throw new Error("boom"); }), true);
  assert.equal(state(ctx, "running"), false);
  assert.equal(element("term-title").textContent, "Falhou.");

  const b = new FakeElement("button");
  ctx.beginRun("E", () => Promise.resolve(), { buttons: [b], maxSteps: 2, announce: "Olá" });
  assert.ok(b.disabled);
  assert.equal(element("term-progress").max, 2);
  ctx.finishRun({ ok: true });
  assert.ok(!b.disabled);
});

test("doneHandlerFor picks the handler by screen family", () => {
  const { ctx } = load();
  const none = ctx.doneHandlerFor("unknown-screen");
  assert.equal(typeof none, "function");
  none({ buttons: [], actionId: "unknown-screen" });
  const run = { buttons: [], actionId: "x" };
  for (const id of ["terminals", "androidstudio", "prereqs", "ides"]) {
    assert.equal(typeof ctx.doneHandlerFor(id), "function", id);
    assert.notEqual(ctx.doneHandlerFor(id), none, id);
    ctx.__selectedItem = { actionId: id, title: id, desc: "" };
    ctx.doneHandlerFor(id)(run);
  }
});

test("screenFor resolves each family to its container", () => {
  const { ctx, element } = load();
  const el = (item) => ctx.screenFor(item).el;
  assert.equal(el({ actionId: "dashboard" }), element("dashboard-container"));
  assert.equal(el({ actionId: "terminals" }), element("multiselect-container"));
  assert.equal(el({ actionId: "androidstudio" }), element("fileapp-container"));
  assert.equal(el({ actionId: "prereqs" }), element("itemtable-container"));
  assert.equal(el({ actionId: "ides" }), element("itemtable-container"));
  assert.equal(el({ actionId: "docker-manage" }), element("dockerlist-container"));
  // Perci82 System's screens are not registered here.
  for (const id of ["fonts", "system-update", "postinstall"]) assert.equal(ctx.screenFor({ actionId: id }), null, id);
  assert.equal(ctx.screenFor({ actionId: "git-transfer" }).alsoShow[0], element("tab-label-git-import"));
  assert.equal(ctx.screenFor({ actionId: "moodleplugins" }).alsoShow[0], element("tab-label-mp-mounted"));
  assert.equal(ctx.screenFor({ actionId: "self-config" }).noTerminal, true);
  assert.equal(evalIn(ctx, "typeof SINGLE_APP_SCREENS"), "undefined");
  assert.equal(evalIn(ctx, "typeof IDE_SCREENS"), "undefined");
});

test("the screen-only tab labels are hidden when another screen is selected", () => {
  const { ctx, element } = load();
  const hidden = ctx.screenContainers();
  assert.ok(hidden.includes(element("tab-label-mp-mounted")));
  assert.ok(hidden.includes(element("tab-label-git-import")));
});

test("quit-blocked: the window running the action opens its Execução tab", () => {
  const { ctx, element, emit } = load();
  ctx.startExecution("Atualizar Sistema", []);
  element("tab-radio-terminal").checked = false;
  emit("quit-blocked");
  assert.equal(element("tab-radio-terminal").checked, true);
  assert.match(element("error-modal-message").textContent, /Aguarde ela terminar na aba Execução/);
});

test("quit-blocked: the main window without the action points to the tray window", () => {
  const { element, emit } = load();
  element("tab-radio-terminal").checked = false;
  emit("quit-blocked");
  assert.notEqual(element("tab-radio-terminal").checked, true);
  assert.match(element("error-modal-message").textContent, /janela da bandeja/);
});

test("showDialog opens a closed dialog once and leaves an open one alone", () => {
  const { ctx } = load();
  let opened = 0;
  const dialog = { open: false, showModal() { opened++; this.open = true; } };
  ctx.showDialog(dialog);
  ctx.showDialog(dialog);
  assert.equal(opened, 1);
});

// FE-01: the checklist's "Executar" is not armed with the previous screen's
// handler while the new items load.
test("checklist: Executar is disabled and unarmed until the items arrive", async () => {
  const slow = deferred();
  const { ctx, element, calls } = load({
    GetAIAppsInfo: [{ id: "f1", label: "F1", installed: true }],
    GetTerminalsInfo: queued([slow]),
  });
  ctx.renderMultiselectScreen({ actionId: "ai-apps", title: "Aplicativos: IA" });
  await tick();
  const btn = element("btn-executar-multiselect");
  assert.equal(typeof btn.onclick, "function");
  assert.equal(btn.disabled, false);

  ctx.renderMultiselectScreen({ actionId: "terminals", title: "Terminais" });
  assert.equal(btn.disabled, true);
  assert.equal(btn.onclick, null);

  slow.resolve([{ id: "t1", label: "T1" }]);
  await tick();
  assert.equal(btn.disabled, false);
  btn.onclick();
  assert.equal(calls.filter((c) => c.name === "RunAIApps").length, 0, "never runs the old screen");
  assert.equal(calls.filter((c) => c.name === "RunTerminals").length, 1);
});

test("item table: a failed query shows the error instead of waiting forever", async () => {
  const { ctx, element } = load({ GetPrereqsInfo: () => Promise.reject(new Error("sem rede")) });
  ctx.renderItemTableScreen({ actionId: "prereqs", desc: "" });
  await tick();
  await tick();
  const cell = element("itemtable-rows").children[0].children[0];
  assert.match(cell.textContent, /Falha ao verificar o que está instalado: Error: sem rede/);
});

test("item table: the error of an older query does not replace a newer screen", async () => {
  const first = deferred();
  const second = deferred();
  const { ctx, element } = load({ GetPrereqsInfo: queued([first, second]) });
  ctx.renderItemTableScreen({ actionId: "prereqs", desc: "" });
  ctx.renderItemTableScreen({ actionId: "prereqs", desc: "" });
  second.resolve([{ id: "Git", label: "Git", description: "", installed: true }]);
  await tick();
  first.reject(new Error("antigo"));
  await tick();
  await tick();
  const rows = element("itemtable-rows").children;
  assert.equal(rows.length, 1);
  assert.doesNotMatch(rows[0].children[0].textContent, /antigo/);
});

// FE-03: an early return of the render invalidates the pending response.
test("scoped table: a slow Global answer does not fill the Local screen without a folder", async () => {
  const slow = deferred();
  const { ctx } = load();
  const els = {
    select: new FakeElement("select"), alertTitle: new FakeElement(), alertDesc: new FakeElement(),
    pickButton: new FakeElement("button"), rows: new FakeElement("tbody"),
  };
  const table = ctx.createScopedTable({
    els, noun: "x", latestKey: "race-test",
    pickFolder: () => Promise.resolve(""),
    fetchRows: () => slow.promise,
    keyOf: (r) => r.slug, sigOf: (r) => r.slug,
    buildRow: () => new FakeElement("tr"),
  });
  table.render(); // Global, response pending
  els.select.value = "local";
  els.select.listeners.change[0](); // Local without a folder
  slow.resolve([{ slug: "a" }]);
  await tick();
  assert.equal(els.rows.children.length, 0);
});

test("IA Contextos: an answer for a folder that was cleared is dropped", async () => {
  const slow = deferred();
  const { ctx, element } = load({ GetAIContextItems: queued([slow]) });
  evalIn(ctx, 'aicFolder = "/proj"');
  ctx.renderAicItems();
  evalIn(ctx, 'aicFolder = ""');
  ctx.renderAicItems();
  slow.resolve([{ id: "a", label: "A" }]);
  await tick();
  assert.equal(element("aic-items").children.length, 0);
  assert.equal(element("btn-aic-apply").disabled, true);
});

// FE-05
test("an update that ends outside Visão Geral makes the next visit check again", () => {
  const { ctx } = load();
  evalIn(ctx, 'selfUpdateCheckedAt = 12345; dashboardRunningAction = "self-update"');
  ctx.__selectedItem = { actionId: "dashboard" };
  ctx.startExecution("Atualizar Perci");
  ctx.__selectedItem = { actionId: "fonts" };
  ctx.finishRun({ ok: true });
  assert.equal(evalIn(ctx, "selfUpdateCheckedAt"), 0);
});

test("the update check shows a failure instead of staying on Verificando", async () => {
  const { ctx, element } = load({ GetSelfUpdateInfo: () => Promise.reject(new Error("offline")) });
  ctx.loadSelfUpdateStatus(true);
  await tick();
  await tick();
  assert.match(element("su-status").textContent, /Não foi possível verificar atualizações: Error: offline/);
  assert.equal(evalIn(ctx, "selfUpdateCheckedAt"), 0);
});

// FE-06
test("theme grid: two renders in a row leave one button per theme", async () => {
  const { ctx, element } = load({
    GetThemeOptions: [{ id: "system", label: "Definido pelo sistema" }, { id: "a", label: "A", scheme: "a" }, { id: "b", label: "B", scheme: "b" }],
  });
  ctx.renderThemeGrid("a");
  ctx.renderThemeGrid("b");
  await tick();
  await tick();
  assert.equal(element("cfg-theme-grid").children.length, 3);
});

// FE-09
test("Moodle PHP versions: a stale answer does not replace a newer one", async () => {
  const slow = deferred();
  const fast = deferred();
  const { ctx, element } = load({ GetPHPVersionsForMoodleVersion: queued([slow, fast]) });
  ctx.wireMoodlePHPFilter("dc"); // first refresh, pending
  element("dc-moodleVersion").listeners.change[0](); // second refresh
  fast.resolve(["8.3"]);
  await tick();
  slow.resolve(["7.4"]);
  await tick();
  assert.ok(element("dc-phpVersion-wrap").innerHTML.includes("8.3"));
  assert.ok(!element("dc-phpVersion-wrap").innerHTML.includes("7.4"));
});

// FE-10
test("Plugins Moodle: typing the name checks the backend once, after a pause", async () => {
  const { ctx, element, calls } = load({
    InspectDevFolder: { ok: true, destLabel: "x", locked: false },
  });
  element("mp-type").options = [];
  evalIn(ctx, 'mpDev = "/ws/dev"');
  const input = element("mp-name").listeners.input[0];
  input();
  input();
  input();
  assert.equal(element("btn-mp-create").disabled, true);
  assert.equal(calls.filter((c) => c.name === "InspectDevFolder").length, 0);
  await sleep(350);
  assert.equal(calls.filter((c) => c.name === "InspectDevFolder").length, 1);
});

test("Plugins Moodle: the screen load and the table reload use separate request keys", async () => {
  const screen = deferred();
  const rows = deferred();
  const { ctx, element } = load({ GetDevFolders: queued([screen, rows]) });
  element("mp-app").options = [];
  ctx.renderMoodlePluginsScreen();
  ctx.loadMpRows();
  rows.resolve({ rows: [] });
  await tick();
  screen.resolve({ apps: [{ folder: "m", label: "M" }], rows: [] });
  await tick();
  assert.equal(element("mp-app").disabled, false, "the screen's own answer was not discarded");
});

// FE-11
test("form field helpers escape id, label and hint", () => {
  const { ctx } = load();
  const bad = `x"><img src=x onerror=1>`;
  for (const html of [
    ctx.textFieldHTML(bad, bad, {}),
    ctx.selectFieldHTML(bad, bad, [], ""),
    ctx.checkboxFieldHTML(bad, bad, false, false, bad),
  ]) {
    assert.ok(!html.includes("<img"), html);
  }
});

// FE-12
test("shell: section tabs, side list with groups, title and the active entries", async () => {
  const sections = [
    { id: "overview", label: "Visão Geral", icon: "house", items: [{ title: "Visão Geral", actionId: "dashboard", desc: "" }] },
    { id: "repositories", label: "Repositórios", icon: "gitBranch", items: [
      { title: "Configurar Conta", actionId: "git-accounts", tab: "Contas", desc: "" },
      { title: "OITO2", actionId: "git-account", accountId: "oito2", tab: "OITO2", group: "Contas", desc: "" },
      { title: "Labs", actionId: "git-account", accountId: "labs", tab: "Labs", group: "Contas", desc: "" },
    ] },
  ];
  const { ctx, element } = load({ GetSections: sections });
  await tick();
  await tick();
  const tabs = element("section-tabs").children;
  assert.deepEqual(tabs.map((t) => t.children[0].textContent), ["Visão Geral", "Repositórios"]);
  assert.equal(tabs[0].getAttribute("aria-selected"), "true", "the main window opens on the first section");
  assert.equal(element("screen-title").textContent, "Visão Geral");

  click(tabs[1]);
  const side = element("side-list");
  assert.deepEqual(side.children.map((c) => c.className + ":" + c.textContent),
    ["o2-side-item:Configurar Conta", "o2-side-group:Contas", "o2-side-item:OITO2", "o2-side-item:Labs"]);
  assert.equal(tabs[1].getAttribute("aria-selected"), "true");
  assert.equal(tabs[0].getAttribute("aria-selected"), "false");
  assert.equal(element("screen-title").textContent, "Configurar Conta");
  assert.equal(element("tab-desc-label").textContent, "Contas");
  assert.equal(side.children[0].getAttribute("aria-current"), "page");

  click(side.children[3]);
  assert.equal(element("screen-title").textContent, "Labs");
  assert.equal(side.children[3].getAttribute("aria-current"), "page");
  assert.equal(side.children[0].getAttribute("aria-current"), null);
});

test("shell: a language switch reloads the sections and relabels the open screen", async () => {
  let locale = "pt-BR";
  const label = { "pt-BR": "Configurações", en: "Settings" };
  const { ctx, element, calls, emit } = load({
    GetLanguage: { choice: "system", locale: "pt-BR" },
    GetSections: () => [{ id: "settings", label: label[locale], icon: "house", items: [{ title: label[locale], actionId: "self-config", desc: "" }] }],
  });
  await tick();
  await tick();
  const before = calls.filter((c) => c.name === "GetSections").length;
  assert.equal(before, 1, "startup loads the sections once");
  assert.equal(element("screen-title").textContent, "Configurações");

  locale = "en";
  emit("language-changed", { choice: "en", locale: "en" });
  await tick();
  await tick();
  assert.equal(calls.filter((c) => c.name === "GetSections").length, before + 1);
  assert.equal(element("section-tabs").children[0].children[0].textContent, "Settings");
  assert.equal(element("screen-title").textContent, "Settings");
  assert.equal(ctx.__selectedItem.title, "Settings");
});
