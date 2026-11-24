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

// Dev Tools :: IA: Contextos.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// ===========================================================================
// "Contextos de IA": Gerenciar Projetos (#aicdefine-container, plus the
// "Contextos aplicados" tab with the registered folders and the
// "Categorias" tab), Gerenciar
// Contextos (#aiclibrary-container, plus the "Contextos personalizados"
// tab) and Exportar/Importar (#aictransfer-container, plus the "Importar"
// tab).
// ===========================================================================

const aicdefineContainer = document.getElementById("aicdefine-container");
const aiclibraryContainer = document.getElementById("aiclibrary-container");
const aictransferContainer = document.getElementById("aictransfer-container");
const tabLabelAicApplied = document.getElementById("tab-label-aic-applied");
const tabLabelAicCustom = document.getElementById("tab-label-aic-custom");
const tabLabelAicImport = document.getElementById("tab-label-aic-import");
const aicCustomSection = document.getElementById("aic-custom-section");
const aicCustomItems = document.getElementById("aic-custom-items");
const aicLibRows = document.getElementById("aic-lib-rows");
const aicCustomRows = document.getElementById("aic-custom-rows");
const aicCustomEmpty = document.getElementById("aic-custom-empty");
const btnAicCustomAdd = document.getElementById("btn-aic-custom-add");
const aicViewModal = document.getElementById("aic-view-modal");
const aicViewTitle = document.getElementById("aic-view-title");
const aicViewContent = document.getElementById("aic-view-content");
const aicCustomModal = document.getElementById("aic-custom-modal");
const aicCustomTitle = document.getElementById("aic-custom-title");
const aicCustomName = document.getElementById("aic-custom-name");
const aicCustomApplies = document.getElementById("aic-custom-applies");
const aicCustomContent = document.getElementById("aic-custom-content");
const btnAicCustomSave = document.getElementById("btn-aic-custom-save");
const aicRowsEl = document.getElementById("aic-rows");
const aicEmpty = document.getElementById("aic-empty");
const btnAicUpdateAll = document.getElementById("btn-aic-update-all");
const aicName = document.getElementById("aic-name");
const aicUnregistered = document.getElementById("aic-unregistered");
const aicUnregisteredDesc = document.getElementById("aic-unregistered-desc");
const btnAicRegister = document.getElementById("btn-aic-register");
const aicExportList = document.getElementById("aic-export-list");
const aicExportEmpty = document.getElementById("aic-export-empty");
const btnAicExport = document.getElementById("btn-aic-export");
const btnAicImportPick = document.getElementById("btn-aic-import-pick");
const aicImportPathEl = document.getElementById("aic-import-path");
const aicImportList = document.getElementById("aic-import-list");
const aicImportNote = document.getElementById("aic-import-note");
const aicImportNoteText = document.getElementById("aic-import-note-text");
const btnAicImport = document.getElementById("btn-aic-import");
const tabLabelAicCategories = document.getElementById("tab-label-aic-categories");
const aicCategory = document.getElementById("aic-category");
const aicFilter = document.getElementById("aic-filter");
const aicFilterEmpty = document.getElementById("aic-filter-empty");
const aicCatNew = document.getElementById("aic-cat-new");
const btnAicCatAdd = document.getElementById("btn-aic-cat-add");
const aicCatRows = document.getElementById("aic-cat-rows");
const aicCatEmpty = document.getElementById("aic-cat-empty");
const aicCatModal = document.getElementById("aic-cat-modal");
const aicCatName = document.getElementById("aic-cat-name");
const btnAicCatSave = document.getElementById("btn-aic-cat-save");
const aicDetailsModal = document.getElementById("aic-details-modal");
const aicDetailsTitle = document.getElementById("aic-details-title");
const aicDetailsPath = document.getElementById("aic-details-path");
const aicDetailsCategory = document.getElementById("aic-details-category");
const aicDetailsModels = document.getElementById("aic-details-models");
const aicDetailsAgents = document.getElementById("aic-details-agents");

// aicFolder is the folder "Definir Contexto" works on; aicContexts the
// rows of the "Contextos" tab last loaded (GetAIContexts); aicAgentLabels
// the label of each agent ID.
let aicFolder = "";
let aicContexts = [];
let aicAgentLabels = {};
// aicAgentList is every agent ({id, label}), shown disabled before a folder
// is chosen.
let aicAgentList = [];
// aicCategories is the categories ({name, uses}) last loaded; aicFilterValue
// the "Filtrar" choice: AIC_FILTER_ALL, AIC_FILTER_NONE or a category.
let aicCategories = [];
const AIC_FILTER_ALL = "";
const AIC_FILTER_NONE = "\u0000none";
let aicFilterValue = AIC_FILTER_ALL;

const aicAgentLabelsReady = App.GetAIContextAgents("").then(function (agents) {
  aicAgentList = (agents || []).map(function (a) { return { id: a.id, label: a.label }; });
  aicAgentList.forEach(function (a) { aicAgentLabels[a.id] = a.label; });
}).catch(function () {});

// aicModelLabels is the label of each standard ID, in the language of the
// last load (aicLoadModelLabels).
let aicModelLabels = {};
function aicLoadModelLabels() {
  return App.GetAIContextItems("").then(function (items) {
    aicModelLabels = {};
    (items || []).forEach(function (it) { aicModelLabels[it.id] = it.label; });
  }).catch(function () {});
}

// aicFolderName is the last segment of path (path itself when it has none).
function aicFolderName(path) {
  const parts = path.split("/").filter(Boolean);
  return parts.length ? parts[parts.length - 1] : path;
}

// aicAgentsText names the agents of ids, "—" for none.
function aicAgentsText(ids) {
  return (ids || []).map(function (id) { return aicAgentLabels[id] || id; }).join(", ") || "—";
}

// aicModelsText names the standards of models, "—" for none.
function aicModelsText(models) {
  return (models || []).map(function (m) { return aicModelLabels[m] || m; }).join(", ") || "—";
}

