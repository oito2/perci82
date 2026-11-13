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

// Shared state and plumbing: the screen registries, shared DOM references,
// theme, inner tabs, the xterm terminal and the Execução run lifecycle
// (log-line/step/action-done events).
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// Any rejected backend call nobody handled (a Go method returning an error,
// a failed dialog...) is shown instead of vanishing into the webview's
// console. Run-starting calls handle their own (failRun).
window.addEventListener("unhandledrejection", function (ev) {
  ev.preventDefault();
  const reason = ev.reason;
  openError(t("error.unexpected", { error: (reason && reason.message) || reason }));
});


const tabDescLabel = document.getElementById("tab-desc-label");
const tabLabelDesc = document.getElementById("tab-label-desc");
const tabRadioDesc = document.getElementById("tab-radio-desc");
const tabRadioTerminal = document.getElementById("tab-radio-terminal");

// --- screen registries ------------------------------------------------------
// Each application's scripts fill these in: SCREENS maps a menu item's
// actionId to { el, render(item), alsoShow?, noTerminal? }; SCREEN_FAMILIES
// lists screens shared by several actionIds, as { has(actionId), el,
// render(item), done(run) }; ACTION_DONE_HANDLERS maps an actionId to the
// refresh that runs when an action ends while its screen shows; ACTIONS
// maps an actionId to the call its screen's "Executar" button starts.
const SCREENS = {};
const SCREEN_FAMILIES = [];
const ACTION_DONE_HANDLERS = {};
const ACTIONS = {};

// The tray's compact window loads the page with ?compact=<tab>; null in the
// main window.
const compactAction = new URLSearchParams(location.search).get("compact");

// --- theme -----------------------------------------------------------------
// The theme choice ("system" or an oito2 preset ID), the presets offered
// and the scheme "system" resolves to. The scheme shown is the
// data-oito2-scheme attribute of <html>; the selector grid is built by
// renderSelfConfigScreen.
let themeChoice = "system";
let themeOptions = [];
let systemScheme = { scheme: "light-blue", highContrast: false };

// Returns the CSS scheme name of choice: the system's scheme for "system"
// (or an unknown choice), otherwise the preset's scheme, in its high
// contrast variant while the system asks for high contrast.
function schemeFor(choice) {
  const preset = themeOptions.find(function (o) { return o.id === choice; });
  if (choice === "system" || !preset || !preset.scheme) return systemScheme.scheme;
  return preset.scheme + (systemScheme.highContrast ? "-hc" : "");
}

function applyTheme(choice) {
  themeChoice = choice;
  document.documentElement.setAttribute("data-oito2-scheme", schemeFor(choice));
}

Promise.all([App.GetThemeOptions(), App.GetSystemScheme(), App.GetTheme()]).then(function (r) {
  themeOptions = r[0] || [];
  if (r[1] && r[1].scheme) systemScheme = r[1];
  applyTheme(r[2] || "system");
});

// The desktop's appearance changed: "system" (and the contrast of a
// preset) follows it at once.
Events.On("system-scheme-changed", function (ev) {
  if (!ev.data || !ev.data.scheme) return;
  systemScheme = ev.data;
  applyTheme(themeChoice);
});

// --- inner tabs (first tab / Execução) — each tab-content's visibility is
// handled via CSS by the matching radio's :checked; the code only checks
// the right radio. --
function selectTab(which) {
  (which === "desc" ? tabRadioDesc : tabRadioTerminal).checked = true;
}

// --- terminal + run lifecycle ---------------------------------------------

const term = new Terminal({
  fontSize: 13,
  convertEol: true,
  disableStdin: true,
  allowTransparency: true, // needed for the transparent background below
  scrollback: 5000,
  theme: { background: "#00000000" },
});
const terminalEl = document.getElementById("terminal");
term.open(terminalEl);
// The idle line waits for the saved language.
i18nReady.then(function () {
  if (!activeRun) term.writeln(t("term.idle"));
});

