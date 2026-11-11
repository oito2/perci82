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

// This application's test harness: the shared one, bound to its dist/.

import path from "node:path";
import { fileURLToPath } from "node:url";
import { makeHarness } from "../../../../internal/gui/web/test/harness.mjs";

export { FakeElement, evalIn } from "../../../../internal/gui/web/test/harness.mjs";

const harness = makeHarness(path.join(path.dirname(fileURLToPath(import.meta.url)), "..", "dist"));

export const loadScripts = harness.loadScripts;
export const scriptOrder = harness.scriptOrder;
export const resolveScript = harness.resolveScript;
export const distDir = harness.distDir;
export const commonDir = harness.commonDir;