// aicBadges returns a wrapping list of small badges, one per text, or "—"
// when there is none.
function aicBadges(texts) {
  const wrap = document.createElement("div");
  wrap.className = "flex flex-wrap gap-1";
  if (!texts.length) {
    wrap.textContent = "—";
    return wrap;
  }
  texts.forEach(function (text) {
    const b = document.createElement("span");
    b.className = "badge badge-sm badge-soft";
    b.textContent = text;
    wrap.appendChild(b);
  });
  return wrap;
}

// aicModelLabelList and aicAgentLabelList name the standards and agents of
// a folder.
function aicModelLabelList(models) {
  return (models || []).map(function (m) { return aicModelLabels[m] || m; });
}
function aicAgentLabelList(ids) {
  return (ids || []).map(function (id) { return aicAgentLabels[id] || id; });
}

// aicNameError tells what is wrong with name for the folder at path
// ("" when nothing): empty, or used by another registered folder.
function aicNameError(name, path) {
  if (!name) return t("aic.err.noName");
  const lower = name.toLowerCase();
  const other = aicContexts.find(function (c) { return c.path !== path && c.name.toLowerCase() === lower; });
  return other ? t("aic.err.nameTaken", { name: name, path: other.path }) : "";
}

// AIC_ACTIONS maps a change's action to the message key of its verb.
const AIC_ACTIONS = { create: "aic.act.create", update: "aic.act.update", remove: "aic.act.remove", skip: "aic.act.skip" };

// aicChangeLines lists changes ({path, action}), one per line, indented by
// indent.
function aicChangeLines(changes, indent) {
  return changes.map(function (c) { return indent + "• " + (AIC_ACTIONS[c.action] ? t(AIC_ACTIONS[c.action]) : c.action) + ": " + c.path; }).join("\n");
}

// aicFolderChangesText lists the changes of each folder of previews
// (AIContextFolderChanges), marking the folders of edited (path → true);
// pendingText describes a folder that does not exist.
function aicFolderChangesText(previews, edited, pendingText) {
  return previews.map(function (p) {
    const head = p.name + " — " + p.path + (edited && edited[p.path] ? " " + t("aic.hasEdited") : "");
    let body;
    if (p.pending) body = "  • " + pendingText;
    else if (!p.changes.length) body = "  • " + t("aic.nothingToChange");
    else body = aicChangeLines(p.changes, "  ");
    return head + "\n" + body;
  }).join("\n\n");
}

// ===========================================================================
// "Contextos aplicados" tab.
// ===========================================================================

// AIC_ROW_STATE describes each folder state (managerai.ContextState).
const AIC_ROW_STATE = {
  applied: { get label() { return t("gac.ch.current"); }, cls: "badge-success", get title() { return t("aic.row.appliedHint"); } },
  outdated: { get label() { return t("aic.st.outdated"); }, cls: "badge-warning", get title() { return t("aic.row.outdatedHint"); } },
  edited: { get label() { return t("aic.st.edited"); }, cls: "badge-error", get title() { return t("aic.row.editedHint"); } },
  pending: { get label() { return t("aic.st.pending"); }, cls: "badge-ghost", get title() { return t("aic.row.pendingHint"); } },
};

// aicNeedsUpdate reports whether Atualizar todos covers row.
function aicNeedsUpdate(row) {
  return row.state === "outdated" || row.state === "edited";
}

// aicUpdate previews Atualizar on the folders of paths and, after the
// confirmation, regenerates the ones with changes.
function aicUpdate(paths) {
  if (running) return;
  App.PreviewAIContextUpdate(paths).then(function (previews) {
    previews = previews || [];
    const todo = previews.filter(function (p) { return !p.pending && p.changes.length; });
    if (todo.length === 0) {
      openMessage(t("aic.nothingTitle"), t("aic.nothingRegistered"), false);
      return;
    }
    const edited = {};
    aicContexts.forEach(function (c) { if (c.state === "edited") edited[c.path] = true; });
    const msg = aicFolderChangesText(previews, edited, t("aic.skippedMissing")) + "\n\n" + t("aic.backupContinue");
    openConfirm(t("aic.updateTitle"), msg, function () {
      beginRun(t("aic.updateTitle"), function () {
        return App.UpdateAIContexts(todo.map(function (p) { return p.path; }));
      }, { buttons: [btnAicUpdateAll] });
    });
  }).catch(function (err) { openError(t("aic.previewFailed", { error: err })); });
}

// aicKeepFolder makes the next Aplicar Contexto render keep aicFolder
// (set by Alterar) instead of starting without a folder.
let aicKeepFolder = false;

// aicEdit opens row's folder on the "Aplicar contexto" tab.
function aicEdit(row) {
  aicFolder = row.path;
  aicKeepFolder = true;
  // Selecting the menu item renders the screen once; it is rendered
  // directly only when the item is missing.
  if (!selectMenuItem("aicontext-define|")) renderAicDefineScreen();
}

// aicRemove previews Remover on row's folder and, after the confirmation,
// removes its context and drops it from the list.
function aicRemove(row) {
  if (running) return;
  App.PreviewAIContextRemove(row.path).then(function (changes) {
    changes = changes || [];
    const files = changes.length
      ? t("aic.removeFiles", { path: row.path }) + "\n\n" + aicChangeLines(changes, "") + "\n\n" + t("aic.removeKept")
      : t("aic.removeNone", { path: row.path });
    openConfirm(t("aic.removeTitle"), files + "\n\n" + t("aic.removeConfirm", { name: row.name }), function () {
      beginRun(t("aic.removeTitle"), function () { return App.RemoveAIContext(row.path); });
    });
  }).catch(function (err) { openError(t("aic.previewFailed", { error: err })); });
}

// aicShowDetails opens the "+" window of row: its category, standards and
// agents.
function aicShowDetails(row) {
  aicDetailsTitle.textContent = row.name;
  aicDetailsPath.textContent = row.path;
  aicDetailsCategory.textContent = row.category || t("aic.noCategory");
  aicDetailsModels.innerHTML = "";
  aicDetailsModels.appendChild(aicBadges(aicModelLabelList(row.models)));
  aicDetailsAgents.innerHTML = "";
  aicDetailsAgents.appendChild(aicBadges(aicAgentLabelList(row.agents)));
  showDialog(aicDetailsModal);
}

