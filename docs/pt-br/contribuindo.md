# Contribuindo com o Perci82

🌐 **Idioma:** [English](../../CONTRIBUTING.md) · Português

Obrigado por considerar uma contribuição. Este documento explica como compilar e verificar o projeto, o que um pull request precisa e como as releases são feitas.

Ao participar deste projeto, você concorda em seguir o [Código de Conduta](codigo-de-conduta.md).

## Antes de começar

Para qualquer coisa além de uma correção pequena (uma tela nova, um item novo num catálogo, uma dependência nova), abra antes uma issue para discutir a abordagem. O projeto mantém poucas dependências, todas justificadas: veja o bloco `require` do [`go.mod`](../../go.mod). O Wails v3 é atualizado à mão, numa mudança própria.

## Ambiente de desenvolvimento

Requisitos:

- Linux em amd64.
- Go 1.27.4 ou mais recente.
- Os pacotes de desenvolvimento do GTK 4 e do WebKitGTK 6.0:

  ```bash
  # Debian/Ubuntu
  sudo apt install -y build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
  # Fedora
  sudo dnf install -y gcc-c++ pkgconf-pkg-config gtk4-devel webkitgtk6.0-devel
  ```

- Node.js, para os testes do frontend.
- A CLI `wails3` da versão do Wails do `go.mod`, para regerar e conferir os bindings: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28`.
- ShellCheck e shfmt, para o `make lint-shell`.

```bash
git clone https://github.com/oito2/perci82.git
cd perci82
make build          # gera ./perci82-system e ./perci82-dev
```

### Verificações

Rode-as antes de abrir um pull request; o CI roda as mesmas.

```bash
make lint             # gofmt, go vet, golangci-lint
make test             # go test -race ./...
make test-frontend    # node --test, Node puro
make check-generated  # CSS e catálogos gerados estão em dia
make check-bindings   # os bindings do frontend batem com os serviços Go
make lint-shell       # ShellCheck e shfmt no install.sh e no scripts/build-pages.sh
```

`make generate` e `make generate-bindings` regravam o que o `check-generated` e o `check-bindings` comparam. `make install-nfpm` instala o nfpm na versão fixada, e `make release-<app> VERSION=vX.Y.Z` gera o `.deb` e o `.rpm` desse aplicativo em `dist/`.

## Estrutura do projeto

| Caminho | O que contém |
|---|---|
| `cmd/perci82-system`, `cmd/perci82-dev` | O ponto de entrada e o frontend de cada aplicativo (`frontend/dist`, com os bindings gerados) |
| `internal/` | Todo o resto: serviços da interface (`internal/gui`), stack Docker (`internal/appstack`), contextos de IA (`internal/manager/ai`), ações do sistema, configuração, catálogos de i18n… |
| `packaging/` | Ícones, entradas `.desktop`, configurações do nfpm e a chave pública de assinatura do repositório |
| `install.sh` | O instalador: adiciona o repositório de pacotes e instala os aplicativos |
| `scripts/build-pages.sh` | Monta o repositório apt/dnf assinado publicado no GitHub Pages |
| `tools/` | `imagetools` (versões fixadas das ferramentas das imagens PHP), `parity-sync` (fixtures do oito2_ui), `installtest` (testes do `install.sh`) |
| `docs/en`, `docs/pt-br` | A documentação, em inglês e em português |

## Fazendo uma mudança

- Mantenha a mudança no escopo do que foi discutido; refatorações grandes não pedidas tendem a ser recusadas.
- Comentários de código são em inglês e descrevem o que o código faz.
- Uma mudança visível ao usuário atualiza a documentação na mesma mudança, nas **duas** línguas: `docs/en/` e `docs/pt-br/`, e o `README.md` junto com o `docs/pt-br/leiame.md`.
- Uma mudança visível ao usuário ganha uma entrada em `## [Unreleased]` no [`CHANGELOG.md`](../../CHANGELOG.md) (Keep a Changelog); seções já lançadas nunca são alteradas.
- Escreva um resumo de commit conciso, que diga por que a mudança foi feita.

## Pull requests

- Descreva o que mudou e por quê, e ligue a issue correspondente.
- O check `Lint and test` precisa passar; a branch padrão mantém o histórico linear (squash ou rebase).
- O [Dependabot](../../.github/dependabot.yml) abre pull requests semanais para as GitHub Actions e os módulos Go. Ele só propõe uma versão 14 dias depois do lançamento (atualizações de segurança não esperam), e os commits dele começam com `ci` ou `deps`. Ferramentas fixadas à mão seguem a mesma regra dos 14 dias; o workflow semanal **Image tools** avisa de versões novas das ferramentas fixadas nas imagens PHP.

## Relatando bugs e pedindo funcionalidades

Abra uma [issue no GitHub](https://github.com/oito2/perci82/issues) com o aplicativo e a versão (Visão Geral → Versão), a sua distribuição e o que aconteceu.

Relate vulnerabilidades de segurança em particular, nunca numa issue pública: use **Report a vulnerability** na [aba Security](https://github.com/oito2/perci82/security/advisories/new) do repositório.

## Releases

Cada aplicativo é lançado por conta própria, a partir da `main`:

1. Acrescente ao `CHANGELOG.md` a seção `## [System X.Y.Z] - data` ou `## [Dev X.Y.Z] - data` (tirando as entradas de `[Unreleased]`) e a linha de link dela.
2. Envie o commit e espere o CI.
3. Envie a tag `system-vX.Y.Z` ou `dev-vX.Y.Z`.

O [workflow de release](../../.github/workflows/release.yml) então:

1. Roda as mesmas verificações do CI.
2. Confere se o commit da tag está na `main` e gera o `.deb`, o `.rpm` e o `checksums.txt`.
3. Espera a aprovação do dono no environment `release`.
4. Atesta a procedência de build dos arquivos e cria a release no GitHub com eles numa única chamada `gh release create`. As notas da release são as instruções de instalação mais a seção do CHANGELOG.
5. Remonta o repositório de pacotes em `https://oito2.github.io/perci82/` ([`pages.yml`](../../.github/workflows/pages.yml)), assinado com a chave do repositório. Quando os dois aplicativos recebem tag no mesmo commit, só a última execução de release publica o site.

Proteções:

- **Tags de release:** só admins do repositório criam, movem ou apagam as tags `system-v*` e `dev-v*` (ruleset `release tags`).
- **Releases imutáveis:** os arquivos e a tag de uma release publicada não mudam. Uma release com problema é corrigida com uma nova versão de patch.
- **Chave de assinatura do repositório:** o apt e o dnf só verificam assinaturas GPG, então a chave que assina o repositório de pacotes fica em `REPO_SIGNING_KEY` e `REPO_SIGNING_PASSPHRASE`, no environment `github-pages`, que só publica a partir da `main`, de `system-v*` e de `dev-v*`. É o único secret do projeto. Todo o resto é assinado por OIDC (as attestations de build).

## Licença

Ao contribuir, você concorda que as suas contribuições são licenciadas sob a [GPL-3.0](../../LICENSE).
