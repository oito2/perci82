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

// Perci82 System screen registrations: which container each menu item
// shows, how it renders and what refreshes it after an action. Loaded last,
// once every screen script has defined its render functions.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

SCREENS.postinstall = { el: postinstallContainer, render: renderPostinstallScreen };
SCREENS["system-update"] = {
  el: simpleContainer,
  render: function (it) {
    contentDesc.textContent = it.desc;
    loadSimpleSystemStat();
    updateExecutarButton();
  },
};

// Re-fetches which items are applied now.
ACTION_DONE_HANDLERS.postinstall = function () { renderPostinstallScreen(); };
// Re-evaluates the "Executar" button.
ACTION_DONE_HANDLERS["system-update"] = function () { updateExecutarButton(); };
