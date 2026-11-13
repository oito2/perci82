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

// Generic UI helpers shared by every screen: confirm/message modals, the run lifecycle helpers (latestOnly, startExecution, beginRun) and escaping (escHTML/escAttr).
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// --- generic modals (confirm / message) ------------------------------

// showDialog opens dialog modally unless it is already open.
function showDialog(dialog) {
  if (!dialog.open) dialog.showModal();
}

let __confirmCallback = null;
btnConfirmOk.addEventListener("click", function () {
  confirmModal.close();
  const cb = __confirmCallback;
  __confirmCallback = null;
  if (cb) cb();
});
function openConfirm(title, message, onConfirm) {
  confirmModalTitle.textContent = title;
  confirmModalMessage.textContent = message;
  __confirmCallback = onConfirm;
  showDialog(confirmModal);
}

// error-modal doubles as a generic message modal (title and color class
// vary), e.g. for success messages like Exportar/Importar's export, which
// doesn't go through the Execução tab.
const errorModalTitleEl = errorModal.querySelector("h3");
function openMessage(title, message, isError) {
  errorModalTitleEl.textContent = title;
  errorModalTitleEl.className = "font-bold text-lg mb-2" + (isError ? " text-error" : "");
  errorModalMessage.textContent = message;
  showDialog(errorModal);
}
function openError(message) { openMessage(t("oito2.infoDanger"), String(message), true); }

// startExecution is the shared "begin a long-running action" sequence,
// switching to the Execução tab — used by every runAction-based screen
// that has no dedicated "Executar" button of its own (Docker:
// Recriar/Editar/Backup/Restore/Importar; Dev Tools :: Repositórios:
// Aplicar Identidade Global/Clonar/Iniciar/Identidade Local/Arquivos —
// only Iniciar/Parar/Reiniciar/Remover on containers stay outside this,
// since those run synchronously with no terminal panel involved).
// buttonsToDisable/maxSteps are optional — omit for the single-step,
// no-extra-button case.
//
// announceText: the line written to the terminal sometimes differs from
// the default "Iniciando <label>..." pattern — a suffix (app-local-
// file: "IDE: Android Studio (instalar)...") or even a different verb
// (self-uninstall: "Desinstalando o Perci...", not "Iniciando..."), while
// termTitle always uses just label ("Executando: IDE: Android Studio..."). This
// parameter lets those screens reuse the shared sequence instead of
// hand-rolling the whole block just for that one line; when omitted, the
// terminal uses the default pattern.
// latestOnly guards an async render against responses arriving out of
// order: each call starts a new request for key and returns fn wrapped so
// it only runs if no newer request for the same key started meanwhile.
// Without it, switching quickly between screens that share one container
// (the 7 checklist screens, the Docker table, the SKILLs/MCPs tables when
// changing scope) would mix items from two screens and leave "Executar"
// bound to whichever response arrived last.
const renderTokens = {};
function latestOnly(key, fn) {
  const token = (renderTokens[key] = (renderTokens[key] || 0) + 1);
  return function (value) {
    if (renderTokens[key] === token) return fn(value);
  };
}

// invalidate drops every pending response of key's latestOnly requests —
// for a render that returns early without starting a new request.
function invalidate(key) {
  renderTokens[key] = (renderTokens[key] || 0) + 1;
}

function startExecution(label, buttonsToDisable, maxSteps, announceText) {
  running = true;
  const buttons = (buttonsToDisable || []).filter(Boolean);
  buttons.forEach(function (btn) { btn.disabled = true; });
  // Remembered so finishRun releases exactly these buttons and
  // knows which screen started the run — see activeRun.
  activeRun = {
    actionId: window.__selectedItem ? window.__selectedItem.actionId : null,
    buttons: buttons,
    runId: null,
  };
  term.clear();
  term.writeln(announceText || t("term.starting", { label: label }));
  termTitle.textContent = t("term.running", { label: label + "..." });
  termProgress.value = 0;
  termProgress.max = maxSteps || 1;
  selectTab("terminal");
}

// beginRun starts a run unless one is active (this window's or, through
// running, any other): startExecution with opts.buttons, opts.maxSteps and
// opts.announce, then call() — which returns the App promise — with a
// refusal or failure handled by failRun. Returns false when it did not start.
function beginRun(label, call, opts) {
  opts = opts || {};
  if (running || activeRun) return false;
  startExecution(label, opts.buttons, opts.maxSteps, opts.announce);
  let result;
  try {
    result = call();
  } catch (err) {
    failRun(err);
    return true;
  }
  Promise.resolve(result).catch(failRun);
  return true;
}

// --- escaping (a convention for the whole file, not just this section) -
// Any text that isn't a fixed literal
// (a folder name, a path chosen in a native dialog, command output,
// whatever it is) must never go into innerHTML without passing through
// escHTML() (loose text) or escAttr() (inside an attribute, e.g.
// data-url="...") first. A folder path chosen via
// App.PickRepoWorkingFolder can contain any character on Linux
// (including `<`/`>`/`"`), and since this is a Wails app, injected script
// has direct access to every bound App method — it isn't "trapped" in a
// browser tab sandbox the way a typical XSS would be. Prefer
// .textContent/document.createElement when there's no real need for
// embedded HTML.
function escHTML(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}
// escAttr also escapes &, so a value like "a&amp;b" (a password, a dev
// command) survives an Editar round trip unchanged.
function escAttr(s) {
  return escHTML(s).replace(/"/g, "&quot;").replace(/'/g, "&#39;");
}
