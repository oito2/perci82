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

// Skills & MCPs :: SKILLs.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// ===========================================================================
// "Skills & MCPs :: SKILLs" (npx skills).
// ===========================================================================

asSkillsShLink.addEventListener("click", function (ev) {
  ev.preventDefault();
  Browser.OpenURL("https://www.skills.sh/");
});

// Skills are written by third parties and run with the agent's permissions:
// a note after the intro says so.
(function addThirdPartyNote() {
  const intro = asSkillsShLink.parentNode;
  if (!intro || !intro.parentNode) return;
  const note = document.createElement("p");
  note.className = "text-sm opacity-80 mb-4";
  note.setAttribute("data-i18n", "skills.thirdParty");
  note.textContent = t("skills.thirdParty");
  intro.parentNode.insertBefore(note, intro.nextSibling);
})();

// partialBadge is the "parcial: Claude, Codex" tag of a row whose record
// says some agent did not accept it (the names are the ones that did), or
// null for any other row.
function partialBadge(row) {
  if (!row.partial) return null;
  const badge = document.createElement("span");
  badge.className = "badge badge-warning badge-sm ml-2";
  badge.textContent = t("badge.partial", { agents: (row.agents || []).join(", ") });
  return badge;
}

// retiredBadge is the "fora do catálogo" tag of an entry that left the
// catalog and is listed only so it can be removed, or null for any other row.
function retiredBadge(row) {
  if (!row.retired) return null;
  const badge = document.createElement("span");
  badge.className = "badge badge-ghost badge-sm ml-2";
  badge.textContent = t("badge.retired");
  return badge;
}

const asTable = createScopedTable({
  els: { select: asScopeSelect, alertTitle: asScopeAlertTitle, alertDesc: asScopeAlertDesc, pickButton: btnAsPickFolder, rows: asRows },
  globalDesc: "skills.globalDesc",
  localDesc: "skills.localDesc",
  latestKey: "agentskills",
  pickFolder: function () { return App.PickAgentSkillsFolder(); },
  fetchRows: function (global, folder) { return App.GetAgentSkillsRows(global, folder); },
  keyOf: function (row) { return row.slug; },
  sigOf: function (row) { return [row.slug, row.name, row.installed, row.partial, row.retired, (row.agents || []).join(",")].join("|"); },
  buildRow: function (row) { return buildAgentSkillRow(row); },
});

function renderAgentSkillsTable() { asTable.render(); }

function buildAgentSkillRow(row) {
  const tr = document.createElement("tr");

  const tdName = document.createElement("td");
  tdName.textContent = row.name;
  const badge = partialBadge(row);
  if (badge) tdName.appendChild(badge);
  const retired = retiredBadge(row);
  if (retired) tdName.appendChild(retired);
  tr.appendChild(tdName);

  const tdActions = document.createElement("td");
  const wrap = document.createElement("div");
  wrap.className = "flex gap-2";

  const btnToggle = document.createElement("button");
  btnToggle.type = "button";
  btnToggle.className = "btn btn-sm " + (row.installed ? "btn-outline btn-error" : "btn-primary");
  btnToggle.textContent = t(row.installed ? "action.remove" : "action.install");

  const btnUpdate = document.createElement("button");
  btnUpdate.type = "button";
  btnUpdate.className = "btn btn-sm btn-outline";
  btnUpdate.textContent = t("action.update");
  btnUpdate.disabled = !row.installed;

  btnToggle.addEventListener("click", function () {
    if (running) return;
    const global = asTable.isGlobal();
    if (row.installed) {
      beginRun(t("skills.run.remove", { name: row.name }), function () { return App.RemoveAgentSkill(row.slug, global, asTable.folder); });
    } else {
      beginRun(t("skills.run.install", { name: row.name }), function () { return App.InstallAgentSkill(row.slug, global, asTable.folder); });
    }
  });
  btnUpdate.addEventListener("click", function () {
    if (running || btnUpdate.disabled) return;
    beginRun(t("skills.run.update", { name: row.name }), function () { return App.UpdateAgentSkill(row.slug, asTable.isGlobal(), asTable.folder); });
  });

  wrap.appendChild(btnToggle);
  // A retired entry can only be removed.
  if (!row.retired) wrap.appendChild(btnUpdate);
  tdActions.appendChild(wrap);
  tr.appendChild(tdActions);
  return tr;
}

function renderAgentSkillsScreen() { asTable.reset(); }
