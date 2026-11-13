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

// Perci82 System parts of Visão Geral (detected system and desktop,
// Desinstalar) and of Configurações (Flatpak scope).
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// renderAppDashboardStats fills the "Sistema" and "Ambiente" cards of
// Visão Geral.
const dashboardDesktopLabel = document.getElementById("dashboard-desktop-label");
function renderAppDashboardStats() {
  App.GetSystemInfo().then(function (info) {
    dashboardSystemLabel.textContent = (info && info.system) || t("shell.unknownSystem");
    dashboardDesktopLabel.textContent = (info && info.desktop) || t("shell.unknownSystem");
  });
}

// --- "Desinstalar Perci82 System" (Visão Geral): the modal's only option
// is "Também remover configurações", unchecked each time (wireUninstall). -
wireUninstall({
  options: [su2RemoveConfig],
  collect: function () { return [su2RemoveConfig.checked]; },
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