// buildAicRow builds the table row of one registered folder.
function buildAicRow(row) {
  const tr = document.createElement("tr");
  const cell = function (text, cls) {
    const td = document.createElement("td");
    if (cls) td.className = cls;
    td.textContent = text;
    tr.appendChild(td);
    return td;
  };
  cell(row.name, "font-medium");
  cell(row.path, "break-all text-sm");

  const tdState = document.createElement("td");
  const st = AIC_ROW_STATE[row.state] || { label: row.state, cls: "badge-ghost", title: "" };
  const badge = document.createElement("span");
  badge.className = "badge badge-sm whitespace-nowrap " + st.cls;
  badge.textContent = st.label;
  badge.title = st.title;
  tdState.appendChild(badge);
  if (row.divergent && row.state !== "pending") {
    const div = document.createElement("span");
    div.className = "badge badge-sm badge-warning badge-outline whitespace-nowrap ml-1";
    div.textContent = t("aic.divergent");
    div.title = t("aic.divergentHint");
    tdState.appendChild(div);
  }
  tr.appendChild(tdState);

  const tdActions = document.createElement("td");
  const wrap = document.createElement("div");
  wrap.className = "flex gap-1";
  const pending = row.state === "pending";

  const btnUpdate = iconButton("refreshCw", t("action.update"));
  btnUpdate.disabled = pending;
  btnUpdate.addEventListener("click", function () { aicUpdate([row.path]); });
  wrap.appendChild(btnUpdate);

  const btnEdit = iconButton("pencil", t("aic.change"));
  btnEdit.disabled = pending;
  btnEdit.addEventListener("click", function () { aicEdit(row); });
  wrap.appendChild(btnEdit);

  const btnDetails = iconButton("plus", t("aic.details"));
  btnDetails.addEventListener("click", function () { aicShowDetails(row); });
  wrap.appendChild(btnDetails);

  const btnRemove = iconButton("trash2", t("docker.remove"));
  btnRemove.addEventListener("click", function () { aicRemove(row); });
  wrap.appendChild(btnRemove);

  tdActions.appendChild(wrap);
  tr.appendChild(tdActions);
  return tr;
}

// aicVisibleRows lists the registered folders the "Filtrar" choice shows.
function aicVisibleRows() {
  if (aicFilterValue === AIC_FILTER_ALL) return aicContexts;
  const want = aicFilterValue === AIC_FILTER_NONE ? "" : aicFilterValue;
  return aicContexts.filter(function (c) { return (c.category || "") === want; });
}

// renderAicTable fills the "Contextos aplicados" table with the folders the
// filter shows; Atualizar todos covers only those.
function renderAicTable() {
  const visible = aicVisibleRows();
  aicRowsEl.innerHTML = "";
  visible.forEach(function (row) { aicRowsEl.appendChild(buildAicRow(row)); });
  aicEmpty.classList.toggle("hidden", aicContexts.length > 0);
  aicFilterEmpty.classList.toggle("hidden", aicContexts.length === 0 || visible.length > 0);
  btnAicUpdateAll.disabled = !visible.some(aicNeedsUpdate);
}

// renderAicRows fills the "Contextos aplicados" table and the Exportar checklist
// with rows (AIContextRow).
function renderAicRows(rows) {
  aicContexts = rows || [];
  renderAicTable();
  renderAicExport();
}

function loadAicRows() {
  Promise.all([aicAgentLabelsReady, App.GetAIContexts(), aicLoadModelLabels()]).then(latestOnly("aic-rows", function (res) {
    renderAicRows(res[1]);
  })).catch(function (err) { openError(t("aic.listFailed", { error: err })); });
}

btnAicUpdateAll.addEventListener("click", function () {
  const paths = aicVisibleRows().filter(aicNeedsUpdate).map(function (c) { return c.path; });
  if (paths.length) aicUpdate(paths);
});

aicFilter.addEventListener("change", function () {
  aicFilterValue = aicFilter.value;
  renderAicTable();
});

// ===========================================================================
// Categories: the "Categoria" select of "Aplicar contexto", the "Filtrar"
// select of "Contextos aplicados" and the "Categorias" tab.
// ===========================================================================

// aicOption builds one <option>.
function aicOption(value, text) {
  const opt = document.createElement("option");
  opt.value = value;
  opt.textContent = text;
  return opt;
}

// renderAicCategorySelect fills the "Categoria" select, keeping its value
// when it still exists.
function renderAicCategorySelect() {
  const keep = aicCategory.value;
  aicCategory.innerHTML = "";
  aicCategory.appendChild(aicOption("", t("aic.noCategory")));
  aicCategories.forEach(function (c) { aicCategory.appendChild(aicOption(c.name, c.name)); });
  aicCategory.value = aicCategories.some(function (c) { return c.name === keep; }) ? keep : "";
}

// renderAicFilterSelect fills the "Filtrar" select; a filter on a category
// that no longer exists goes back to "Todas".
function renderAicFilterSelect() {
  aicFilter.innerHTML = "";
  aicFilter.appendChild(aicOption(AIC_FILTER_ALL, t("aic.filterAll")));
  aicFilter.appendChild(aicOption(AIC_FILTER_NONE, t("aic.noCategory")));
  aicCategories.forEach(function (c) { aicFilter.appendChild(aicOption(c.name, c.name)); });
  if (aicFilterValue !== AIC_FILTER_ALL && aicFilterValue !== AIC_FILTER_NONE &&
      !aicCategories.some(function (c) { return c.name === aicFilterValue; })) {
    aicFilterValue = AIC_FILTER_ALL;
  }
  aicFilter.value = aicFilterValue;
}

// aicCatRenaming is the category the Renomear window works on.
let aicCatRenaming = "";

