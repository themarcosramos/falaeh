# falaeh

Aplicação PWA gamificada para exercícios fonoaudiológicos, com progressão por níveis, interação por voz, gamificação, geração de relatório final (PDF/PNG) e backend em Go.

> Três mundos (iniciante, intermediário e avançado), quatro tipos de exercício (incluindo interação por voz com Web Speech API e fallback por toque), XP/streak/conquistas calculados centralmente no backend, suporte offline PWA e emissão local de certificados e relatórios em PDF e imagem PNG de alta resolução.

---

## Stack

- **Frontend**: HTML5, CSS3, Bootstrap 5, JavaScript moderno (sem framework SPA), PWA (manifest + service worker)
- **Backend**: Go 1.27 com biblioteca padrão `net/http` (Clean Architecture pragmática)
- **Dados**: arquivos JSON versionados em `backend/data` (sem banco de dados relacional nesta versão)
- **Infra local**: Docker, Docker Compose, Nginx e Make

---

## Arquitetura

O projeto adota uma arquitetura limpa e pragmática, sem abstrações desnecessárias ou overengineering.
Para detalhes conceituais, separação de camadas e diagramas Mermaid, consulte [docs/architecture.md](docs/architecture.md).

---

## Pré-requisitos

- Docker
- Docker Compose v2
- Make

Não é necessário instalar Go localmente: tudo executa dentro dos containers.

---

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
| Frontend PWA | http://localhost:3000               |
| API Health   | http://localhost:8080/health        |
| Swagger UI   | http://localhost:8080/docs          |
| Spec Swagger | http://localhost:8080/swagger.yaml  |

O frontend consome a API pelo prefixo `/api/`, que o Nginx repassa para o backend Go. A documentação interativa também fica acessível em `http://localhost:3000/api/docs`.

Para encerrar os containers:

```bash
make down
```

---

## Documentação da API

A especificação Swagger é gerada a partir das anotações no código:

```bash
make swagger
```

O comando lê as anotações em [backend/cmd/api/main.go](backend/cmd/api/main.go) e nos handlers em [backend/internal/httpapi/router.go](backend/internal/httpapi/router.go), gerando [backend/docs/swagger.yaml](backend/docs/swagger.yaml). O arquivo é embutido no binário e servido pelo pacote [backend/docs/docs.go](backend/docs/docs.go).

---

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
| `make coverage-check`     | Falha se a cobertura ficar abaixo de `COVERAGE_MIN` (80%) |
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

---

## Estrutura de pastas

```
frontend/
  index.html          # casca da aplicação PWA
  manifest.json       # Web App Manifest (standalone, ícones, tema)
  sw.js               # Service Worker (cache de assets essenciais + fallback offline)
  nginx.conf          # servidor web estático + proxy reverso para /api
  assets/
    css/ app.css      # estilos do jogo, microanimações, HUD e regras @media print
    js/ app.js        # lógica do cliente, SpeechRecognition, PWA e geração de relatórios
    icons/ icon.svg   # ícone adaptável de alta definição

backend/
  cmd/api/main.go     # ponto de entrada HTTP e inicialização com injeção de dependência
  docs/               # swagger.yaml gerado + Swagger UI (embed)
  internal/
    httpapi/          # camada delivery REST (rotas e handlers)
    game/             # máquina de estados da partida em memória e relatório final
    exercise/         # carregamento JSON, repositório e normalização textual
    gamification/     # regras de XP, streak, bônus e conquistas
  data/               # exercícios organizados por nível (JSON)

docs/
  architecture.md     # descrição detalhada da arquitetura e diagramas Mermaid

docker-compose.yml      # configuração base para produção
docker-compose.dev.yml  # volumes e hot-reload para desenvolvimento
Makefile
```

---

## API

Documentação interativa disponível em `http://localhost:8080/docs` (spec em `/swagger.yaml`).