// FitAddon sizes the terminal to its panel
// instead of xterm's fixed 80x24. The observer also fires when the
// Execução tab becomes visible (0 → real size); a hidden panel is skipped.
const termFit = new FitAddon.FitAddon();
term.loadAddon(termFit);
new ResizeObserver(function () {
  if (terminalEl.clientWidth > 0 && terminalEl.clientHeight > 0) termFit.fit();
}).observe(terminalEl);

const termTitle = document.getElementById("term-title");
const termProgress = document.getElementById("term-progress");
const multiselectContainer = document.getElementById("multiselect-container");
const multiselectIntro = document.getElementById("multiselect-intro");
const multiselectItemsList = document.getElementById("multiselect-items");
const btnExecutarMultiselect = document.getElementById("btn-executar-multiselect");
const multiselectExtras = document.getElementById("multiselect-extras");
const confirmModal = document.getElementById("confirm-modal");
const confirmModalTitle = document.getElementById("confirm-modal-title");
const confirmModalMessage = document.getElementById("confirm-modal-message");
const btnConfirmOk = document.getElementById("btn-confirm-ok");
const errorModal = document.getElementById("error-modal");
const errorModalMessage = document.getElementById("error-modal-message");
const tabLabelTerminal = document.getElementById("tab-label-terminal");
const dashboardContainer = document.getElementById("dashboard-container");
const dashboardVersion = document.getElementById("dashboard-version");
const suStatus = document.getElementById("su-status");
const btnSuCheck = document.getElementById("btn-su-check");
const btnSuUpdate = document.getElementById("btn-su-update");
const su2RemoveConfig = document.getElementById("su2-remove-config");
const btnSu2Uninstall = document.getElementById("btn-su2-uninstall");
const selfconfigContainer = document.getElementById("selfconfig-container");
const cfgThemeGrid = document.getElementById("cfg-theme-grid");
// Buttons built from MULTISELECT_SCREENS[actionId].extras for the screen
// currently shown (renderMultiselectScreen).
let multiselectExtraButtons = [];

let running = false;

// activeRun describes the action THIS window started (startExecution):
// the actionId of the screen it was started from, the buttons it disabled
// and runId, the number the backend gave the action ("action-start"; null
// until it arrives). null when this window isn't running anything — the
// log-line/step/action-done events reach every window (main + the tray's
// compact window), and only the one that started the action handles them,
// by its number: another window's action never writes here.
let activeRun = null;

// ownsRunEvent reports whether an event of a run (log-line, step,
// action-done) belongs to this window's action.
function ownsRunEvent(data) {
  return !!activeRun && activeRun.runId !== null && data && data.runId === activeRun.runId;
}

// The first action started after this window asked for one is its own:
// the backend runs one action at a time and refuses the others.
Events.On("action-start", function (ev) {
  if (activeRun && activeRun.runId === null) activeRun.runId = ev.data.runId;
});

// Events.On delivers the payload inside ev.data.
// Output is written once per frame instead of once per event: apt or
// docker build emit thousands of small events.
let pendingLog = "";
Events.On("log-line", function (ev) {
  if (!ownsRunEvent(ev.data)) return;
  if (!pendingLog) requestAnimationFrame(flushLog);
  pendingLog += ev.data.text;
});
function flushLog() {
  if (pendingLog) term.write(pendingLog);
  pendingLog = "";
}

Events.On("step", function (ev) {
  if (!ownsRunEvent(ev.data)) return;
  const step = ev.data;
  termTitle.textContent = t("term.running", { label: step.label });
  termProgress.value = step.index;
  termProgress.max = step.total;
});

Events.On("action-done", function (ev) {
  if (!ownsRunEvent(ev.data)) return;
  finishRun(ev.data);
});

