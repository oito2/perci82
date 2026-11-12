🌐 [English](../../en/system/uninstallation.md) | **Português** | 🏠 [Índice](../index.md)

---

# Desinstalar o Perci82 System

## O que a instalação coloca no sistema

| Item | Caminho |
|---|---|
| Binário | `/usr/local/bin/perci82-system` |
| Entrada no menu de aplicativos | `/usr/share/applications/io.github.oito2.perci82.system.desktop` |
| Ícones | `/usr/share/icons/hicolor/<tamanho>x<tamanho>/apps/io.github.oito2.perci82.system.png` (16, 22, 24, 32, 48, 64, 128, 256 e 512), `/usr/share/icons/hicolor/scalable/apps/io.github.oito2.perci82.system.svg`, `/usr/share/icons/hicolor/symbolic/apps/io.github.oito2.perci82.system-symbolic.svg` |
| Configurações | `~/.perci/config.yaml` — compartilhado com o Perci82 Dev; as chaves do Perci82 System estão em [Configurações](settings.md#onde-as-configurações-ficam-salvas) |

O Perci82 System não tem entrada de abertura no login, ícone na bandeja nem serviço.

## Pelo aplicativo

1. Abra a **Visão Geral** e clique em **Desinstalar Perci82 System**, na seção **Desinstalar**.
2. Na janela de confirmação, marque **Também remover as configurações do Perci82 System** se quiser apagar também as configurações (idioma, tema, escopo do Flatpak e o registro da pós-instalação). As configurações do Perci82 Dev são mantidas.
3. Clique em **Desinstalar** e digite a sua senha quando pedida. A aba **Execução** mostra o andamento.

O binário, a entrada no menu e os ícones são removidos. Com a opção de configurações, as chaves do Perci82 System saem do `~/.perci/config.yaml`; a pasta `~/.perci` só é apagada quando nem o Perci82 Dev nem o Perci antigo estão instalados e não sobrou mais nada nela.

## Manualmente

```bash
sudo rm -f /usr/local/bin/perci82-system \
  /usr/share/applications/io.github.oito2.perci82.system.desktop \
  /usr/share/icons/hicolor/*/apps/io.github.oito2.perci82.system.png \
  /usr/share/icons/hicolor/scalable/apps/io.github.oito2.perci82.system.svg \
  /usr/share/icons/hicolor/symbolic/apps/io.github.oito2.perci82.system-symbolic.svg
sudo gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor
```

Para remover as configurações, apague as chaves `postinstall_applied`, `postinstall_machine_id`, `flatpak_scope`, `system_gui_theme` e `system_language` do `~/.perci/config.yaml`. Apague a pasta `~/.perci` inteira só quando o Perci82 Dev não estiver instalado: ele guarda nela as próprias configurações e dados.

## O que as ações do aplicativo deixam

Desinstalar remove o aplicativo, não o que as telas dele instalaram. Desfaça isso antes, se quiser remover:

| Tela | Deixa | Como remover |
|---|---|---|
| [Fontes](fonts.md) | Pacotes e `~/.local/share/fonts/perci/` da JetBrains Mono | Desmarque as fontes e clique em **Executar** |
| [Templates](templates.md) | Arquivos na sua pasta de modelos | Desmarque os templates e clique em **Executar** |
| [Flatpaks](flatpaks.md) | Aplicativos Flatpak | Desmarque-os e clique em **Executar** |
| [Pós-instalação](post-install.md) | Pacotes, repositórios, `/etc/sysctl.d/99-perci.conf`, `/swapfile` e a linha dele no `/etc/fstab`, o serviço `fstrim.timer` | Manualmente, com as ferramentas da sua distribuição — a pós-instalação não tem como desfazer |
