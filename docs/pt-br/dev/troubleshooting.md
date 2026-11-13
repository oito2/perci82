🌐 [English](../../en/dev/troubleshooting.md) | **Português** | 🏠 [Índice](../index.md)

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

## O instalador diz "No Perci82 Dev release has been published yet."

O script não encontrou nenhuma versão `dev-vX.Y.Z` no GitHub (rascunhos e pré-lançamentos não contam). Compile a partir do código-fonte — veja [Instalar](installation.md#compilar-a-partir-do-código-fonte).

## O instalador manteve o Perci antigo: "No terminal to ask on"

O script pergunta no terminal antes de remover o Perci antigo; sem um terminal para perguntar, ele o mantém. Rode o script de novo num terminal ou remova manualmente os arquivos que ele listou.

## Abrir o Perci82 Dev só traz de volta uma janela antiga

O Perci82 Dev roda uma vez por sessão, então abri-lo de novo traz para a frente a janela que já está rodando — inclusive uma aberta oculta na bandeja no login. Depois de instalar ou compilar uma versão nova, encerre a que está rodando com **Sair** no menu da bandeja (ou feche a janela, quando a bandeja está desligada) e abra de novo.

## O ícone da bandeja não aparece

O ícone precisa de um ambiente gráfico com suporte a StatusNotifierItem. KDE, XFCE e COSMIC o mostram nativamente; no GNOME, instale e ative a extensão "AppIndicator and KStatusNotifierItem Support".

## "Outra ação do Perci82 System está em andamento — aguarde ela terminar."

Ações que precisam de permissão de administrador nunca rodam nos dois aplicativos ao mesmo tempo. Espere a ação no Perci82 System terminar e tente de novo.

## A janela não fecha

A janela fica aberta enquanto uma ação está rodando. Acompanhe-a na aba **Execução** e feche a janela depois que ela terminar.

## Uma ação falha logo depois da janela de senha

A autenticação foi cancelada ou a senha estava errada. Rode a ação de novo. O aplicativo usa a janela de autenticação do ambiente gráfico (`pkexec`), então um agente de autenticação do PolicyKit precisa estar rodando, como nas sessões gráficas comuns.

## A Visão Geral mostra "Docker inacessível"

O Docker não está instalado, não está rodando ou o seu usuário não tem acesso a ele. Instale o Docker em [Ambiente → Pré-requisitos](environment.md): ele adiciona o seu usuário ao grupo `docker`, o que só vale depois de você sair e entrar de novo na sessão.

## "secret-tool não encontrado — instale o Libsecret Tools em Ambiente → Pré-requisitos"

O token de API de uma conta Git é guardado no chaveiro do sistema pelo `secret-tool`. Instale o **Libsecret Tools** em [Ambiente → Pré-requisitos](environment.md) e salve a conta de novo.

## "Caminho não escolhido no seletor de arquivos"

Pastas e arquivos em que uma ação grava precisam ser escolhidos no seletor de arquivos do próprio aplicativo, na mesma sessão. Escolha a pasta de novo pelo botão da tela em vez de digitá-la.

## "Não foi possível verificar atualizações"

A Visão Geral consulta a API do GitHub para procurar uma versão nova. Confira a conexão de rede e clique em **Verificar** de novo; a API do GitHub também limita quantas vezes um mesmo endereço pode consultar.
