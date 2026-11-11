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

// Perci82 Dev "Desenvolvimento" checklists (Aplicativos: IA, Terminais) and
// the local-file app screens (Android Studio, Antigravity IDE).
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

MULTISELECT_SCREENS["ai-apps"] = {
  fetch: function () { return App.GetAIAppsInfo(); },
  run: function (ids) { return App.RunAIApps(ids); },
};

MULTISELECT_SCREENS.terminals = {
  fetch: function () { return App.GetTerminalsInfo(); },
  run: function (ids) { return App.RunTerminals(ids); },
  // Extra buttons — not part of the checklist (Starship isn't a
  // checklist item), run
  // independent of selection. confirm (optional) is asked first.
  extras: [
    {
      label: "Aplicar Starship",
      run: function () { return App.ApplyStarship(); },
    },
    {
      label: "Remover Starship",
      confirm: "Remove o binário do Starship e a inicialização dele no bash, zsh e fish. " +
        "A configuração (~/.config/starship.toml) é preservada como starship.toml.perci-bak.",
      run: function () { return App.RemoveStarship(); },
    },
  ],
};

// runScreenAction starts one of a screen's install/update/uninstall
// actions: nothing while another runs, then the Execução lifecycle with
// the screen's buttons disabled, and fn()'s refusal handled by failRun.
function runScreenAction(item, label, buttons, fn) {
  beginRun(item.title, fn, {
    buttons: buttons,
    maxSteps: 1,
    announce: "Iniciando " + item.title + " (" + label + ")...",
  });
}

// --- "app externo via arquivo local install/(atualizar/)desinstalar"
// screens: the user points at a manually downloaded .tar.gz (Android
// Studio, Antigravity IDE) before being able to install/update —
// Desinstalar doesn't depend on any file. FILEAPP_SCREENS maps actionId
// to the bound methods; update is optional (Android Studio doesn't have
// one — it updates itself). ---------------------------------------------
const FILEAPP_SCREENS = {
  androidstudio: {
    pick: function () { return App.PickAndroidStudioTarball(); },
    fetch: function () { return App.GetAndroidStudioInfo(); },
    install: function (path) { return App.InstallAndroidStudio(path); },
    uninstall: function () { return App.UninstallAndroidStudio(); },
  },
  antigravityide: {
    pick: function () { return App.PickAntigravityIDETarball(); },
    fetch: function () { return App.GetAntigravityIDEInfo(); },
    install: function (path) { return App.InstallAntigravityIDE(path); },
    update: function (path) { return App.UpdateAntigravityIDE(path); },
    uninstall: function () { return App.UninstallAntigravityIDE(); },
  },
};

// linkifyDesc escapes the text and turns http(s) URLs into clickable
// links — used only by renderFileAppScreen (Android Studio/Antigravity
// IDE carry the download link inside their own desc). The click is
// intercepted (see the fileappDesc listener right below) and opens in
// the system's default browser via Browser.OpenURL — a plain <a href>
// would navigate the Wails window itself away from the app.
// Every piece is escaped exactly once, from the raw text: the text between
// URLs with escHTML, each URL with escAttr (attribute) and escHTML (label).
function linkifyDesc(text) {
  let out = "";
  let last = 0;
  text.replace(/https?:\/\/[^\s<>"]+/g, function (url, idx) {
    out += escHTML(text.slice(last, idx)) +
      '<a href="#" class="link link-primary" data-url="' + escAttr(url) + '">' + escHTML(url) + "</a>";
    last = idx + url.length;
    return url;
  });
  return out + escHTML(text.slice(last));
}

fileappDesc.addEventListener("click", function (ev) {
  const a = ev.target.closest("a[data-url]");
  if (!a) return;
  ev.preventDefault();
  Browser.OpenURL(a.dataset.url);
});

function renderFileAppScreen(item) {
  fileappDesc.innerHTML = linkifyDesc(item.desc || "");
  fileappNotes.innerHTML = "";
  (item.notes || []).forEach(function (n) {
    const li = document.createElement("li");
    li.textContent = n;
    fileappNotes.appendChild(li);
  });

  const cfg = FILEAPP_SCREENS[item.actionId];
  window.__fileappPath = "";
  fileappPath.textContent = "Nenhum arquivo selecionado.";
  btnFileappInstall.disabled = true;
  btnFileappUninstall.disabled = true;

  if (cfg.update) {
    btnFileappUpdate.classList.remove("hidden");
    btnFileappUpdate.disabled = true;
  } else {
    btnFileappUpdate.classList.add("hidden");
  }

  cfg.fetch().then(latestOnly("fileapp", function (info) {
    btnFileappUninstall.disabled = !info.installed;
  }));

  btnFileappPick.onclick = function () {
    cfg.pick().then(function (path) {
      if (!path) return; // user cancelled the dialog
      window.__fileappPath = path;
      fileappPath.textContent = path;
      btnFileappInstall.disabled = false;
      if (cfg.update) btnFileappUpdate.disabled = false;
    });
  };

  const buttons = [btnFileappInstall, btnFileappUpdate, btnFileappUninstall];
  btnFileappInstall.onclick = function () {
    if (!window.__fileappPath) return;
    runScreenAction(item, "instalar", buttons, function () { return cfg.install(window.__fileappPath); });
  };
  btnFileappUpdate.onclick = function () {
    if (!window.__fileappPath || !cfg.update) return;
    runScreenAction(item, "atualizar", buttons, function () { return cfg.update(window.__fileappPath); });
  };
  btnFileappUninstall.onclick = function () {
    runScreenAction(item, "desinstalar", buttons, cfg.uninstall);
  };
}

SCREEN_FAMILIES.push({
  has: function (id) { return !!FILEAPP_SCREENS[id]; },
  el: fileappContainer,
  render: renderFileAppScreen,
  // Shares its container's buttons with the screen's own render, so it
  // uses the render's latestOnly key.
  done: function (run) {
    const cfg = FILEAPP_SCREENS[window.__selectedItem.actionId];
    const hasPath = !!window.__fileappPath;
    cfg.fetch().then(latestOnly("fileapp", function (info) {
      btnFileappInstall.disabled = !hasPath;
      if (cfg.update) btnFileappUpdate.disabled = !hasPath;
      btnFileappUninstall.disabled = !info.installed;
    }));
  },
});
