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

// Perci82 Dev parts of Home: the Docker summary of Visão Geral, Desinstalar
// with its backups, and the workspace and tray settings of Configurações.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// renderAppDashboardStats fills the "Contêineres Docker" stat of Visão Geral.
function renderAppDashboardStats() {
  App.GetContainerRows().then(function (rows) {
    rows = rows || [];
    if (rows.length === 0) {
      dashboardDockerSummary.textContent = "Nenhum";
      return;
    }
    if (rows[0].dockerError) {
      dashboardDockerSummary.textContent = "Docker inacessível";
      return;
    }
    const runningCount = rows.filter(function (r) { return r.status === "running"; }).length;
    dashboardDockerSummary.textContent = runningCount + " / " + rows.length;
  });
}

// --- "Desinstalar Perci" (embedded in Visão Geral) — five independent
// checkboxes (back up config / back up repositories / back up AI contexts /
// remove Docker containers / also remove config), a "Desinstalar" button
// after them. The config backup is a full export, in the format "Docker ::
// Exportar/Importar" reads (App.PickUninstallBackupPath); the repositories
// and AI contexts backups are separate files, in the formats "Repositórios
// :: Gerenciar Repositórios" (App.PickUninstallReposBackupPath) and "Dev
// Tools :: IA: Contextos" (App.PickUninstallAIContextsBackupPath) read. If
// the user checks a backup and then cancels its "Salvar Como" dialog, the
// whole uninstall is aborted instead of proceeding without the requested
// backup. Success
// closes the GUI's own window (App.RunSelfUninstall handles that). ------
btnSu2Uninstall.addEventListener("click", function () {
  if (running) return;
  const removeConfig = su2RemoveConfig.checked;
  const removeDocker = su2RemoveDocker.checked;
  const backupConfig = su2BackupConfig.checked;
  const backupRepos = su2BackupRepos.checked;
  const backupAI = su2BackupAI.checked;

  let message = "O binário do Perci82 Dev será removido do sistema.";
  if (backupConfig) message += "\nUm backup das configurações será salvo antes.";
  if (backupRepos) message += "\nUm backup das contas Git e dos repositórios será salvo antes.";
  if (backupAI) message += "\nUm backup dos contextos de IA será salvo antes.";
  if (removeDocker) message += "\nOs contêineres Docker registrados serão removidos (dados em disco são mantidos).";
  message += removeConfig
    ? "\nAs configurações do Perci82 Dev (no ~/.perci/config.yaml e as pastas appstack e git do ~/.perci) também serão removidas, junto com as regras das contas Git no ~/.gitconfig e os tokens do chaveiro (chaves SSH e repositórios ficam). As do Perci82 System são mantidas."
    : "\nAs configurações em ~/.perci serão mantidas.";
  message += "\n\nEsta ação não pode ser desfeita.";

  openConfirm("Desinstalar o Perci82 Dev?", message, function () {
    function startUninstall(backupPath, reposBackupPath, aiBackupPath) {
      if (running || activeRun) return; // re-checked: the dialogs are async
      dashboardRunningAction = "self-uninstall";
      beginRun("Desinstalar", function () {
        return App.RunSelfUninstall(removeConfig, removeDocker, backupPath, reposBackupPath, aiBackupPath);
      }, {
        buttons: [btnSu2Uninstall],
        maxSteps: 1,
        announce: "Desinstalando o Perci82 Dev...",
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
      [backupConfig, App.PickUninstallBackupPath],
      [backupRepos, App.PickUninstallReposBackupPath],
      [backupAI, App.PickUninstallAIContextsBackupPath],
    ];
    wanted.reduce(function (chain, w) {
      return chain.then(function (paths) {
        if (paths.indexOf(null) >= 0) return paths.concat([null]);
        return pick(w[0], w[1]).then(function (path) { return paths.concat([path]); });
      });
    }, Promise.resolve([])).then(function (paths) {
      if (paths.indexOf(null) >= 0) {
        openMessage("Desinstalação cancelada", "Nenhum caminho foi escolhido para o backup — a desinstalação não foi iniciada.", false);
        return;
      }
      startUninstall(paths[0], paths[1], paths[2]);
    }).catch(function (err) { openError("Falha ao escolher o arquivo de backup: " + err); });
  });
});

// Caminho do Workspace — same alert-card pattern as "Dev Tools :: IA:
// Contextos", but unlike that screen (which always starts over with no
// folder), this card reflects the value already persisted in
// config.yaml as soon as the screen opens.
let scWorkspacePath = "";       // persisted value (config.yaml)
let scPickedWorkspacePath = ""; // picked this session, not saved yet

function renderScWorkspaceAlert() {
  if (scWorkspacePath) {
    scWorkspaceAlertTitle.textContent = "O workspace está criado";
    scWorkspaceAlertDesc.textContent = scWorkspacePath;
    btnScCreateWorkspace.classList.add("hidden");
    return;
  }
  scWorkspaceAlertTitle.textContent = "O workspace não foi criado";
  scWorkspaceAlertDesc.textContent = scPickedWorkspacePath || "Selecione uma pasta";
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
    openError("Falha ao criar workspace: " + err);
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
    openError("Falha ao " + (enabled ? "ligar" : "desligar") + " a bandeja do sistema: " + err);
  });
});

// renderAppSelfConfig fills the tray toggle and the workspace card of
// Configurações.
function renderAppSelfConfig() {
  App.GetTrayEnabled().then(function (enabled) {
    scTrayEnabled.checked = enabled;
  });

  scPickedWorkspacePath = "";
  App.GetWorkspacePath().then(function (path) {
    scWorkspacePath = path || "";
    renderScWorkspaceAlert();
  });
}