// buildAicCatRow builds the row of one category of the "Categorias" tab.
function buildAicCatRow(cat) {
  const tr = document.createElement("tr");
  const tdName = document.createElement("td");
  tdName.className = "font-medium";
  tdName.textContent = cat.name;
  tr.appendChild(tdName);
  const tdUses = document.createElement("td");
  tdUses.textContent = String(cat.uses);
  tr.appendChild(tdUses);
  const tdActions = document.createElement("td");
  const wrap = document.createElement("div");
  wrap.className = "flex gap-1";
  const btnRename = iconButton("pencil", t("aic.cat.rename"));
  btnRename.addEventListener("click", function () {
    aicCatRenaming = cat.name;
    aicCatName.value = cat.name;
    showDialog(aicCatModal);
  });
  wrap.appendChild(btnRename);
  const btnDelete = iconButton("trash2", t("aic.cat.delete"));
  btnDelete.addEventListener("click", function () {
    const msg = cat.uses
      ? t("aic.cat.deleteUsed", { name: cat.name, count: cat.uses })
      : t("aic.cat.deleteConfirm", { name: cat.name });
    openConfirm(t("aic.cat.deleteTitle"), msg, function () {
      App.DeleteAICategory(cat.name).then(aicCategoriesChanged)
        .catch(function (err) { openError(String(err)); });
    });
  });
  wrap.appendChild(btnDelete);
  tdActions.appendChild(wrap);
  tr.appendChild(tdActions);
  return tr;
}

// renderAicCatRows fills the "Categorias" tab.
function renderAicCatRows() {
  aicCatRows.innerHTML = "";
  aicCategories.forEach(function (c) { aicCatRows.appendChild(buildAicCatRow(c)); });
  aicCatEmpty.classList.toggle("hidden", aicCategories.length > 0);
}

// loadAicCategories reads the categories and refreshes everything that
// shows them.
function loadAicCategories() {
  return App.GetAICategories().then(latestOnly("aic-categories", function (cats) {
    aicCategories = Array.isArray(cats) ? cats : [];
    renderAicCategorySelect();
    renderAicFilterSelect();
    renderAicCatRows();
    renderAicTable();
  })).catch(function (err) { openError(t("aic.listFailed", { error: err })); });
}

// aicCategoriesChanged reloads the categories and the registered folders
// after a category was added, renamed or deleted.
function aicCategoriesChanged() {
  loadAicCategories();
  loadAicRows();
}

btnAicCatAdd.addEventListener("click", function () {
  const name = aicCatNew.value.trim();
  if (!name) {
    openError(t("aic.cat.emptyName"));
    return;
  }
  App.AddAICategory(name).then(function () {
    aicCatNew.value = "";
    aicCategoriesChanged();
  }).catch(function (err) { openError(String(err)); });
});

btnAicCatSave.addEventListener("click", function () {
  const name = aicCatName.value.trim();
  if (!name) {
    openError(t("aic.cat.emptyName"));
    return;
  }
  App.RenameAICategory(aicCatRenaming, name).then(function () {
    aicCatModal.close();
    aicCategoriesChanged();
  }).catch(function (err) { openError(String(err)); });
});

// ===========================================================================
// "Aplicar contexto" tab.
// ===========================================================================

function renderAicAlert() {
  if (!aicFolder) {
    aicAlertTitle.textContent = t("aic.pickTitle");
    aicAlertDesc.textContent = t("aic.pickDesc");
    btnAicPickFolder.textContent = t("devfolders.pick");
    return;
  }
  aicAlertTitle.textContent = t("aic.folderName", { name: aicFolderName(aicFolder) });
  aicAlertDesc.textContent = aicFolder;
  btnAicPickFolder.textContent = t("gt.changeFolder");
}

btnAicPickFolder.addEventListener("click", function () {
  App.PickAIContextFolder().then(function (path) {
    if (!path) return; // user cancelled the dialog
    aicFolder = path;
    renderAicAlert();
    renderAicItems();
  });
});

// aicPresets holds the project types (name, models, hint) from the backend.
let aicPresets = [];

// setAicPresetHint shows the hint of the selected project type, if any.
function setAicPresetHint(preset) {
  const hint = preset && preset.hint ? preset.hint : "";
  aicPresetHint.textContent = hint;
  aicPresetHint.classList.toggle("hidden", !hint);
}

// renderAicPresetOptions fills the project type select: "Customizado"
// plus one option per preset.
function renderAicPresetOptions() {
  aicPreset.innerHTML = "";
  const custom = document.createElement("option");
  custom.value = "";
  custom.textContent = t("aic.custom");
  aicPreset.appendChild(custom);
  aicPresets.forEach(function (p) {
    const opt = document.createElement("option");
    opt.value = p.name;
    opt.textContent = p.label || p.name;
    aicPreset.appendChild(opt);
  });
  aicPreset.value = "";
  setAicPresetHint(null);
}

Promise.all([App.GetAIContextPresets(), i18nReady]).then(function (res) {
  aicPresets = res[0] || [];
  renderAicPresetOptions();
}).catch(function () {
  aicPresets = [];
});

// renderAicAgents fills the agent checkboxes from agents
// ({id,label,selected}); disabled shows them without letting them change.
function renderAicAgents(agents, disabled) {
  aicAgents.innerHTML = "";
  (agents || []).forEach(function (a) {
    const label = document.createElement("label");
    label.className = "label cursor-pointer gap-2";
    const box = document.createElement("input");
    box.type = "checkbox";
    box.className = "checkbox checkbox-sm";
    box.dataset.agent = a.id;
    box.checked = !!a.selected;
    box.disabled = !!disabled;
    label.appendChild(box);
    const text = document.createElement("span");
    text.className = "label-text";
    text.textContent = a.label;
    label.appendChild(text);
    aicAgents.appendChild(label);
  });
}

// aicCheckedAgents lists the IDs of the checked agents.
function aicCheckedAgents() {
  return Array.prototype.filter.call(aicAgents.querySelectorAll("input[data-agent]"), function (box) { return box.checked; })
    .map(function (box) { return box.dataset.agent; });
}

// AIC_STATUS describes each standard state (managerai.GuideStatus).
const AIC_STATUS = {
  applied: { get label() { return t("aic.st.applied"); }, cls: "badge-success", get title() { return t("aic.item.appliedHint"); } },
  outdated: { get label() { return t("aic.st.outdated"); }, cls: "badge-warning", get title() { return t("aic.item.outdatedHint"); } },
  edited: { get label() { return t("aic.st.edited"); }, cls: "badge-error", get title() { return t("aic.item.editedHint"); } },
};

