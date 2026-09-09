
#  Falaêh — PWA gamificada de exercícios fonoaudiológicos
#  Todos os comandos Go rodam dentro do container: nada é instalado no host.

#  configuração
APP_NAME     ?= Falaêh
COMPOSE      ?= docker compose
SWAG_VERSION ?= v1.16.6
COVERAGE_MIN ?= 80

# Imagens de ferramentas externas e imagem de produção auditada.
GOLANGCI_IMAGE ?= golangci/golangci-lint:v2.13.2
TRIVY_IMAGE    ?= aquasec/trivy:latest
DOCKER_IMAGE          ?= falaeh-api
DOCKER_TAG            ?= latest
DESKTOP_BUILDER_IMAGE ?= falaeh-desktop-builder

COMPOSE_DEV := $(COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml

# Caminhos relativos a ./backend, montado como /app no container de dev.
TMP_DIR   := .tmp
TESTS_DIR := $(TMP_DIR)/tests
COVERAGE  := $(TESTS_DIR)/coverage.out

# Pacotes de teste por suíte.
UNIT_PKGS       ?= internal/exercise internal/httpapi internal/gamification internal/game internal/feedback
ACCEPTANCE_PKGS ?= internal/exercise internal/httpapi

# Executa comandos Go no container com o usuário do host, para não gerar
# arquivos como root, e com os caches em ./backend/tmp (ignorado pelo git).
GO_RUN := $(COMPOSE_DEV) run --rm --no-deps -T --user $$(id -u):$$(id -g) \
	-e HOME=/tmp \
	-e GOPATH=/app/$(TMP_DIR)/go \
	-e GOCACHE=/app/$(TMP_DIR)/go-build \
	api

# golangci-lint não está no compose: roda pela imagem oficial sobre o projeto.
LINT_RUN := docker run --rm -t --user $$(id -u):$$(id -g) \
	-v $$(pwd):/app -w /app/backend \
	-e HOME=/tmp \
	-e GOPATH=/app/backend/$(TMP_DIR)/go \
	-e GOCACHE=/app/backend/$(TMP_DIR)/go-build \
	-e GOLANGCI_LINT_CACHE=/app/backend/$(TMP_DIR)/golangci-lint \
	$(GOLANGCI_IMAGE)

C_TITLE := \033[1;36m
C_OK    := \033[1;32m
C_FAIL  := \033[1;31m
C_WARN  := \033[1;33m
C_DIM   := \033[2m
C_OFF   := \033[0m

.DEFAULT_GOAL := help

.PHONY: help setup dev up down stop restart build logs bash \
	test test-unit test-acceptance acceptance test-run coverage coverage-check \
	fmt vet lint lint-fix style-fix tidy generate audit audit-image check ci \
	wails-dev swagger docs \
	desktop-builder desktop-icons desktop-sync desktop-linux desktop-appimage desktop-windows desktop-darwin desktop-all desktop-dev desktop-clean \
	clean docker-clean docker-down-volumes

#  macros
# $(1) suíte (unit|acceptance)   $(2) título   $(3) pacotes
define run_test_suite
	@printf '\n  $(C_TITLE)%s — $(APP_NAME)$(C_OFF)\n\n' '$(2)'
	@$(GO_RUN) mkdir -p $(TESTS_DIR)
	@failed=''; \
	for pkg in $(3); do \
		printf '  ./test/$(1)/%s\n' "$$pkg"; \
		output=$$($(GO_RUN) sh -c 'go test -c -o $(TESTS_DIR)/$(1)_testbin ./test/$(1)/'"$$pkg"' && ./$(TESTS_DIR)/$(1)_testbin -test.v -test.count=1; ret=$$?; rm -f $(TESTS_DIR)/$(1)_testbin; exit $$ret' 2>&1); \
		ret=$$?; \
		printf '%s\n' "$$output"; \
		if [ $$ret -ne 0 ]; then \
			failed="$$failed ./test/$(1)/$$pkg"; \
			printf '%s\n' "$$output" | grep -E '^[[:space:]]*--- FAIL:' | sed -E 's/^[[:space:]]*--- FAIL: //; s/ \(.*//' | while IFS= read -r name; do \
				printf '  $(C_FAIL)✗ FAIL  %s$(C_OFF)\n' "$$name"; \
			done; \
			printf '%s\n' "$$output" | grep -E '^# |panic:' | head -5 | while IFS= read -r line; do \
				printf '  $(C_WARN)⚠ %s$(C_OFF)\n' "$$line"; \
			done; \
		fi; \
	done; \
	if [ -n "$$failed" ]; then \
		printf '\n  $(C_FAIL)✗ %s — falhas em:%s$(C_OFF)\n\n' '$(2)' "$$failed"; \
		exit 1; \
	fi; \
	printf '\n  $(C_OK)✔ %s — todos os testes passaram$(C_OFF)\n\n' '$(2)'
endef

# $(1) suíte (unit|acceptance)   $(2) pacotes
define run_coverage_suite
	@for pkg in $(2); do \
		pkg_name=$$(printf '%s' "$$pkg" | sed 's|internal/||'); \
		pkg_safe=$$(printf '%s' "$$pkg" | tr '/' '_'); \
		printf '  %-11s %-38s' '$(1)' "$$pkg_name"; \
		result=$$($(GO_RUN) sh -c 'go test -c -cover -covermode=atomic -coverpkg=./internal/... -o $(TESTS_DIR)/covbin ./test/$(1)/'"$$pkg"' && ./$(TESTS_DIR)/covbin -test.coverprofile=$(TESTS_DIR)/coverage-$(1)-'"$$pkg_safe"'.out 2>&1 | sed -n "s/.*coverage: \([0-9.]*%\).*/\1/p"; rm -f $(TESTS_DIR)/covbin' 2>&1 | tr -d '\r\n'); \
		printf '$(C_OK)%s$(C_OFF)\n' "$$result"; \
	done
endef

##@ Geral

help: ## Lista os comandos disponíveis
	@awk 'BEGIN {FS = ":.*##"; printf "\n  $(C_TITLE)$(APP_NAME) — comandos$(C_OFF)\n"} \
		/^##@/ {printf "\n  $(C_DIM)%s$(C_OFF)\n", substr($$0, 5)} \
		/^[a-zA-Z_-]+:.*?##/ {printf "    $(C_TITLE)%-21s$(C_OFF) %s\n", $$1, $$2} \
		END {printf "\n"}' $(MAKEFILE_LIST)

##@ Ambiente

setup: ## Cria o .env a partir do .env.example
	@if [ -f .env ]; then \
		printf '  $(C_WARN)⚠ .env já existe — nada a fazer$(C_OFF)\n'; \
	else \
		cp .env.example .env; \
		printf '  $(C_OK)✔ .env criado a partir de .env.example$(C_OFF)\n'; \
	fi

dev: ## Sobe a aplicação em modo desenvolvimento (foreground)
	$(COMPOSE_DEV) up --build

up: ## Sobe a aplicação em modo desenvolvimento (background)
	$(COMPOSE_DEV) up -d --build

down: ## Derruba os containers
	$(COMPOSE_DEV) down --remove-orphans

stop: ## Para os containers sem removê-los
	$(COMPOSE_DEV) stop

restart: ## Reinicia os containers
	$(COMPOSE_DEV) restart

build: ## Constrói as imagens de produção
	$(COMPOSE) build

logs: ## Acompanha os logs dos containers
	$(COMPOSE_DEV) logs -f

bash: ## Abre um shell dentro do container da api
	$(COMPOSE_DEV) exec api sh

##@ Testes

test: test-unit test-acceptance ## Executa todas as suítes de teste

test-unit: ## Executa os testes unitários
	$(call run_test_suite,unit,Testes Unitários,$(UNIT_PKGS))

test-acceptance: ## Executa os testes de aceitação
	$(call run_test_suite,acceptance,Testes de Aceitação,$(ACCEPTANCE_PKGS))

acceptance: test-acceptance ## Alias de test-acceptance

test-run: ## Executa um teste específico (uso: make test-run TEST=NomeDoTeste [PKG=caminho])
ifndef TEST
	$(error TEST é obrigatório: make test-run TEST=<nome_do_teste>)
endif
	@$(GO_RUN) go test -v -count=1 -vet=off -run $(TEST) $(if $(PKG),$(PKG),./...)

coverage: ## Gera o relatório de cobertura (unitários + aceitação)
	@printf '\n  $(C_TITLE)Cobertura — $(APP_NAME)$(C_OFF)\n\n'
	@$(GO_RUN) sh -c 'mkdir -p $(TESTS_DIR) && rm -f $(TESTS_DIR)/coverage*.out'
	$(call run_coverage_suite,unit,$(UNIT_PKGS))
	$(call run_coverage_suite,acceptance,$(ACCEPTANCE_PKGS))
	@$(GO_RUN) sh -c "echo 'mode: atomic' > $(COVERAGE) && \
		tail -q -n +2 $(TESTS_DIR)/coverage-*.out 2>/dev/null >> $(COVERAGE); \
		rm -f $(TESTS_DIR)/coverage-*.out"
	@$(GO_RUN) go tool cover -html=$(COVERAGE) -o $(TESTS_DIR)/coverage.html
	@total=$$($(GO_RUN) sh -c 'go tool cover -func=$(COVERAGE) | tail -1' | awk '{print $$NF}'); \
	printf '\n  $(C_TITLE)%-50s$(C_OK)%s$(C_OFF)\n' 'TOTAL' "$$total"; \
	printf '  $(C_DIM)HTML → backend/$(TESTS_DIR)/coverage.html$(C_OFF)\n\n'

coverage-check: coverage ## Falha se a cobertura global ficar abaixo de COVERAGE_MIN
	@total=$$($(GO_RUN) sh -c 'go tool cover -func=$(COVERAGE) | tail -1' | awk '{print $$NF}' | tr -d '%\r\n'); \
	if awk "BEGIN {exit !($$total < $(COVERAGE_MIN))}"; then \
		printf '  $(C_FAIL)✗ Cobertura %s%% abaixo do mínimo de $(COVERAGE_MIN)%%$(C_OFF)\n\n' "$$total"; \
		exit 1; \
	fi; \
	printf '  $(C_OK)✔ Cobertura %s%% ≥ $(COVERAGE_MIN)%%$(C_OFF)\n\n' "$$total"

##@ Qualidade

fmt: ## Formata o código Go
	$(GO_RUN) sh -c 'gofmt -w -s cmd internal test docs'

vet: ## Executa go vet no backend
	$(GO_RUN) go vet ./cmd/... ./internal/... ./docs/... ./test/...

tidy: ## Organiza e verifica as dependências do módulo
	$(GO_RUN) sh -c 'go mod tidy && go mod verify'

generate: ## Executa go generate no backend
	$(GO_RUN) go generate ./cmd/... ./internal/... ./docs/... ./test/...

lint: ## Analisa o código com golangci-lint
	@printf '\n  $(C_TITLE)Lint — $(APP_NAME)$(C_OFF)\n\n'
	@$(LINT_RUN) golangci-lint run -c /app/.golangci.yml --show-stats ./cmd/... ./internal/... ./docs/... ./test/...

lint-fix: fmt ## Corrige automaticamente o que o golangci-lint souber corrigir
	@printf '\n  $(C_TITLE)Lint Fix — $(APP_NAME)$(C_OFF)\n\n'
	@$(LINT_RUN) golangci-lint run -c /app/.golangci.yml --fix --show-stats ./cmd/... ./internal/... ./docs/... ./test/...

style-fix: lint-fix ## Alias de lint-fix

audit: ## Procura vulnerabilidades conhecidas nas dependências (govulncheck)
	@printf '\n  $(C_TITLE)Auditoria — $(APP_NAME)$(C_OFF)\n\n'
	@output=$$($(GO_RUN) sh -c 'go install golang.org/x/vuln/cmd/govulncheck@latest && $$GOPATH/bin/govulncheck ./...' 2>&1); \
	printf '%s\n' "$$output" | awk '\
		/^Vulnerability #/ {printf "$(C_FAIL)%s$(C_OFF)\n", $$0; next} \
		/More info:/       {printf "$(C_DIM)%s$(C_OFF)\n", $$0; next} \
		/Found in:/        {printf "$(C_WARN)%s$(C_OFF)\n", $$0; next} \
		/Fixed in:/        {printf "$(C_OK)%s$(C_OFF)\n", $$0; next} \
		                   {print}'; \
	count=$$(printf '%s\n' "$$output" | grep -c '^Vulnerability #' || true); \
	if [ "$$count" -gt 0 ]; then \
		printf '\n  $(C_FAIL)✗ %d vulnerabilidade(s) encontrada(s) — veja o relatório acima$(C_OFF)\n\n' "$$count"; \
		exit 1; \
	fi; \
	printf '\n  $(C_OK)✔ Nenhuma vulnerabilidade encontrada$(C_OFF)\n\n'

audit-image: ## Escaneia a imagem de produção com trivy (requer make build)
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		$(TRIVY_IMAGE) image \
		--severity HIGH,CRITICAL \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

check: vet lint test coverage-check ## Quality Gate: vet + lint + testes + cobertura mínima

ci: check ## Alias do Quality Gate para CI

##@ Desktop (Wails)

desktop-builder: ## Constrói a imagem Docker para compilação multiplataforma com Wails
	@printf '\n  $(C_TITLE)Desktop Builder — $(APP_NAME)$(C_OFF)\n\n'
	docker build -t $(DESKTOP_BUILDER_IMAGE) -f build/desktop/Dockerfile .

desktop-icons: desktop-builder ## Gera os ícones da aplicação (PNG, ICO) a partir do SVG do navegador
	@printf '\n  $(C_TITLE)Ícones Desktop — $(APP_NAME)$(C_OFF)\n\n'
	@docker run --rm --user $$(id -u):$$(id -g) \
		-v $$(pwd):/workspace -w /workspace \
		-e HOME=/tmp \
		$(DESKTOP_BUILDER_IMAGE) sh -c '\
		mkdir -p /tmp/icons build/windows build/darwin backend/cmd/desktop/build/windows backend/cmd/desktop/build/darwin && \
		rsvg-convert -w 512 -h 512 frontend/assets/icons/icon.svg -o build/appicon.png && \
		rsvg-convert -w 16 -h 16 frontend/assets/icons/icon.svg -o /tmp/icons/16.png && \
		rsvg-convert -w 32 -h 32 frontend/assets/icons/icon.svg -o /tmp/icons/32.png && \
		rsvg-convert -w 48 -h 48 frontend/assets/icons/icon.svg -o /tmp/icons/48.png && \
		rsvg-convert -w 64 -h 64 frontend/assets/icons/icon.svg -o /tmp/icons/64.png && \
		rsvg-convert -w 128 -h 128 frontend/assets/icons/icon.svg -o /tmp/icons/128.png && \
		rsvg-convert -w 256 -h 256 frontend/assets/icons/icon.svg -o /tmp/icons/256.png && \
		icotool -c -o build/windows/icon.ico /tmp/icons/*.png && \
		cp -f build/appicon.png backend/cmd/desktop/build/appicon.png && \
		cp -f build/windows/icon.ico backend/cmd/desktop/build/windows/icon.ico'
	@printf '  $(C_OK)✔ Ícones PNG e Windows ICO gerados com sucesso a partir de frontend/assets/icons/icon.svg$(C_OFF)\n\n'

desktop-sync: desktop-icons ## Sincroniza frontend, dados e ícones para o pacote desktop
	@mkdir -p backend/cmd/desktop/frontend backend/cmd/desktop/data build/bin
	@rsync -a --delete frontend/ backend/cmd/desktop/frontend/
	@rsync -a --delete backend/data/ backend/cmd/desktop/data/

desktop-linux: desktop-builder desktop-sync ## Compila o binário Falaêh Desktop Linux (AMD64)
	@printf '\n  $(C_TITLE)Desktop Linux (AMD64) — $(APP_NAME)$(C_OFF)\n\n'
	@docker run --rm --user $$(id -u):$$(id -g) \
		-v $$(pwd):/workspace -w /workspace/backend/cmd/desktop \
		-e HOME=/tmp -e GOPATH=/tmp/go -e GOCACHE=/tmp/go-build \
		$(DESKTOP_BUILDER_IMAGE) \
		wails build -s -skipbindings -tags webkit2_41 -o falaeh
	@cp -f backend/cmd/desktop/build/bin/falaeh build/bin/falaeh
	@printf '  $(C_OK)✔ Binário Linux gerado em: build/bin/falaeh$(C_OFF)\n\n'

desktop-appimage: desktop-linux ## Empacota o Falaêh Desktop Linux em formato portátil AppImage (x86_64)
	@printf '\n  $(C_TITLE)Desktop Linux AppImage (x86_64) — $(APP_NAME)$(C_OFF)\n\n'
	@docker run --rm --user $$(id -u):$$(id -g) \
		-v $$(pwd):/workspace -w /workspace \
		-e HOME=/tmp \
		$(DESKTOP_BUILDER_IMAGE) sh -c '\
		mkdir -p build/AppDir/usr/bin build/bin && \
		cp -f build/bin/falaeh build/AppDir/usr/bin/falaeh && \
		chmod +x build/AppDir/usr/bin/falaeh && \
		cp -f build/appicon.png build/AppDir/falaeh.png && \
		cp -f build/appicon.png build/AppDir/.DirIcon && \
		printf "[Desktop Entry]\nType=Application\nName=Falaêh\nComment=Jogo educativo de exercícios fonoaudiológicos\nExec=falaeh\nIcon=falaeh\nCategories=Education;Game;\nTerminal=false\n" > build/AppDir/falaeh.desktop && \
		printf "#!/bin/sh\nSELF=\$$(readlink -f \"\$$0\")\nHERE=\$${SELF%%/*}\nexec \"\$$HERE/usr/bin/falaeh\" \"\$$@\"\n" > build/AppDir/AppRun && \
		chmod +x build/AppDir/AppRun && \
		ARCH=x86_64 appimagetool build/AppDir build/bin/Falaeh-x86_64.AppImage && \
		rm -rf build/AppDir'
	@printf '  $(C_OK)✔ Pacote AppImage gerado em: build/bin/Falaeh-x86_64.AppImage$(C_OFF)\n\n'

desktop-windows: desktop-builder desktop-sync ## Compila o executável portátil Falaêh Desktop Windows (AMD64) via MinGW
	@printf '\n  $(C_TITLE)Desktop Windows Portável (AMD64) — $(APP_NAME)$(C_OFF)\n\n'
	@docker run --rm --user $$(id -u):$$(id -g) \
		-v $$(pwd):/workspace -w /workspace/backend/cmd/desktop \
		-e HOME=/tmp -e GOPATH=/tmp/go -e GOCACHE=/tmp/go-build \
		$(DESKTOP_BUILDER_IMAGE) \
		wails build -s -skipbindings -platform windows/amd64 -o falaeh.exe
	@cp -f backend/cmd/desktop/build/bin/falaeh.exe build/bin/falaeh.exe
	@cp -f backend/cmd/desktop/build/bin/falaeh.exe build/bin/falaeh-portable.exe
	@printf '  $(C_OK)✔ Executável Portátil Windows gerado em: build/bin/falaeh.exe e build/bin/falaeh-portable.exe$(C_OFF)\n\n'

desktop-darwin: desktop-sync ## Prepara ou compila Falaêh Desktop macOS (requer macOS runner para .app/.dmg)
	@printf '\n  $(C_TITLE)Desktop macOS — $(APP_NAME)$(C_OFF)\n\n'
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		cd backend/cmd/desktop && wails build -s -skipbindings -platform darwin/universal -o falaeh; \
		cp -rf backend/cmd/desktop/build/bin/falaeh.app build/bin/; \
		printf '  $(C_OK)✔ Aplicativo macOS gerado em: build/bin/falaeh.app$(C_OFF)\n\n'; \
	else \
		printf '  $(C_WARN)⚠ O empacotamento oficial de macOS (.app/.dmg) requer Cocoa/WebKit nativos da Apple.$(C_OFF)\n'; \
		printf '  $(C_DIM)A configuração está pronta e é executada no GitHub Actions runner macos-latest.$(C_OFF)\n\n'; \
	fi

desktop-all: desktop-linux desktop-appimage desktop-windows ## Compila todas as distribuições desktop suportadas no ambiente atual
	@printf '\n  $(C_TITLE)Artefatos gerados em build/bin/:$(C_OFF)\n'
	@file build/bin/* 2>/dev/null || true

desktop-dev: desktop-builder desktop-sync ## Inicia wails dev para hot-reload da interface desktop
	@printf '\n  $(C_TITLE)Wails Dev — $(APP_NAME)$(C_OFF)\n\n'
	@docker run --rm -it --user $$(id -u):$$(id -g) \
		-v $$(pwd):/workspace -w /workspace/backend/cmd/desktop \
		-e HOME=/tmp -e GOPATH=/tmp/go -e GOCACHE=/tmp/go-build \
		-p 34115:34115 \
		$(DESKTOP_BUILDER_IMAGE) \
		wails dev -s

desktop-clean: ## Limpa os artefatos de build do desktop
	@rm -rf backend/cmd/desktop/build backend/cmd/desktop/frontend backend/cmd/desktop/data build/bin/falaeh*
	@printf '  $(C_OK)✔ Artefatos desktop limpos$(C_OFF)\n'

wails-dev: desktop-dev ## Alias de desktop-dev

##@ Documentação

swagger: ## Regenera a documentação Swagger a partir das anotações
	$(GO_RUN) sh -c 'go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) && \
		$$GOPATH/bin/swag init \
			--generalInfo cmd/api/main.go \
			--dir ./ \
			--exclude tmp,test \
			--output docs \
			--outputTypes yaml \
			--generatedTime=false'

docs: swagger ## Alias de swagger

##@ Manutenção

clean: ## Remove artefatos temporários de teste e cobertura
	$(GO_RUN) rm -rf $(TESTS_DIR)

docker-down-volumes: ## Derruba os containers e remove os volumes
	$(COMPOSE_DEV) down --volumes --remove-orphans

docker-clean: ## Derruba containers, volumes e imagens do projeto
	$(COMPOSE_DEV) down --volumes --remove-orphans --rmi all
