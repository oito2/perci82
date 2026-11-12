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

// Repositórios :: Configurar Conta, and the per-account screens.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

const gitaccountsContainer = document.getElementById("gitaccounts-container");
const gitaccountContainer = document.getElementById("gitaccount-container");
const tabLabelGitIdentity = document.getElementById("tab-label-git-identity");
const gaListView = document.getElementById("ga-list-view");
const gaFormView = document.getElementById("ga-form-view");
const gaDetailView = document.getElementById("ga-detail-view");
const gaRows = document.getElementById("ga-rows");
const gaEmpty = document.getElementById("ga-empty");
const btnGaNew = document.getElementById("btn-ga-new");
const gaFormTitle = document.getElementById("ga-form-title");
const gaName = document.getElementById("ga-name");
const gaProvider = document.getElementById("ga-provider");
const gaHost = document.getElementById("ga-host");
const gaPortWrap = document.getElementById("ga-port-wrap");
const gaPort = document.getElementById("ga-port");
const gaUsername = document.getElementById("ga-username");
const gaGitName = document.getElementById("ga-gitname");
const gaGitEmail = document.getElementById("ga-gitemail");
const gaFolderPath = document.getElementById("ga-folder-path");
const btnGaPickFolder = document.getElementById("btn-ga-pick-folder");
const gaKeyMode = document.getElementById("ga-key-mode");
const gaPassWrap = document.getElementById("ga-pass-wrap");
const gaPass = document.getElementById("ga-pass");
const gaPass2 = document.getElementById("ga-pass2");
const gaKeyExistingWrap = document.getElementById("ga-key-existing-wrap");
const gaKeyPathEl = document.getElementById("ga-key-path");
const btnGaPickKey = document.getElementById("btn-ga-pick-key");
const gaToken = document.getElementById("ga-token");
const gaTokenHint = document.getElementById("ga-token-hint");
const gaKeyHint = document.getElementById("ga-key-hint");
const btnGaFormKeysPage = document.getElementById("btn-ga-form-keys-page");
const gaTokenSteps = document.getElementById("ga-token-steps");
const btnGaTokenPage = document.getElementById("btn-ga-token-page");
const gaClearTokenWrap = document.getElementById("ga-clear-token-wrap");
const gaClearToken = document.getElementById("ga-clear-token");
const btnGaSave = document.getElementById("btn-ga-save");
const btnGaCancel = document.getElementById("btn-ga-cancel");
const gaDetailTitle = document.getElementById("ga-detail-title");
const btnGaEdit = document.getElementById("btn-ga-edit");
const btnGaBack = document.getElementById("btn-ga-back");
const gaDetailSummary = document.getElementById("ga-detail-summary");
const gaKeyBadge = document.getElementById("ga-key-badge");
const gaPubkey = document.getElementById("ga-pubkey");
const btnGaCopy = document.getElementById("btn-ga-copy");
const btnGaKeysPage = document.getElementById("btn-ga-keys-page");
const gaUnlockWrap = document.getElementById("ga-unlock-wrap");
const gaUnlockPass = document.getElementById("ga-unlock-pass");
const btnGaUnlock = document.getElementById("btn-ga-unlock");
const gaNoAgentNote = document.getElementById("ga-noagent-note");
const gaHostWrap = document.getElementById("ga-host-wrap");
const gaHostKeys = document.getElementById("ga-host-keys");
const btnGaScan = document.getElementById("btn-ga-scan");
const btnGaTrust = document.getElementById("btn-ga-trust");
const btnGaTest = document.getElementById("btn-ga-test");
const gaTestResult = document.getElementById("ga-test-result");
const gaTestResultText = document.getElementById("ga-test-result-text");
const gaIdName = document.getElementById("ga-id-name");
const gaIdEmail = document.getElementById("ga-id-email");
const btnGaIdApply = document.getElementById("btn-ga-id-apply");
const btnGacConfigure = document.getElementById("btn-gac-configure");

const GA_PROVIDERS = {
  github: {
    label: "GitHub", host: "github.com",
    keysPath: "/settings/keys",
    tokenPath: "/settings/personal-access-tokens/new",
    get keyHint() { return t("ga.github.keyHint"); },
    get tokenSteps() { return t("ga.github.tokenSteps"); },
  },
  gitlab: {
    label: "GitLab", host: "gitlab.com",
    keysPath: "/-/user_settings/ssh_keys",
    tokenPath: "/-/user_settings/personal_access_tokens",
    get keyHint() { return t("ga.gitlab.keyHint"); },
    get tokenSteps() { return t("ga.gitlab.tokenSteps"); },
  },
};

// GA_HOST_PATTERN is the host format the backend accepts; only such a
// host is opened in the browser.
const GA_HOST_PATTERN = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$/;

// gaProviderPage is the https URL of path on the form's provider and host,
// or "" while the typed host is not a valid host name.
function gaProviderPage(path) {
  const provider = GA_PROVIDERS[gaProvider.value] || GA_PROVIDERS.github;
  const host = gaHost.value.trim().toLowerCase() || provider.host;
  return GA_HOST_PATTERN.test(host) ? "https://" + host + path : "";
}

// Badge per GitAccountView.keyState: the message key of its label and its
// color.
const GA_KEY_STATES = {
  missing: { key: "ga.key.missing", color: "error" },
  ready: { key: "ga.key.ready", color: "success" },
  unlocked: { key: "ga.key.unlocked", color: "success" },
  locked: { key: "ga.key.locked", color: "warning" },
  "no-agent": { key: "ga.key.noAgent", color: "warning" },
};

// Key states in which a connection test can't authenticate.
const GA_KEY_UNUSABLE = ["missing", "locked", "no-agent"];

let gaEditing = null; // the account being edited; null for a new one
let gaFolder = ""; // the folder chosen in the form
let gaKeyPath = ""; // the key chosen in the form ("existing" mode)
let gaDetail = null; // the account the details view shows
let gaOpenOnRender = ""; // account ID whose details the next render opens

function gaErrText(err) {
  return String((err && err.message) || err);
}

function gaProviderLabel(v) {
  const p = GA_PROVIDERS[v.provider];
  return p ? p.label : v.provider;
}

// gaHostLabel is the host, with the port when it isn't 22.
function gaHostLabel(v) {
  return v.host + (v.sshPort ? ":" + v.sshPort : "");
}

function gaKeyState(state) {
  const s = GA_KEY_STATES[state];
  return s ? { label: t(s.key), color: s.color } : { label: state, color: "neutral" };
}

function gaKeyBadgeHTML(state) {
  const s = gaKeyState(state);
  return '<span class="badge badge-soft badge-' + s.color + '">' + escHTML(s.label) + "</span>";
}

