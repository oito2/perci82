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

// Perci82 Dev screen registrations: which container each menu item shows,
// how it renders and what refreshes it after an action. Loaded last, once
// every screen script has defined its render functions.
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

SCREENS["docker-create"] = { el: dockercreateContainer, render: renderDockerCreateScreen };
SCREENS["docker-manage"] = { el: dockerlistContainer, render: renderDockerListScreen };
SCREENS["docker-transfer"] = { el: transferContainer, alsoShow: [tabLabelTransferImport], render: renderTransferScreen };
SCREENS.moodleplugins = { el: moodlepluginsContainer, alsoShow: [tabLabelMpMounted], render: renderMoodlePluginsScreen };
SCREENS["git-accounts"] = { el: gitaccountsContainer, alsoShow: [tabLabelGitIdentity], render: renderGitAccountsScreen };
SCREENS["git-account"] = { el: gitaccountContainer, render: renderGitAccountScreen };
SCREENS["git-transfer"] = { el: gittransferContainer, alsoShow: [tabLabelGitImport], render: renderGitTransferScreen };
SCREENS.agentskills = { el: agentskillsContainer, render: renderAgentSkillsScreen };
SCREENS.aicontext = { el: aicontextContainer, alsoShow: [tabLabelAicDefine, tabLabelAicTransfer], render: renderAIContextScreen };
SCREENS.mcpservers = { el: mcpserversContainer, render: renderMCPServersScreen };

// Item table screens (Pré-requisitos, Linguagens e SDKs, IDEs): re-fetches
// what is installed now after an action.
SCREEN_FAMILIES.push({
  has: function (id) { return !!ITEM_TABLE_SCREENS[id]; },
  el: itemTableContainer,
  render: renderItemTableScreen,
  done: function () { renderItemTableScreen(window.__selectedItem); },
});

// Creating Nginx/MariaDB may have just disabled that option in the select
// (singleton) — re-fetches the whole catalog.
ACTION_DONE_HANDLERS["docker-create"] = function () { renderDockerCreateScreen(); };
// Recriar/Editar/Backup/Restore — re-fetches the whole table.
ACTION_DONE_HANDLERS["docker-manage"] = function () { renderDockerListScreen(); };
// Importar — re-reads what can be exported and the import file's items
// (an imported plugin mount now shows as already mounted).
ACTION_DONE_HANDLERS["docker-transfer"] = function () { refreshTransferScreen(); };
// Criar/Reiniciar/Remover — re-fetches the table and re-checks the form.
ACTION_DONE_HANDLERS.moodleplugins = function () { loadMpRows(); checkMpForm(); };
// Identidade padrão — re-fetches the global identity.
ACTION_DONE_HANDLERS["git-accounts"] = function () { renderGitIdentityTab(); };
// Verificar/Baixar — re-reads the account's repositories.
ACTION_DONE_HANDLERS["git-account"] = function () { gacSetReady(gacReady); loadGacRepos(); };
// Importar — re-reads both checklists and the menu (new accounts).
ACTION_DONE_HANDLERS["git-transfer"] = function () { refreshGitTransferScreen(); };
ACTION_DONE_HANDLERS.agentskills = function () { renderAgentSkillsTable(); };
ACTION_DONE_HANDLERS.aicontext = function () { refreshAIContextScreen(); };
ACTION_DONE_HANDLERS.mcpservers = function () { renderMCPServersTable(); };
