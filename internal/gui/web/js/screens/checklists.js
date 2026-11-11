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

// Shared "checklist simples" screens (installed ⇄ unchecked): the
// MULTISELECT_SCREENS registry each application fills in, and its renderer.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// --- "checklist simples" screens (installed ⇄ unchecked, no dependency
// between items) — MULTISELECT_SCREENS maps actionId to the two bound
// methods the screen needs: fetch() brings the items with their current
// state (installed), run(ids) executes whatever got checked. Each
// application adds one entry per screen of this type. -----------------------
const MULTISELECT_SCREENS = {};

// buildChecklistItemLI builds the <li> (checkbox + label [+ description])
// shared by the "checklist simples" variants (multiselect/aicontext),
// instead of each one hand-rolling the same DOM construction. Returns
// {li, checkbox} — callers keep the checkbox reference from creation time
// instead of re-querying it later by id (avoids repeating
// getElementById("prefix-" + id) in loops/filters for an element created
// a few lines earlier in the very same pass).
//   idPrefix     — e.g. "ms-check-"/"aic-check-"
//   item         — { id, label, description?, installed? }
//   opts.withDescription — wraps label (+ description, when present) in a <div>
function buildChecklistItemLI(idPrefix, item, opts) {
  opts = opts || {};
  const li = document.createElement("li");
  li.className = "flex items-start gap-3";

  const cb = document.createElement("input");
  cb.type = "checkbox";
  cb.className = "checkbox checkbox-sm mt-1";
  cb.id = idPrefix + item.id;
  cb.checked = !!item.installed;

  const label = document.createElement("label");
  label.setAttribute("for", cb.id);
  label.className = "font-medium cursor-pointer";
  label.textContent = item.label;

  li.appendChild(cb);
  if (opts.withDescription) {
    const textWrap = document.createElement("div");
    textWrap.appendChild(label);
    if (item.description) {
      const desc = document.createElement("p");
      desc.className = "text-sm opacity-70";
      desc.textContent = item.description;
      textWrap.appendChild(desc);
    }
    li.appendChild(textWrap);
  } else {
    li.appendChild(label);
  }
  return { li: li, checkbox: cb };
}

// multiselectButtons lists every button of the current checklist screen —
// disabled together while one of its runs is in progress.
function multiselectButtons() {
  return [btnExecutarMultiselect].concat(multiselectExtraButtons);
}

function renderMultiselectScreen(item) {
  multiselectIntro.textContent = item.desc || "";
  multiselectItemsList.innerHTML = "";

  const cfg = MULTISELECT_SCREENS[item.actionId];

  multiselectExtras.innerHTML = "";
  multiselectExtraButtons = (cfg.extras || []).map(function (extra) {
    const btn = document.createElement("button");
    btn.className = "btn btn-outline";
    btn.textContent = extra.label;
    btn.onclick = function () {
      if (running) return;
      const run = function () {
        beginRun(extra.label, extra.run, { buttons: multiselectButtons() });
      };
      if (extra.confirm) {
        openConfirm(extra.label, extra.confirm, run);
        return;
      }
      run();
    };
    multiselectExtras.appendChild(btn);
    return btn;
  });

  // The previous screen's handler must not stay armed while this screen's
  // items load.
  btnExecutarMultiselect.disabled = true;
  btnExecutarMultiselect.onclick = null;
  cfg.fetch().then(latestOnly("multiselect", function (items) {
    items = items || [];
    const checkboxes = {}; // it.id -> checkbox, captured at creation time
    items.forEach(function (it) {
      const built = buildChecklistItemLI("ms-check-", it, { withDescription: true });
      checkboxes[it.id] = built.checkbox;
      multiselectItemsList.appendChild(built.li);
    });

    btnExecutarMultiselect.disabled = running;
    btnExecutarMultiselect.onclick = function () {
      if (running) return;
      const selected = items
        .filter(function (it) { return checkboxes[it.id].checked; })
        .map(function (it) { return it.id; });
      const run = function () {
        beginRun(item.title, function () { return cfg.run(selected); }, { buttons: multiselectButtons() });
      };

      // Items whose removal takes more than the item itself (e.g. Node.js
      // deletes ~/.nvm) are confirmed before anything runs.
      const warnings = items
        .filter(function (it) { return it.installed && it.removeWarning && !checkboxes[it.id].checked; })
        .map(function (it) { return it.removeWarning; });
      if (warnings.length) {
        openConfirm("Confirmar remoção", warnings.join("\n\n"), run);
        return;
      }
      run();
    };
  }));
}

SCREEN_FAMILIES.push({
  has: function (id) { return !!MULTISELECT_SCREENS[id]; },
  el: multiselectContainer,
  render: renderMultiselectScreen,
  done: function () {
    btnExecutarMultiselect.disabled = false;
    multiselectExtraButtons.forEach(function (btn) { btn.disabled = false; });
  },
});
