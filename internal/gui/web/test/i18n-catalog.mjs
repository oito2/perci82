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

// Catalog checks shared by both applications' tests: which message keys
// a page uses (data-i18n* attributes in its index.html, t("…") and
// setAttribute("data-i18n…", "…") in its scripts), and the catalogs of
// each layer (oito2 base, shared, application) per locale.

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

export const LOCALES = ["pt-BR", "en"];

const commonDir = path.join(path.dirname(fileURLToPath(import.meta.url)), "..");

// appDistDirs lists the dist/ folder of every application.
export function appDistDirs() {
  const cmdDir = path.join(commonDir, "..", "..", "..", "cmd");
  return fs.readdirSync(cmdDir)
    .map((name) => path.join(cmdDir, name, "frontend", "dist"))
    .filter((dir) => fs.existsSync(path.join(dir, "index.html")));
}

// readCatalog returns the messages of one catalog file ({} when missing).
export function readCatalog(file) {
  return fs.existsSync(file) ? JSON.parse(fs.readFileSync(file, "utf8")) : {};
}

// layers returns the catalog folders merged for the application of
// distDir, in the order i18n.js merges them.
export function layers(distDir) {
  return {
    oito2: path.join(commonDir, "i18n", "oito2"),
    shared: path.join(commonDir, "i18n"),
    app: path.join(distDir, "i18n", "app"),
  };
}

// scriptFiles returns the paths of the scripts the page of distDir loads
// (the application's own file first, else the shared one).
function scriptFiles(distDir) {
  const html = fs.readFileSync(path.join(distDir, "index.html"), "utf8");
  return [...html.matchAll(/<script (?:defer |type="module" )?src="\.\/js\/([^"]+)"/g)].map((m) => {
    const own = path.join(distDir, "js", m[1]);
    return fs.existsSync(own) ? own : path.join(commonDir, "js", m[1]);
  });
}

// usedKeys returns the message keys the page of distDir uses.
export function usedKeys(distDir) {
  const keys = new Set();
  const html = fs.readFileSync(path.join(distDir, "index.html"), "utf8");
  for (const m of html.matchAll(/data-i18n(?:-[a-z-]+)?="([^"]+)"/g)) keys.add(m[1]);
  for (const file of scriptFiles(distDir)) {
    const src = fs.readFileSync(file, "utf8");
    for (const m of src.matchAll(/\bt\(\s*"([^"]+)"/g)) keys.add(m[1]);
    for (const m of src.matchAll(/"data-i18n(?:-[a-z-]+)?",\s*"([^"]+)"/g)) keys.add(m[1]);
  }
  return keys;
}

// placeholders returns the sorted {name} placeholders of msg.
export function placeholders(msg) {
  return [...msg.matchAll(/\{([A-Za-z0-9_]+)\}/g)].map((m) => m[1]).sort();
}
