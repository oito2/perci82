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

// Perci82 Dev parts of Home: the Docker and repositories summary of Visão
// Geral, Desinstalar with its backups, and the workspace, project notes and
// tray settings of Configurações.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

const dashboardReposSummary = document.getElementById("dashboard-repos-summary");

// renderAppDashboardStats fills the "Contêineres Docker" and "Repositórios"
// cards of Visão Geral.
function renderAppDashboardStats() {
  App.GetContainerRows().then(function (rows) {
    rows = rows || [];
    if (rows.length === 0) {
      dashboardDockerSummary.textContent = t("dashboard.dockerNone");
      return;
    }
    if (rows[0].dockerError) {
      dashboardDockerSummary.textContent = t("dashboard.dockerUnreachable");
      return;
    }
    const running = rows.filter(function (r) { return r.status === "running"; }).length;
    dashboardDockerSummary.textContent = t("dashboard.dockerRunning", { running: running, total: rows.length });
  });
  App.GetRepositorySummary().then(function (sum) {
    dashboardReposSummary.textContent = !sum || sum.accounts === 0
      ? t("dashboard.reposNone")
      : t("dashboard.reposSummary", { accounts: sum.accounts, repos: sum.repos });
  }).catch(function () {
    dashboardReposSummary.textContent = "—";
  });
}

// --- "Desinstalar Perci82 Dev" (Visão Geral): the button opens
// #uninstall-modal with every option unchecked; its "Desinstalar" button
// asks, one after the other, where to save each backup that was checked
// (Docker in the format "Docker :: Exportar/Importar" reads, repositories
// in the one of "Repositórios :: Gerenciar Repositórios", AI contexts in
// the one of "Contextos de IA :: Exportar/Importar") and then starts the
// uninstall. A cancelled "Salvar Como" dialog aborts the whole uninstall
// instead of going on without the requested backup. Success closes the
// window (App.RunSelfUninstall handles that). ---------------------------
const uninstallModal = document.getElementById("uninstall-modal");
const btnUninstallConfirm = document.getElementById("btn-uninstall-confirm");
const uninstallOptions = [su2BackupConfig, su2BackupRepos, su2BackupAI, su2RemoveDocker, su2RemoveConfig];

btnSu2Uninstall.addEventListener("click", function () {
  if (running) return;
  uninstallOptions.forEach(function (box) { box.checked = false; });
  showDialog(uninstallModal);
});

btnUninstallConfirm.addEventListener("click", function () {
  uninstallModal.close();
  if (running || activeRun) return;
  const removeConfig = su2RemoveConfig.checked;
  const removeDocker = su2RemoveDocker.checked;

  function startUninstall(backupPath, reposBackupPath, aiBackupPath) {
    if (running || activeRun) return; // re-checked: the dialogs are async
    dashboardRunningAction = "self-uninstall";
    beginRun(t("dashboard.uninstall"), function () {
      return App.RunSelfUninstall(removeConfig, removeDocker, backupPath, reposBackupPath, aiBackupPath);
    }, {
      buttons: [btnSu2Uninstall],
      maxSteps: 1,
      announce: t("uninstall.starting"),
    });
  }

  // pick asks for one backup's path when wanted is set; "" means not
  // wanted, null means the dialog was cancelled.
  function pick(wanted, picker) {
    if (!wanted) return Promise.resolve("");
    return picker().then(function (path) { return path || null; });
  }

  // Each picker opens only after the previous one got a path; the first
  // cancelled dialog stops the chain.
  const wanted = [
    [su2BackupConfig.checked, App.PickUninstallBackupPath],
    [su2BackupRepos.checked, App.PickUninstallReposBackupPath],
    [su2BackupAI.checked, App.PickUninstallAIContextsBackupPath],
  ];
  wanted.reduce(function (chain, w) {
    return chain.then(function (paths) {
      if (paths.indexOf(null) >= 0) return paths.concat([null]);
      return pick(w[0], w[1]).then(function (path) { return paths.concat([path]); });
    });
  }, Promise.resolve([])).then(function (paths) {
    if (paths.indexOf(null) >= 0) {
      openMessage(t("uninstall.cancelledTitle"), t("uninstall.cancelled"), false);
      return;
    }
    startUninstall(paths[0], paths[1], paths[2]);
  }).catch(function (err) { openError(t("uninstall.pickFailed", { error: err })); });
});

