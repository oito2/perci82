🌐 [English](../../en/system/overview.md) | **Português** | 🏠 [Índice](../index.md)

---

# Visão Geral

A primeira tela do Perci82 System: o que ele detectou, a versão, as atualizações e a desinstalação.

![Visão Geral](../../img/system-overview.png)

## A janela

As seções do Perci82 System são abas no alto da janela: **Visão Geral**, **Pós-instalação**, **Atualizar**, **Fontes**, **Templates**, **Flatpaks** e **Configurações**. As telas que executam ações têm duas abas internas:

- **Itens** — o que aplicar, instalar ou remover, e o botão que inicia a ação.
- **Execução** — a ação em andamento: a barra de progresso e a saída dos comandos.

Só uma ação roda por vez, e a janela não fecha enquanto uma está rodando. Ações que precisam de permissão de administrador nunca rodam no Perci82 System e no Perci82 Dev ao mesmo tempo: enquanto um aplicativo executa uma delas, o outro recusa com uma mensagem que diz qual aplicativo está ocupado.

## Cards

| Card | Mostra |
|---|---|
| **Sistema** | O perfil de [pós-instalação](post-install.md) da máquina (ex.: "ZorinOS 18.1 Lite"); num sistema sem perfil, o nome e a versão da distribuição |
| **Ambiente** | O ambiente gráfico (GNOME, KDE Plasma, Cinnamon, XFCE, Budgie, COSMIC…) |
| **Versão** | A versão do Perci82 System, com o selo **em dia** ou **nova versão** depois de cada verificação |

## Atualização

O aplicativo procura uma versão mais nova quando a tela abre (de novo, no máximo a cada 10 minutos); **Verificar** procura na hora. Ele lê a versão mais recente do Perci82 System no site do repositório de pacotes (`https://oito2.github.io/perci82/latest/system.json`) e, quando isso falha, nas releases do Perci82 System no GitHub (tags `system-vX.Y.Z`).

Quando existe uma versão mais nova e o aplicativo foi instalado pelo pacote, a tela mostra o comando que a instala, com **Copiar**. Rode-o num terminal:

```bash
sudo apt update && sudo apt install --only-upgrade perci82-system   # apt
sudo dnf upgrade --refresh perci82-system                             # dnf
```

O `apt upgrade` e o `dnf upgrade` também o atualizam junto com o resto do sistema. O aplicativo nunca substitui os próprios arquivos. Uma compilação a partir do código-fonte informa a versão `dev` e não mostra comando.

## Desinstalar

**Desinstalar Perci82 System** abre uma janela de confirmação. **Também remover as configurações do Perci82 System** (desmarcada sempre que a janela abre) remove as configurações; depois, uma janela mostra o comando que remove o pacote — o binário, a entrada no menu de aplicativos e os ícones. Veja [Desinstalar](uninstallation.md).
