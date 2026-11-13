🌐 [English](../../en/dev/overview.md) | **Português** | 🏠 [Índice](../index.md)

---

# Visão Geral

A primeira tela do Perci82 Dev: um resumo dos contêineres Docker e dos repositórios, a versão, as atualizações e a desinstalação.

![Visão Geral](../../img/dev-overview.png)
<!-- TODO: capture a screenshot of Perci82 Dev's Overview -->

## A janela

As seções do Perci82 Dev são abas no alto da janela: **Visão Geral**, **Ambiente**, **Docker**, **Repositórios**, **Contextos de IA**, **Skills & MCPs** e **Configurações**. Cada seção tem uma lista lateral com as telas dela; em **Repositórios**, as contas Git ficam agrupadas em **Contas**, um item por conta. As telas que executam ações têm as próprias abas e mais **Execução**, que mostra a ação em andamento: a barra de progresso e a saída dos comandos.

Só uma ação roda por vez, e a janela não fecha enquanto uma está rodando. Ações que precisam de permissão de administrador nunca rodam no Perci82 Dev e no Perci82 System ao mesmo tempo: enquanto um aplicativo executa uma delas, o outro recusa com uma mensagem que diz qual aplicativo está ocupado.

O Perci82 Dev roda uma vez por sessão: abri-lo de novo traz para a frente a janela que já está aberta.

## Cards

| Card | Mostra |
|---|---|
| **Contêineres Docker** | Quantos dos contêineres registrados estão rodando (ex.: "2 de 3 rodando"); **Nenhum** sem contêineres, **Docker inacessível** quando não é possível consultar o Docker |
| **Repositórios** | O número de contas Git e de repositórios encontrados nas pastas delas; **Nenhuma conta** sem contas |
| **Versão** | A versão do Perci82 Dev, com o selo **em dia** ou **nova versão** depois de cada verificação |

## Atualização

O aplicativo procura uma versão mais nova quando a tela abre (de novo, no máximo a cada 10 minutos); **Verificar** procura na hora. Ele olha só as versões do próprio Perci82 Dev (tags `dev-vX.Y.Z`) no GitHub.

Quando existe uma versão mais nova, aparece **Atualizar**. Ele baixa o novo binário, confere o SHA-256 pelo `checksums.txt` da release, substitui o binário instalado e mostra o andamento na aba **Execução**. Depois, reinicie o aplicativo.

Uma compilação a partir do código-fonte informa a versão `dev` e não se atualiza sozinha.

## Desinstalar

**Desinstalar Perci82 Dev** abre uma janela de confirmação com os backups do Docker, dos repositórios e dos contextos de IA, e as opções de remover os contêineres Docker e as configurações do Perci82 Dev. Veja [Desinstalar](uninstallation.md).

## Bandeja do sistema

Com **Mostrar o Perci82 Dev na bandeja do sistema** ligado ([Configurações](settings.md#bandeja-do-sistema)), o Perci82 Dev coloca um ícone na bandeja do sistema. O menu dele tem:

- **Abrir Aplicativo** — mostra a janela principal.
- **Contêineres** — abre uma janela compacta só com **Docker → Gerenciar Contêineres** e um botão **Abrir Aplicativo**.
- **Sair** — encerra o Perci82 Dev (recusado enquanto uma ação está rodando).

Fechar a janela principal passa a escondê-la na bandeja em vez de encerrar o aplicativo, e o Perci82 Dev abre oculto na bandeja quando você entra na sessão.