// gaSummaryHTML lists an account's settings — shared by the details view
// and the account screen.
function gaSummaryHTML(v) {
  const rows = [
    [t("ga.sum.provider"), gaProviderLabel(v) + " — " + gaHostLabel(v)],
    [t("ga.sum.user"), v.username],
    [t("ga.sum.commits"), v.gitName + " <" + v.gitEmail + ">"],
    [t("ga.sum.folder"), v.folder],
    [t("ga.sum.key"), v.sshKey],
    [t("ga.sum.token"), v.hasToken ? t("ga.sum.tokenSaved") : t("ga.sum.none")],
  ];
  return '<dl class="grid gap-x-4 gap-y-1 text-sm" style="grid-template-columns: max-content 1fr;">' +
    rows.map(function (r) {
      return '<dt class="opacity-70">' + escHTML(r[0]) + '</dt><dd class="break-all">' + escHTML(r[1]) + "</dd>";
    }).join("") +
    "</dl>";
}

// gaShowView shows one of the screen's three views.
function gaShowView(view) {
  gaListView.classList.toggle("hidden", view !== gaListView);
  gaFormView.classList.toggle("hidden", view !== gaFormView);
  gaDetailView.classList.toggle("hidden", view !== gaDetailView);
}

function renderGitAccountsScreen() {
  renderGitIdentityTab();
  const open = gaOpenOnRender;
  gaOpenOnRender = "";
  if (open) gaShowDetail(open);
  else gaShowList();
}

// --- list view ----------------------------------------------------------

function gaShowList() {
  gaDetail = null;
  gaShowView(gaListView);
  App.ListGitAccounts().then(latestOnly("ga-list", function (accounts) {
    accounts = accounts || [];
    gaRows.innerHTML = "";
    gaEmpty.classList.toggle("hidden", accounts.length > 0);
    accounts.forEach(function (v) { gaRows.appendChild(buildGaRow(v)); });
  }));
}

function buildGaRow(v) {
  const tr = document.createElement("tr");

  const tdName = document.createElement("td");
  const link = document.createElement("button");
  link.type = "button";
  link.className = "link link-hover font-semibold";
  link.textContent = v.name;
  link.addEventListener("click", function () { gaShowDetail(v.id); });
  tdName.appendChild(link);
  tr.appendChild(tdName);

  const tdProvider = document.createElement("td");
  tdProvider.textContent = gaProviderLabel(v) + (v.pinnedHost ? "" : " (" + gaHostLabel(v) + ")");
  tr.appendChild(tdProvider);

  const tdFolder = document.createElement("td");
  tdFolder.className = "text-xs break-all";
  tdFolder.textContent = v.folder;
  tr.appendChild(tdFolder);

  const tdKey = document.createElement("td");
  tdKey.innerHTML = gaKeyBadgeHTML(v.keyState);
  tr.appendChild(tdKey);

  const tdActions = document.createElement("td");
  tdActions.className = "whitespace-nowrap";
  const btnEdit = iconButton("pencil", t("docker.edit"));
  btnEdit.addEventListener("click", function () { gaShowForm(v); });
  const btnRemove = iconButton("trash2", t("docker.remove"));
  btnRemove.addEventListener("click", function () { gaConfirmRemove(v); });
  tdActions.appendChild(btnEdit);
  tdActions.appendChild(btnRemove);
  tr.appendChild(tdActions);
  return tr;
}

function gaConfirmRemove(v) {
  openConfirm(
    t("ga.removeTitle"),
    t("ga.removeConfirm", { name: v.name, folder: v.folder, key: v.sshKey }) +
      (v.hasToken ? "\n\n" + t("ga.removeToken") : ""),
    function () {
      App.RemoveGitAccount(v.id).then(function () {
        loadMenu();
        gaShowList();
      }).catch(function (err) {
        openError(t("ga.removeFailed", { error: gaErrText(err) }));
      });
    }
  );
}

btnGaNew.addEventListener("click", function () { gaShowForm(null); });

// --- form view ----------------------------------------------------------

function gaShowForm(v) {
  gaEditing = v;
  gaFolder = v ? v.folder : "";
  gaKeyPath = "";
  gaFormTitle.textContent = v ? t("ga.editTitle", { name: v.name }) : t("ga.newTitle");
  gaName.value = v ? v.name : "";
  gaProvider.value = v ? v.provider : "github";
  gaHost.value = v && !v.pinnedHost ? v.host : "";
  gaPort.value = v && v.sshPort ? String(v.sshPort) : "";
  gaUsername.value = v ? v.username : "";
  gaGitName.value = v ? v.gitName : "";
  gaGitEmail.value = v ? v.gitEmail : "";
  gaPass.value = "";
  gaPass2.value = "";
  gaToken.value = "";
  gaClearToken.checked = false;
  gaClearTokenWrap.classList.toggle("hidden", !(v && v.hasToken));
  gaToken.placeholder = v && v.hasToken ? t("ga.tokenNew") : t("ga.tokenOptional");

  const modes = [];
  if (v) modes.push(["keep", t("ga.keyMode.keep", { key: v.sshKey })]);
  modes.push(["generate", t("ga.keyMode.generate")]);
  modes.push(["existing", t("ga.keyMode.existing")]);
  gaKeyMode.innerHTML = modes.map(function (m) {
    return '<option value="' + escAttr(m[0]) + '">' + escHTML(m[1]) + "</option>";
  }).join("");
  gaKeyMode.value = modes[0][0];

  gaUpdateForm();
  gaShowView(gaFormView);
}

// gaUpdateForm shows the fields that apply to the current choices: the SSH
// port only for a server of its own, and the inputs of the key mode.
function gaUpdateForm() {
  const provider = GA_PROVIDERS[gaProvider.value] || GA_PROVIDERS.github;
  gaHost.placeholder = provider.host;
  const host = gaHost.value.trim().toLowerCase();
  gaPortWrap.classList.toggle("hidden", host === "" || host === provider.host);
  gaPassWrap.classList.toggle("hidden", gaKeyMode.value !== "generate");
  gaKeyExistingWrap.classList.toggle("hidden", gaKeyMode.value !== "existing");
  gaFolderPath.textContent = gaFolder || t("devfolders.noFolder");
  gaKeyPathEl.textContent = gaKeyPath || t("ga.noKey");
  gaKeyHint.textContent = provider.keyHint;
  gaTokenSteps.textContent = provider.tokenSteps;
  btnGaFormKeysPage.disabled = !gaProviderPage(provider.keysPath);
  btnGaTokenPage.disabled = !gaProviderPage(provider.tokenPath);
}
btnGaFormKeysPage.addEventListener("click", function () {
  const url = gaProviderPage((GA_PROVIDERS[gaProvider.value] || GA_PROVIDERS.github).keysPath);
  if (url) Browser.OpenURL(url);
});
btnGaTokenPage.addEventListener("click", function () {
  const url = gaProviderPage((GA_PROVIDERS[gaProvider.value] || GA_PROVIDERS.github).tokenPath);
  if (url) Browser.OpenURL(url);
});

gaProvider.addEventListener("change", gaUpdateForm);
gaHost.addEventListener("input", gaUpdateForm);
gaKeyMode.addEventListener("change", gaUpdateForm);

btnGaPickFolder.addEventListener("click", function () {
  App.PickGitAccountFolder().then(function (path) {
    if (!path) return; // dialog cancelled
    gaFolder = path;
    gaUpdateForm();
  });
});

btnGaPickKey.addEventListener("click", function () {
  App.PickGitAccountKey().then(function (path) {
    if (!path) return; // dialog cancelled
    gaKeyPath = path;
    gaUpdateForm();
  });
});