| Método | Rota | Descrição |
| --- | --- | --- |
| `GET` | `/health` / `/api/health` | Verificação de disponibilidade |
| `GET` | `/api/v1/levels` | Lista os mundos/níveis do jogo |
| `GET` | `/api/v1/levels/{level}/exercises` | Exercícios públicos de um nível (sem gabarito) |
| `GET` | `/api/v1/exercises` | Lista exercícios com filtro opcional `?level=` |
| `GET` | `/api/v1/exercises/{id}` | Exercício por ID |
| `POST` | `/api/v1/exercises/{id}/answer` | Valida uma resposta isolada |
| `POST` | `/api/v1/game/start` | Inicia (ou retoma) uma partida e devolve o primeiro exercício |
| `POST` | `/api/v1/game/{sessionId}/answer` | Responde o exercício atual e devolve XP, streak e o próximo exercício |

### Fluxo da partida

1. `POST /api/v1/game/start` cria a partida em memória e devolve `sessionId`, o primeiro exercício e o progresso.
2. Cada resposta é submetida a `POST /api/v1/game/{sessionId}/answer`. O backend normaliza a entrada, valida contra o gabarito, calcula o XP (com bônus de 1ª tentativa e streak) e devolve o próximo exercício.
3. Ao concluir todos os desafios do nível, o backend retorna `phaseCompleted: true`, desbloqueia o próximo mundo e anexa o `report` de desempenho.

---

## Como testar

```bash
make test              # unitários + aceitação
make test-unit
make test-acceptance
make test-run TEST=TestGameFlow_AcertoAvancaExercicio
make coverage          # gera o relatório de cobertura em HTML
make coverage-check    # valida se a cobertura está >= 80% (meta atingida: 97.5%)
make ci                # pipeline completa de validação
```

---

## Modo por voz

Exercícios do tipo `voice` desafiam a criança ou adolescente a pronunciar fonemas, palavras ou trava-línguas.
O reconhecimento ocorre **localmente no navegador** via Web Speech API (`SpeechRecognition` / `webkitSpeechRecognition`), configurada para `pt-BR`.

### Privacidade e Segurança da Fala:
- **Nenhum áudio é gravado, armazenado ou transmitido para servidores.**
- Apenas a transcrição textual é normalizada localmente e enviada para conferência no backend.
- **Fallback total por toque**: opções de resposta por clique permanecem disponíveis caso o navegador não ofereça suporte a fala, a permissão do microfone seja recusada ou ocorra ruído ambiente.

---

## Relatório final e Certificado

Ao concluir uma fase, o jogador recebe um relatório completo de desempenho com opção de personalizar seu nome:
- **Nível** concluído;
- **Total de Exercícios Realizados**, **Acertos** e **Tentativas** com taxa de acurácia;
- **XP Total Conquistado** e **Maior Sequência (*Streak*)**;
- **Status da Fase** e **Próximo Nível Desbloqueado**;
- **Gerar PDF**: imprime ou salva localmente o certificado oficial em folha A4;
- **Salvar Imagem**: gera e baixa instantaneamente um cartão comemorativo PNG em alta resolução via HTML5 Canvas.

Toda a geração ocorre no cliente de forma privada, sem enviar informações pessoais a nenhum serviço.

---

## Como estender o jogo

### Adicionar um novo exercício

Edite o arquivo do nível desejado em `backend/data/` (`beginner.json`, `intermediate.json` ou `advanced.json`):

```json
{
    "id": "beg-007",
    "level": "beginner",
    "type": "voice",
    "instruction": "Pronuncie a palavra SAPO em voz alta:",
    "targetWord": "Sapo",
    "options": ["Sapo", "Pato", "Gato"],
    "correctAnswer": "Sapo"
}
```

### Alterar as regras de XP

As constantes de pontuação ficam centralizadas em [backend/internal/gamification/rules.go](backend/internal/gamification/rules.go):

| Regra | Constante | Valor padrão |
| --- | --- | --- |
| Resposta correta | `DefaultBaseCorrectXP` | 100 |
| Acerto na 1ª tentativa | `DefaultFirstAttemptBonusXP` | +50 |
| Sequência de 3 acertos | `DefaultStreak3BonusXP` | +50 |
| Sequência de 5 acertos | `DefaultStreak5BonusXP` | +100 |
| Conclusão de fase | `DefaultPhaseCompletionBonusXP` | +300 |
| Conclusão de nível | `DefaultLevelCompletionBonusXP` | +500 |

---

## Licença

Ver [LICENSE](LICENSE).
