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

// Perci82 Dev widgets: the container logs modal, the container status
// badge, the per-type container fields shared by Criar and Editar, and the
// Global/Local scoped tables of SKILLs and MCPs.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// --- container logs modal ---------------------------------------------------
let __logsFolder = "";
// The fetched log itself, kept apart from what the modal shows: "Exportar"
// saves this, never a "Carregando..."/"(sem saída)"/error placeholder.
let __logsText = "";
function refreshLogs() {
  const folder = __logsFolder;
  __logsText = "";
  btnLogsExport.disabled = true;
  logsModalContent.textContent = t("logs.loading");
  App.GetContainerLogsSnapshot(folder).then(function (text) {
    if (folder !== __logsFolder) return; // another container's logs opened meanwhile
    __logsText = text || "";
    btnLogsExport.disabled = __logsText === "";
    logsModalContent.textContent = __logsText || t("logs.empty");
  }).catch(function (err) {
    if (folder !== __logsFolder) return;
    logsModalContent.textContent = t("logs.failed", { error: err });
  });
}
function openLogs(folder, title) {
  __logsFolder = folder;
  logsModalTitle.textContent = t("logs.title", { name: title });
  showDialog(logsModal);
  refreshLogs();
}
btnLogsRefresh.addEventListener("click", refreshLogs);
btnLogsExport.addEventListener("click", function () {
  App.PickLogsExportPath(__logsFolder + "-logs.txt").then(function (path) {
    if (!path) return;
    return App.SaveTextFile(path, __logsText);
  }).catch(function (err) { openError(t("logs.exportFailed", { error: err })); });
});

// --- status → daisyUI "Status with ping animation" ---

// CONTAINER_TYPE_LABELS maps each container type to its name; "generic"
// is translated, so the map has getters.
const CONTAINER_TYPE_LABELS = {
  nginx: "Nginx", mariadb: "MariaDB", moodle: "Moodle", php: "PHP",
  get generic() { return t("docker.type.generic"); }, node: "Node", php_node: "PHP + Node",
};

function statusInfo(status) {
  switch (status) {
    case "running": return { label: t("docker.status.running"), color: "success", ping: true };
    case "restarting": return { label: t("docker.status.restarting"), color: "warning", ping: true };
    case "paused": return { label: t("docker.status.paused"), color: "warning", ping: false };
    case "created": return { label: t("docker.status.created"), color: "info", ping: false };
    case "exited": return { label: t("docker.status.exited"), color: "neutral", ping: false };
    case "dead": return { label: t("docker.status.dead"), color: "error", ping: false };
    case "": return { label: t("docker.status.missing"), color: "error", ping: false };
    default: return { label: status, color: "neutral", ping: false };
  }
}
// dockerError (GetContainerRows' row.dockerError) means Docker couldn't be
// queried: the badge says so and shows the reason as a tooltip instead of
// "não encontrado".
function statusBadgeHTML(status, dockerError) {
  const info = dockerError
    ? { label: t("docker.status.unreachable"), color: "warning", ping: false }
    : statusInfo(status);
  const dot = "status status-" + info.color;
  const indicator = info.ping
    ? '<span class="inline-grid *:[grid-area:1/1]"><span class="' + dot + ' animate-ping"></span><span class="' + dot + '"></span></span>'
    : '<span class="' + dot + '"></span>';
  // The default label is Docker's raw status — escaped like any other
  // non-literal text.
  const title = dockerError ? ' title="' + escAttr(dockerError) + '"' : "";
  return '<span class="inline-flex items-center gap-2"' + title + ">" + indicator + "<span>" + escHTML(info.label) + "</span></span>";
}

// --- dynamic per-type field construction (shared between Criar and the
// Editar modal) -----------------------------------------------------

