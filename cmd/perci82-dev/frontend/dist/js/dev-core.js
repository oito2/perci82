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

// Perci82 Dev plumbing: the DOM references of its screens and the tray's
// compact window.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

const fileappContainer = document.getElementById("fileapp-container");
const fileappDesc = document.getElementById("fileapp-desc");
const btnFileappPick = document.getElementById("btn-fileapp-pick");
const fileappPath = document.getElementById("fileapp-path");
const btnFileappInstall = document.getElementById("btn-fileapp-install");
const btnFileappUpdate = document.getElementById("btn-fileapp-update");
const btnFileappUninstall = document.getElementById("btn-fileapp-uninstall");
const fileappNotes = document.getElementById("fileapp-notes");
const dockercreateContainer = document.getElementById("dockercreate-container");
const dcKind = document.getElementById("dc-kind");
const dcAppTypeWrap = document.getElementById("dc-apptype-wrap");
const dcAppType = document.getElementById("dc-apptype");
const dcFields = document.getElementById("dc-fields");
const dcMkcertAlert = document.getElementById("dc-mkcert-alert");
const btnDcCreate = document.getElementById("btn-dc-create");
const dcHint = document.getElementById("dc-hint");
const dockerlistContainer = document.getElementById("dockerlist-container");
const dockerlistRows = document.getElementById("dockerlist-rows");
const dockerlistEmpty = document.getElementById("dockerlist-empty");
const dockerlistToolbar = document.getElementById("dockerlist-toolbar");
const btnDockerlistUpdateAll = document.getElementById("btn-dockerlist-update-all");
const transferContainer = document.getElementById("transfer-container");
const transferExportList = document.getElementById("transfer-export-list");
const transferExportEmpty = document.getElementById("transfer-export-empty");
const transferExportNote = document.getElementById("transfer-export-note");
const transferExportNoteText = document.getElementById("transfer-export-note-text");
const btnTransferExport = document.getElementById("btn-transfer-export");
const tabLabelTransferImport = document.getElementById("tab-label-transfer-import");
const btnTransferPick = document.getElementById("btn-transfer-pick");
const transferImportPathEl = document.getElementById("transfer-import-path");
const transferImportWorkspace = document.getElementById("transfer-import-workspace");
const transferImportList = document.getElementById("transfer-import-list");
const transferImportNote = document.getElementById("transfer-import-note");
const transferImportNoteText = document.getElementById("transfer-import-note-text");
const btnTransferImport = document.getElementById("btn-transfer-import");
const logsModal = document.getElementById("logs-modal");
const logsModalTitle = document.getElementById("logs-modal-title");
const logsModalContent = document.getElementById("logs-modal-content");
const btnLogsRefresh = document.getElementById("btn-logs-refresh");
const btnLogsExport = document.getElementById("btn-logs-export");
const editModal = document.getElementById("edit-modal");
const editModalTitle = document.getElementById("edit-modal-title");
const editFields = document.getElementById("edit-fields");
const btnEditSave = document.getElementById("btn-edit-save");
const moodlepluginsContainer = document.getElementById("moodleplugins-container");
const mpDevPath = document.getElementById("mp-dev-path");
const btnMpPickDev = document.getElementById("btn-mp-pick-dev");
const mpApp = document.getElementById("mp-app");
const mpType = document.getElementById("mp-type");
const mpName = document.getElementById("mp-name");
const mpLocked = document.getElementById("mp-locked");
const mpStatus = document.getElementById("mp-status");
const mpStatusText = document.getElementById("mp-status-text");
const btnMpCreate = document.getElementById("btn-mp-create");
const tabLabelMpMounted = document.getElementById("tab-label-mp-mounted");
const mpRows = document.getElementById("mp-rows");
const mpEmpty = document.getElementById("mp-empty");
const agentskillsContainer = document.getElementById("agentskills-container");
const itemTableContainer = document.getElementById("itemtable-container");
const itemTableIntro = document.getElementById("itemtable-intro");
const itemTableHeader = document.getElementById("itemtable-header");
const itemTableRows = document.getElementById("itemtable-rows");
const asScopeSelect = document.getElementById("as-scope");
const asScopeAlertTitle = document.getElementById("as-scope-alert-title");
const asScopeAlertDesc = document.getElementById("as-scope-alert-desc");
const btnAsPickFolder = document.getElementById("btn-as-pick-folder");
const asRows = document.getElementById("as-rows");
const asSkillsShLink = document.getElementById("as-skills-sh-link");
const aicontextContainer = document.getElementById("aicontext-container");
const aicAlertTitle = document.getElementById("aic-alert-title");
const aicAlertDesc = document.getElementById("aic-alert-desc");
const btnAicPickFolder = document.getElementById("btn-aic-pick-folder");
const aicItems = document.getElementById("aic-items");
const aicPreset = document.getElementById("aic-preset");
const aicPresetHint = document.getElementById("aic-preset-hint");
const aicAgents = document.getElementById("aic-agents");
const btnAicApply = document.getElementById("btn-aic-apply");
const mcpserversContainer = document.getElementById("mcpservers-container");
const mcpScopeSelect = document.getElementById("mcp-scope");
const mcpScopeAlertTitle = document.getElementById("mcp-scope-alert-title");
const mcpScopeAlertDesc = document.getElementById("mcp-scope-alert-desc");
const btnMcpPickFolder = document.getElementById("btn-mcp-pick-folder");
const mcpCodexNote = document.getElementById("mcp-codex-note");
const mcpRows = document.getElementById("mcp-rows");
const dashboardDockerSummary = document.getElementById("dashboard-docker-summary");
const su2BackupConfig = document.getElementById("su2-backup-config");
const su2BackupRepos = document.getElementById("su2-backup-repos");
const su2BackupAI = document.getElementById("su2-backup-ai");
const su2RemoveDocker = document.getElementById("su2-remove-docker");
const scWorkspaceAlertTitle = document.getElementById("sc-workspace-alert-title");
const scWorkspaceAlertDesc = document.getElementById("sc-workspace-alert-desc");
const btnScPickWorkspace = document.getElementById("btn-sc-pick-workspace");
const btnScCreateWorkspace = document.getElementById("btn-sc-create-workspace");
const scTrayEnabled = document.getElementById("sc-tray-enabled");