// Caminho do Workspace — same alert-card pattern as "Contextos de IA ::
// Configurar Projeto", but unlike that screen (which always starts over with no
// folder), this card reflects the value already persisted in
// config.yaml as soon as the screen opens.
let scWorkspacePath = "";       // persisted value (config.yaml)
let scPickedWorkspacePath = ""; // picked this session, not saved yet

function renderScWorkspaceAlert() {
  if (scWorkspacePath) {
    scWorkspaceAlertTitle.textContent = t("settings.workspace.created");
    scWorkspaceAlertDesc.textContent = scWorkspacePath;
    btnScCreateWorkspace.classList.add("hidden");
    return;
  }
  scWorkspaceAlertTitle.textContent = t("settings.workspace.missing");
  scWorkspaceAlertDesc.textContent = scPickedWorkspacePath || t("settings.workspace.pickHint");
  btnScCreateWorkspace.classList.toggle("hidden", !scPickedWorkspacePath);
}

btnScPickWorkspace.addEventListener("click", function () {
  App.PickWorkspaceFolder().then(function (path) {
    if (!path) return; // user cancelled the dialog
    scPickedWorkspacePath = path;
    renderScWorkspaceAlert();
  });
});

btnScCreateWorkspace.addEventListener("click", function () {
  if (!scPickedWorkspacePath) return;
  btnScCreateWorkspace.disabled = true;
  App.SetWorkspacePath(scPickedWorkspacePath).then(function () {
    scWorkspacePath = scPickedWorkspacePath;
    scPickedWorkspacePath = "";
    btnScCreateWorkspace.disabled = false;
    renderScWorkspaceAlert();
  }).catch(function (err) {
    btnScCreateWorkspace.disabled = false;
    openError(t("settings.workspace.createFailed", { error: err }));
  });
});

// Bandeja do sistema — applies right away, same "no Salvar button"
// convention as the rest of the screen; unlike SetAppIcon, SetTrayEnabled
// has real immediate effect (SystemTray.New()/Destroy() at runtime, not
// just on the next launch). Reverts the checkbox if the call fails, so
// it doesn't lie about the actual state.
scTrayEnabled.addEventListener("change", function () {
  const enabled = scTrayEnabled.checked;
  App.SetTrayEnabled(enabled).catch(function (err) {
    scTrayEnabled.checked = !enabled;
    openError(t(enabled ? "settings.trayOnFailed" : "settings.trayOffFailed", { error: err }));
  });
});

// Anotações de projeto: the folder is saved as soon as it is picked, and
// Desativar clears it.
const scNotesPath = document.getElementById("sc-notes-path");
const btnScPickNotes = document.getElementById("btn-sc-pick-notes");
const btnScClearNotes = document.getElementById("btn-sc-clear-notes");

function renderScNotes(path) {
  scNotesPath.textContent = path || t("notes.off");
  scNotesPath.classList.toggle("opacity-60", !path);
  btnScClearNotes.classList.toggle("hidden", !path);
}

function saveScNotes(path) {
  App.SetProjectNotesPath(path).then(function () {
    renderScNotes(path);
  }).catch(function (err) {
    openError(t("notes.saveFailed", { error: err }));
  });
}

btnScPickNotes.addEventListener("click", function () {
  App.PickProjectNotesFolder().then(function (path) {
    if (path) saveScNotes(path); // "" when the dialog was cancelled
  });
});
btnScClearNotes.addEventListener("click", function () { saveScNotes(""); });

// renderAppSelfConfig fills the tray toggle, the workspace card and the
// project notes folder of Configurações.
function renderAppSelfConfig() {
  App.GetProjectNotesPath().then(renderScNotes);

  App.GetTrayEnabled().then(function (enabled) {
    scTrayEnabled.checked = enabled;
  });

  scPickedWorkspacePath = "";
  App.GetWorkspacePath().then(function (path) {
    scWorkspacePath = path || "";
    renderScWorkspaceAlert();
  });
}