// buildAicItem builds the row of one standard: checkbox, name, state
// badge and description; a complement is indented under its base.
function buildAicItem(it) {
  const built = buildChecklistItemLI("aic-check-", { id: it.id, label: it.label, description: it.description, installed: !!it.status }, { withDescription: true });
  if (it.base) built.li.className += " ml-8";
  const st = AIC_STATUS[it.status];
  if (st) {
    const badge = document.createElement("span");
    badge.className = "badge badge-sm ml-2 " + st.cls;
    badge.textContent = st.label;
    badge.title = st.title;
    built.li.children[1].children[0].appendChild(badge);
  }
  return built;
}

// aicPreviewMessage lists the changes of an Aplicar for the confirmation.
function aicPreviewMessage(folder, changes) {
  return t("aic.changeFiles", { path: folder }) + "\n\n" + aicChangeLines(changes, "") + "\n\n" + t("aic.backupContinue");
}

// aicRecordOf is the AIContextRecord of rec, or an unregistered folder
// without context when the backend sent none.
function aicRecordOf(rec) {
  if (!rec || Array.isArray(rec)) return { registered: false, hasContext: false, name: "", category: "", models: [], agents: [] };
  return rec;
}

// aicSameSet reports whether a and b hold the same strings.
function aicSameSet(a, b) {
  return a.length === b.length && a.every(function (x) { return b.indexOf(x) >= 0; });
}

// resetAicDefine empties the "Aplicar contexto" form.
function resetAicDefine() {
  aicItems.innerHTML = "";
  aicCustomItems.innerHTML = "";
  aicCustomSection.classList.add("hidden");
  renderAicAgents(aicAgentList, true);
  aicCategory.value = "";
  aicCategory.disabled = true;
  btnAicApply.disabled = true;
  btnAicApply.onclick = null;
  aicPreset.disabled = true;
  aicPreset.onchange = null;
  aicPreset.value = "";
  setAicPresetHint(null);
  aicName.value = "";
  aicName.disabled = true;
  aicUnregistered.classList.add("hidden");
}

function renderAicItems() {
  resetAicDefine();
  if (!aicFolder) {
    invalidate("aicontext");
    invalidate("aicontext-suggest");
    return;
  }
  const folder = aicFolder;
  Promise.all([App.GetAIContextRecord(folder), App.GetAIContextAgents(folder), App.GetAIContextItems(folder)])
    .then(latestOnly("aicontext", function (res) {
      renderAicDefine(folder, aicRecordOf(res[0]), res[1] || [], res[2] || []);
    }))
    .catch(function (err) { openError(t("aic.readFailed", { error: err })); });
}

// renderAicDefine fills the "Aplicar contexto" form for folder: rec is its
// record (a registered folder starts on its recorded name, agents and
// standards), agents and items the agents and standards of the screen;
// custom contexts go to their own section, shown only when there is one.
function renderAicDefine(folder, rec, agents, items) {
  aicName.disabled = false;
  aicName.value = rec.registered ? rec.name : "";
  aicCategory.disabled = false;
  aicCategory.value = rec.registered && aicCategories.some(function (c) { return c.name === rec.category; }) ? rec.category : "";
  aicUnregistered.classList.toggle("hidden", rec.registered || !rec.hasContext);
  if (!rec.registered && rec.hasContext) {
    aicUnregisteredDesc.textContent = t("aic.unregistered", { models: aicModelsText(rec.models), agents: aicAgentsText(rec.agents) });
  }

  renderAicAgents(rec.registered ? agents.map(function (a) {
    return { id: a.id, label: a.label, selected: rec.agents.indexOf(a.id) >= 0 };
  }) : agents);

  const checkboxes = {}; // it.id -> checkbox, captured at creation time
  items.forEach(function (it) {
    const built = buildAicItem(it);
    checkboxes[it.id] = built.checkbox;
    if (rec.registered) built.checkbox.checked = rec.models.indexOf(it.id) >= 0;
    built.checkbox.addEventListener("change", function () {
      // A complement needs its base.
      if (built.checkbox.checked && it.base && checkboxes[it.base]) checkboxes[it.base].checked = true;
      // A manual change no longer matches the chosen project type.
      if (it.custom) return;
      aicPreset.value = "";
      setAicPresetHint(null);
    });
    (it.custom ? aicCustomItems : aicItems).appendChild(built.li);
  });
  aicCustomSection.classList.toggle("hidden", !items.some(function (it) { return it.custom; }));

  aicPreset.disabled = aicPresets.length === 0;
  aicPreset.onchange = function () {
    const preset = aicPresets.find(function (p) { return p.name === aicPreset.value; });
    setAicPresetHint(preset);
    if (!preset) return;
    items.forEach(function (it) {
      if (!it.custom) checkboxes[it.id].checked = preset.models.indexOf(it.id) >= 0;
    });
  };

  // A folder with no standard yet starts on the project type its files
  // point to.
  if (!rec.registered && !items.some(function (it) { return it.status && !it.custom; })) {
    App.GetAIContextSuggestedPreset(folder).then(latestOnly("aicontext-suggest", function (name) {
      const preset = aicPresets.find(function (p) { return p.name === name; });
      if (!preset || folder !== aicFolder) return;
      aicPreset.value = name;
      aicPreset.onchange();
      aicPresetHint.textContent = t("aic.suggested") + (preset.hint ? " " + preset.hint : "");
      aicPresetHint.classList.remove("hidden");
    }));
  }

  btnAicApply.disabled = false;
  btnAicApply.onclick = function () {
    if (running) return;
    const name = aicName.value.trim();
    const nameErr = aicNameError(name, folder);
    if (nameErr) {
      openError(nameErr);
      return;
    }
    const selected = items
      .filter(function (it) { return checkboxes[it.id].checked; })
      .map(function (it) { return it.id; });
    const agentIDs = aicCheckedAgents();
    if (agentIDs.length === 0) {
      openError(t("aic.err.noAgent"));
      return;
    }
    const category = aicCategory.value;
    const apply = function () {
      beginRun(t("aic.applyTitle"), function () { return App.ApplyAIContext(folder, name, category, selected, agentIDs); });
    };
    App.PreviewAIContext(folder, selected, agentIDs).then(function (changes) {
      changes = changes || [];
      if (changes.length) {
        openConfirm(t("aic.applyTitle"), aicPreviewMessage(folder, changes), apply);
        return;
      }
      if (rec.registered && rec.name === name && (rec.category || "") === category &&
          aicSameSet(rec.models, selected) && aicSameSet(rec.agents, agentIDs)) {
        openMessage(t("aic.nothingTitle"), t("aic.nothingSelection", { path: folder }), false);
        return;
      }
      openConfirm(t("aic.applyTitle"), t("aic.recordOnly", { path: folder, name: name }), apply);
    }).catch(function (err) { openError(t("aic.previewFailed", { error: err })); });
  };
}