// opts.secret renders a password input plus a "Mostrar"/"Ocultar" toggle
// (the delegated listener below) — a generated password still has to be
// readable once, to be written down.
function textFieldHTML(id, label, opts) {
  opts = opts || {};
  const input =
    '<input id="' + escAttr(id) + '" type="' + escAttr(opts.secret ? "password" : (opts.type || "text")) + '" class="input input-bordered' + (opts.secret ? " join-item flex-1" : "") + '" ' +
    (opts.placeholder ? 'placeholder="' + escAttr(opts.placeholder) + '" ' : "") +
    (opts.value !== undefined && opts.value !== null ? 'value="' + escAttr(opts.value) + '" ' : "") +
    (opts.secret ? 'autocomplete="off" ' : "") +
    "/>";
  return (
    '<label class="form-control">' +
    '<div class="label"><span class="label-text">' + escHTML(label) + "</span></div>" +
    (opts.secret
      ? '<div class="join w-full">' + input +
        '<button type="button" class="btn join-item" data-reveal="' + escAttr(id) + '" aria-pressed="false">' + escHTML(t("field.show")) + "</button></div>"
      : input) +
    "</label>"
  );
}
document.addEventListener("click", function (ev) {
  const btn = ev.target.closest("button[data-reveal]");
  if (!btn) return;
  ev.preventDefault(); // inside a <label>: don't focus/submit anything else
  const input = document.getElementById(btn.dataset.reveal);
  if (!input) return;
  const show = input.type === "password";
  input.type = show ? "text" : "password";
  btn.textContent = show ? t("field.hide") : t("field.show");
  btn.setAttribute("aria-pressed", show ? "true" : "false");
});
function selectFieldHTML(id, label, options, selected) {
  const opts = (options || []).map(function (v) {
    return '<option value="' + escAttr(v) + '"' + (v === selected ? " selected" : "") + ">" + escHTML(v) + "</option>";
  }).join("");
  return (
    '<label class="form-control">' +
    '<div class="label"><span class="label-text">' + escHTML(label) + "</span></div>" +
    '<select id="' + escAttr(id) + '" class="select select-bordered">' + opts + "</select></label>"
  );
}
function checkboxFieldHTML(id, label, checked, disabled, hint) {
  return (
    '<label class="label cursor-pointer justify-start gap-3">' +
    '<input id="' + escAttr(id) + '" type="checkbox" class="checkbox"' + (checked ? " checked" : "") + (disabled ? " disabled" : "") + " />" +
    '<span class="label-text">' + escHTML(label) + (hint ? '<br><span class="text-xs opacity-60">' + escHTML(hint) + "</span>" : "") + "</span>" +
    "</label>"
  );
}

// reconcileKeyedList keeps container in sync with items (array), avoiding
// innerHTML="" + full rebuild on every update — used by
// renderAgentSkillsTable/renderMCPServersTable. keyOf(item)
// identifies each node across calls (e.g. slug/path); signatureOf(item)
// summarizes every field that affects the node (content, button state,
// text) — when it hasn't changed since the last render, the existing node
// is left untouched (buildNode is never called); when it has changed, or
// the item is new, buildNode(item) builds a fresh node from scratch
// (handlers are always current, never at risk of referencing stale
// state); items that disappeared from items are removed. The final node
// order follows items' own order.
function reconcileKeyedList(container, items, keyOf, signatureOf, buildNode) {
  const existing = new Map();
  Array.prototype.forEach.call(container.children, function (el) {
    if (el.dataset && el.dataset.rkKey !== undefined) existing.set(el.dataset.rkKey, el);
  });

  let afterNode = null; // last node already in its right place in this pass
  items.forEach(function (item) {
    const key = String(keyOf(item));
    const sig = String(signatureOf(item));
    let node = existing.get(key);
    if (node) {
      existing.delete(key);
      if (node.dataset.rkSig !== sig) {
        const fresh = buildNode(item);
        fresh.dataset.rkKey = key;
        fresh.dataset.rkSig = sig;
        node.replaceWith(fresh);
        node = fresh;
      }
    } else {
      node = buildNode(item);
      node.dataset.rkKey = key;
      node.dataset.rkSig = sig;
    }
    const wantedNext = afterNode ? afterNode.nextSibling : container.firstChild;
    if (wantedNext !== node) container.insertBefore(node, wantedNext);
    afterNode = node;
  });

  // leftovers: keys that existed before and no longer appear in items
  existing.forEach(function (node) { node.remove(); });
}