// --- compact window (system tray: a single window with tabs) -----------
// Opened with ?compact=<tab>. The section tabs, side list and screen title
// are hidden and the compact header ("Perci82 Dev" + "Abrir Aplicativo")
// and its tab bar are shown instead; the rest of the screen (inner tabs,
// containers, render*Screen) is shared with the main window — the compact
// tabs just pick, via selectItem, WHICH item to show. The
// "compact-switch-tab" event (see Events.On below) switches the tab of an
// already open window.
const compactHeader = document.getElementById("compact-header");
const btnCompactOpenApp = document.getElementById("btn-compact-open-app");
const compactTabsEl = document.getElementById("compact-tabs");
const compactTabRadios = {
  "docker-manage": document.getElementById("compact-tab-docker-manage"),
};
let compactSections = null; // catalog (App.GetSections()) cached to switch tabs without reloading

if (compactAction) {
  document.getElementById("section-tabs").classList.add("hidden");
  document.getElementById("side-list").classList.add("hidden");
  document.getElementById("screen-title").classList.add("hidden");
  compactHeader.classList.remove("hidden");
  compactHeader.classList.add("flex");
  compactTabsEl.classList.remove("hidden");
  // #dashboard-container is visible by default (no class="hidden" in the
  // HTML) — hidden right here, even before App.GetSections() resolves, so
  // the full dashboard doesn't flash in a compact window that's about to
  // show something else.
  document.getElementById("dashboard-container").classList.add("hidden");
}

btnCompactOpenApp.addEventListener("click", function () {
  App.FocusMainWindow();
});

// Finds, in the catalog already loaded by App.GetSections(), the item
// that corresponds to a tab — by actionId (the compact window's tab uses
// the item's actionId as its key: "docker-manage").
function findCompactTarget(sections, tab) {
  for (let sectionIdx = 0; sectionIdx < sections.length; sectionIdx++) {
    const section = sections[sectionIdx];
    for (let itemIdx = 0; itemIdx < section.items.length; itemIdx++) {
      const item = section.items[itemIdx];
      if (item.actionId === tab) {
        return { section: section, item: item, sectionIdx: sectionIdx, itemIdx: itemIdx };
      }
    }
  }
  return null;
}

// Switches the compact window's active tab — checks the right radio and
// calls selectItem to build the content, both on initial load
// (?compact=<tab>) and when the user clicks a tab, and when the
// "compact-switch-tab" event arrives (window already open, another tray
// item clicked).
function switchCompactTab(tab) {
  if (!compactSections) return;
  const target = findCompactTarget(compactSections, tab);
  if (!target) return;
  const radio = compactTabRadios[tab];
  if (radio) radio.checked = true;
  selectItem(target.section, target.item, target.sectionIdx, target.itemIdx);
}

Object.keys(compactTabRadios).forEach(function (tab) {
  compactTabRadios[tab].addEventListener("change", function () {
    if (compactTabRadios[tab].checked) switchCompactTab(tab);
  });
});

Events.On("compact-switch-tab", function (ev) {
  switchCompactTab(ev.data);
});

// Activates the requested compact window — called after App.GetSections()
// resolves (selectItem depends on the loaded catalog).
function activateCompactMode(sections) {
  compactSections = sections;
  switchCompactTab(compactAction);
}
