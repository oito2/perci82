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

// This application's message catalogs and the live language switch.

import { distDir, evalIn, loadScripts, scriptOrder } from "./harness.mjs";
import { registerI18nTests } from "../../../../internal/gui/web/test/i18n-tests.mjs";

registerI18nTests({ distDir, evalIn, loadScripts, scriptOrder });