// buildContainerFieldsHTML renders the fields for one final container
// kind ("nginx"|"mariadb"|"moodle"|"php"|"generic"|"node"|"php_node")
// into prefix-scoped ids (ex. "dc-folder", "edit-folder") — reused by
// both #dc-fields (Criar) and #edit-fields (Editar), values pre-fills
// when editing.
function buildContainerFieldsHTML(prefix, kind, catalog, values) {
  values = values || {};
  const p = prefix;
  if (kind === "nginx") {
    return '<p class="opacity-70 text-sm">' + escHTML(t("docker.nginxNoParams")) + "</p>";
  }
  if (kind === "mariadb") {
    return (
      textFieldHTML(p + "-dbUser", t("docker.field.dbUser"), { value: values.dbUser || "dev_user" }) +
      textFieldHTML(p + "-dbPass", t("docker.field.dbPass"), { value: values.dbPass || "", secret: true })
    );
  }
  const isPHPFamily = kind === "moodle" || kind === "php" || kind === "generic";
  const isNodeFamily = kind === "node" || kind === "php_node";
  let html = "";
  html += textFieldHTML(p + "-name", t("docker.field.name"), { value: values.name || "" });
  html += textFieldHTML(p + "-folder", t("docker.field.folder"), { value: values.folder || "", placeholder: t("docker.field.folderPlaceholder") });
  html += textFieldHTML(p + "-url", "URL", { value: values.url || "" });

  if (kind === "moodle") {
    const mv = values.moodleVersion || catalog.moodleVersions[catalog.moodleVersions.length - 1];
    // A saved value no longer offered (e.g. "4.x") stays selectable, so
    // editing the container keeps it unless the user picks another one.
    const moodleOptions = catalog.moodleVersions.indexOf(mv) >= 0 ? catalog.moodleVersions : catalog.moodleVersions.concat([mv]);
    html += selectFieldHTML(p + "-moodleVersion", t("docker.field.moodleVersion"), moodleOptions, mv);
  }
  if (isPHPFamily) {
    html += '<div id="' + p + '-phpVersion-wrap">' + selectFieldHTML(p + "-phpVersion", t("docker.field.phpVersion"), catalog.phpVersions, values.phpVersion || catalog.phpVersions[0]) + "</div>";
    html += textFieldHTML(p + "-phpMemoryLimit", t("docker.field.phpMemoryLimit"), { value: values.phpMemoryLimit || "", placeholder: "512M" });
  }
  if (isNodeFamily) {
    html += selectFieldHTML(p + "-nodeVersion", t("docker.field.nodeVersion"), catalog.nodeVersions, values.nodeVersion || catalog.nodeVersions[0]);
    html += textFieldHTML(p + "-devCommand", t("docker.field.devCommand"), { value: values.devCommand || "npm run dev" });
    html += textFieldHTML(p + "-devPort", t("docker.field.devPort"), { type: "number", value: values.devPort || 5173 });
  }
  if (kind === "php_node") {
    html += textFieldHTML(p + "-workerCommand", t("docker.field.workerCommand"), { value: values.workerCommand || "" });
  }
  if (isPHPFamily || kind === "php_node") {
    html += checkboxFieldHTML(
      p + "-dbAccess", t("docker.field.dbAccess"), !!values.dbAccess, !catalog.mariadbReady,
      catalog.mariadbReady ? "" : t("docker.field.dbAccessHint")
    );
  }
  return html;
}

// wireMoodlePHPFilter: "Versão PHP" only shows the versions compatible
// with the chosen "Versão do Moodle" (appstack.PHPVersionsForMoodleVersion).
function wireMoodlePHPFilter(prefix) {
  const moodleSel = document.getElementById(prefix + "-moodleVersion");
  const phpWrap = document.getElementById(prefix + "-phpVersion-wrap");
  if (!moodleSel || !phpWrap) return;
  function refresh() {
    const current = document.getElementById(prefix + "-phpVersion");
    const currentVal = current ? current.value : "";
    App.GetPHPVersionsForMoodleVersion(moodleSel.value).then(latestOnly("php-versions-" + prefix, function (versions) {
      phpWrap.innerHTML = selectFieldHTML(prefix + "-phpVersion", t("docker.field.phpVersion"), versions, versions.indexOf(currentVal) >= 0 ? currentVal : versions[0]);
    }));
  }
  moodleSel.addEventListener("change", refresh);
  refresh();
}

// wireFolderURLAutofill: "URL" follows "Pasta" (folder + ".localhost")
// until the user edits the URL by hand.
function wireFolderURLAutofill(prefix) {
  const folderEl = document.getElementById(prefix + "-folder");
  const urlEl = document.getElementById(prefix + "-url");
  if (!folderEl || !urlEl) return;
  let urlEdited = !!urlEl.value;
  urlEl.addEventListener("input", function () { urlEdited = true; });
  folderEl.addEventListener("input", function () {
    if (urlEdited) return;
    const slug = folderEl.value.trim().toLowerCase().replace(/[^a-z0-9-]/g, "-");
    urlEl.value = slug ? slug + ".localhost" : "";
  });
}

