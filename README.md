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

| Comando        | Descrição                                     |
| -------------- | --------------------------------------------- |
| `make dev`     | Sobe a aplicação em modo desenvolvimento      |
| `make down`    | Derruba os containers                         |
| `make build`   | Constrói as imagens de produção               |
| `make test`    | Executa os testes com cobertura               |
| `make lint`    | Executa `go vet`                              |
| `make fmt`     | Formata o código Go                           |
| `make swagger` | Regenera a documentação Swagger               |
| `make logs`    | Acompanha os logs                             |

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