btnAicRegister.addEventListener("click", function () {
  if (!aicFolder) return;
  const folder = aicFolder;
  const name = aicName.value.trim();
  const nameErr = aicNameError(name, folder);
  if (nameErr) {
    openError(nameErr);
    return;
  }
  App.RegisterAIContext(folder, name, aicCategory.value).then(function () {
    openMessage(t("aic.registered"), t("aic.registeredText", { name: name }), false);
    loadAicRows();
    if (folder === aicFolder) renderAicItems();
  }).catch(function (err) { openError(t("aic.registerFailed", { error: err })); });
});

// ===========================================================================
// "Exportar/Importar" tab.
// ===========================================================================

// aicCheckRow builds one checklist row: a checkbox with a title and a
// detail line. A disabled row is unchecked. Returns { row, box, text }.
function aicCheckRow(title, detail, disabled) {
  const row = document.createElement("label");
  row.className = "flex items-start gap-3 py-1" + (disabled ? " opacity-60" : " cursor-pointer");
  const box = document.createElement("input");
  box.type = "checkbox";
  box.className = "checkbox checkbox-sm mt-1";
  box.checked = !disabled;
  box.disabled = !!disabled;
  row.appendChild(box);
  const text = document.createElement("div");
  text.className = "min-w-0 flex-1";
  const head = document.createElement("span");
  head.className = "font-medium";
  head.textContent = title;
  text.appendChild(head);
  if (detail) {
    const d = document.createElement("div");
    d.className = "text-xs opacity-70 break-all";
    d.textContent = detail;
    text.appendChild(d);
  }
  row.appendChild(text);
  return { row: row, box: box, text: text };
}

// ---- Exportar ---------------------------------------------------------------

// aicExportBadges adds to an Exportar row a labelled line of badges.
function aicExportBadges(parent, labelKey, texts) {
  const line = document.createElement("div");
  line.className = "flex flex-wrap items-center gap-1 mt-1 text-xs";
  const label = document.createElement("span");
  label.className = "opacity-70 mr-1";
  label.textContent = t(labelKey);
  line.appendChild(label);
  line.appendChild(aicBadges(texts));
  parent.appendChild(line);
}

// aicExportRows is the Exportar checklist: { path, box } per folder.
let aicExportRows = [];

function renderAicExport() {
  aicExportList.innerHTML = "";
  aicExportRows = aicContexts.map(function (c) {
    const built = aicCheckRow(c.name, c.path, false);
    aicExportBadges(built.text, "aic.exp.agents", aicAgentLabelList(c.agents));
    aicExportBadges(built.text, "aic.exp.contexts", aicModelLabelList(c.models));
    built.box.addEventListener("change", updateAicExport);
    aicExportList.appendChild(built.row);
    return { path: c.path, box: built.box };
  });
  aicExportEmpty.classList.toggle("hidden", aicContexts.length > 0);
  updateAicExport();
}

// aicExportPaths lists the folders checked on the Exportar checklist.
function aicExportPaths() {
  return aicExportRows.filter(function (r) { return r.box.checked; }).map(function (r) { return r.path; });
}

function updateAicExport() {
  btnAicExport.disabled = aicExportPaths().length === 0;
}

btnAicExport.addEventListener("click", function () {
  const paths = aicExportPaths();
  if (paths.length === 0) return;
  App.PickAIContextExportPath().then(function (path) {
    if (!path) return; // user cancelled the dialog
    return App.ExportAIContexts(path, paths).then(function () {
      openMessage(t("transfer.exported"), t("aic.exportedTo", { count: paths.length, path: path }), false);
    });
  }).catch(function (err) { openError(t("transfer.exportFailed", { error: err })); });
});

// ---- Importar ---------------------------------------------------------------


// AIC_IMPORT_STATE labels each import state (managerai.ImportContext.State).
const AIC_IMPORT_STATE = {
  get apply() { return t("aic.imp.apply"); },
  get pending() { return t("aic.imp.pending"); },
  get registered() { return t("aic.imp.registered"); },
  get invalid() { return t("aic.imp.invalid"); },
};

// aicImportFile is the file the Importar section shows; aicImportPaths the
// folders picked to replace the file's and aicImportNames the names typed,
// both by the folder's name in the file; aicImportPreview the last
// AIContextImportPreview and aicImportRows its checklist
// ({ source, box, input }).
let aicImportFile = "";
let aicImportPaths = {};
let aicImportNames = {};
let aicImportPreview = null;
let aicImportRows = [];

// aicImportPicks reads the checked rows into ImportPicks.
function aicImportPicks() {
  return aicImportRows.filter(function (r) { return r.box.checked; }).map(function (r) {
    return { source: r.source, name: r.input.value.trim(), path: aicImportPaths[r.source] || "" };
  });
}

// aicImportNotes tells what blocks importing picks: an empty name, a name
// in taken (the names registered here) or one given to two picks.
function aicImportNotes(picks, taken) {
  const notes = [];
  const lowerTaken = (taken || []).map(function (n) { return n.toLowerCase(); });
  const seen = {};
  picks.forEach(function (p) {
    const lower = p.name.toLowerCase();
    if (!p.name) notes.push(t("aic.imp.noName", { name: p.source }));
    else if (lowerTaken.indexOf(lower) >= 0) notes.push(t("aic.imp.nameTaken", { name: p.name }));
    else if (seen[lower]) notes.push(t("aic.imp.nameTwice", { name: p.name }));
    seen[lower] = true;
  });
  return notes;
}

