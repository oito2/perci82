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

// Shared state and plumbing: the screen registries, shared DOM references, theme, sidebar logo/collapse, tabs, the xterm terminal and the Execução run lifecycle (log-line/step/action-done events).
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
  openError("Erro inesperado: " + ((reason && reason.message) || reason));
});


const breadcrumb = document.getElementById("breadcrumb");
const tabDescLabel = document.getElementById("tab-desc-label");
const accordionEl = document.getElementById("accordion");
const tabRadioDesc = document.getElementById("tab-radio-desc");
const tabRadioTerminal = document.getElementById("tab-radio-terminal");
const sidebar = document.getElementById("sidebar");

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
// Applies the saved theme (data-theme) on load; the swatch grid is built
// by renderSelfConfigScreen.
function applyTheme(name) {
  document.documentElement.setAttribute("data-theme", name);
  // If the logo "easter egg" (below) already has an active mask, its
  // color depends on the theme — reapply it right away, without
  // waiting for a new click.
  if (sidebarLogoMaskIndex >= 0) logoImgEl.style.backgroundColor = sidebarLogoMaskColor();
}

App.GetTheme().then(applyTheme);

// --- sidebar logo --------------------------------------------------------
// Applies the saved sidebar logo on load; the clickable grid is built by
// renderSelfConfigScreen.
App.GetSidebarLogo().then(function (name) {
  sidebarLogoImg.src = "./assets/perci-" + name + ".png";
});

// --- sidebar logo easter egg: each click on the image applies a
// different daisyUI mask, cycling through a fixed list of shapes. Both
// mascot variants (perci-blue.png/perci-pink.png) have a transparent
// background, so coloring the <img> itself (background-color) shows that
// color clipped exactly to the mask's shape. Color per theme:
// "synthwave"/"retro"/"valentine"/"halloween"/"garden" use the theme's
// primary color (`var(--color-primary)`); every other theme uses the
// fixed pink #FEC5C4. Purely client-side, no persistence — resets (back
// to the original circle) every time Perci reopens. Uses
// `document.getElementById` (not the `sidebarLogoImg` const declared
// further below) because this snippet runs before that const exists;
// `logoImgEl` has its own name so it doesn't collide with that const. ---
const SIDEBAR_LOGO_MASKS = [
  "mask-squircle", "mask-heart", "mask-hexagon", "mask-hexagon-2",
  "mask-decagon", "mask-pentagon", "mask-diamond", "mask-star",
  "mask-star-2", "mask-triangle", "mask-circle",
];
const SIDEBAR_LOGO_PRIMARY_THEMES = ["synthwave", "retro", "valentine", "halloween", "garden"];
let sidebarLogoMaskIndex = -1; // -1 = still the original circle, no mask
const logoImgEl = document.getElementById("sidebar-logo-img");

function sidebarLogoMaskColor() {
  const theme = document.documentElement.getAttribute("data-theme");
  return SIDEBAR_LOGO_PRIMARY_THEMES.indexOf(theme) >= 0 ? "var(--color-primary)" : "#FEC5C4";
}

function applySidebarLogoMask() {
  logoImgEl.classList.remove("rounded-full", "mask", ...SIDEBAR_LOGO_MASKS);
  if (sidebarLogoMaskIndex < 0) {
    logoImgEl.classList.add("rounded-full");
    logoImgEl.style.backgroundColor = "";
    return;
  }
  logoImgEl.classList.add("mask", SIDEBAR_LOGO_MASKS[sidebarLogoMaskIndex]);
  logoImgEl.style.backgroundColor = sidebarLogoMaskColor();
}

// Keyboard: the logo is focusable and Enter/Space act as a click.
logoImgEl.addEventListener("keydown", function (ev) {
  if (ev.key === "Enter" || ev.key === " ") {
    ev.preventDefault();
    logoImgEl.click();
  }
});
logoImgEl.addEventListener("click", function () {
  sidebarLogoMaskIndex = (sidebarLogoMaskIndex + 1) % SIDEBAR_LOGO_MASKS.length;
  applySidebarLogoMask();
});

// --- collapse sidebar -------------------------------------------------------
document.getElementById("btn-collapse-sidebar").addEventListener("click", function () {
  const collapsed = sidebar.getAttribute("data-collapsed") === "true";
  sidebar.setAttribute("data-collapsed", collapsed ? "false" : "true");
});

// --- tabs (Definições / Execução) — each tab-content's visibility is
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
term.writeln("Nenhuma ação em execução.");

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
const cfgLogoGrid = document.getElementById("cfg-logo-grid");
const sidebarLogoImg = document.getElementById("sidebar-logo-img");
// Buttons built from MULTISELECT_SCREENS[actionId].extras for the screen
// currently shown (renderMultiselectScreen).
let multiselectExtraButtons = [];

let running = false;

// activeRun describes the action THIS window started (startExecution):
// the actionId of the screen it was started from and the
// buttons it disabled. null when this window isn't running anything — the
// log-line/step/action-done events reach every window (main + the tray's
// compact window), and only the one that started the action handles them.
let activeRun = null;

// Events.On delivers the payload inside ev.data.
// Output is written once per frame instead of once per event: apt or
// docker build emit thousands of small events.
let pendingLog = "";
Events.On("log-line", function (ev) {
  if (!activeRun) return;
  if (!pendingLog) requestAnimationFrame(flushLog);
  pendingLog += ev.data;
});
function flushLog() {
  if (pendingLog) term.write(pendingLog);
  pendingLog = "";
}

Events.On("step", function (ev) {
  if (!activeRun) return;
  const step = ev.data;
  termTitle.textContent = "Executando: " + step.label;
  termProgress.value = step.index;
  termProgress.max = step.total;
});

Events.On("action-done", function (ev) {
  if (!activeRun) return;
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
    openMessage(
      "Ação em andamento",
      "O aplicativo não será fechado enquanto uma ação está em execução. Aguarde ela terminar na aba Execução.",
      false
    );
    return;
  }
  if (compactAction) return;
  openMessage(
    "Ação em andamento",
    "O aplicativo não será fechado enquanto uma ação está em execução na janela da bandeja. Aguarde ela terminar.",
    false
  );
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
// the sidebar isn't locked during a run, and those buttons are shared by
// several screens), then the screen showing now refreshes its own state
// (ACTION_DONE_HANDLERS). The screen the run started from re-renders
// anyway the next time it's selected.
function finishRun(result) {
  flushLog(); // output still waiting for its frame comes before the status line
  const run = activeRun;
  activeRun = null;
  running = false;
  term.writeln("");
  term.writeln(result.ok ? "--- concluído com sucesso ---" : "--- falhou: " + result.error + " ---");
  termTitle.textContent = result.ok ? "Concluído." : "Falhou.";
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