// gaFormInput reads the form into a GitAccountInput, or returns an error
// message for what the backend can't check (the repeated passphrase).
function gaFormInput() {
  const mode = gaKeyMode.value;
  if (mode === "generate" && gaPass.value !== gaPass2.value) {
    return { error: t("ga.passMismatch") };
  }
  const portShown = !gaPortWrap.classList.contains("hidden");
  return {
    input: {
      id: gaEditing ? gaEditing.id : "",
      name: gaName.value,
      provider: gaProvider.value,
      host: gaHost.value,
      sshPort: portShown ? (parseInt(gaPort.value, 10) || 0) : 0,
      username: gaUsername.value,
      gitName: gaGitName.value,
      gitEmail: gaGitEmail.value,
      folder: gaFolder,
      keyMode: mode === "keep" ? "" : mode,
      keyPath: mode === "existing" ? gaKeyPath : "",
      passphrase: mode === "generate" ? gaPass.value : "",
      token: gaToken.value.trim(),
      clearToken: gaClearToken.checked,
    },
  };
}

btnGaSave.addEventListener("click", function () {
  if (btnGaSave.disabled) return;
  const form = gaFormInput();
  if (form.error) {
    openError(form.error);
    return;
  }
  btnGaSave.disabled = true;
  App.SaveGitAccount(form.input).then(function (v) {
    gaPass.value = "";
    gaPass2.value = "";
    gaToken.value = "";
    loadMenu();
    gaShowDetail(v.id, v);
  }).catch(function (err) {
    openError(t("ga.saveFailed", { error: gaErrText(err) }));
  }).finally(function () {
    btnGaSave.disabled = false;
  });
});

btnGaCancel.addEventListener("click", function () {
  if (gaEditing) gaShowDetail(gaEditing.id);
  else gaShowList();
});

// --- details view -------------------------------------------------------

// gaShowDetail shows account id's details: v when the caller already has
// them, fetched otherwise.
function gaShowDetail(id, v) {
  gaDetail = { id: id };
  gaShowView(gaDetailView);
  gaTestResult.classList.add("hidden");
  gaHostKeys.innerHTML = "";
  btnGaTrust.classList.add("hidden");
  gaUnlockPass.value = "";
  if (v) {
    invalidate("ga-detail");
    gaRenderDetail(v);
    return;
  }
  App.GetGitAccount(id).then(latestOnly("ga-detail", gaRenderDetail)).catch(function (err) {
    openError(t("ga.readFailed", { error: gaErrText(err) }));
    gaShowList();
  });
}

function gaRenderDetail(v) {
  if (!gaDetail || gaDetail.id !== v.id) return;
  gaDetail = v;
  gaDetailTitle.textContent = v.name;
  gaDetailSummary.innerHTML = gaSummaryHTML(v);
  const state = gaKeyState(v.keyState);
  gaKeyBadge.className = "badge badge-soft badge-" + state.color;
  gaKeyBadge.textContent = state.label;
  gaPubkey.textContent = v.publicKey || t("ga.noPubkey", { path: v.sshKey + ".pub" });
  btnGaCopy.disabled = !v.publicKey;
  btnGaCopy.textContent = t("ga.copy");
  btnGaKeysPage.textContent = t("ga.openKeys", { provider: gaProviderLabel(v) });
  btnGaKeysPage.classList.toggle("hidden", !v.keysUrl);
  gaUnlockWrap.classList.toggle("hidden", v.keyState !== "locked");
  gaNoAgentNote.classList.toggle("hidden", v.keyState !== "no-agent");
  gaHostWrap.classList.toggle("hidden", v.hostTrusted);
  btnGaTest.disabled = !v.hostTrusted || GA_KEY_UNUSABLE.indexOf(v.keyState) >= 0;
}

btnGaEdit.addEventListener("click", function () {
  if (gaDetail && gaDetail.name) gaShowForm(gaDetail);
});
btnGaBack.addEventListener("click", gaShowList);

btnGaCopy.addEventListener("click", function () {
  if (!gaDetail || !gaDetail.publicKey) return;
  Clipboard.SetText(gaDetail.publicKey).then(function () {
    btnGaCopy.textContent = t("ga.copied");
  }).catch(function (err) {
    openError(t("ga.copyFailed", { error: gaErrText(err) }));
  });
});

btnGaKeysPage.addEventListener("click", function () {
  if (gaDetail && gaDetail.keysUrl) Browser.OpenURL(gaDetail.keysUrl);
});

btnGaUnlock.addEventListener("click", function () {
  if (!gaDetail || btnGaUnlock.disabled) return;
  const id = gaDetail.id;
  btnGaUnlock.disabled = true;
  App.UnlockGitAccountKey(id, gaUnlockPass.value).then(function () {
    gaShowDetail(id);
  }).catch(function (err) {
    openError(gaErrText(err));
  }).finally(function () {
    gaUnlockPass.value = "";
    btnGaUnlock.disabled = false;
  });
});

// The server's host keys are listed for the user to compare; only the
// keys of this scan can then be trusted (the backend keeps them).
btnGaScan.addEventListener("click", function () {
  if (!gaDetail || btnGaScan.disabled) return;
  const id = gaDetail.id;
  btnGaScan.disabled = true;
  btnGaTrust.classList.add("hidden");
  gaHostKeys.textContent = t("ga.scanning");
  App.ScanGitAccountHostKeys(id).then(function (keys) {
    if (!gaDetail || gaDetail.id !== id) return;
    gaHostKeys.innerHTML = "<ul>" + (keys || []).map(function (k) {
      return '<li><span class="opacity-70">' + escHTML(k.type) + "</span> <code>" + escHTML(k.fingerprint) + "</code></li>";
    }).join("") + "</ul>";
    btnGaTrust.classList.remove("hidden");
  }).catch(function (err) {
    gaHostKeys.textContent = "";
    openError(gaErrText(err));
  }).finally(function () {
    btnGaScan.disabled = false;
  });
});

btnGaTrust.addEventListener("click", function () {
  if (!gaDetail) return;
  const id = gaDetail.id;
  App.TrustGitAccountHostKeys(id).then(function () {
    gaShowDetail(id);
  }).catch(function (err) {
    openError(gaErrText(err));
  });
});

btnGaTest.addEventListener("click", function () {
  if (!gaDetail || btnGaTest.disabled) return;
  const id = gaDetail.id;
  btnGaTest.disabled = true;
  gaShowTestResult("info", t("ga.connecting", { host: gaHostLabel(gaDetail) }));
  App.TestGitAccountConnection(id).then(function (user) {
    if (gaDetail && gaDetail.id === id) gaShowTestResult("success", t("ga.connected", { user: user }));
  }).catch(function (err) {
    if (gaDetail && gaDetail.id === id) gaShowTestResult("error", gaErrText(err));
  }).finally(function () {
    btnGaTest.disabled = false;
  });
});

function gaShowTestResult(kind, text) {
  gaTestResult.className = "alert alert-soft alert-" + kind;
  gaTestResultText.textContent = text;
}

// --- "Identidade padrão" tab --------------------------------------------

