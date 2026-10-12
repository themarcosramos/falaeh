COMPOSE ?= docker compose
COMPOSE_DEV := $(COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml
# Executa comandos Go no container com o usuário do host para não gerar arquivos como root.
GO_RUN := $(COMPOSE_DEV) run --rm --no-deps --user $$(id -u):$$(id -g) \
	-e HOME=/tmp -e GOPATH=/tmp/go -e GOCACHE=/tmp/go-build api
SWAG_VERSION ?= v1.16.6

.DEFAULT_GOAL := help

.PHONY: help dev down build test lint fmt vet logs swagger

help: ## Lista os comandos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "%-10s %s\n", $$1, $$2}'

dev: ## Sobe a aplicação em modo desenvolvimento
	$(COMPOSE_DEV) up --build

down: ## Derruba os containers
	$(COMPOSE_DEV) down --remove-orphans

build: ## Constrói as imagens de produção
	$(COMPOSE) build

test: ## Executa os testes do backend com cobertura
	$(GO_RUN) go test ./... -coverprofile=coverage.out

lint: ## Executa go vet no backend
	$(GO_RUN) go vet ./...

fmt: ## Formata o código Go
	$(GO_RUN) gofmt -w .

swagger: ## Regenera a documentação Swagger a partir das anotações
	$(GO_RUN) sh -c 'go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) && \
		$$GOPATH/bin/swag init \
			--generalInfo cmd/api/main.go \
			--dir ./ \
			--output docs \
			--outputTypes yaml \
			--generatedTime=false'

vet: lint ## Alias de lint

logs: ## Acompanha os logs dos containers
	$(COMPOSE_DEV) logs -f
