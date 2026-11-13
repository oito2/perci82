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

// Docker :: Criar Container and Docker :: Gerenciar Containers.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// ===========================================================================
// "Docker :: Criar Container" / "Docker :: Gerenciar Containers".
// ===========================================================================

// --- "Docker :: Criar Container" ------------------------------------------

let __dcCatalog = null;

// dockerQuickActionBusy is true while a Start, Stop or Restart of a row
// runs.
let dockerQuickActionBusy = false;

function dcFinalKind() {
  return dcKind.value === "__app__" ? dcAppType.value : dcKind.value;
}

function renderDcFields() {
  const kind = dcFinalKind();
  const prefill = kind === "mariadb" ? { dbPass: (__dcCatalog && __dcCatalog.__genPass) || "" } : {};
  dcFields.innerHTML = buildContainerFieldsHTML("dc", kind, __dcCatalog, prefill);
  wireMoodlePHPFilter("dc");
  wireFolderURLAutofill("dc");
}

function dcKindChanged() {
  dcAppTypeWrap.classList.toggle("hidden", dcKind.value !== "__app__");
  if (dcKind.value === "mariadb") {
    App.GenMariaDBPassword().then(function (pass) {
      __dcCatalog.__genPass = pass;
      renderDcFields();
    });
  } else {
    renderDcFields();
  }
}
dcKind.addEventListener("change", dcKindChanged);
dcAppType.addEventListener("change", renderDcFields);

// renderMkcertAlert shows the warning at the top of the screen while the
// Nginx container, whose creation runs "mkcert -install" for the first
// time (catalog.mkcertReady=false), is still to be created. That command
// can prompt for an admin password on its own, outside Perci's pkexec,
// and the GUI has no way to answer that prompt.
function renderMkcertAlert(catalog) {
  const show = !catalog.mkcertReady && !catalog.nginxExists;
  dcMkcertAlert.classList.toggle("hidden", !show);
  dcMkcertAlert.innerHTML = show
    ? LUCIDE.svg("triangleAlert", { size: 18 }) +
      "<span>" + escHTML(t("docker.mkcertAlert")) + "</span>"
    : "";
}

function renderDockerCreateScreen() {
  dcHint.textContent = "";
  App.GetDockerCreateCatalog().then(function (catalog) {
    __dcCatalog = catalog;
    renderMkcertAlert(catalog);
    dcKind.querySelector('option[value="nginx"]').disabled = catalog.nginxExists;
    dcKind.querySelector('option[value="mariadb"]').disabled = catalog.mariadbExists;
    if (dcKind.selectedOptions[0] && dcKind.selectedOptions[0].disabled) {
      dcKind.value = catalog.nginxExists ? (catalog.mariadbExists ? "__app__" : "mariadb") : "nginx";
    }
    dcKindChanged();
  });
}

btnDcCreate.addEventListener("click", function () {
  if (running) return;
  const kind = dcFinalKind();
  const req = readContainerFieldsValues("dc", kind, __dcCatalog);
  beginRun(t("docker.run.create"), function () { return App.CreateDockerContainer(req); });
});

// --- "Docker :: Gerenciar Containers" --------------------------------------

function iconButton(name, title) {
  const btn = document.createElement("button");
  btn.type = "button";
  btn.className = "btn btn-ghost btn-xs";
  btn.title = title;
  btn.setAttribute("aria-label", title); // icon only: title alone isn't announced reliably
  btn.innerHTML = LUCIDE.svg(name, { size: 16 });
  return btn;
}