function renderGitIdentityTab() {
  App.GetGlobalGitIdentity().then(function (identity) {
    gaIdName.value = identity.name || "";
    gaIdEmail.value = identity.email || "";
  });
}

btnGaIdApply.addEventListener("click", function () {
  const name = gaIdName.value;
  const email = gaIdEmail.value;
  beginRun(t("ga.run.identity"), function () { return App.ApplyGlobalGitIdentity(name, email); },
    { buttons: [btnGaIdApply] });
});

// --- "Repositórios :: <conta>" ------------------------------------------

const gacHeader = document.getElementById("gac-header");
const btnGacReload = document.getElementById("btn-gac-reload");
const btnGacVerify = document.getElementById("btn-gac-verify");
const btnGacPull = document.getElementById("btn-gac-pull");
const btnGacFixUrls = document.getElementById("btn-gac-fix-urls");
const btnGacClonePending = document.getElementById("btn-gac-clone-pending");
const gacPending = document.getElementById("gac-pending");
const gacPendingList = document.getElementById("gac-pending-list");
const btnGacNew = document.getElementById("btn-gac-new");
const newRepoModal = document.getElementById("new-repo-modal");
const nrMode = document.getElementById("nr-mode");
const nrAccount = document.getElementById("nr-account");
const btnNrLoad = document.getElementById("btn-nr-load");
const nrListNote = document.getElementById("nr-list-note");
const nrList = document.getElementById("nr-list");
const nrClone = document.getElementById("nr-clone");
const nrSource = document.getElementById("nr-source");
const nrInit = document.getElementById("nr-init");
const nrRemoteName = document.getElementById("nr-remote-name");
const nrDest = document.getElementById("nr-dest");
const nrDestPreview = document.getElementById("nr-dest-preview");
const btnNrGo = document.getElementById("btn-nr-go");
const repoActionsModal = document.getElementById("repo-actions-modal");
const raTitle = document.getElementById("ra-title");
const raSubtitle = document.getElementById("ra-subtitle");
const raUrl = document.getElementById("ra-url");
const raUrlText = document.getElementById("ra-url-text");
const btnRaUrl = document.getElementById("btn-ra-url");
const raRemote = document.getElementById("ra-remote");
const raRemoteName = document.getElementById("ra-remote-name");
const btnRaRemote = document.getElementById("btn-ra-remote");
const raRemotePreview = document.getElementById("ra-remote-preview");
const raIdent = document.getElementById("ra-ident");
const raIdentText = document.getElementById("ra-ident-text");
const btnRaIdent = document.getElementById("btn-ra-ident");
const raTabSettings = document.getElementById("ra-tab-settings");
const raAccountEmail = document.getElementById("ra-account-email");
const btnRaAccountEmail = document.getElementById("btn-ra-account-email");
const raRepoEmail = document.getElementById("ra-repo-email");
const btnRaRepoEmail = document.getElementById("btn-ra-repo-email");
const raEmailEffective = document.getElementById("ra-email-effective");
const raSSHOk = document.getElementById("ra-ssh-ok");
const raSSHOkText = document.getElementById("ra-ssh-ok-text");
const raLicense = document.getElementById("ra-license");
const raLicenseHolderWrap = document.getElementById("ra-license-holder-wrap");
const raLicenseHolder = document.getElementById("ra-license-holder");
const raLicenseYearWrap = document.getElementById("ra-license-year-wrap");
const raLicenseYear = document.getElementById("ra-license-year");
const raFilesMsg = document.getElementById("ra-files-msg");
const raFilesRows = document.getElementById("ra-files-rows");
const gacAlert = document.getElementById("gac-alert");
const gacAlertText = document.getElementById("gac-alert-text");
const gacLoading = document.getElementById("gac-loading");
const gacRows = document.getElementById("gac-rows");
const gacEmpty = document.getElementById("gac-empty");
const gacHidden = document.getElementById("gac-hidden");
const gacHiddenSummary = document.getElementById("gac-hidden-summary");
const gacHiddenList = document.getElementById("gac-hidden-list");

let gacAccountId = "";
let gacAccount = null; // the account's GitAccountView
let raRepo = null; // the repository the actions dialog is for
// gacReady: the account can reach its server (no alert), so the network
// buttons are enabled.
let gacReady = false;

// relTime renders an RFC 3339 time relative to now ("há 5 min"), or as a
// date once it is more than a month old.
function relTime(iso, now) {
  const ms = Date.parse(iso);
  if (isNaN(ms)) return "";
  const secs = Math.max(0, Math.round(((now || Date.now()) - ms) / 1000));
  if (secs < 60) return t("time.now");
  const mins = Math.round(secs / 60);
  if (mins < 60) return t("time.minutes", { count: mins });
  const hours = Math.round(mins / 60);
  if (hours < 24) return t("time.hours", { count: hours });
  const days = Math.round(hours / 24);
  if (days === 1) return t("time.yesterday");
  if (days < 30) return t("time.days", { count: days });
  const d = new Date(ms);
  return String(d.getDate()).padStart(2, "0") + "/" + String(d.getMonth() + 1).padStart(2, "0") + "/" + d.getFullYear();
}

// gacChanges lists the "Alterações" badges of a repository, from its
// local state.
function gacChanges(st) {
  const out = [];
  if (st.error) out.push({ label: t("gac.ch.error"), color: "error", title: st.error });
  if (st.behind > 0) out.push({ label: "↓ " + st.behind, color: "info", title: t("gac.ch.behind", { count: st.behind }) });
  if (st.ahead > 0) out.push({ label: "↑ " + st.ahead, color: "warning", title: t("gac.ch.ahead", { count: st.ahead }) });
  if (st.modified > 0) out.push({ label: t("gac.ch.modified", { count: st.modified }), color: "warning", title: t("gac.ch.modifiedHint") });
  if (st.untracked > 0) out.push({ label: t("gac.ch.untracked", { count: st.untracked }), color: "neutral", title: t("gac.ch.untrackedHint") });
  if (st.remoteClass === "none") out.push({ label: t("gac.ch.noRemote"), color: "neutral", title: t("gac.ch.noRemoteHint") });
  if (st.remoteClass === "https") out.push({ label: t("gac.ch.https"), color: "warning", title: t("gac.ch.httpsHint") });
  if (st.remoteClass === "ok" && !st.upstream && !st.detached && st.lastCommit) {
    out.push({ label: t("gac.ch.noUpstream"), color: "neutral", title: t("gac.ch.noUpstreamHint") });
  }
  if (st.identityDiverges) {
    out.push({
      label: t("gac.ch.identity"), color: "warning",
      title: t("gac.ch.identityHint", { identity: [st.localName, st.localEmail].filter(Boolean).join(" ") }),
    });
  }
  if (!out.length) out.push({ label: t("gac.ch.current"), color: "success", title: t("gac.ch.currentHint") });
  return out;
}