function updateAicImport() {
  if (!aicImportPreview) {
    btnAicImport.disabled = true;
    return;
  }
  const picks = aicImportPicks();
  const notes = aicImportNotes(picks, aicImportPreview.taken);
  aicImportNote.classList.toggle("hidden", notes.length === 0);
  aicImportNoteText.textContent = notes.map(function (n) { return "• " + n; }).join("\n");
  btnAicImport.disabled = running || picks.length === 0 || notes.length > 0;
}

// buildAicImportRow builds the checklist row of one folder of the file
// (ImportContext): checkbox, editable name, folder with "Trocar pasta",
// state and what is left out. A folder that can't be imported is
// disabled, with the reason.
function buildAicImportRow(item) {
  const disabled = item.state !== "apply" && item.state !== "pending";
  let detail = item.path + " · " + aicModelsText(item.models) + " · " + aicAgentsText(item.agents);
  if (item.reason) detail += "\n" + t("transfer.unavailable", { reason: item.reason });
  if ((item.unknown || []).length) detail += "\n" + t("aic.imp.unknown", { names: item.unknown.join(", ") });
  const built = aicCheckRow(item.name, detail, disabled);
  built.text.children[1].className += " whitespace-pre-line";

  const badge = document.createElement("span");
  badge.className = "badge badge-sm badge-ghost ml-2";
  badge.textContent = AIC_IMPORT_STATE[item.state] || item.state;
  built.text.children[0].appendChild(badge);

  const controls = document.createElement("div");
  controls.className = "flex items-center gap-2 mt-1 flex-wrap";
  const input = document.createElement("input");
  input.type = "text";
  input.className = "input input-bordered input-sm max-w-xs";
  input.maxLength = 80;
  input.placeholder = t("aic.imp.namePlaceholder");
  input.value = aicImportNames[item.name] !== undefined ? aicImportNames[item.name] : item.name;
  input.disabled = disabled;
  input.addEventListener("input", function () {
    aicImportNames[item.name] = input.value;
    updateAicImport();
  });
  controls.appendChild(input);
  const change = document.createElement("button");
  change.className = "btn btn-xs";
  change.textContent = t("gt.changeFolder");
  change.addEventListener("click", function (e) {
    e.preventDefault();
    App.PickAIContextFolder().then(function (folder) {
      if (!folder) return;
      aicImportPaths[item.name] = folder;
      loadAicImport(aicImportFile);
    }).catch(function (err) { openError(t("gt.pickFailed", { error: err })); });
  });
  controls.appendChild(change);
  built.text.appendChild(controls);

  built.box.addEventListener("change", updateAicImport);
  return { row: built.row, entry: { source: item.name, box: built.box, input: input } };
}

function renderAicImport(preview) {
  aicImportList.innerHTML = "";
  aicImportRows = (preview.items || []).map(function (item) {
    const built = buildAicImportRow(item);
    aicImportList.appendChild(built.row);
    return built.entry;
  });
  updateAicImport();
}

// resetAicImport clears the Importar section back to "no file".
function resetAicImport() {
  invalidate("aic-import");
  aicImportFile = "";
  aicImportPaths = {};
  aicImportNames = {};
  aicImportPreview = null;
  aicImportRows = [];
  aicImportPathEl.textContent = t("transfer.noFile");
  aicImportPathEl.classList.add("opacity-60");
  aicImportList.innerHTML = "";
  aicImportNote.classList.add("hidden");
  btnAicImport.disabled = true;
}

// loadAicImport reads the export file at path, with the folders picked so
// far, and shows its folders.
function loadAicImport(path) {
  App.PreviewAIContextImport(path, aicImportPaths).then(latestOnly("aic-import", function (preview) {
    aicImportPreview = preview || { items: [], taken: [] };
    // Custom contexts carried by the file and not on this machine yet.
    Object.keys(aicImportPreview.labels || {}).forEach(function (id) {
      if (!aicModelLabels[id]) aicModelLabels[id] = aicImportPreview.labels[id];
    });
    aicImportFile = path;
    aicImportPathEl.textContent = path;
    aicImportPathEl.classList.remove("opacity-60");
    renderAicImport(aicImportPreview);
  })).catch(function (err) {
    resetAicImport();
    openError(t("transfer.readFailed", { error: err }));
  });
}

btnAicImportPick.addEventListener("click", function () {
  App.PickAIContextImportPath().then(function (path) {
    if (!path) return;
    aicImportPaths = {};
    aicImportNames = {};
    loadAicImport(path);
  }).catch(function (err) { openError(t("transfer.pickFailed", { error: err })); });
});

btnAicImport.addEventListener("click", function () {
  if (running || !aicImportPreview) return;
  const picks = aicImportPicks();
  if (picks.length === 0 || aicImportNotes(picks, aicImportPreview.taken).length) return;
  const file = aicImportFile;
  App.PreviewAIContextImportChanges(file, picks).then(function (previews) {
    const msg = t("aic.importWhat", { count: picks.length }) + "\n\n" +
      aicFolderChangesText(previews || [], null, t("aic.imp.pendingNote")) + "\n\n" + t("aic.backupContinue");
    openConfirm(t("aic.importTitle"), msg, function () {
      beginRun(t("aic.importTitle"), function () {
        return App.ImportAIContexts(file, picks);
      }, { buttons: [btnAicImport] });
    });
  }).catch(function (err) { openError(t("aic.previewFailed", { error: err })); });
});

// ===========================================================================

// renderAicDefineScreen shows Aplicar Contexto: the form without a
// folder, unless Alterar just chose one, and the registered folders.
function renderAicDefineScreen() {
  if (!aicKeepFolder) aicFolder = "";
  aicKeepFolder = false;
  renderAicPresetOptions();
  renderAicAlert();
  // The category select needs the categories before the folder's record
  // picks its value.
  loadAicCategories().then(renderAicItems);
  loadAicRows();
}

// renderAicTransferScreen shows Exportar/Importar.
function renderAicTransferScreen() {
  loadAicRows();
  resetAicImport();
}

