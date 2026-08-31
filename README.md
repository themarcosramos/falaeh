# falaeh

Aplicação PWA gamificada para exercícios fonoaudiológicos, com progressão por níveis, interação por voz, gamificação e backend em Go.

> Estado atual: **estrutura inicial**. Nenhuma funcionalidade de jogo foi implementada ainda — apenas o esqueleto do projeto, o servidor HTTP com `/health` e a casca da PWA.

## Stack

- **Frontend**: HTML5, CSS3, Bootstrap 5, JavaScript moderno (sem framework SPA), PWA (manifest + service worker)
- **Backend**: Go 1.27 com biblioteca padrão `net/http`
- **Dados**: arquivos JSON versionados em `backend/data` (sem banco de dados nesta versão)
- **Infra local**: Docker, Docker Compose, Nginx e Make

## Pré-requisitos

- Docker
- Docker Compose v2
- Make

Não é necessário instalar Go localmente: tudo executa dentro dos containers.

## Como executar

```bash
git clone git@github.com:themarcosramos/falaeh.git
cd falaeh
cp .env.example .env
make dev
```

Serviços disponíveis:

| Serviço      | URL                                 |
| ------------ | ----------------------------------- |
| Frontend     | http://localhost:3000               |
| API          | http://localhost:8080/health        |
| Swagger UI   | http://localhost:8080/docs          |
| Spec Swagger | http://localhost:8080/swagger.yaml  |

O frontend chama a API pelo caminho `/api/`, que o Nginx encaminha para o backend. A documentação também fica acessível por ele em http://localhost:3000/api/docs.

## Documentação da API

A spec é **gerada a partir de anotações no código** com o [swag](https://github.com/swaggo/swag), executado via Docker (nenhuma dependência é adicionada ao `go.mod`):

```bash
make swagger
```

O comando lê as anotações gerais em [backend/cmd/api/main.go](backend/cmd/api/main.go) e as dos handlers em [backend/internal/httpapi/router.go](backend/internal/httpapi/router.go), e escreve [backend/docs/swagger.yaml](backend/docs/swagger.yaml). O arquivo é versionado e embutido no binário pelo pacote [backend/docs/docs.go](backend/docs/docs.go); o Swagger UI é carregado por CDN com Subresource Integrity.

Ao criar um endpoint novo, anote o handler e rode `make swagger`. Não edite o `swagger.yaml` à mão.

Para encerrar:

```bash
make down
```

## Comandos

Use `make help` para listar todos os comandos disponíveis.

| Comando                   | Descrição                                            |
| ------------------------- | ---------------------------------------------------- |
| `make setup`              | Cria o `.env` a partir do `.env.example`             |
| `make dev`                | Sobe a aplicação em desenvolvimento (foreground)     |
| `make up`                 | Sobe a aplicação em desenvolvimento (background)     |
| `make down`               | Derruba os containers                                |
| `make stop`               | Para os containers sem removê-los                    |
| `make restart`            | Reinicia os containers                               |
| `make build`              | Constrói as imagens de produção                      |
| `make logs`               | Acompanha os logs                                    |
| `make bash`               | Abre um shell no container da api                    |
| `make test`               | Executa testes unitários e de aceitação              |
| `make test-unit`          | Executa apenas os testes unitários                   |
| `make test-acceptance`    | Executa apenas os testes de aceitação                |
| `make test-run`           | Executa um teste específico (`TEST=Nome [PKG=path]`) |
| `make coverage`           | Gera o relatório de cobertura                        |
| `make coverage-check`     | Falha se a cobertura ficar abaixo de `COVERAGE_MIN`  |
| `make fmt`                | Formata o código Go (`gofmt -w -s`)                  |
| `make vet`                | Executa `go vet`                                     |
| `make lint`               | Analisa o código com golangci-lint                   |
| `make lint-fix`           | Corrige automaticamente o que o linter souber        |
| `make tidy`               | Organiza e verifica as dependências do módulo        |
| `make audit`              | Procura vulnerabilidades com govulncheck             |
| `make audit-image`        | Escaneia a imagem de produção com trivy              |
| `make ci`                 | Quality gate: vet + lint + testes + cobertura        |
| `make swagger` / `docs`   | Regenera a documentação Swagger                      |
| `make clean`              | Remove artefatos de teste e cobertura                |
| `make docker-down-volumes`| Derruba containers e remove volumes                  |
| `make docker-clean`       | Derruba containers, volumes e imagens do projeto     |

## Estrutura de pastas

```
frontend/
  index.html          # casca da aplicação
  manifest.json       # Web App Manifest
  sw.js               # service worker (cache dos assets essenciais)
  nginx.conf          # servidor estático + proxy /api -> backend
  assets/
    css/ js/ icons/

backend/
  cmd/api/            # ponto de entrada HTTP
  docs/               # swagger.yaml gerado + Swagger UI (embed)
  internal/
    httpapi/          # delivery HTTP (rotas e handlers)
    game/             # regras da partida e progressão
    exercise/         # exercícios e carregamento dos JSON
    gamification/     # XP, bônus, streak e conquistas
    report/           # relatório final da missão
  data/               # exercícios por nível (JSON)

docker-compose.yml      # configuração base (produção)
docker-compose.dev.yml  # overrides de desenvolvimento
Makefile
```

## Próximos passos

1. Modelar exercícios e o carregamento dos arquivos JSON.
2. Implementar as regras de gamificação (XP, streak, bônus, conquistas).
3. Implementar a máquina de estados da partida e o desbloqueio de níveis.
4. Expor os endpoints da API v1 e conectar o frontend.
5. Implementar a interface do jogo, o modo por voz e o relatório final.

## Licença

Ver [LICENSE](LICENSE).
