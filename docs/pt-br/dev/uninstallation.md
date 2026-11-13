🌐 [English](../../en/dev/uninstallation.md) | **Português** | 🏠 [Índice](../index.md)

---

# Desinstalar o Perci82 Dev

## O que a instalação coloca no sistema

| Item | Caminho |
|---|---|
| Binário | `/usr/local/bin/perci82-dev` |
| Entrada no menu de aplicativos | `/usr/share/applications/io.github.oito2.perci82.dev.desktop` |
| Ícones | `/usr/share/icons/hicolor/<tamanho>x<tamanho>/apps/io.github.oito2.perci82.dev.png` (16, 22, 24, 32, 48, 64, 128, 256 e 512), `/usr/share/icons/hicolor/scalable/apps/io.github.oito2.perci82.dev.svg`, `/usr/share/icons/hicolor/symbolic/apps/io.github.oito2.perci82.dev-symbolic.svg` |
| Abertura no login | `~/.config/autostart/io.github.oito2.perci82.dev.desktop` — só enquanto a bandeja do sistema está ligada ([Configurações](settings.md#bandeja-do-sistema)) |
| Configurações | `~/.perci/config.yaml` — compartilhado com o Perci82 System; as chaves do Perci82 Dev estão em [Configurações](settings.md#onde-as-configurações-ficam-salvas) |
| Dados | `~/.perci/appstack/` (arquivos do stack Docker) e `~/.perci/git/` (os arquivos gitconfig das contas Git) |

## Pelo aplicativo

1. Abra a **Visão Geral** e clique em **Desinstalar Perci82 Dev** na seção **Desinstalar**.
2. Na janela de confirmação, marque o que quiser (todas as opções começam desmarcadas):
   - **Backups antes de remover** — **Docker (contêineres e pastas de desenvolvimento)**, **Repositórios (contas Git)** e **Contextos de IA (pastas registradas)**. Cada backup é salvo num arquivo separado, no formato que a tela de Exportar/Importar correspondente lê. As janelas de salvar abrem uma depois da outra depois que você confirma; se você cancelar uma delas, nada é removido. Um backup que falha também interrompe a desinstalação antes de remover qualquer coisa.
   - **Remover também** — **Os contêineres Docker (Nginx, MariaDB e apps)**: os contêineres e o comando de cada app em `~/.local/bin` são removidos; os dados em disco, dentro do workspace, são mantidos.
   - **Remover também** — **As configurações do Perci82 Dev**: as chaves dele no `~/.perci/config.yaml`, a pasta `~/.perci/git` e, só quando os contêineres Docker também são removidos (eles ainda a usam), `~/.perci/appstack` — menos `~/.perci/appstack/backups`, os backups do MariaDB, que são sempre mantidos —, as regras `includeIf` das contas Git no `~/.gitconfig` e os tokens das contas no chaveiro.
3. Clique em **Desinstalar** e digite a sua senha quando for pedida. A aba **Execução** mostra o andamento, e a janela fecha quando termina.

O binário, a entrada no menu, os ícones e a abertura no login são removidos. A pasta `~/.perci` em si só é removida quando nem o Perci82 System nem o Perci antigo estão instalados e não sobrou mais nada nela. Chaves SSH, repositórios e as configurações do Perci82 System são sempre mantidos.

## Manualmente

```bash
sudo rm -f /usr/local/bin/perci82-dev \
  /usr/share/applications/io.github.oito2.perci82.dev.desktop \
  /usr/share/icons/hicolor/*/apps/io.github.oito2.perci82.dev.png \
  /usr/share/icons/hicolor/scalable/apps/io.github.oito2.perci82.dev.svg \
  /usr/share/icons/hicolor/symbolic/apps/io.github.oito2.perci82.dev-symbolic.svg
sudo gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor
rm -f ~/.config/autostart/io.github.oito2.perci82.dev.desktop
```

Para remover as configurações, apague as chaves `workspace_path`, `docker`, `git_accounts`, `agent_skills`, `mcp_servers`, `mcp_binaries`, `moodle_plugins`, `ai_contexts`, `tray_enabled`, `gui_theme`, `language` e `project_notes_path` do `~/.perci/config.yaml`, e as pastas `~/.perci/appstack` e `~/.perci/git`. Apague a pasta `~/.perci` inteira só quando o Perci82 System não estiver instalado: ele guarda as configurações dele lá.

## O que as ações do aplicativo deixam

Desinstalar remove o aplicativo, não o que as telas dele instalaram ou criaram. Desfaça isso antes, se quiser removê-los:

| Tela | Deixa | Como remover |
|---|---|---|
| [Ambiente](environment.md) | Pacotes, SDKs, IDEs, terminais e apps de IA | Remova-os nas telas deles antes de desinstalar |
| [Docker](docker.md) | Os contêineres (a menos que a desinstalação os remova), as imagens `perci-php*`, os dados em `<workspace>/localhost/`, a autoridade certificadora do mkcert | **Gerenciar Contêineres** remove os contêineres; imagens e dados com `docker image rm` e manualmente |
| [Docker → Pastas de Desenvolvimento](docker.md) | Unidades de montagem em `/etc/systemd/system/` | Remova cada pasta nessa tela antes de desinstalar |
| [Repositórios](repositories.md) | Chaves SSH e repositórios clonados | Manualmente; a desinstalação nunca os remove |
| [Contextos de IA](ai-contexts.md) | Os arquivos de contexto em cada projeto e a pasta de anotações de projeto | Manualmente; a desinstalação nunca mexe neles |
| [Skills & MCPs](skills-mcps.md) | Skills instaladas, os registros de MCP nos agentes e `~/.local/bin/build82` / `~/.local/bin/sync82` | **Remover** em cada linha antes de desinstalar — depois que as configurações são removidas, o Perci82 Dev não sabe mais quais desses arquivos ele instalou |
