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

// The window shell: the section tab bar, the side list of the active
// section's items, the screen title, and selectItem (which screen shows
// for each item).
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

const sectionTabsEl = document.getElementById("section-tabs");
const sideListEl = document.getElementById("side-list");
const screenTitleEl = document.getElementById("screen-title");

// menuKey identifies an item across catalog reloads: its actionId plus,
// for an account item, the account ID.
function menuKey(item) {
  return item.actionId + "|" + (item.accountId || "");
}

// The sections the window shows now (App.GetSections()) and the index of
// the active one.
let menuSections = [];
let currentSectionIdx = 0;

// sectionIcon returns the SVG of a lucide icon, or "" when the vendored
// set lacks it.
function sectionIcon(name) {
  try {
    return name ? LUCIDE.svg(name, { size: 24 }) : "";
  } catch (err) {
    return "";
  }
}

// renderSectionTabs (re)builds the top tab bar from menuSections.
function renderSectionTabs() {
  sectionTabsEl.innerHTML = "";
  menuSections.forEach(function (section, idx) {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "o2-tab";
    btn.setAttribute("role", "tab");
    btn.dataset.section = section.id;
    btn.innerHTML = sectionIcon(section.icon);
    const label = document.createElement("span");
    label.textContent = section.label;
    btn.appendChild(label);
    btn.addEventListener("click", function () { selectSection(idx); });
    sectionTabsEl.appendChild(btn);
  });
}

// renderSideList (re)builds the side list of the active section: one
// entry per item, with a group heading where an item's group differs from
// the previous one's. A section with a single item hides the list.
function renderSideList() {
  const section = menuSections[currentSectionIdx];
  sideListEl.innerHTML = "";
  if (!section || section.items.length < 2) {
    sideListEl.classList.add("hidden");
    return;
  }
  sideListEl.classList.remove("hidden");
  let group = "";
  section.items.forEach(function (item, itemIdx) {
    if (item.group && item.group !== group) {
      const heading = document.createElement("div");
      heading.className = "o2-side-group";
      heading.textContent = item.group;
      sideListEl.appendChild(heading);
    }
    group = item.group || "";
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "o2-side-item";
    btn.textContent = item.title;
    btn.title = item.title;
    btn.dataset.key = menuKey(item);
    btn.addEventListener("click", function () {
      selectItem(section, item, currentSectionIdx, itemIdx);
    });
    sideListEl.appendChild(btn);
  });
}

// loadMenu (re)builds the tab bar and side list from App.GetSections() —
// on start, on a language switch and whenever the catalog changes (a Git
// account added, renamed or removed). A rebuild keeps the active section
// and item, and relabels the screen title and first inner tab.
function loadMenu() {
  return App.GetSections().then(function (sections) {
    const activeID = menuSections[currentSectionIdx] && menuSections[currentSectionIdx].id;
    menuSections = sections || [];
    const idx = menuSections.findIndex(function (s) { return s.id === activeID; });
    currentSectionIdx = idx >= 0 ? idx : 0;
    renderSectionTabs();
    renderSideList();
    const current = window.__selectedItem && findMenuItem(menuKey(window.__selectedItem));
    if (current) {
      window.__selectedItem = current.item;
      showItemLabels(current.item);
    }
    markActiveMenuItem();
    return menuSections;
  });
}

// findMenuItem returns { section, item, sectionIdx, itemIdx } of the item
// whose menuKey is key, or null.
function findMenuItem(key) {
  for (let sectionIdx = 0; sectionIdx < menuSections.length; sectionIdx++) {
    const section = menuSections[sectionIdx];
    for (let itemIdx = 0; itemIdx < section.items.length; itemIdx++) {
      if (menuKey(section.items[itemIdx]) === key) {
        return { section: section, item: section.items[itemIdx], sectionIdx: sectionIdx, itemIdx: itemIdx };
      }
    }
  }
  return null;
}

// selectSection shows the section at idx, on its first item.
function selectSection(idx) {
  const section = menuSections[idx];
  if (!section || !section.items.length) return;
  selectItem(section, section.items[0], idx, 0);
}