function buildGacRow(st) {
  const tr = document.createElement("tr");

  const tdRepo = document.createElement("td");
  const name = document.createElement("div");
  name.className = "font-semibold break-all";
  name.textContent = st.path;
  tdRepo.appendChild(name);
  if (st.remote) {
    const remote = document.createElement("div");
    remote.className = "text-xs opacity-60 break-all";
    remote.textContent = st.remote;
    tdRepo.appendChild(remote);
  }
  tr.appendChild(tdRepo);

  const tdBranch = document.createElement("td");
  tdBranch.textContent = st.detached ? t("gac.detached") : st.branch;
  if (st.upstream) tdBranch.title = t("gac.tracks", { branch: st.upstream });
  tr.appendChild(tdBranch);

  const tdCommit = document.createElement("td");
  tdCommit.className = "whitespace-nowrap";
  if (st.lastCommit) {
    tdCommit.textContent = relTime(st.lastCommit);
    tdCommit.title = new Date(Date.parse(st.lastCommit)).toLocaleString(i18nLocale) + " — " + st.lastSubject;
  } else {
    tdCommit.textContent = t("gac.noCommits");
    tdCommit.className += " opacity-60";
  }
  tr.appendChild(tdCommit);

  const tdChanges = document.createElement("td");
  const badges = document.createElement("div");
  badges.className = "flex flex-wrap gap-1";
  gacChanges(st).forEach(function (c) {
    const b = document.createElement("span");
    b.className = "badge badge-soft badge-sm badge-" + c.color;
    b.textContent = c.label;
    b.title = c.title;
    badges.appendChild(b);
  });
  tdChanges.appendChild(badges);
  if (st.remoteClass !== "none") {
    const fetched = document.createElement("div");
    fetched.className = "text-xs opacity-60 mt-1";
    fetched.textContent = st.lastFetch ? t("gac.fetched", { when: relTime(st.lastFetch) }) : t("gac.neverFetched");
    tdChanges.appendChild(fetched);
  }
  tr.appendChild(tdChanges);

  const tdActions = document.createElement("td");
  tdActions.className = "whitespace-nowrap";
  const canNet = gacReady && st.remoteClass !== "none" && !st.error;
  const btnVerify = iconButton("refreshCw", t("gac.verifyOne", { path: st.path }));
  btnVerify.disabled = !canNet;
  btnVerify.addEventListener("click", function () { gacRun("verify", st); });
  const btnPull = iconButton("download", t("gac.pullOne", { path: st.path }));
  btnPull.disabled = !canNet;
  btnPull.addEventListener("click", function () { gacRun("pull", st); });
  const btnMore = document.createElement("button");
  btnMore.type = "button";
  btnMore.className = "btn btn-ghost btn-xs";
  btnMore.title = t("gac.more", { path: st.path });
  btnMore.setAttribute("aria-label", btnMore.title);
  btnMore.innerHTML = LUCIDE.svg("plus", { size: 16 });
  btnMore.disabled = !!st.error;
  btnMore.addEventListener("click", function () { openRepoActions(st); });
  tdActions.appendChild(btnVerify);
  tdActions.appendChild(btnPull);
  tdActions.appendChild(btnMore);
  tr.appendChild(tdActions);
  return tr;
}

// The toolbar buttons a long action of this screen disables.
const GAC_RUN_BUTTONS = [btnGacVerify, btnGacPull, btnGacReload, btnGacFixUrls, btnGacNew, btnGacClonePending];

// gacRun starts Verificar ("verify") or Baixar ("pull") on the Execução
// tab, for every listed repository or only st's.
function gacRun(kind, st) {
  const id = gacAccountId;
  const dir = st ? st.dir : "";
  const what = st ? st.path : t("gac.allRepos");
  const label = t(kind === "verify" ? "gac.verifyOne" : "gac.pullOne", { path: what });
  beginRun(label, function () {
    return kind === "verify" ? App.VerifyGitAccountRepos(id, dir) : App.PullGitAccountRepos(id, dir);
  }, { buttons: GAC_RUN_BUTTONS });
}

btnGacFixUrls.addEventListener("click", function () {
  if (!gacAccountId || btnGacFixUrls.disabled) return;
  const id = gacAccountId;
  beginRun(t("gac.run.fixUrls"), function () { return App.FixGitAccountRepoURLs(id, ""); },
    { buttons: GAC_RUN_BUTTONS });
});

// --- actions of one repository --------------------------------------------

// gacSSHURL mirrors the backend's SSH URL of name under the account's
// user, for the "Definir remote" preview.
function gacSSHURL(v, name) {
  if (v.sshPort && v.sshPort !== 22) return "ssh://git@" + v.host + ":" + v.sshPort + "/" + v.username + "/" + name + ".git";
  return "git@" + v.host + ":" + v.username + "/" + name + ".git";
}

function repoBaseName(path) {
  const parts = path.split("/").filter(function (p) { return p && p !== "."; });
  return parts.length ? parts[parts.length - 1] : "";
}

function openRepoActions(st) {
  if (!gacAccount) return;
  raRepo = st;
  raTitle.textContent = st.path;
  raSubtitle.textContent = st.dir;
  raTabSettings.checked = true;

  raSSHOk.classList.toggle("hidden", st.remoteClass !== "ok");
  raSSHOkText.textContent = t("ra.sshOk", { remote: st.remote });
  raUrl.classList.toggle("hidden", st.remoteClass !== "https");
  raUrlText.textContent = t("ra.https", { remote: st.remote, ssh: st.sshUrl || "" });
  btnRaUrl.disabled = !gacReady || running;

  raRemote.classList.toggle("hidden", st.remoteClass !== "none");
  raRemoteName.value = repoBaseName(st.path);
  raUpdateRemotePreview();

  raIdent.classList.toggle("hidden", !st.identityDiverges);
  raIdentText.textContent = t("ra.identity", {
    local: [st.localName, st.localEmail].filter(Boolean).join(" "),
    identity: gacAccount.gitName + " <" + gacAccount.gitEmail + ">",
  });

  raFiles = null;
  raAccountEmail.value = "";
  raRepoEmail.value = "";
  raEmailEffective.textContent = "";
  raFilesRows.innerHTML = "";
  raShowFilesMsg("", false);
  raLoadFiles("");
  showDialog(repoActionsModal);
}

function raUpdateRemotePreview() {
  const name = raRemoteName.value.trim().replace(/\.git$/, "");
  raRemotePreview.textContent = name && gacAccount ? t("ra.remotePreview", { url: gacSSHURL(gacAccount, name) }) : "";
}
raRemoteName.addEventListener("input", raUpdateRemotePreview);

// raLaunch closes the dialog and starts a long action on the Execução tab.
function raLaunch(label, call) {
  repoActionsModal.close();
  beginRun(label, call, { buttons: GAC_RUN_BUTTONS });
}

btnRaUrl.addEventListener("click", function () {
  if (!raRepo || btnRaUrl.disabled) return;
  const id = gacAccountId;
  const dir = raRepo.dir;
  raLaunch(t("ra.run.fixUrl", { path: raRepo.path }), function () { return App.FixGitAccountRepoURLs(id, dir); });
});

btnRaRemote.addEventListener("click", function () {
  if (!raRepo || btnRaRemote.disabled) return;
  btnRaRemote.disabled = true;
  App.SetGitAccountRepoRemote(gacAccountId, raRepo.dir, raRemoteName.value).then(function () {
    repoActionsModal.close();
    loadGacRepos();
  }).catch(function (err) {
    openError(t("ra.remoteFailed", { error: gaErrText(err) }));
  }).finally(function () {
    btnRaRemote.disabled = false;
  });
});

