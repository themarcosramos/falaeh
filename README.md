# falaeh

Aplicação PWA gamificada para exercícios fonoaudiológicos, com progressão por níveis, interação por voz, gamificação e backend em Go.

> Estado atual: **jogável de ponta a ponta**. Três mundos (iniciante, intermediário e avançado), quatro tipos de
> exercício (incluindo o modo por voz), XP/streak/conquistas calculados no backend e relatório final da missão.
> Pendente: exportação do relatório em PDF/imagem.

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

## API

Documentação interativa em `http://localhost:8080/docs` (spec em `/swagger.yaml`).

| Método | Rota | Descrição |
| --- | --- | --- |
| `GET` | `/health` | Disponibilidade do serviço |
| `GET` | `/api/v1/levels` | Lista os níveis do jogo |
| `GET` | `/api/v1/levels/{level}/exercises` | Exercícios públicos de um nível (sem gabarito) |
| `GET` | `/api/v1/exercises` | Lista exercícios, com filtro opcional `?level=` |
| `GET` | `/api/v1/exercises/{id}` | Exercício por ID |
| `POST` | `/api/v1/exercises/{id}/answer` | Valida uma resposta isolada |
| `POST` | `/api/v1/game/start` | Inicia (ou retoma) uma partida e devolve o primeiro exercício |
| `POST` | `/api/v1/game/{sessionId}/answer` | Responde o exercício atual e devolve XP, streak e o próximo exercício |

### Fluxo da partida

`POST /api/v1/game/start` cria uma partida em memória e devolve `sessionId`, o primeiro
exercício e o progresso. Cada resposta é enviada para `POST /api/v1/game/{sessionId}/answer`,
que valida no backend, aplica as regras de XP e devolve o próximo exercício. Ao final do nível
a resposta inclui `phaseCompleted`, `completion` (com o desbloqueio do próximo mundo) e
`report`, usado pela tela de resultado.

Toda a pontuação, o streak e o desbloqueio de níveis são decididos pelo backend — o frontend
apenas exibe o que recebe. O gabarito nunca é enviado ao cliente. A partida vive somente em
memória, expira por inatividade e não guarda nenhum dado pessoal.

## Como testar

```bash
make test              # unitários + aceitação
make test-unit
make test-acceptance
make test-run TEST=TestAplicaXP PKG=./test/unit/...
make coverage          # gera o perfil e o HTML em backend/tmp/tests
make coverage-check    # falha se a cobertura global ficar abaixo de COVERAGE_MIN (80%)
make ci                # quality gate completo: vet + lint + testes + cobertura
```

Os testes ficam separados por finalidade em `backend/test/unit` e `backend/test/acceptance`, e usam
table-driven tests. Toda alteração relevante deve manter a cobertura global em pelo menos 80%.

## Modo por voz

Exercícios do tipo `voice` pedem que a criança pronuncie uma palavra. O reconhecimento acontece
**inteiramente no navegador**, pela Web Speech API (`SpeechRecognition` / `webkitSpeechRecognition`),
configurada em `pt-BR`. O fluxo é:

1. o jogador toca no botão de microfone;
2. o navegador transcreve a fala em texto;
3. o texto é normalizado (minúsculas, sem pontuação e sem espaços extras);
4. apenas esse **texto** é enviado para `POST /api/v1/game/{sessionId}/answer`;
5. o backend normaliza novamente e compara com a resposta esperada.

Nenhum áudio é gravado, armazenado ou enviado para o servidor.

### Limitações do reconhecimento de fala

- **Não é avaliação clínica.** O recurso apenas verifica se a palavra reconhecida corresponde à
  palavra solicitada; não mede articulação, fonemas ou qualidade da produção da fala, e não
  substitui a avaliação de um profissional de Fonoaudiologia. Um aviso equivalente é exibido na tela.
- A disponibilidade varia por navegador; em vários casos a transcrição depende de conexão com a
  internet e de serviço externo do próprio navegador.
