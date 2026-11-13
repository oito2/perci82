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

// Docker :: Pastas de Desenvolvimento: a development folder mounted into
// an app container — a plugin of a Moodle app, or a folder the user names
// inside the project root of the other app types.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

let mpDev = "";

// mpApps holds the apps of the last GetDevFolders, by folder.
let mpApps = {};

// MP_STATUS maps each mount status to the message keys of its badge label
// and tooltip, and its color.
const MP_STATUS = {
  applied: ["devfolders.status.applied", "badge-success", "devfolders.status.appliedHint"],
  pending: ["devfolders.status.pending", "badge-warning", "devfolders.status.pendingHint"],
  stopped: ["devfolders.status.stopped", "badge-ghost", "devfolders.status.stoppedHint"],
  inactive: ["devfolders.status.inactive", "badge-error", "devfolders.status.inactiveHint"],
  missing: ["devfolders.status.missing", "badge-error", "devfolders.status.missingHint"],
};

// setMpStatus shows text in the status alert with the daisyUI alert kind
// ("info", "warning", "error"), or hides it when text is empty.
function setMpStatus(text, kind) {
  mpStatus.classList.remove("alert-info", "alert-warning", "alert-error");
  mpStatus.classList.toggle("hidden", !text);
  if (!text) return;
  mpStatus.classList.add("alert-" + kind);
  mpStatusText.textContent = text;
}

// fillMpSelect replaces select's options with values (value = label),
// after an optional placeholder, keeping the previous value when present.
function fillMpSelect(select, options, placeholder) {
  const previous = select.value;
  select.innerHTML = "";
  if (placeholder) {
    const opt = document.createElement("option");
    opt.value = "";
    opt.textContent = placeholder;
    select.appendChild(opt);
  }
  options.forEach(function (o) {
    const opt = document.createElement("option");
    opt.value = o.value;
    opt.textContent = o.label;
    select.appendChild(opt);
  });
  if (options.some(function (o) { return o.value === previous; })) select.value = previous;
}

// mpIsMoodle reports whether the selected app is a Moodle app (plugin type
// and name) rather than one taking a destination folder.
function mpIsMoodle() {
  const app = mpApps[mpApp.value];
  return !!app && app.type === "moodle";
}

// showMpFields shows the plugin fields for a Moodle app and the destination
// field for the others, with the hint of where the destination is relative
// to.
function showMpFields() {
  const app = mpApps[mpApp.value];
  const moodle = mpIsMoodle();
  mpMoodleFields.classList.toggle("hidden", !app || !moodle);
  mpDestField.classList.toggle("hidden", !app || moodle);
  if (app && !moodle) {
    mpDestHint.textContent = app.type === "php_node"
      ? t("devfolders.destHintCombo", { root: app.root })
      : t("devfolders.destHint", { root: app.root });
  }
}

// setMpLocked locks the type/name fields to the values read from the
// development folder's version.php, or unlocks them.
function setMpLocked(check) {
  const locked = !!(check && check.locked);
  if (locked) {
    if (!Array.prototype.some.call(mpType.options, function (o) { return o.value === check.type; })) {
      fillMpSelect(mpType, [{ value: check.type, label: check.type }], "");
    }
    mpType.value = check.type;
    mpName.value = check.name;
    mpLocked.textContent = t("devfolders.locked", { component: check.type + "_" + check.name });
  }
  mpType.disabled = locked;
  mpName.disabled = locked;
  mpLocked.classList.toggle("hidden", !locked);
}

// checkMpForm asks the backend whether the form can be created and
// enables "Criar" only when it can.
function checkMpForm() {
  clearTimeout(mpCheckTimer);
  btnMpCreate.disabled = true;
  if (!mpDev) {
    invalidate("moodleplugins-check");
    setMpLocked(null);
    setMpStatus("", "");
    return;
  }
  App.InspectDevFolder(mpDev, mpApp.value, mpType.value, mpName.value.trim(), mpDest.value.trim()).then(latestOnly("moodleplugins-check", function (check) {
    setMpLocked(check);
    if (check.ok) {
      setMpStatus(t("devfolders.willMount", { dest: check.destLabel }), "info");
      btnMpCreate.disabled = running;
      return;
    }
    setMpStatus(check.message, "warning");
  })).catch(function (err) { setMpStatus(String(err), "error"); });
}

// loadMpTypes shows the fields of the selected app and, for a Moodle app,
// fills "Tipo do plugin" with the types it accepts.
function loadMpTypes() {
  showMpFields();
  if (!mpIsMoodle()) {
    checkMpForm();
    return;
  }
  App.GetMoodlePluginTypes(mpApp.value).then(latestOnly("moodleplugins-types", function (types) {
    fillMpSelect(mpType, (types || []).map(function (ty) { return { value: ty, label: ty }; }), t("devfolders.chooseType"));
    checkMpForm();
  })).catch(function (err) {
    fillMpSelect(mpType, [], t("devfolders.noTypes"));
    setMpStatus(String(err), "error");
  });
}

btnMpPickDev.addEventListener("click", function () {
  App.PickDevFolder().then(function (path) {
    if (!path) return; // user cancelled the dialog
    mpDev = path;
    mpDevPath.textContent = path;
    mpDevPath.classList.remove("opacity-60");
    checkMpForm();
  });
});
mpApp.addEventListener("change", loadMpTypes);
mpType.addEventListener("change", checkMpForm);