// The main window opens on the first section once the catalog and the
// saved language are loaded; the compact window shows its own tab.
Promise.all([loadMenu(), i18nReady]).then(function (r) {
  if (compactAction) {
    activateCompactMode(r[0]);
    return;
  }
  selectSection(0);
});

// A language switch relabels the tabs, the side list and the titles, then
// renders the open screen again so the texts built in Go come in the new
// language (skipped while an action runs, to keep its screen as it is);
// the first locale shown at startup needs no reload (GetSections already
// labels in the saved language).
let menuLocale = null;
onLocaleChange(function (locale) {
  if (menuLocale !== null && locale !== menuLocale) {
    loadMenu().then(function () {
      const item = window.__selectedItem;
      const screen = item && screenFor(item);
      if (screen && !running) screen.render(item);
    });
  }
  menuLocale = locale;
});

// markActiveMenuItem marks the active section tab and side list entry.
function markActiveMenuItem() {
  const key = window.__selectedItem ? menuKey(window.__selectedItem) : "";
  Array.prototype.forEach.call(sectionTabsEl.children, function (el, idx) {
    el.setAttribute("aria-selected", idx === currentSectionIdx ? "true" : "false");
  });
  sideListEl.querySelectorAll("button[data-key]").forEach(function (el) {
    if (el.dataset.key === key) el.setAttribute("aria-current", "page");
    else el.removeAttribute("aria-current");
  });
}

// selectMenuItem shows the item whose menuKey is key, as a click would.
// Returns false when the current catalog has no such item.
function selectMenuItem(key) {
  const found = findMenuItem(key);
  if (!found) return false;
  selectItem(found.section, found.item, found.sectionIdx, found.itemIdx);
  return true;
}

// screenFor returns the screen an item shows: its SCREENS entry, else the
// first SCREEN_FAMILIES entry that has its actionId, else null.
function screenFor(item) {
  const id = item.actionId;
  if (SCREENS[id]) return SCREENS[id];
  for (let i = 0; i < SCREEN_FAMILIES.length; i++) {
    if (SCREEN_FAMILIES[i].has(id)) return SCREEN_FAMILIES[i];
  }
  return null;
}

// screenContainers lists every screen container plus the extra inner tab
// labels only some screens show — all hidden before the selected screen
// shows its own.
function screenContainers() {
  const els = [];
  Object.keys(SCREENS).forEach(function (k) {
    els.push(SCREENS[k].el);
    (SCREENS[k].alsoShow || []).forEach(function (el) { els.push(el); });
  });
  SCREEN_FAMILIES.forEach(function (f) { els.push(f.el); });
  return els;
}

// showItemLabels writes the item's title and first inner tab label.
function showItemLabels(item) {
  screenTitleEl.textContent = item.title;
  tabDescLabel.textContent = item.tab || item.title;
}

function selectItem(section, item, sectionIdx, itemIdx) {
  const sectionChanged = sectionIdx !== currentSectionIdx;
  currentSectionIdx = sectionIdx;
  if (sectionChanged) renderSideList();
  showItemLabels(item);
  window.__selectedItem = item;

  screenContainers().forEach(function (el) { el.classList.add("hidden"); });
  // The first inner tab and "Execução" are visible by default; a screen
  // whose actions all apply at once (noTerminal) shows no inner tabs.
  tabLabelDesc.classList.remove("hidden");
  tabLabelTerminal.classList.remove("hidden");

  const screen = screenFor(item);
  if (!screen) {
    openError(t("shell.noScreen", { id: item.actionId }));
    return;
  }
  screen.el.classList.remove("hidden");
  (screen.alsoShow || []).forEach(function (el) { el.classList.remove("hidden"); });
  if (screen.noTerminal) {
    tabLabelDesc.classList.add("hidden");
    tabLabelTerminal.classList.add("hidden");
  }
  screen.render(item);
  selectTab("desc");

  markActiveMenuItem();
}
