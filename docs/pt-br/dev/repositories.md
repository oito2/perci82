🌐 [English](../../en/dev/repositories.md) | **Português** | 🏠 [Índice](../index.md)

---

# Repositórios

Contas Git e os seus repositórios. Cada conta tem uma pasta: todo repositório dentro dela, em qualquer subpasta, usa o nome, o e-mail e a chave SSH da conta nos commits — também no terminal. A lista lateral tem **Configurar Conta**, **Gerenciar Repositórios** e, em **Contas**, um item por conta cadastrada.

![Repositórios](../../img/dev-repositories.png)
<!-- TODO: capture a screenshot of an account in Repositories -->

## Como uma conta funciona

Uma conta é uma conta pessoal do GitHub ou do GitLab, no serviço público ou num servidor próprio. Ao salvá-la:

- a chave SSH é configurada (uma chave ed25519 nova em `~/.ssh/perci_<id>`, ou uma existente);
- é gravado `~/.perci/git/<id>.gitconfig` com `user.name`, `user.email` e um `core.sshCommand` que usa só a chave da conta e o arquivo `known_hosts` do próprio Perci82 Dev;
- é adicionada uma regra `includeIf` à configuração global do Git, para que esse arquivo valha em todo repositório dentro da pasta da conta:

```ini
[includeIf "gitdir:/home/voce/dev/github/"]
	path = /home/voce/.perci/git/pessoal.gitconfig
```

- as chaves de host publicadas de `github.com` e `gitlab.com` (porta 22) passam a ser confiáveis; qualquer outro servidor precisa ter a chave confirmada nos detalhes da conta;
- o token da API, quando informado, é guardado no chaveiro do sistema.

O `<id>` vem do nome da conta: letras minúsculas e dígitos unidos por `-`.

## Configurar Conta

A aba **Contas** lista as contas com **Conta**, **Provedor**, **Pasta** e o estado da **Chave SSH**. Clique no nome de uma conta para abrir os detalhes dela; os ícones da linha servem para **Editar** e **Remover**. **Nova conta** abre o formulário.

### Formulário da conta

| Campo | Observações |
|---|---|
| **Nome da conta** | Aparece no menu Repositórios. Até 40 caracteres, sem repetir |
| **Provedor** | GitHub ou GitLab |
| **Host** | Vazio para o serviço público (`github.com`, `gitlab.com`); preencha para um servidor próprio |
| **Porta SSH** | Só para um servidor próprio; vazio significa 22 |
| **Usuário** | O dono dos repositórios no provedor |
| **Nome nos commits** / **E-mail nos commits** | A identidade dos commits dentro da pasta da conta |
| **Pasta da conta** | Escolhida em **Selecionar pasta**. Precisa existir e não pode conter nem estar dentro da pasta de outra conta |
| **Chave SSH** | **Gerar uma chave nova (ed25519)**, com senha opcional; **Usar uma chave existente** (a `.pub` precisa estar ao lado dela); ou, ao editar, **Manter a chave atual** |
| **Token da API (opcional)** | Fica no chaveiro, nunca no `config.yaml`. Ao editar, **Remover o token salvo** apaga o token |

**Abrir página de chaves SSH** e **Abrir página de tokens** abrem as páginas do provedor no host digitado. O token só é usado para listar os repositórios da conta em **Novo Repositório**: sem ele, só os públicos aparecem. O formulário descreve o token a criar:

- GitHub: um token fine-grained com Repository access = All repositories e só **Metadata: Read-only**.
- GitLab: um personal access token só com o escopo `read_api`.

O token é guardado com o `secret-tool`, que precisa estar instalado.

### Detalhes da conta

Os detalhes mostram as configurações da conta e a **Chave pública** com um selo de estado:

| Estado | Significado |
|---|---|
| **pronta** | A chave não tem senha |
| **desbloqueada** | A chave tem senha e está carregada no agente SSH da sessão |
| **bloqueada** | A chave tem senha e não está carregada no agente |
| **sem agente SSH** | A chave tem senha e não há agente SSH nesta sessão |
| **não encontrada** | Falta o arquivo da chave ou a sua `.pub` |

- **Copiar** copia a chave pública; **Abrir chaves SSH no GitHub/GitLab** abre a página onde ela é cadastrada (como chave de autenticação).
- **Desbloquear chave** (só quando **bloqueada**): digite a senha e clique em **Desbloquear** para carregar a chave no agente SSH nesta sessão.
- **Chave do servidor** (só enquanto a chave do servidor não foi confirmada): **Verificar chave do servidor** pede as chaves ao servidor (`ssh-keyscan`) e mostra as impressões digitais; **Confiar nestas chaves** adiciona-as a `~/.perci/git/known_hosts`. Confira as impressões digitais com o administrador do servidor antes.
- **Testar conexão** conecta por SSH e passa quando o servidor cumprimenta o **Usuário** da conta. Fica desativado até a chave do servidor ser confirmada e a chave estar utilizável.

