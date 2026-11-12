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

// Visão Geral (dashboard, Atualizar) and Configurações (theme,
// language), shared by both applications. Each application
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

// --- "Home :: Configurar" (self-config) ---------------------------------
// Tema — "Definido pelo sistema" or one of the oito2 presets; a click
// applies and persists right away, with no "Salvar" button involved.
function renderThemeGrid(current) {
  App.GetThemeOptions().then(function (options) {
    cfgThemeGrid.innerHTML = "";
    (options || []).forEach(function (option) {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "btn btn-sm justify-start" + (option.id === current ? " btn-active" : "");
      if (option.id === "system") {
        // Relabeled by applyI18n on a language switch.
        btn.setAttribute("data-i18n", "settings.system");
        btn.textContent = t("settings.system");
      } else {
        btn.textContent = option.label;
      }
      btn.addEventListener("click", function () {
        applyTheme(option.id);
        App.SetTheme(option.id).catch(function (err) {
          openError(t("settings.themeSaveFailed", { error: err }));
        });
        renderThemeGrid(option.id);
      });
      cfgThemeGrid.appendChild(btn);
    });
  });
}

// Idioma — "Definido pelo sistema" or one of the supported languages,
// each under its own name; a change persists it and switches the
// interface at once.
const cfgLanguage = document.getElementById("cfg-language");

function renderLanguageSelect() {
  Promise.all([App.GetLanguageOptions(), App.GetLanguage()]).then(function (r) {
    const current = (r[1] && r[1].choice) || "system";
    cfgLanguage.innerHTML = "";
    (r[0] || []).forEach(function (option) {
      const opt = document.createElement("option");
      opt.value = option.id;
      if (option.id === "system") {
        // Relabeled by applyI18n on a language switch.
        opt.setAttribute("data-i18n", "settings.system");
        opt.textContent = t("settings.system");
      } else {
        opt.textContent = option.label;
      }
      cfgLanguage.appendChild(opt);
    });
    cfgLanguage.value = current;
  });
}

cfgLanguage.addEventListener("change", function () {
  App.SetLanguage(cfgLanguage.value).then(function (state) {
    return setLocale(state.locale);
  }).catch(function (err) {
    openError(t("settings.languageSaveFailed", { error: err }));
    renderLanguageSelect();
  });
});

function renderSelfConfigScreen() {
  App.GetTheme().then(renderThemeGrid);
  renderLanguageSelect();
  renderAppSelfConfig();
}

// Re-runs the whole Visão Geral screen — on every selection of the
// section (the buttons' click listeners were attached once, outside
// selectItem's dispatch, since this screen isn't recreated like the
// others).
function renderDashboardScreen() {
  renderDashboardStats();
  loadSelfUpdateStatus();
}

SCREENS.dashboard = { el: dashboardContainer, render: renderDashboardScreen };
SCREENS["self-config"] = { el: selfconfigContainer, noTerminal: true, render: renderSelfConfigScreen };
