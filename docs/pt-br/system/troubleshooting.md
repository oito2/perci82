🌐 [English](../../en/system/troubleshooting.md) | **Português** | 🏠 [Índice](../index.md)

---

# Solução de problemas

## O aplicativo não abre: `libwebkitgtk-6.0.so.4` ou `libgtk-4.so.1` não encontrada

Falta a biblioteca de execução do GTK 4 ou do WebKitGTK 6.0. Instale-a:

```bash
# Debian/Ubuntu
sudo apt install -y libgtk-4-1 libwebkitgtk-6.0-4
# Fedora
sudo dnf install -y gtk4 webkitgtk6.0
```

## O instalador diz "This release has no WebKitGTK 6.0"

A versão da distribuição é mais antiga que as suportadas (Debian 13, Ubuntu 24.04, Linux Mint 22 ou mais recentes): ela não tem o `libwebkitgtk-6.0-4`, de que o Perci82 System precisa. Atualize a distribuição.

## O instalador diz que a chave do repositório "does not have the expected fingerprint"

A chave baixada de `https://oito2.github.io/perci82/perci82.asc` não é a que o script espera, então ele para antes de alterar qualquer coisa. Confira a rede (um proxy pode estar respondendo no lugar do site) e rode de novo; se continuar falhando, avise nas issues do repositório.

## Uma versão antiga continua abrindo depois de instalar o pacote

O Perci82 System 1.1 e anteriores eram instalados como `/usr/local/bin/perci82-system`, que vem antes do `/usr/bin/perci82-system` do pacote no `PATH`. Remova o arquivo antigo:

```bash
sudo rm -f /usr/local/bin/perci82-system
```

## O instalador manteve o Perci antigo: "No terminal to ask on"

O script pergunta no terminal antes de remover o Perci antigo; sem terminal para perguntar, ele o mantém. Rode o script de novo num terminal, ou remova manualmente os arquivos que ele listou.

## "Outra ação do Perci82 Dev está em andamento — aguarde ela terminar."

Ações que precisam de permissão de administrador nunca rodam nos dois aplicativos ao mesmo tempo. Espere a ação do Perci82 Dev terminar e tente de novo.

## A janela não fecha

A janela fica aberta enquanto uma ação está rodando. Acompanhe na aba **Execução** e feche a janela quando ela terminar.

## A pós-instalação diz que o sistema não é suportado

A pós-instalação só tem itens para os sistemas listados em [Pós-instalação](post-install.md#sistemas-suportados). As outras telas funcionam em qualquer distribuição com `apt` ou `dnf`.

## Uma ação falha logo depois da janela de senha

A autenticação foi cancelada ou a senha estava errada. Rode a ação de novo. O aplicativo usa a janela de autenticação do ambiente gráfico (`pkexec`), então um agente de autenticação do PolicyKit precisa estar rodando, como acontece nas sessões desktop comuns.

## "Não foi possível verificar atualizações"

A Visão Geral lê a versão mais recente no site do repositório de pacotes e, quando não consegue, na API do GitHub, que limita quantas vezes um mesmo endereço pode consultar (60 vezes por hora sem login; mais com o GitHub CLI logado). Confira a conexão de rede e clique em **Verificar** de novo.