**Remover** esquece a conta: a regra `includeIf`, o arquivo gitconfig e o token dela são apagados. Os repositórios da pasta e a chave SSH continuam no disco; os repositórios passam a usar a identidade padrão.

### Identidade padrão

A aba **Identidade padrão** define o **Nome** e o **E-mail** dos commits feitos fora das pastas das contas (`git config --global user.name` / `user.email`). **Aplicar identidade padrão** grava os dois e, quando ainda não há `credential.helper` configurado, define um (libsecret quando disponível; senão, `cache`).

## Telas das contas

Cada item em **Contas** abre os repositórios daquela conta. O cabeçalho mostra o provedor, o usuário e a pasta, com **Recarregar** e **Configurar conta** (abre os detalhes da conta).

Quando a conta ainda não alcança o servidor — chave não encontrada, bloqueada, sem agente SSH ou chave do servidor não confirmada —, um aviso diz o motivo e os botões que usam a rede ficam desativados.

### Lista de repositórios

A lista traz todo repositório Git encontrado dentro da pasta da conta (pastas ocultas, `node_modules` e `vendor` são ignoradas). Abrir a tela não acessa a rede.

| Coluna | Mostra |
|---|---|
| **Repositório** | O caminho relativo à pasta da conta e a URL do `origin` |
| **Branch** | A branch atual, ou **(HEAD solto)** |
| **Último commit** | Há quanto tempo; a data e o assunto ao passar o mouse |
| **Alterações** | Selos de estado e quando o repositório foi verificado pela última vez |

| Selo | Significado |
|---|---|
| **em dia** | Nada a enviar nem a baixar desde a última verificação |
| **↓ n** / **↑ n** | Commits para baixar (desde a última verificação) / commits não enviados |
| **n modificado(s)** / **n novo(s)** | Arquivos rastreados alterados / arquivos fora do Git |
| **sem remote** | Não há remote `origin` |
| **URL HTTPS** | O `origin` usa HTTPS; com SSH vale a chave da conta |
| **sem upstream** | A branch atual não acompanha uma branch remota |
| **identidade divergente** | O repositório define o próprio `user.name` ou `user.email` |
| **erro ao ler** | Não foi possível ler o repositório |

Repositórios cujo `origin` é de outro dono ou de outro host não aparecem; uma nota recolhida lista esses repositórios.

### Barra de botões e ações da linha

| Botão | O que faz |
|---|---|
| **Verificar Alterações** | `git fetch --prune origin` em todo repositório com remote, sem alterar os arquivos |
| **Baixar Alterações** | Busca e avança a branch atual (só fast-forward). Repositórios com alterações não commitadas, sem upstream, com HEAD solto ou que divergiram do remoto são ignorados, com o motivo na aba **Execução** |
| **Novo Repositório** | Clona um repositório ou inicia um novo dentro da pasta da conta |
| **Ajustar todas as URLs** | Só quando algum repositório usa HTTPS: troca cada `origin` HTTPS da conta pela URL SSH equivalente, depois de confirmar que a chave alcança o repositório |
| **Clonar pendentes** | Só quando há repositórios pendentes (veja abaixo) |

Cada linha também tem **Verificar**, **Baixar** e um botão **+** que abre o diálogo do repositório. Ações longas rodam na aba **Execução**.

### Novo Repositório

**Como** escolhe um de três modos:

- **Escolher da lista da conta** — **Carregar repositórios da conta** lista os repositórios da conta no provedor (também os privados, com token); os já clonados aparecem marcados e não podem ser escolhidos.
- **Clonar por URL ou nome** — um nome (`perci`) ou uma URL de um repositório da conta; uma URL HTTPS é clonada por SSH. URLs de outro dono ou de outro host são recusadas.
- **Iniciar um repositório local** — `git init`; com **Nome no provedor (opcional)**, o remote `origin` também é definido. Crie o repositório vazio no site antes do primeiro push.

**Pasta do repositório** é relativa à pasta da conta (por exemplo `labs/perci`), precisa ser nova ou vazia e não pode estar dentro de outro repositório.

### Diálogo do repositório

O botão **+** abre três abas:

- **Configurações** — o **E-mail de contato** usado no `CODE_OF_CONDUCT.md` e no `SUPPORT.md`: o padrão da conta e, opcionalmente, um só deste repositório. **Corrigir identidade** (quando o repositório define a própria identidade): **Usar a identidade da conta** remove `user.name` e `user.email` da configuração própria do repositório; os commits já feitos não mudam.
- **Remote e SSH** — mostra **Já usa SSH**; ou **Migrar para SSH** com **Trocar para SSH** para um `origin` HTTPS; ou **Definir remote** para um repositório sem `origin` (o nome no provedor; o repositório ainda não precisa existir).
- **Arquivos** — arquivos da comunidade, comparados com o que o Perci82 Dev gera agora:

