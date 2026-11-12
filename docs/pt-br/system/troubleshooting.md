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

## O instalador diz "No Perci82 System release has been published yet."

O script não encontrou nenhuma versão `system-vX.Y.Z` no GitHub (rascunhos e pré-lançamentos não contam). Compile a partir do código-fonte — veja [Instalar](installation.md#compilar-a-partir-do-código-fonte).

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

A Visão Geral consulta a API do GitHub para procurar uma nova versão. Confira a conexão de rede e clique em **Verificar** de novo; a API do GitHub também limita quantas vezes um mesmo endereço pode consultar.