function readContainerFieldsValues(prefix, kind) {
  const p = prefix;
  const get = function (id) { const el = document.getElementById(id); return el ? el.value : ""; };
  const getChecked = function (id) { const el = document.getElementById(id); return !!(el && el.checked); };

  if (kind === "nginx") return { kind: "nginx" };
  if (kind === "mariadb") return { kind: "mariadb", dbUser: get(p + "-dbUser"), dbPass: get(p + "-dbPass") };

  const req = { kind: kind, name: get(p + "-name"), folder: get(p + "-folder"), url: get(p + "-url") };
  if (kind === "moodle") req.moodleVersion = get(p + "-moodleVersion");
  if (kind === "moodle" || kind === "php" || kind === "generic" || kind === "php_node") {
    req.phpVersion = get(p + "-phpVersion");
    req.phpMemoryLimit = get(p + "-phpMemoryLimit");
    req.dbAccess = getChecked(p + "-dbAccess");
  }
  if (kind === "node" || kind === "php_node") {
    req.nodeVersion = get(p + "-nodeVersion");
    req.devCommand = get(p + "-devCommand");
    req.devPort = parseInt(get(p + "-devPort"), 10) || 0;
  }
  if (kind === "php_node") req.workerCommand = get(p + "-workerCommand");
  return req;
}

// createScopedTable drives a "scope + table" screen (IA: SKILLs, IA: MCPs):
// a Global/Local select, the alert describing the scope (with the folder
// picker for Local) and a keyed table of rows fetched for that scope.
//   cfg.els      — { select, alertTitle, alertDesc, pickButton, rows }
//   cfg.noun     — "das SKILLs" / "dos MCPs", used in the alert text
//   cfg.latestKey, cfg.pickFolder(), cfg.fetchRows(global, folder)
//   cfg.keyOf(row), cfg.sigOf(row, state), cfg.buildRow(row, state)
//   cfg.onScope(state) — optional, after every scope/folder change
// Returns the state ({ scope, folder, isGlobal() }) plus render()/reset().
function createScopedTable(cfg) {
  const els = cfg.els;
  const state = {
    scope: "global", // "global" | "local"
    folder: "",
    isGlobal: function () { return state.scope === "global"; },
  };

  function isReady() {
    return state.isGlobal() || !!state.folder;
  }

  function renderAlert() {
    if (cfg.onScope) cfg.onScope(state);
    if (state.isGlobal()) {
      els.alertTitle.textContent = "Instalação Global";
      els.alertDesc.textContent = "A instalação " + cfg.noun + " será feita globalmente.";
      els.pickButton.classList.add("hidden");
      return;
    }
    const desc = "A instalação " + cfg.noun + " será feita exclusivamente para este projeto.";
    els.alertTitle.textContent = "Caminho da pasta";
    if (state.folder) {
      els.alertDesc.innerHTML = escHTML(desc) + '<br><span class="text-xs opacity-70">' + escHTML(state.folder) + "</span>";
    } else {
      els.alertDesc.textContent = desc;
    }
    els.pickButton.textContent = state.folder ? "Alterar Pasta" : "Selecionar Pasta";
    els.pickButton.classList.remove("hidden");
  }

  function render() {
    if (!isReady()) {
      invalidate(cfg.latestKey);
      els.rows.innerHTML = "";
      return;
    }
    cfg.fetchRows(state.isGlobal(), state.folder).then(latestOnly(cfg.latestKey, function (rows) {
      // Reconciled by key instead of innerHTML="" + full rebuild — a row
      // is only recreated when its signature changed.
      reconcileKeyedList(
        els.rows,
        rows || [],
        cfg.keyOf,
        function (row) { return cfg.sigOf(row, state); },
        function (row) { return cfg.buildRow(row, state); }
      );
    }));
  }

  els.select.addEventListener("change", function () {
    state.scope = els.select.value;
    renderAlert();
    render();
  });
  els.pickButton.addEventListener("click", function () {
    cfg.pickFolder().then(function (path) {
      if (!path) return; // user cancelled the dialog
      state.folder = path;
      renderAlert();
      render();
    });
  });

  state.render = render;
  state.reset = function () {
    state.scope = "global";
    state.folder = "";
    els.select.value = "global";
    renderAlert();
    render();
  };
  return state;
}