| Arquivo | Observações |
|---|---|
| `.gitignore` | Padrões por tipo de projeto detectado na raiz (Go, Flutter, Dart, Moodle, PHP, Node.js, Python, Ruby, Rust, Java, Shell), mais editores e sistema operacional; padrões genéricos quando nenhum é detectado. **Atualizar** só acrescenta os padrões que faltam |
| `LICENSE` | GNU GPL v3.0 (padrão), GNU AGPL v3.0, GNU LGPL v3.0, MIT, Apache 2.0, Mozilla Public License 2.0 ou BSD 3-Clause; MIT e BSD 3-Clause também pedem **Titular dos direitos** e **Ano** |
| `CODE_OF_CONDUCT.md` | Com espelho em português em `docs/pt-br/codigo-de-conduta.md`; precisa do e-mail de contato |
| `SUPPORT.md` | Com espelho em português em `docs/pt-br/suporte.md`; precisa do e-mail de contato |
| Modelos de issue | `.github/ISSUE_TEMPLATE/` no GitHub, `.gitlab/issue_templates/` no GitLab |
| Modelo de pull/merge request | `.github/PULL_REQUEST_TEMPLATE.md` no GitHub, `.gitlab/merge_request_templates/Default.md` no GitLab |
| `.github/FUNDING.yml` | **em breve** |

Cada linha oferece **Criar**, **Atualizar** ou **Remover**, conforme o estado. **Atualizar** substitui o arquivo e guarda a versão anterior em `<arquivo>.perci-bak`.

### Repositórios pendentes

Repositórios importados em **Gerenciar Repositórios** e ainda não clonados aparecem em **Pendentes**. Antes, cadastre a chave pública da conta no servidor e use **Testar conexão**. **Clonar pendentes** testa a conexão e clona os repositórios nas suas pastas; **Descartar** tira um deles da lista sem apagar nada.

## Gerenciar Repositórios

Leva contas e repositórios para outra máquina por um arquivo YAML. Esse arquivo é separado da exportação do Docker.

### Exportar

A aba **Exportar** lista cada conta com os seus repositórios para marcar. O arquivo guarda o nome, o provedor, o host, a porta SSH, o usuário, a identidade dos commits, a pasta e os e-mails de contato de cada conta e, de cada repositório, a pasta e a URL SSH. Chaves SSH, tokens e os arquivos dos repositórios não são exportados. Repositórios sem remote ficam de fora (não há de onde cloná-los); os pendentes entram.

**Exportar…** pergunta onde salvar (nome padrão `perci-repositorios.yaml`); só você pode ler o arquivo. Ele é gravado na versão 2 do formato quando tem algum e-mail de contato; senão, na versão 1.

```yaml
version: 2
accounts:
  - name: Pessoal
    provider: github
    host: github.com
    username: seu-usuario
    git_name: Seu Nome
    git_email: voce@example.com
    folder: /home/voce/dev/github
    contact_email: voce@example.com
    repos:
      - path: labs/projeto
        url: git@github.com:seu-usuario/projeto.git
```

### Importar

Na aba **Importar**, **Selecionar arquivo** lê um arquivo exportado e lista o que seria feito:

- **conta nova** — ganha a pasta (criada se não existir; **Trocar pasta** escolhe outra), uma chave SSH sem senha em `~/.ssh/perci_<id>` (uma chave que já esteja lá é reaproveitada), as chaves de servidor embutidas e a regra `includeIf`.
- **já cadastrada — mescla com …** — mesmo usuário no mesmo servidor: a conta mantém os dados dela e só ganha os repositórios.
- Repositórios que já estão pendentes ou já estão clonados aparecem marcados; um cuja pasta tem outro conteúdo é ignorado, com o motivo.

**Importar** adiciona os repositórios marcados como pendentes na tela de cada conta; nada é clonado ainda.

## Onde cada coisa fica

| O quê | Onde |
|---|---|
| Contas, repositórios pendentes e e-mails de contato | `~/.perci/config.yaml`, chave `git_accounts` |
| Identidade dos commits e comando SSH de cada conta | `~/.perci/git/<id>.gitconfig` |
| As regras `includeIf` e a identidade padrão | Configuração global do Git (`~/.gitconfig`) |
| Chaves de servidor confirmadas | `~/.perci/git/known_hosts` |
| Chaves SSH geradas | `~/.ssh/perci_<id>` e `~/.ssh/perci_<id>.pub` |
| Tokens da API | Chaveiro do sistema (`secret-tool`, serviço `perci-git`) |

A desinstalação do Perci82 Dev pode antes salvar as contas e os repositórios num arquivo de exportação; quando também remove as configurações, ela esquece as contas do mesmo jeito que **Remover**. Veja [Desinstalação](uninstallation.md).
