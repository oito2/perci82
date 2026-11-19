🌐 [English](../../en/system/uninstallation.md) | **Português** | 🏠 [Índice](../index.md)

---

# Desinstalar o Perci82 System

## O que a instalação coloca no sistema

O pacote `perci82-system` contém o binário, a entrada no menu, os ícones e a licença; o gerenciador de pacotes os remove. O resto fica até ser removido como abaixo.

| Item | Caminho |
|---|---|
| Binário | `/usr/bin/perci82-system` |
| Entrada no menu de aplicativos | `/usr/share/applications/io.github.oito2.perci82.system.desktop` |
| Ícones | `/usr/share/icons/hicolor/<tamanho>x<tamanho>/apps/io.github.oito2.perci82.system.png` (16, 22, 24, 32, 48, 64, 128, 256 e 512), `/usr/share/icons/hicolor/scalable/apps/io.github.oito2.perci82.system.svg`, `/usr/share/icons/hicolor/symbolic/apps/io.github.oito2.perci82.system-symbolic.svg` |
| Licença | `/usr/share/doc/perci82-system/copyright` (`.deb`) ou `/usr/share/licenses/perci82-system/LICENSE` (`.rpm`) |
| Repositório de pacotes | Compartilhado com o Perci82 Dev. `apt`: `/etc/apt/sources.list.d/perci82.sources` e `/usr/share/keyrings/perci82-archive-keyring.asc`; `dnf`: `/etc/yum.repos.d/perci82.repo`, `/etc/pki/rpm-gpg/RPM-GPG-KEY-perci82` e a chave importada no rpm (`gpg-pubkey-3c2a545f`) |
| Configurações | `~/.perci/config.yaml` — compartilhado com o Perci82 Dev; as chaves do Perci82 System estão em [Configurações](settings.md#onde-as-configurações-ficam-salvas) |

O Perci82 System não tem entrada de abertura no login, ícone na bandeja nem serviço.

## Pelo aplicativo

1. Abra a **Visão Geral** e clique em **Desinstalar Perci82 System**, na seção **Desinstalar**.
2. Na janela de confirmação, marque **Também remover as configurações do Perci82 System** se quiser apagar também as configurações (idioma, tema, escopo do Flatpak e o registro da pós-instalação). As configurações do Perci82 Dev são mantidas.
3. Clique em **Desinstalar**. A aba **Execução** mostra o andamento.
4. Ao terminar, a janela **Limpeza concluída** mostra o comando que remove o pacote, com **Copiar**. Copie o comando, clique em **Fechar o aplicativo** e rode o comando num terminal:

   ```bash
   sudo apt remove perci82-system   # apt
   sudo dnf remove perci82-system   # dnf
   ```

Com a opção de configurações, as chaves do Perci82 System saem do `~/.perci/config.yaml`; a pasta `~/.perci` só é apagada quando nem o Perci82 Dev nem o Perci antigo estão instalados e não sobrou mais nada nela.

## Manualmente

```bash
sudo apt remove perci82-system   # ou: sudo dnf remove perci82-system
```

Para remover as configurações, apague as chaves `postinstall_applied`, `postinstall_machine_id`, `flatpak_scope`, `system_gui_theme` e `system_language` do `~/.perci/config.yaml`. Apague a pasta `~/.perci` inteira só quando o Perci82 Dev não estiver instalado: ele guarda nela as próprias configurações e dados.

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

Desinstalar remove o aplicativo, não o que as telas dele instalaram. Desfaça isso antes, se quiser remover:

| Tela | Deixa | Como remover |
|---|---|---|
| [Fontes](fonts.md) | Pacotes e `~/.local/share/fonts/perci/` da JetBrains Mono | Desmarque as fontes e clique em **Executar** |
| [Templates](templates.md) | Arquivos na sua pasta de modelos | Desmarque os templates e clique em **Executar** |
| [Flatpaks](flatpaks.md) | Aplicativos Flatpak | Desmarque-os e clique em **Executar** |
| [Pós-instalação](post-install.md) | Pacotes, repositórios, `/etc/sysctl.d/99-perci.conf`, `/swapfile` e a linha dele no `/etc/fstab`, o serviço `fstrim.timer` | Manualmente, com as ferramentas da sua distribuição — a pós-instalação não tem como desfazer |
