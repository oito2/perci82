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

// Sidebar accordion + collapsed rail flyout, and selectItem (which screen shows for each menu item).
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// --- accordion (expanded) + rail flyout (collapsed) ------------------------
// Each category has 2 representations in the DOM, one visible at a time
// (CSS, based on #sidebar[data-collapsed]): the normal one (accordion,
// inline items) and the rail one (just the category icon, items in a
// flyout dropdown beside it).
function catInitials(label) {
  return label.split(" ").map(function (w) { return w[0]; }).join("").slice(0, 2).toUpperCase();
}

// Rail icon per category — a category missing
// here falls back to its initials.
const CATEGORY_ICONS = {
  Home: "house",
  Linux: "terminal",
  Desenvolvimento: "code",
  Docker: "container",
  "Repositórios": "gitBranch",
  "Dev Tools": "wrench",
};

// menuKey identifies a menu item across menu reloads: its actionId plus,
// for an account item, the account ID.
function menuKey(item) {
  return item.actionId + "|" + (item.accountId || "");
}

// Builds the <ul class="menu"> of one category's items — reused both by
// the accordion and the rail dropdown, so the two views always stay in
// sync (same content, same click).
function buildItemsMenu(cat, catIdx, fullClass) {
  const ul = document.createElement("ul");
  ul.className = fullClass;
  cat.items.forEach(function (item, itemIdx) {
    const li = document.createElement("li");
    li.className = "w-full";
    const a = document.createElement("a");
    a.className = "w-full";
    a.href = "#"; // focusable and activated by Enter, like any link
    a.textContent = item.title;
    a.dataset.cat = catIdx;
    a.dataset.item = itemIdx;
    a.dataset.key = menuKey(item);
    a.addEventListener("click", function (ev) {
      ev.preventDefault();
      selectItem(cat, item, catIdx, itemIdx);
    });
    li.appendChild(a);
    ul.appendChild(li);
  });
  return ul;
}

// The catalog the sidebar shows now (App.GetCategories()).
let menuCats = [];

// loadMenu (re)builds the sidebar from App.GetCategories() — on start and
// whenever the catalog changes (a Git account added, renamed or removed).
// A rebuild keeps the categories that were open and the active item.
function loadMenu() {
  return App.GetCategories().then(function (cats) {
    const wasOpen = {};
    document.querySelectorAll("#accordion .accordion-view > input[type=checkbox]").forEach(function (cb) {
      wasOpen[cb.getAttribute("aria-label")] = cb.checked;
    });
    const firstBuild = menuCats.length === 0;
    menuCats = cats;
    accordionEl.innerHTML = "";
    buildMenu(cats, function (cat, catIdx) {
      return firstBuild ? catIdx === 0 : !!wasOpen[cat.label];
    });
    markActiveMenuItem();
    return cats;
  });
}

// buildMenu appends one accordion entry and one rail entry per category to
// #accordion; isOpen(cat, catIdx) says which accordion entries start open.
function buildMenu(cats, isOpen) {
  cats.forEach(function (cat, catIdx) {
    // --- normal view: accordion, checkbox (not radio) so every
    // category can be closed, not just swapped for another. No border
    // between categories — just daisyUI's collapse-arrow, no
    // border-b/border-base-300.
    const accWrap = document.createElement("div");
    accWrap.className = "accordion-view collapse collapse-arrow";

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.setAttribute("aria-label", cat.label);
    if (isOpen(cat, catIdx)) checkbox.checked = true;
    checkbox.addEventListener("change", function () {
      if (!checkbox.checked) return;
      document.querySelectorAll("#accordion .accordion-view > input[type=checkbox]").forEach(function (other) {
        if (other !== checkbox) other.checked = false;
      });
    });

    const accTitle = document.createElement("div");
    accTitle.className = "collapse-title font-medium";
    accTitle.textContent = cat.label;

    const accContent = document.createElement("div");
    accContent.className = "collapse-content";
    accContent.appendChild(buildItemsMenu(cat, catIdx, "menu menu-md w-full p-0"));

    accWrap.appendChild(checkbox);
    accWrap.appendChild(accTitle);
    accWrap.appendChild(accContent);

    // --- rail view: icon + flyout with the items (custom CSS, not
    // daisyUI's `dropdown` component).
    const railWrap = document.createElement("div");
    railWrap.className = "rail-view justify-center p-2";

    const railItem = document.createElement("div");
    railItem.className = "rail-item";
    railItem.tabIndex = 0;
    railItem.setAttribute("aria-label", cat.label);

    const railTrigger = document.createElement("div");
    railTrigger.title = cat.label;
    railTrigger.className = "cat-badge cursor-pointer";
    railTrigger.setAttribute("aria-hidden", "true"); // railItem carries the label
    // An icon missing from the vendored set renders nothing (or throws):
    // the initials stand in for it.
    const icon = CATEGORY_ICONS[cat.label];
    let svg = "";
    try {
      svg = icon ? LUCIDE.svg(icon, { size: 22 }) : "";
    } catch (err) {
      svg = "";
    }
    if (svg) railTrigger.innerHTML = svg;
    else railTrigger.textContent = catInitials(cat.label);

    const railMenu = buildItemsMenu(cat, catIdx, "rail-flyout menu menu-md bg-base-200 rounded-box w-56 p-2 shadow border border-base-300");

    railItem.appendChild(railTrigger);
    railItem.appendChild(railMenu);
    railWrap.appendChild(railItem);

    accordionEl.appendChild(accWrap);
    accordionEl.appendChild(railWrap);
  });
}

