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

import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";

import { commonDir, distDir } from "./harness.mjs";

// usedIcons collects every icon name the frontend asks for: iconButton("x")
// and LUCIDE.svg("x") in the scripts, and the section icons of the Go
// catalog (icon: "x").
function usedIcons() {
  const names = new Set();
  const dirs = [distDir, commonDir].flatMap((root) => [path.join(root, "js"), path.join(root, "js", "screens")]);
  for (const dir of dirs) {
    for (const f of fs.readdirSync(dir).filter((n) => n.endsWith(".js"))) {
      const src = fs.readFileSync(path.join(dir, f), "utf8");
      for (const m of src.matchAll(/(?:iconButton|LUCIDE\.svg)\("([A-Za-z0-9]+)"/g)) names.add(m[1]);
    }
  }
  const catalog = fs.readFileSync(path.join(commonDir, "..", "catalog.go"), "utf8");
  for (const m of catalog.matchAll(/icon:\s*"([A-Za-z0-9]+)"/g)) names.add(m[1]);
  return [...names].sort();
}

test("every icon the frontend uses is in the vendored lucide set", () => {
  const icons = fs.readFileSync(path.join(commonDir, "vendor", "lucide", "icons.js"), "utf8");
  const used = usedIcons();
  assert.ok(used.includes("circleArrowUp") && used.includes("gitBranch") && used.includes("house"), "the scan finds the icons in use");
  const missing = used.filter((name) => !new RegExp("^\\s+" + name + ":\\s*'", "m").test(icons));
  assert.deepEqual(missing, []);
});
