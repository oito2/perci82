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

// Perci82 System parts of Home: the detected system in Visão Geral,
// Desinstalar, and the Flatpak scope of Configurações.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// renderAppDashboardStats fills the "Sistema" stat of Visão Geral.
function renderAppDashboardStats() {
  getPostinstallProfile().then(function (profile) {
    dashboardSystemLabel.textContent = (profile && profile.supported) ? profile.label : t("shell.unknownSystem");
  });
}

// --- "Desinstalar Perci82 System" (embedded in Visão Geral): the "Também
// remover configurações" checkbox, then the "Desinstalar" button. --------
btnSu2Uninstall.addEventListener("click", function () {
  if (running) return;
  const removeConfig = su2RemoveConfig.checked;
  const message = t("uninstall.binary") + "\n" +
    (removeConfig ? t("uninstall.removeConfig") : t("uninstall.keepConfig")) +
    "\n\n" + t("uninstall.irreversible");
  openConfirm(t("uninstall.confirmTitle"), message, function () {
    if (running || activeRun) return;
    dashboardRunningAction = "self-uninstall";
    beginRun(t("dashboard.uninstall"), function () { return App.RunSelfUninstall(removeConfig); }, {
      buttons: [btnSu2Uninstall],
      maxSteps: 1,
      announce: t("uninstall.starting"),
    });
  });
});

// Escopo de instalação Flatpak — applies right away, like Tema,
// with no "Salvar" button at all.
scFlatpak.addEventListener("change", function () {
  App.SetFlatpakScope(scFlatpak.value).catch(function (err) {
    openError(t("settings.flatpakScopeSaveFailed", { error: err }));
  });
});

// renderAppSelfConfig shows the persisted Flatpak scope in Configurações.
function renderAppSelfConfig() {
  App.GetFlatpakScope().then(function (scope) {
    scFlatpak.value = scope === "user" ? "user" : "system";
  });
}