function openEditModal(row) {
  Promise.all([
    App.GetContainerForEdit(row.family, row.folder),
    App.GetDockerCreateCatalog(),
  ]).then(function (results) {
    const values = results[0];
    const catalog = results[1];
    editModalTitle.textContent = t("docker.editTitle", { name: row.name });
    editFields.innerHTML = buildContainerFieldsHTML("edit", values.kind, catalog, values);
    wireMoodlePHPFilter("edit");
    wireFolderURLAutofill("edit");
    const folderEl = document.getElementById("edit-folder");
    if (folderEl && row.family === "app") folderEl.disabled = true; // the folder is the container's identity: not editable

    btnEditSave.onclick = function () {
      const req = readContainerFieldsValues("edit", values.kind, catalog);
      req.folder = values.folder; // the original folder, even though its field is disabled
      editModal.close();
      beginRun(t("docker.run.edit", { name: row.name }), function () { return App.UpdateDockerContainer(req); });
    };
    editModal.showModal();
  }).catch(function (err) { openError(t("docker.editLoadFailed", { error: err })); });
}

function buildDockerRow(row) {
  const tr = document.createElement("tr");

  const tdType = document.createElement("td");
  tdType.textContent = row.family === "app"
    ? row.name + " (" + (CONTAINER_TYPE_LABELS[row.type] || row.type) + ")"
    : (CONTAINER_TYPE_LABELS[row.type] || row.type);
  tr.appendChild(tdType);

  const tdStatus = document.createElement("td");
  tdStatus.innerHTML = statusBadgeHTML(row.status, row.dockerError);
  tr.appendChild(tdStatus);

  function setRowLoading() {
    tdStatus.innerHTML = '<span class="loading loading-spinner loading-sm"></span>';
  }
  function refreshRowStatus() {
    App.GetContainerRows().then(function (fresh) {
      const updated = (fresh || []).find(function (r) { return r.folder === row.folder; });
      tdStatus.innerHTML = updated ? statusBadgeHTML(updated.status, updated.dockerError) : statusBadgeHTML("");
    });
  }
  // A quick action runs one at a time, on any row: the backend refuses a
  // second one while the first runs.
  function wireQuickAction(btn, fn) {
    btn.addEventListener("click", function () {
      if (running || dockerQuickActionBusy) return;
      dockerQuickActionBusy = true;
      setRowLoading();
      fn().then(refreshRowStatus).catch(function (err) {
        refreshRowStatus();
        openError(String(err));
      }).finally(function () { dockerQuickActionBusy = false; });
    });
  }

  const tdActions = document.createElement("td");
  const wrap = document.createElement("div");
  wrap.className = "flex gap-1 flex-wrap";

  const btnStart = iconButton("play", t("docker.start"));
  wireQuickAction(btnStart, function () { return App.StartDockerContainer(row.folder); });
  wrap.appendChild(btnStart);

  const btnStop = iconButton("square", t("docker.stop"));
  wireQuickAction(btnStop, function () { return App.StopDockerContainer(row.folder); });
  wrap.appendChild(btnStop);

  const btnRestart = iconButton("rotateCw", t("docker.restart"));
  wireQuickAction(btnRestart, function () { return App.RestartDockerContainer(row.folder); });
  wrap.appendChild(btnRestart);

  const btnRecreate = iconButton("refreshCw", t("docker.recreate"));
  btnRecreate.addEventListener("click", function () {
    if (running) return;
    openConfirm(t("docker.recreate"), t("docker.recreateConfirm", { name: row.name }), function () {
      if (running) return; // re-checked: the confirm dialog is async
      beginRun(t("docker.run.recreate", { name: row.name }), function () { return App.RecreateDockerContainer(row.family, row.folder); });
    });
  });
  wrap.appendChild(btnRecreate);

  const btnUpdateImage = iconButton("circleArrowUp", t("docker.updateImage"));
  btnUpdateImage.addEventListener("click", function () {
    if (running) return;
    openConfirm(t("docker.updateImage"), updateImageMessage(row), function () {
      if (running) return; // re-checked: the confirm dialog is async
      beginRun(t("docker.run.updateImage", { name: row.name }), function () { return App.UpdateDockerContainerImage(row.family, row.folder); });
    });
  });
  wrap.appendChild(btnUpdateImage);

  if (row.legacyCommands) {
    const btnMigrate = iconButton("terminal", t("docker.migrate"));
    btnMigrate.classList.add("text-warning");
    btnMigrate.addEventListener("click", function () {
      if (running) return;
      openConfirm(t("docker.migrate"), t("docker.migrateConfirm", { name: row.name, folder: row.folder }), function () {
        if (running) return; // re-checked: the confirm dialog is async
        beginRun(t("docker.run.migrate", { folder: row.folder }), function () { return App.MigrateAppCommand(row.folder); });
      });
    });
    wrap.appendChild(btnMigrate);
  }

  const btnLogs = iconButton("scrollText", t("docker.logs"));
  btnLogs.addEventListener("click", function () { openLogs(row.folder, row.name); });
  wrap.appendChild(btnLogs);

  if (row.canEdit) {
    const btnEdit = iconButton("pencil", t("docker.edit"));
    btnEdit.addEventListener("click", function () {
      if (running) return;
      openEditModal(row);
    });
    wrap.appendChild(btnEdit);
  }

  if (row.hasDb) {
    const btnBackup = iconButton("download", t("docker.backup"));
    btnBackup.addEventListener("click", function () {
      if (running) return;
      App.PickBackupPath().then(function (path) {
        if (!path || running) return; // re-checked: the dialog is async
        beginRun(t("docker.run.backup"), function () { return App.BackupMariaDBContainer(path); });
      });
    });
    wrap.appendChild(btnBackup);

    const btnRestore = iconButton("upload", t("docker.restore"));
    btnRestore.addEventListener("click", function () {
      if (running) return;
      App.PickRestorePath().then(function (path) {
        if (!path) return;
        openConfirm(t("docker.restore"), t("docker.restoreConfirm"), function () {
          if (running) return; // re-checked: the dialogs are async
          beginRun(t("docker.run.restore"), function () { return App.RestoreMariaDBContainer(path); });
        });
      });
    });
    wrap.appendChild(btnRestore);
  }

  const btnRemove = iconButton("trash2", t("docker.remove"));
  btnRemove.addEventListener("click", function () {
    if (running) return;
    openConfirm(t("docker.remove"), t("docker.removeConfirm", { name: row.name }), function () {
      setRowLoading();
      App.RemoveDockerContainer(row.family, row.folder).then(function () {
        renderDockerListScreen();
      }).catch(function (err) {
        refreshRowStatus();
        openError(String(err));
      });
    });
  });
  wrap.appendChild(btnRemove);

  tdActions.appendChild(wrap);
  tr.appendChild(tdActions);
  return tr;
}

// updateImageMessage is the confirmation text of a row's "Atualizar
// imagem": what is downloaded and recreated, and what is kept.
function updateImageMessage(row) {
  const kept = " " + t("docker.updateImageKept");
  if (row.family === "mariadb") return t("docker.updateImageMariaDB") + kept;
  if (row.family === "nginx") return t("docker.updateImageNginx") + kept;
  return t("docker.updateImageApp", { name: row.name }) + kept;
}

btnDockerlistUpdateAll.addEventListener("click", function () {
  if (running) return;
  openConfirm(t("docker.updateImages"), t("docker.updateImagesConfirm"),
    function () {
      if (running) return; // re-checked: the confirm dialog is async
      beginRun(t("docker.updateImages"), function () { return App.UpdateAllDockerImages(); });
    });
});

function renderDockerListScreen() {
  dockerlistRows.innerHTML = "";
  App.GetContainerRows().then(latestOnly("dockerlist", function (rows) {
    rows = rows || [];
    dockerlistEmpty.classList.toggle("hidden", rows.length > 0);
    dockerlistToolbar.classList.toggle("hidden", rows.length === 0);
    rows.forEach(function (row) { dockerlistRows.appendChild(buildDockerRow(row)); });
  }));
}