- Ruído, sotaque e microfone influenciam o resultado.
- **Fallback sempre disponível**: as opções por toque continuam visíveis em todos os exercícios de voz.
  Se o navegador não suportar a API, se a permissão do microfone for negada ou se ocorrer erro de rede,
  a aplicação exibe uma mensagem amigável e o jogo continua normalmente pelo toque.

## Relatório final

Ao concluir um nível, a resposta da API traz o objeto `report` (exercícios, acertos, tentativas, XP,
precisão, maior sequência, conquistas e próximo nível desbloqueado). A tela de comemoração monta o
relatório no frontend, sem enviar nada para serviços externos. A exportação em PDF/imagem ainda não
foi implementada.

## Como estender o jogo

### Adicionar um novo exercício

Edite o JSON do nível correspondente em `backend/data/` (`beginner.json`, `intermediate.json` ou
`advanced.json`) e acrescente um item ao array `exercises`:

```json
{
    "id": "beg-005",
    "level": "beginner",
    "type": "multiple_choice",
    "instruction": "Qual palavra começa com o som /F/?",
    "targetWord": "Foca",
    "options": ["Foca", "Bota", "Mala"],
    "correctAnswer": "Foca"
}
```

Regras: o `id` deve ser único, o `level` precisa coincidir com o do arquivo, `correctAnswer` deve
estar entre as `options` e `type` deve ser um dos tipos suportados — `multiple_choice`,
`image_word_match`, `sound_identification` ou `voice`. Os arquivos são validados no carregamento;
um JSON inválido faz a API falhar na inicialização. Reinicie com `make restart` para recarregar.

### Adicionar uma nova fase/nível

Nesta versão cada arquivo de dados corresponde a uma fase (um mundo). Para acrescentar outra:

1. crie o novo `Level` em [backend/internal/exercise/exercise.go](backend/internal/exercise/exercise.go);
2. inclua-o na progressão em `NextLevel` ([backend/internal/gamification/rules.go](backend/internal/gamification/rules.go));
3. adicione o arquivo `backend/data/<nivel>.json`;
4. registre o mundo correspondente em `WORLDS` no [frontend/assets/js/app.js](frontend/assets/js/app.js) e o card em [frontend/index.html](frontend/index.html);
5. atualize os testes de progressão e desbloqueio.

O diretório dos dados pode ser trocado pela variável de ambiente `DATA_DIR`.

### Alterar as regras de XP

Todos os valores ficam centralizados em [backend/internal/gamification/rules.go](backend/internal/gamification/rules.go)
(`DefaultRules`), sem números mágicos espalhados pelo código:

| Regra | Constante | Valor padrão |
| --- | --- | --- |
| Resposta correta | `DefaultBaseCorrectXP` | 100 |
| Acerto na primeira tentativa | `DefaultFirstAttemptBonusXP` | +50 |
| Sequência de 3 acertos | `DefaultStreak3BonusXP` | +50 |
| Sequência de 5 acertos | `DefaultStreak5BonusXP` | +100 |
| Conclusão de fase | `DefaultPhaseCompletionBonusXP` | +300 |
| Conclusão de nível | `DefaultLevelCompletionBonusXP` | +500 |

As regras são injetadas no `game.Manager` em [backend/cmd/api/main.go](backend/cmd/api/main.go), o que
permite usar um conjunto alternativo sem alterar o domínio. Ao mudar qualquer valor, atualize os
testes de gamificação.

## Privacidade

- Sem cadastro, login ou coleta de dados pessoais.
- Sem banco de dados, analytics, tracking ou telemetria.
- Áudio não é gravado nem enviado ao backend.
- A partida existe apenas em memória e expira por inatividade.

## Próximos passos

1. ~~Modelar exercícios e o carregamento dos arquivos JSON.~~
2. ~~Implementar as regras de gamificação (XP, streak, bônus, conquistas).~~
3. ~~Implementar a máquina de estados da partida e o desbloqueio de níveis.~~
4. ~~Expor os endpoints da API v1 e conectar o frontend.~~
5. ~~Implementar o modo por voz (Web Speech API) com fallback por toque.~~
6. Implementar a exportação do relatório final (PDF/imagem).

## Licença

Ver [LICENSE](LICENSE).