btnRaIdent.addEventListener("click", function () {
  if (!raRepo) return;
  const st = raRepo;
  const id = gacAccountId;
  repoActionsModal.close();
  openConfirm(
    t("ra.identityTitle"),
    t("ra.identityConfirm", { path: st.path, identity: gacAccount.gitName + " <" + gacAccount.gitEmail + ">" }),
    function () {
      App.ResetGitAccountRepoIdentity(id, st.dir).then(loadGacRepos).catch(function (err) {
        openError(t("ra.identityFailed", { error: gaErrText(err) }));
      });
    }
  );
});

// --- Configurações and Arquivos tabs ----------------------------------------

let raFiles = null; // the last RepoFilesView of the open repository
let raFilesSeq = 0; // drops answers of an older load

// raShowFilesMsg shows text above the files table (an error in red), or
// hides it when text is empty.
function raShowFilesMsg(text, isError) {
  raFilesMsg.textContent = text;
  raFilesMsg.classList.toggle("hidden", !text);
  raFilesMsg.classList.toggle("alert-error", !!isError);
  raFilesMsg.classList.toggle("alert-success", !isError);
}

// raLoadFiles reads the Arquivos tab of the open repository, comparing
// LICENSE against license ("" lets the backend pick the license found in
// the file, or the default one).
function raLoadFiles(license) {
  if (!raRepo) return;
  const seq = ++raFilesSeq;
  const first = raFiles === null;
  App.GetGitAccountRepoFiles(gacAccountId, raRepo.dir, license).then(function (view) {
    if (seq !== raFilesSeq) return;
    raFiles = view;
    raRenderEmails(view);
    raRenderLicense(view, first);
    raRenderFiles(view);
  }).catch(function (err) {
    if (seq !== raFilesSeq) return;
    raShowFilesMsg(t("ra.filesFailed", { error: gaErrText(err) }), true);
  });
}

function raRenderEmails(v) {
  raAccountEmail.value = v.accountEmail || "";
  raRepoEmail.value = v.repoEmail || "";
  if (!v.contactEmail) {
    raEmailEffective.textContent = t("ra.noEmail");
  } else if (v.repoEmail) {
    raEmailEffective.textContent = t("ra.emailRepo", { email: v.contactEmail });
  } else {
    raEmailEffective.textContent = t("ra.emailAccount", { email: v.contactEmail });
  }
}

// raLicenseInfo returns the offered license id of the open view.
function raLicenseInfo(id) {
  const list = (raFiles && raFiles.licenses) || [];
  for (let i = 0; i < list.length; i++) {
    if (list[i].id === id) return list[i];
  }
  return null;
}

// raRenderLicense fills the license select with v.license selected, and
// the holder and year with the suggested values on the first load of the
// dialog (later loads keep what was typed).
function raRenderLicense(v, first) {
  raLicense.innerHTML = "";
  (v.licenses || []).forEach(function (l) {
    const opt = document.createElement("option");
    opt.value = l.id;
    opt.textContent = l.name;
    raLicense.appendChild(opt);
  });
  raLicense.value = v.license;
  if (first) {
    raLicenseHolder.value = v.holder || "";
    raLicenseYear.value = String(v.year || "");
  }
  const info = raLicenseInfo(v.license);
  const copyright = !!(info && info.copyright);
  raLicenseHolderWrap.classList.toggle("hidden", !copyright);
  raLicenseYearWrap.classList.toggle("hidden", !copyright);
}

// raStatus returns the Status badge of a files row.
function raStatus(row, v) {
  switch (row.state) {
    case "missing":
      return { label: t("ra.st.missing"), color: "neutral", title: t("ra.st.missingHint") };
    case "current":
      return {
        label: row.license ? row.license + " — " + t("gac.ch.current") : t("gac.ch.current"), color: "success",
        title: t("ra.st.currentHint"),
      };
    case "differs":
      if (row.id === "gitignore") {
        return { label: t("ra.st.patterns", { count: row.missing }), color: "warning", title: t("ra.st.patternsHint") };
      }
      if (row.id === "license") {
        return row.license
          ? { label: row.license, color: "warning", title: t("ra.st.otherLicense", { license: v.license }) }
          : { label: t("ra.st.unknownLicense"), color: "warning", title: t("ra.st.unknownLicenseHint") };
      }
      return {
        label: t("ra.st.differs"), color: "warning",
        title: t("ra.st.differsHint"),
      };
    case "present":
      return { label: t("ra.st.present"), color: "info", title: t("ra.st.presentHint") };
    default:
      return { label: t("ra.st.soon"), color: "ghost", title: t("ra.st.soonHint") };
  }
}

// raActions lists the actions of a files row: "apply" (Criar or
// Atualizar) and "remove".
function raActions(row) {
  switch (row.state) {
    case "missing": return [{ kind: "apply", label: t("devfolders.create") }];
    case "current": return [{ kind: "remove", label: t("docker.remove") }];
    case "differs":
    case "present": return [{ kind: "apply", label: t("action.update") }, { kind: "remove", label: t("docker.remove") }];
    default: return [];
  }
}

function raRenderFiles(v) {
  raFilesRows.innerHTML = "";
  (v.rows || []).forEach(function (row) {
    const tr = document.createElement("tr");

    const tdName = document.createElement("td");
    const name = document.createElement("div");
    name.className = "font-mono text-sm break-all";
    name.textContent = row.label;
    tdName.appendChild(name);
    if (row.files && (row.files.length > 1 || row.files[0] !== row.label)) {
      const files = document.createElement("div");
      files.className = "text-xs opacity-60 break-all";
      files.textContent = row.files.join(", ");
      tdName.appendChild(files);
    }
    (row.notes || []).forEach(function (note) {
      const n = document.createElement("div");
      n.className = "text-xs text-warning";
      n.textContent = note;
      tdName.appendChild(n);
    });
    tr.appendChild(tdName);

    const tdStatus = document.createElement("td");
    const st = raStatus(row, v);
    const badge = document.createElement("span");
    badge.className = "badge badge-soft badge-sm whitespace-nowrap badge-" + st.color;
    badge.textContent = st.label;
    badge.title = st.title;
    tdStatus.appendChild(badge);
    tr.appendChild(tdStatus);

    const tdActions = document.createElement("td");
    tdActions.className = "whitespace-nowrap";
    raActions(row).forEach(function (a) {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "btn btn-xs mr-1 " + (a.kind === "remove" ? "btn-ghost text-error" : row.state === "missing" ? "btn-primary" : "btn-outline");
      btn.textContent = a.label;
      btn.dataset.row = row.id;
      btn.dataset.kind = a.kind;
      btn.disabled = running;
      if (a.kind === "apply" && row.needsEmail && !v.contactEmail) {
        btn.disabled = true;
        btn.title = t("ra.needEmail");
      }
      btn.addEventListener("click", function () {
        if (btn.disabled) return;
        if (a.kind === "apply") raApply(row); else raRemove(row);
      });
      tdActions.appendChild(btn);
    });
    tr.appendChild(tdActions);
    raFilesRows.appendChild(tr);
  });
}

