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

// Perci82 System plumbing: the DOM references of its screens and the
// memoized post-install profile.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

const contentDesc = document.getElementById("content-desc");
const simpleSystemLabel = document.getElementById("simple-system-label");
const btnExecutar = document.getElementById("btn-executar");
const executarHint = document.getElementById("executar-hint");
const simpleContainer = document.getElementById("simple-container");
const postinstallContainer = document.getElementById("postinstall-container");
const postinstallUnsupported = document.getElementById("postinstall-unsupported");
const postinstallSupported = document.getElementById("postinstall-supported");
const postinstallTitle = document.getElementById("postinstall-title");
const postinstallActionsList = document.getElementById("postinstall-actions");
const dashboardSystemLabel = document.getElementById("dashboard-system-label");
const scFlatpak = document.getElementById("sc-flatpak");

// getPostinstallProfile memoizes App.GetPostinstallProfile — the detected
// distribution can't change while the application runs, and three screens
// read it. A failed call isn't cached, and a language switch drops it (its
// labels are translated).
let postinstallProfilePromise = null;
onLocaleChange(function () { postinstallProfilePromise = null; });
function getPostinstallProfile() {
  if (!postinstallProfilePromise) {
    postinstallProfilePromise = App.GetPostinstallProfile().catch(function (err) {
      postinstallProfilePromise = null;
      throw err;
    });
  }
  return postinstallProfilePromise;
}

const suOptJournal = document.getElementById("su-opt-journal");
const suOptAutoremove = document.getElementById("su-opt-autoremove");