// Typing in "Nome" or "Destino" asks the backend only once the typing
// pauses.
const MP_CHECK_DEBOUNCE_MS = 250;
let mpCheckTimer = null;
function checkMpFormLater() {
  btnMpCreate.disabled = true;
  clearTimeout(mpCheckTimer);
  mpCheckTimer = setTimeout(checkMpForm, MP_CHECK_DEBOUNCE_MS);
}
mpName.addEventListener("input", checkMpFormLater);
mpDest.addEventListener("input", checkMpFormLater);

btnMpCreate.addEventListener("click", function () {
  if (running || !mpDev) return;
  beginRun(t("devfolders.run.mount"), function () {
    return App.AddDevFolder(mpDev, mpApp.value, mpType.value, mpName.value.trim(), mpDest.value.trim());
  }, { buttons: [btnMpCreate] });
});

// buildMpRow returns the table row for one mounted folder.
function buildMpRow(row) {
  const tr = document.createElement("tr");
  const name = row.component || row.destLabel;

  const tdName = document.createElement("td");
  tdName.textContent = name;
  if (row.declaredAs) {
    const warn = document.createElement("div");
    warn.className = "text-xs text-warning";
    warn.textContent = t("devfolders.declares", { component: row.declaredAs });
    tdName.appendChild(warn);
  }
  tr.appendChild(tdName);

  const tdApp = document.createElement("td");
  tdApp.textContent = row.appLabel;
  tdApp.title = row.destLabel;
  tr.appendChild(tdApp);

  const tdDev = document.createElement("td");
  tdDev.className = "break-all";
  tdDev.textContent = row.devLabel;
  tdDev.title = row.devPath;
  tr.appendChild(tdDev);

  const info = MP_STATUS[row.status];
  const tdStatus = document.createElement("td");
  const badge = document.createElement("span");
  badge.className = "badge badge-sm whitespace-nowrap " + (info ? info[1] : "badge-ghost");
  badge.textContent = info ? t(info[0]) : row.status;
  badge.title = info ? t(info[2]) : "";
  tdStatus.appendChild(badge);
  tr.appendChild(tdStatus);

  const tdActions = document.createElement("td");
  const wrap = document.createElement("div");
  wrap.className = "flex gap-1";

  const btnRestart = iconButton("refreshCw", t("devfolders.restart"));
  btnRestart.addEventListener("click", function () {
    beginRun(t("devfolders.run.restart", { app: row.appLabel }), function () { return App.ApplyDevFolders(row.app); });
  });
  wrap.appendChild(btnRestart);

  const btnRemove = iconButton("trash2", t("devfolders.remove"));
  btnRemove.addEventListener("click", function () {
    if (running) return;
    openConfirm(t("devfolders.removeTitle"),
      t("devfolders.removeConfirm", { name: name, app: row.appLabel }),
      function () {
        beginRun(t("devfolders.run.remove", { name: name }), function () { return App.RemoveDevFolder(row.app, row.relPath); });
      });
  });
  wrap.appendChild(btnRemove);

  tdActions.appendChild(wrap);
  tr.appendChild(tdActions);
  return tr;
}

// showMpRows fills the table with the mounted folders of screen, and
// shows the update warning when some of their units predate the checks
// against links.
function showMpRows(screen) {
  mpRows.innerHTML = "";
  const rows = screen.rows || [];
  rows.forEach(function (row) { mpRows.appendChild(buildMpRow(row)); });
  mpEmpty.classList.toggle("hidden", rows.length > 0);
  const outdated = screen.outdatedUnits || 0;
  mpUnitsText.textContent = t("devfolders.unitsOutdated", { count: outdated });
  mpUnitsAlert.classList.toggle("hidden", outdated === 0);
}

btnMpUnits.addEventListener("click", function () {
  beginRun(t("devfolders.run.units"), function () { return App.UpdateDevFolderUnits(); });
});

// loadMpRows fills the table with every mounted folder and its status.
function loadMpRows() {
  App.GetDevFolders().then(latestOnly("moodleplugins-rows", showMpRows))
    .catch(function (err) { openError(t("devfolders.listFailed", { error: err })); });
}

function renderMoodlePluginsScreen() {
  mpDev = "";
  mpDevPath.textContent = t("devfolders.noFolder");
  mpDevPath.classList.add("opacity-60");
  mpName.value = "";
  mpDest.value = "";
  setMpLocked(null);
  setMpStatus("", "");
  btnMpCreate.disabled = true;
  // The rows share loadMpRows' key: whichever answer is older is dropped.
  const showRows = latestOnly("moodleplugins-rows", showMpRows);
  App.GetDevFolders().then(latestOnly("moodleplugins-screen", function (screen) {
    mpApps = {};
    (screen.apps || []).forEach(function (a) { mpApps[a.folder] = a; });
    const apps = (screen.apps || []).map(function (a) { return { value: a.folder, label: a.label }; });
    fillMpSelect(mpApp, apps, apps.length ? "" : t("devfolders.noApps"));
    if (apps.length && !mpApp.value) mpApp.value = apps[0].value;
    mpApp.disabled = !apps.length;
    loadMpTypes();
    showRows(screen);
  })).catch(function (err) { openError(t("devfolders.loadFailed", { error: err })); });
}