// raChoice returns the LICENSE choices typed in the Arquivos tab.
function raChoice() {
  return {
    license: raLicense.value,
    holder: raLicenseHolder.value.trim(),
    year: parseInt(raLicenseYear.value, 10) || 0,
  };
}

// raApply creates or updates the files of row, asking first when an
// existing file is replaced (the .gitignore only gains patterns).
function raApply(row) {
  const id = gacAccountId;
  const dir = raRepo.dir;
  const choice = raChoice();
  const go = function () {
    App.ApplyGitAccountRepoFile(id, dir, row.id, choice).then(function (msg) {
      raShowFilesMsg(msg, false);
      raLoadFiles(choice.license);
    }).catch(function (err) {
      raShowFilesMsg(t("ra.writeFailed", { name: row.label, error: gaErrText(err) }), true);
    });
  };
  if (row.state === "missing" || row.id === "gitignore") {
    go();
    return;
  }
  let message = t("ra.replaceConfirm", { files: row.files.join(", ") });
  if (row.id === "license" && row.license && row.license !== choice.license) {
    message = t("ra.replaceLicense", { from: row.license, to: choice.license });
  }
  openConfirm(t("ra.updateTitle", { name: row.label }), message, go);
}

// raRemove deletes the files of row after a confirmation.
function raRemove(row) {
  const id = gacAccountId;
  const dir = raRepo.dir;
  openConfirm(
    t("ra.removeTitle", { name: row.label }),
    t("ra.removeConfirm", { files: row.files.join("\n- ") }),
    function () {
      App.RemoveGitAccountRepoFile(id, dir, row.id).then(function () {
        raShowFilesMsg(t("ra.removed", { name: row.label }), false);
        raLoadFiles(raLicense.value);
      }).catch(function (err) {
        raShowFilesMsg(t("ra.removeFailed", { name: row.label, error: gaErrText(err) }), true);
      });
    }
  );
}

raLicense.addEventListener("change", function () {
  raShowFilesMsg("", false);
  raLoadFiles(raLicense.value);
});

// raSaveEmail saves the account's contact e-mail (forRepo false) or the
// open repository's own, then re-reads the tabs.
function raSaveEmail(forRepo) {
  if (!raRepo) return;
  const btn = forRepo ? btnRaRepoEmail : btnRaAccountEmail;
  if (btn.disabled) return;
  const value = (forRepo ? raRepoEmail : raAccountEmail).value.trim();
  btn.disabled = true;
  App.SetGitAccountContactEmail(gacAccountId, forRepo ? raRepo.dir : "", value).then(function () {
    raLoadFiles(raLicense.value);
  }).catch(function (err) {
    openError(t("ra.emailFailed", { error: gaErrText(err) }));
  }).finally(function () {
    btn.disabled = false;
  });
}
btnRaAccountEmail.addEventListener("click", function () { raSaveEmail(false); });
btnRaRepoEmail.addEventListener("click", function () { raSaveEmail(true); });

btnGacVerify.addEventListener("click", function () {
  if (gacAccountId && !btnGacVerify.disabled) gacRun("verify", null);
});
btnGacPull.addEventListener("click", function () {
  if (gacAccountId && !btnGacPull.disabled) gacRun("pull", null);
});

// gacSetReady enables the network buttons when the account can reach its
// server and nothing is running.
function gacSetReady(ready) {
  gacReady = ready;
  btnGacVerify.disabled = !ready || running;
  btnGacPull.disabled = !ready || running;
  btnGacFixUrls.disabled = !ready || running;
  btnGacNew.disabled = !ready || running;
  btnGacClonePending.disabled = !ready || running;
}

// gacAccountAlert says why the account can't reach its server yet, or ""
// when it can.
function gacAccountAlert(v) {
  if (v.keyState === "missing") return t("gac.alert.keyMissing");
  if (v.keyState === "locked") return t("gac.alert.keyLocked");
  if (v.keyState === "no-agent") return t("gac.alert.noAgent");
  if (!v.hostTrusted) return t("gac.alert.hostUntrusted", { host: v.host });
  return "";
}

// renderGitAccountScreen reads the account first, so the rows know
// whether the network buttons apply, then the repositories.
function renderGitAccountScreen(item) {
  gacAccountId = item.accountId;
  gacHeader.textContent = "";
  gacAlert.classList.add("hidden");
  gacRows.innerHTML = "";
  gacAccount = null;
  gacSetReady(false);
  App.GetGitAccount(item.accountId).then(latestOnly("gac", function (v) {
    gacAccount = v;
    gacHeader.textContent = gaProviderLabel(v) + " · " + v.username + " · " + v.folder;
    const alert = gacAccountAlert(v);
    gacAlertText.textContent = alert;
    gacAlert.classList.toggle("hidden", !alert);
    gacSetReady(!alert);
    loadGacRepos();
  })).catch(function (err) {
    openError(t("ga.readFailed", { error: gaErrText(err) }));
  });
}

function loadGacRepos() {
  const id = gacAccountId;
  gacRows.innerHTML = "";
  gacEmpty.classList.add("hidden");
  gacHidden.classList.add("hidden");
  renderGacPending([]);
  gacLoading.classList.remove("hidden");
  btnGacReload.disabled = true;
  // One guard for both outcomes: only the latest request's answer counts.
  const latest = latestOnly("gac-repos", function (fn) { fn(); });
  App.ListGitAccountRepos(id).then(function (res) { latest(function () {
    gacLoading.classList.add("hidden");
    btnGacReload.disabled = running;
    const repos = (res && res.repos) || [];
    const hidden = (res && res.hidden) || [];
    repos.forEach(function (st) { gacRows.appendChild(buildGacRow(st)); });
    btnGacFixUrls.classList.toggle("hidden", !repos.some(function (st) { return st.remoteClass === "https"; }));
    gacEmpty.classList.toggle("hidden", repos.length > 0);
    gacHidden.classList.toggle("hidden", hidden.length === 0);
    gacHiddenSummary.textContent = t("gac.hidden", { count: hidden.length });
    gacHiddenList.innerHTML = "";
    hidden.forEach(function (st) {
      const li = document.createElement("li");
      li.className = "break-all";
      li.textContent = st.path + " — " + st.remote;
      gacHiddenList.appendChild(li);
    });
    renderGacPending((res && res.pending) || []);
  }); }, function (err) { latest(function () {
    gacLoading.classList.add("hidden");
    btnGacReload.disabled = running;
    openError(t("gac.listFailed", { error: gaErrText(err) }));
  }); });
}