// The backend refused to quit (window closed or tray "Sair") because an
// action is running: the main window shows the Execução tab with a notice.
// The compact window ignores it.
// quit-blocked reaches every window: the one running the action shows its
// Execução tab; the main window without the action points to the other one.
Events.On("quit-blocked", function () {
  if (activeRun) {
    selectTab("terminal");
    openMessage(t("quit.title"), t("quit.message"), false);
    return;
  }
  if (compactAction) return;
  openMessage(t("quit.title"), t("quit.messageTray"), false);
});

// failRun ends this window's run when the backend refused to start it
// ("outra ação já está em execução") or the call itself failed — no
// "action-done" event comes in that case. Every App call that follows a
// startExecution() chains .catch(failRun).
function failRun(err) {
  if (!activeRun) return;
  finishRun({ ok: false, error: String((err && err.message) || err) });
}

// finishRun closes the run lifecycle: terminal status, then every button
// startExecution disabled is released (whatever screen is showing now —
// the navigation isn't locked during a run, and those buttons are shared by
// several screens), then the screen showing now refreshes its own state
// (ACTION_DONE_HANDLERS). The screen the run started from re-renders
// anyway the next time it's selected.
function finishRun(result) {
  flushLog(); // output still waiting for its frame comes before the status line
  const run = activeRun;
  activeRun = null;
  running = false;
  term.writeln("");
  term.writeln(result.ok ? t("term.doneLine") : t("term.failedLine", { error: result.error }));
  termTitle.textContent = result.ok ? t("term.done") : t("term.failed");
  if (result.ok) termProgress.value = termProgress.max;
  run.buttons.forEach(function (b) { b.disabled = false; });
  // An update that ended outside Visão Geral leaves its status stale: the
  // next visit checks again.
  if (run.actionId === "dashboard" && dashboardRunningAction === "self-update") selfUpdateCheckedAt = 0;
  const current = window.__selectedItem && window.__selectedItem.actionId;
  doneHandlerFor(current)(run);
  if (run.actionId === "dashboard") dashboardRunningAction = null;
}

// "Atualizar" and "Desinstalar" both live in Visão Geral —
// dashboardRunningAction (set before each one) says which one finished.
// A successful uninstall closes the window on its own.
ACTION_DONE_HANDLERS.dashboard = function () {
  if (dashboardRunningAction === "self-update") loadSelfUpdateStatus(true);
};

// doneHandlerFor returns the refresh for the screen showing actionId: its
// ACTION_DONE_HANDLERS entry, else its family's done, else nothing.
function doneHandlerFor(actionId) {
  if (ACTION_DONE_HANDLERS[actionId]) return ACTION_DONE_HANDLERS[actionId];
  for (let i = 0; i < SCREEN_FAMILIES.length; i++) {
    if (SCREEN_FAMILIES[i].has(actionId)) return SCREEN_FAMILIES[i].done;
  }
  return function () {};
}

// wireUninstall connects Visão Geral's uninstall, the same in both
// applications: "Desinstalar" opens #uninstall-modal with every box of
// opts.options unchecked; the modal's own "Desinstalar" calls
// opts.collect(), which returns a promise of App.RunSelfUninstall's
// arguments (null stops without uninstalling), and starts the uninstall
// in the Execução tab. It lives here, in the first script, because the
// home-app.js of each application calls it while loading.
function wireUninstall(opts) {
  const modal = document.getElementById("uninstall-modal");
  btnSu2Uninstall.addEventListener("click", function () {
    if (running) return;
    opts.options.forEach(function (box) { box.checked = false; });
    showDialog(modal);
  });
  document.getElementById("btn-uninstall-confirm").addEventListener("click", function () {
    modal.close();
    if (running || activeRun) return;
    Promise.resolve(opts.collect()).then(function (args) {
      if (!args || running || activeRun) return; // re-checked: the dialogs are async
      dashboardRunningAction = "self-uninstall";
      beginRun(t("dashboard.uninstall"), function () { return App.RunSelfUninstall.apply(null, args); }, {
        buttons: [btnSu2Uninstall],
        maxSteps: 1,
        announce: t("uninstall.starting"),
      });
    }).catch(function (err) { openError(String(err)); });
  });
}