// refreshAIContextScreen re-reads the list, the folder open on "Aplicar
// contexto" and the file shown on Importar after a run.
function refreshAIContextScreen() {
  loadAicRows();
  if (aicFolder) renderAicItems();
  if (aicImportFile) loadAicImport(aicImportFile);
}

// ===========================================================================
// "Gerenciar Contextos": the Perci82 contexts (read only) and the
// "Contextos personalizados" tab.
// ===========================================================================

// aicTextButton builds a small text button.
function aicTextButton(label, cls) {
  const btn = document.createElement("button");
  btn.type = "button";
  btn.className = "btn btn-xs" + (cls ? " " + cls : "");
  btn.textContent = label;
  return btn;
}

// aicShowText opens the read-only view of a context's text.
function aicShowText(title, text) {
  aicViewTitle.textContent = title;
  aicViewContent.textContent = text;
  aicViewModal.showModal();
}

// aicLibraryRow builds the row of one Perci82 context (AIContextItem).
function aicLibraryRow(it) {
  const tr = document.createElement("tr");
  const name = document.createElement("td");
  name.className = "font-medium" + (it.base ? " pl-8" : "");
  name.textContent = it.label;
  tr.appendChild(name);
  const desc = document.createElement("td");
  desc.className = "text-sm opacity-80";
  desc.textContent = it.description;
  tr.appendChild(desc);
  const actions = document.createElement("td");
  const view = aicTextButton(t("aic.lib.view"));
  view.addEventListener("click", function () {
    App.GetAIContextStandardText(it.id).then(function (text) {
      aicShowText(it.label, text);
    }).catch(function (err) { openError(t("aic.lib.readFailed", { error: err })); });
  });
  actions.appendChild(view);
  tr.appendChild(actions);
  return tr;
}

// aicCustomList is the custom contexts last loaded (CustomContext).
let aicCustomList = [];

// aicCustomRow builds the row of one custom context, with Visualizar,
// Editar and Apagar.
function aicCustomRow(c) {
  const tr = document.createElement("tr");
  const name = document.createElement("td");
  name.className = "font-medium";
  name.textContent = c.name;
  tr.appendChild(name);
  const applies = document.createElement("td");
  applies.className = "text-sm opacity-80";
  applies.textContent = c.appliesTo || "—";
  tr.appendChild(applies);
  const actions = document.createElement("td");
  const wrap = document.createElement("div");
  wrap.className = "flex gap-1";
  const view = aicTextButton(t("aic.lib.view"));
  view.addEventListener("click", function () { aicShowText(c.name, c.content); });
  wrap.appendChild(view);
  const edit = aicTextButton(t("aic.lib.edit"));
  edit.addEventListener("click", function () { aicOpenCustom(c); });
  wrap.appendChild(edit);
  const del = aicTextButton(t("aic.lib.delete"), "btn-error btn-outline");
  del.addEventListener("click", function () { aicDeleteCustom(c); });
  wrap.appendChild(del);
  actions.appendChild(wrap);
  tr.appendChild(actions);
  return tr;
}

function loadAicLibrary() {
  App.GetAIContextItems("").then(latestOnly("aic-lib", function (items) {
    aicLibRows.innerHTML = "";
    (items || []).filter(function (it) { return !it.custom; }).forEach(function (it) {
      aicLibRows.appendChild(aicLibraryRow(it));
    });
  })).catch(function (err) { openError(t("aic.lib.listFailed", { error: err })); });
}

function loadAicCustom() {
  App.GetCustomContexts().then(latestOnly("aic-custom", function (list) {
    aicCustomList = list || [];
    aicCustomRows.innerHTML = "";
    aicCustomList.forEach(function (c) { aicCustomRows.appendChild(aicCustomRow(c)); });
    aicCustomEmpty.classList.toggle("hidden", aicCustomList.length > 0);
  })).catch(function (err) { openError(t("aic.lib.listFailed", { error: err })); });
}

// aicCustomEditing is the ID of the custom context the modal edits, ""
// for a new one.
let aicCustomEditing = "";

// aicOpenCustom opens the custom context modal on c, or empty for a new
// context when c is null.
function aicOpenCustom(c) {
  aicCustomEditing = c ? c.id : "";
  aicCustomTitle.textContent = c ? t("aic.lib.editTitle", { name: c.name }) : t("aic.lib.addTitle");
  aicCustomName.value = c ? c.name : "";
  aicCustomApplies.value = c ? c.appliesTo : "";
  aicCustomContent.value = c ? c.content : "";
  btnAicCustomSave.disabled = false;
  aicCustomModal.showModal();
  aicCustomName.focus();
}

btnAicCustomAdd.addEventListener("click", function () { aicOpenCustom(null); });

btnAicCustomSave.addEventListener("click", function () {
  const editing = aicCustomEditing;
  const name = aicCustomName.value.trim();
  if (!name) {
    openError(t("aic.err.noName"));
    return;
  }
  if (!aicCustomContent.value.trim()) {
    openError(t("aic.lib.noContent"));
    return;
  }
  btnAicCustomSave.disabled = true;
  App.SaveCustomContext({ id: editing, name: name, appliesTo: aicCustomApplies.value, content: aicCustomContent.value })
    .then(function () {
      aicCustomModal.close();
      loadAicCustom();
      aicLoadModelLabels();
      if (editing) openMessage(t("aic.lib.saved"), t("aic.lib.savedEdit"), false);
    })
    .catch(function (err) { openError(t("aic.lib.saveFailed", { error: err })); })
    .finally(function () { btnAicCustomSave.disabled = false; });
});

// aicDeleteCustom asks for confirmation and deletes c; the backend
// refuses a context a registered folder uses.
function aicDeleteCustom(c) {
  openConfirm(t("aic.lib.deleteTitle"), t("aic.lib.deleteConfirm", { name: c.name }), function () {
    App.DeleteCustomContext(c.id).then(function () {
      loadAicCustom();
    }).catch(function (err) { openError(t("aic.lib.deleteFailed", { error: err })); });
  });
}

// renderAicLibraryScreen shows Gerenciar Contextos.
function renderAicLibraryScreen() {
  loadAicLibrary();
  loadAicCustom();
}