// renderGacPending lists the imported repositories not cloned yet, each
// with a Descartar button, and shows "Clonar pendentes" when there are any.
function renderGacPending(pending) {
  gacPendingList.innerHTML = "";
  gacPending.classList.toggle("hidden", pending.length === 0);
  btnGacClonePending.classList.toggle("hidden", pending.length === 0);
  pending.forEach(function (p) {
    const li = document.createElement("li");
    li.className = "flex items-center gap-2 text-sm";
    const text = document.createElement("span");
    text.className = "flex-1 min-w-0 break-all";
    text.textContent = p.path + " — " + p.url;
    li.appendChild(text);
    if (p.reason) {
      const badge = document.createElement("span");
      badge.className = "badge badge-sm badge-warning";
      badge.textContent = t("gac.blocked");
      badge.title = p.reason;
      li.appendChild(badge);
    }
    const discard = document.createElement("button");
    discard.className = "btn btn-xs btn-ghost";
    discard.textContent = t("gac.discard");
    discard.addEventListener("click", function () {
      const id = gacAccountId;
      openConfirm(t("gac.discardTitle"), t("gac.discardConfirm", { path: p.path }), function () {
        App.DiscardGitAccountPending(id, p.path).then(function () {
          loadGacRepos();
        }).catch(function (err) { openError(t("gac.discardFailed", { error: gaErrText(err) })); });
      });
    });
    li.appendChild(discard);
    gacPendingList.appendChild(li);
  });
}

btnGacClonePending.addEventListener("click", function () {
  if (!gacAccountId || btnGacClonePending.disabled) return;
  const id = gacAccountId;
  const count = gacPendingList.children.length;
  openConfirm(t("gac.clonePending"), t("gac.clonePendingConfirm", { count: count }), function () {
    beginRun(t("gac.clonePending"), function () { return App.CloneGitAccountPending(id); },
      { buttons: GAC_RUN_BUTTONS });
  });
});

btnGacReload.addEventListener("click", function () {
  if (gacAccountId && !btnGacReload.disabled) loadGacRepos();
});

btnGacConfigure.addEventListener("click", function () {
  gaOpenOnRender = gacAccountId;
  if (!selectMenuItem("git-accounts|")) gaOpenOnRender = "";
});

// --- Novo Repositório ---------------------------------------------------

let nrPicked = null; // the RemoteRepo picked from the account's list
let nrDestTouched = false; // the folder was typed, not derived

function openNewRepo() {
  if (!gacAccount) return;
  nrPicked = null;
  nrDestTouched = false;
  nrSource.value = "";
  nrRemoteName.value = "";
  nrDest.value = "";
  nrList.innerHTML = "";
  nrListNote.textContent = "";
  nrMode.value = "account";
  nrUpdate();
  showDialog(newRepoModal);
}

btnGacNew.addEventListener("click", function () {
  if (!btnGacNew.disabled) openNewRepo();
});

// nrSuggestedDest is the folder derived from what was picked or typed:
// the repository's last path segment.
function nrSuggestedDest() {
  let name = "";
  if (nrMode.value === "account" && nrPicked) name = nrPicked.name;
  if (nrMode.value === "clone") name = nrSource.value.trim();
  if (nrMode.value === "init") name = nrRemoteName.value.trim();
  name = name.replace(/\.git$/, "").replace(/\/+$/, "");
  const parts = name.split(/[/:]/).filter(Boolean);
  return parts.length ? parts[parts.length - 1] : "";
}

// nrUpdate shows the section of the chosen mode, derives the folder until
// it is typed, and enables the main button when the form is complete.
function nrUpdate() {
  const mode = nrMode.value;
  nrAccount.classList.toggle("hidden", mode !== "account");
  nrClone.classList.toggle("hidden", mode !== "clone");
  nrInit.classList.toggle("hidden", mode !== "init");
  btnNrGo.textContent = mode === "init" ? t("nr.init") : t("nr.clone");
  if (!nrDestTouched) nrDest.value = nrSuggestedDest();
  const dest = nrDest.value.trim();
  nrDestPreview.textContent = dest && gacAccount ? gacAccount.folder.replace(/\/+$/, "") + "/" + dest.replace(/^\.?\/+/, "") : "";
  const source = mode === "account" ? !!nrPicked : mode === "clone" ? nrSource.value.trim() !== "" : true;
  btnNrGo.disabled = !source || !dest || running;
}
nrMode.addEventListener("change", nrUpdate);
nrSource.addEventListener("input", nrUpdate);
nrRemoteName.addEventListener("input", nrUpdate);
nrDest.addEventListener("input", function () {
  nrDestTouched = nrDest.value.trim() !== "";
  nrUpdate();
});

btnNrLoad.addEventListener("click", function () {
  if (btnNrLoad.disabled || !gacAccountId) return;
  btnNrLoad.disabled = true;
  nrListNote.textContent = t("nr.querying", { host: gacAccount ? gacAccount.host : t("nr.provider") });
  nrList.innerHTML = "";
  App.ListGitAccountRemoteRepos(gacAccountId).then(function (res) {
    const repos = (res && res.repos) || [];
    nrListNote.textContent = t(res.withToken ? "nr.listAll" : "nr.listPublic", { count: repos.length }) + (res.note ? " " + res.note : "");
    repos.forEach(function (r) { nrList.appendChild(buildNrItem(r)); });
  }).catch(function (err) {
    nrListNote.textContent = "";
    openError(t("nr.listFailed", { error: gaErrText(err) }));
  }).finally(function () {
    btnNrLoad.disabled = false;
  });
});

function buildNrItem(r) {
  const li = document.createElement("li");
  const a = document.createElement("a");
  a.href = "#";
  a.className = "flex flex-col items-start gap-0";
  const title = document.createElement("span");
  title.className = "font-semibold";
  title.textContent = r.name + (r.private ? " 🔒" : "");
  a.appendChild(title);
  const sub = document.createElement("span");
  sub.className = "text-xs opacity-70";
  sub.textContent = r.localPath ? t("nr.cloned", { path: r.localPath }) : (r.description || "");
  a.appendChild(sub);
  if (r.localPath) {
    a.classList.add("menu-disabled");
    a.setAttribute("aria-disabled", "true");
  }
  a.addEventListener("click", function (ev) {
    ev.preventDefault();
    if (r.localPath) return;
    nrPicked = r;
    nrList.querySelectorAll("a").forEach(function (el) { el.classList.toggle("menu-active", el === a); });
    nrUpdate();
  });
  li.appendChild(a);
  return li;
}

btnNrGo.addEventListener("click", function () {
  if (btnNrGo.disabled || !gacAccountId) return;
  const id = gacAccountId;
  const dest = nrDest.value.trim();
  const mode = nrMode.value;
  if (mode === "init") {
    btnNrGo.disabled = true;
    App.InitGitAccountRepo(id, dest, nrRemoteName.value.trim()).then(function () {
      newRepoModal.close();
      loadGacRepos();
    }).catch(function (err) {
      openError(t("nr.initFailed", { error: gaErrText(err) }));
    }).finally(nrUpdate);
    return;
  }
  const source = mode === "account" ? nrPicked.sshUrl : nrSource.value.trim();
  // Checked first, so the dialog stays open to fix a refusal; the clone
  // itself then runs on the Execução tab.
  btnNrGo.disabled = true;
  App.CheckGitAccountClone(id, source, dest).then(function () {
    newRepoModal.close();
    beginRun(t("nr.run.clone", { path: dest }), function () { return App.CloneGitAccountRepo(id, source, dest); },
      { buttons: GAC_RUN_BUTTONS });
  }).catch(function (err) {
    openError(t("nr.cloneFailed", { error: gaErrText(err) }));
  }).finally(nrUpdate);
});
