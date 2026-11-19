🌐 [English](../../en/dev/uninstallation.md) | **Português** | 🏠 [Índice](../index.md)

---

# Desinstalar o Perci82 Dev

## O que a instalação coloca no sistema

O pacote `perci82-dev` contém o binário, a entrada no menu, os ícones e a licença; o gerenciador de pacotes os remove. O resto fica até ser removido como abaixo.

| Item | Caminho |
|---|---|
| Binário | `/usr/bin/perci82-dev` |
| Entrada no menu de aplicativos | `/usr/share/applications/io.github.oito2.perci82.dev.desktop` |
| Ícones | `/usr/share/icons/hicolor/<tamanho>x<tamanho>/apps/io.github.oito2.perci82.dev.png` (16, 22, 24, 32, 48, 64, 128, 256 e 512), `/usr/share/icons/hicolor/scalable/apps/io.github.oito2.perci82.dev.svg`, `/usr/share/icons/hicolor/symbolic/apps/io.github.oito2.perci82.dev-symbolic.svg` |
| Licença | `/usr/share/doc/perci82-dev/copyright` (`.deb`) ou `/usr/share/licenses/perci82-dev/LICENSE` (`.rpm`) |
| Repositório de pacotes | Compartilhado com o Perci82 System. `apt`: `/etc/apt/sources.list.d/perci82.sources` e `/usr/share/keyrings/perci82-archive-keyring.asc`; `dnf`: `/etc/yum.repos.d/perci82.repo`, `/etc/pki/rpm-gpg/RPM-GPG-KEY-perci82` e a chave importada no rpm (`gpg-pubkey-3c2a545f`) |
| Abertura no login | `~/.config/autostart/io.github.oito2.perci82.dev.desktop` — só enquanto a bandeja do sistema está ligada ([Configurações](settings.md#bandeja-do-sistema)) |
| Configurações | `~/.perci/config.yaml` — compartilhado com o Perci82 System; as chaves do Perci82 Dev estão em [Configurações](settings.md#onde-as-configurações-ficam-salvas) |
| Dados | `~/.perci/appstack/` (arquivos do stack Docker) e `~/.perci/git/` (os arquivos gitconfig das contas Git) |

## Pelo aplicativo

1. Abra a **Visão Geral** e clique em **Desinstalar Perci82 Dev**, na seção **Desinstalar**.
2. Na janela de confirmação, marque o que quiser (todas as opções começam desmarcadas):
   - **Backups antes de remover** — **Docker (contêineres e pastas de desenvolvimento)**, **Repositórios (contas Git)** e **Contextos de IA (pastas registradas)**. Cada backup é salvo num arquivo separado, no formato que a tela de Exportar/Importar correspondente lê. As janelas de salvar abrem uma depois da outra depois que você confirma; se você cancelar uma delas, nada é removido. Um backup que falha também interrompe a desinstalação antes de remover qualquer coisa.
   - **Remover também** — **Os contêineres Docker (Nginx, MariaDB, Selenium e apps)**: os contêineres e o comando de cada app em `~/.local/bin` são removidos; os dados em disco, dentro do workspace, são mantidos.
   - **Remover também** — **As configurações do Perci82 Dev**: as chaves dele no `~/.perci/config.yaml`, a pasta `~/.perci/git` e, só quando os contêineres Docker também são removidos (eles ainda a usam), `~/.perci/appstack` — menos `~/.perci/appstack/backups`, os backups do MariaDB, que são sempre mantidos —, as regras `includeIf` das contas Git no `~/.gitconfig` e os tokens das contas no chaveiro.
3. Clique em **Desinstalar** e digite a sua senha se for pedida. A aba **Execução** mostra o andamento.
4. Ao terminar, a janela **Limpeza concluída** mostra o comando que remove o pacote, com **Copiar**. Copie o comando, clique em **Fechar o aplicativo** e rode o comando num terminal:

   ```bash
   sudo apt remove perci82-dev   # apt
   sudo dnf remove perci82-dev   # dnf
   ```

A abertura no login é removida. A pasta `~/.perci` em si só é removida quando nem o Perci82 System nem o Perci antigo estão instalados e não sobrou mais nada nela. Chaves SSH, repositórios, os contextos de IA personalizados (`~/.perci/user-contexts`) e as configurações do Perci82 System são sempre mantidos.

## Manualmente

```bash
sudo apt remove perci82-dev   # ou: sudo dnf remove perci82-dev
rm -f ~/.config/autostart/io.github.oito2.perci82.dev.desktop
```

Para remover as configurações, apague as chaves `workspace_path`, `docker`, `git_accounts`, `agent_skills`, `mcp_servers`, `mcp_binaries`, `moodle_plugins`, `ai_contexts`, `tray_enabled`, `gui_theme`, `language` e `project_notes_path` do `~/.perci/config.yaml`, e as pastas `~/.perci/appstack` e `~/.perci/git`. Apague a pasta `~/.perci` inteira só quando o Perci82 System não estiver instalado: ele guarda as configurações dele lá.

## Remover o repositório de pacotes

Quando nenhum dos dois aplicativos Perci82 estiver instalado, remova o repositório e a chave dele:

```bash
# apt
sudo rm -f /etc/apt/sources.list.d/perci82.sources /usr/share/keyrings/perci82-archive-keyring.asc
sudo apt update
# dnf
sudo rm -f /etc/yum.repos.d/perci82.repo /etc/pki/rpm-gpg/RPM-GPG-KEY-perci82
sudo rpm -e gpg-pubkey-3c2a545f
```

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
