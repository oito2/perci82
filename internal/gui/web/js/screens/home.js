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

// Home :: Visão Geral (dashboard, Atualizar) and Home :: Configurações
// (theme, sidebar logo), shared by both applications. Each application
// defines renderAppDashboardStats() and renderAppSelfConfig() for its own
// parts of these screens, and the Desinstalar button's handler.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// --- "Home :: Visão Geral" (dashboard — also the screen's initial
// state: #dashboard-container is visible by default, without going
// through selectItem). Brings together 3 blocks: the stats (GetAppVersion plus
// renderAppDashboardStats), "Atualizar" and "Desinstalar". dashboardRunningAction marks which of the two
// embedded actions is in progress — App's "action-done" event only has
// the actionId "dashboard" to go on, so this extra flag is needed to know
// what to reload. ---------------------------------------------------------
let dashboardRunningAction = null;

function renderDashboardStats() {
  App.GetAppVersion().then(function (v) {
    dashboardVersion.textContent = v;
  });
  renderAppDashboardStats();
}

// --- "Atualizar" (embedded in Visão Geral) ------------------------
// Reopening Visão Geral re-checks GitHub at most every 10 minutes; force
// (the "Verificar novamente" button, after an update) always does.
const SELF_UPDATE_RECHECK_MS = 10 * 60 * 1000;
let selfUpdateCheckedAt = 0;
function loadSelfUpdateStatus(force) {
  if (!force && selfUpdateCheckedAt && Date.now() - selfUpdateCheckedAt < SELF_UPDATE_RECHECK_MS) return;
  selfUpdateCheckedAt = Date.now();
  suStatus.textContent = "Verificando...";
  btnSuUpdate.disabled = true;
  App.GetSelfUpdateInfo().then(function (info) {
    if (info.error) {
      selfUpdateCheckedAt = 0; // retried on the next visit
      suStatus.textContent = "Não foi possível verificar atualizações: " + info.error;
      return;
    }
    if (info.updateAvailable) {
      suStatus.textContent = "Existe uma nova versão do " + document.title + " disponível (" + info.latestVersion + ").";
      btnSuUpdate.disabled = false;
    } else {
      suStatus.textContent = "O " + document.title + " está atualizado.";
    }
  }).catch(function (err) {
    selfUpdateCheckedAt = 0;
    suStatus.textContent = "Falha ao verificar atualizações: " + err;
  });
}

btnSuCheck.addEventListener("click", function () {
  if (running) return;
  loadSelfUpdateStatus(true);
});

btnSuUpdate.addEventListener("click", function () {
  if (running) return;
  dashboardRunningAction = "self-update";
  beginRun("Atualizar " + document.title, function () { return App.RunSelfUpdate(); }, {
    buttons: [btnSuUpdate, btnSuCheck],
    maxSteps: 1,
    announce: "Iniciando atualização do " + document.title + "...",
  });
});

// The dashboard is the main window's initial screen; the tray's compact
// window never shows it.
if (!compactAction) {
  renderDashboardStats();
  loadSelfUpdateStatus();
}

// --- "Home :: Configurar" (self-config) ---------------------------------
// Tema — a click on a swatch applies and persists
// right away, with no "Salvar" button involved.
function renderThemeGrid(current) {
  App.GetGUIThemes().then(function (themes) {
    cfgThemeGrid.innerHTML = "";
    themes.forEach(function (name) {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "btn btn-sm justify-start" + (name === current ? " btn-active" : "");
      btn.textContent = name;
      btn.addEventListener("click", function () {
        applyTheme(name);
        App.SetTheme(name).catch(function (err) {
          openError("Falha ao salvar o tema: " + err);
        });
        renderThemeGrid(name);
      });
      cfgThemeGrid.appendChild(btn);
    });
  });
}

function renderSelfConfigScreen() {
  App.GetTheme().then(renderThemeGrid);
  renderAppSelfConfig();
}

// Re-runs the whole Visão Geral screen — called when "Visão Geral" is
// clicked again in the sidebar (the buttons' click listeners were already
// attached once, outside selectItem's dispatch, since this screen isn't
// dynamically recreated like the others).
function renderDashboardScreen() {
  renderDashboardStats();
  loadSelfUpdateStatus();
}

SCREENS.dashboard = { el: dashboardContainer, render: renderDashboardScreen };
SCREENS["self-config"] = { el: selfconfigContainer, noTerminal: true, render: renderSelfConfigScreen };