// The compact window's first screen is built once the saved language is
// shown (i18nReady), so its texts come out translated.
Promise.all([loadMenu(), i18nReady]).then(function (r) {
  if (compactAction) {
    activateCompactMode(r[0]);
  }
});

// markActiveMenuItem highlights the sidebar entries of the selected item.
function markActiveMenuItem() {
  const key = window.__selectedItem ? menuKey(window.__selectedItem) : "";
  document.querySelectorAll("#accordion a[data-key]").forEach(function (el) {
    const active = el.dataset.key === key;
    el.classList.toggle("menu-active", active);
    if (active) el.setAttribute("aria-current", "page");
    else el.removeAttribute("aria-current");
  });
}

// selectMenuItem shows the item whose menuKey is key, as a sidebar click
// would. Returns false when the current catalog has no such item.
function selectMenuItem(key) {
  for (let catIdx = 0; catIdx < menuCats.length; catIdx++) {
    const cat = menuCats[catIdx];
    for (let itemIdx = 0; itemIdx < cat.items.length; itemIdx++) {
      if (menuKey(cat.items[itemIdx]) === key) {
        selectItem(cat, cat.items[itemIdx], catIdx, itemIdx);
        return true;
      }
    }
  }
  return false;
}

// screenFor returns the screen a menu item shows: its SCREENS entry, else
// the first SCREEN_FAMILIES entry that has its actionId, else null.
function screenFor(item) {
  const id = item.actionId;
  if (SCREENS[id]) return SCREENS[id];
  for (let i = 0; i < SCREEN_FAMILIES.length; i++) {
    if (SCREEN_FAMILIES[i].has(id)) return SCREEN_FAMILIES[i];
  }
  return null;
}

// screenContainers lists every screen container plus the extra tab labels
// only some screens show — all hidden before the selected screen shows its
// own.
function screenContainers() {
  const els = [];
  Object.keys(SCREENS).forEach(function (k) {
    els.push(SCREENS[k].el);
    (SCREENS[k].alsoShow || []).forEach(function (el) { els.push(el); });
  });
  SCREEN_FAMILIES.forEach(function (f) { els.push(f.el); });
  return els;
}

function selectItem(cat, item, catIdx, itemIdx) {
  breadcrumb.innerHTML = "<li>" + escHTML(cat.label) + "</li><li>" + escHTML(item.title) + "</li>";
  tabDescLabel.textContent = item.tab || document.title;
  window.__selectedItem = item;

  screenContainers().forEach(function (el) { el.classList.add("hidden"); });
  // "Execução" is visible by default (almost every screen goes through
  // it) — only "Home :: Configurações" hides it (noTerminal), since none of
  // its actions are async/streamed (Tema/Flatpak apply
  // immediately, Workspace persists on its own when "Criar Workspace"
  // is clicked).
  tabLabelTerminal.classList.remove("hidden");

  const screen = screenFor(item);
  if (!screen) {
    openError("Tela sem registro: " + item.actionId);
    return;
  }
  screen.el.classList.remove("hidden");
  (screen.alsoShow || []).forEach(function (el) { el.classList.remove("hidden"); });
  if (screen.noTerminal) {
    tabLabelTerminal.classList.add("hidden");
    selectTab("desc");
  }
  screen.render(item);
  selectTab("desc");

  markActiveMenuItem();
}
