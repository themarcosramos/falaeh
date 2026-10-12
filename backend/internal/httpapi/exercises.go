package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
)

// ExerciseService define a interface dos casos de uso de exercícios consumida pela API HTTP.
type ExerciseService interface {
	ListLevels(ctx context.Context) []exercise.LevelInfo
	ListAll(ctx context.Context) ([]exercise.PublicExercise, error)
	ListByLevel(ctx context.Context, level exercise.Level) ([]exercise.PublicExercise, error)
	GetExerciseByID(ctx context.Context, id string) (exercise.PublicExercise, error)
	ValidateAnswer(ctx context.Context, exerciseID string, answer string) (exercise.ValidationResult, error)
}

// ErrorResponse representa o formato padrão de resposta com erro da API.
type ErrorResponse struct {
	Error string `json:"error" example:"exercício não encontrado"`
}

// AnswerRequest representa o corpo da requisição para envio de resposta a um exercício.
type AnswerRequest struct {
	Answer string `json:"answer" example:"pato"`
}

// handleListLevels godoc
//
//	@Summary		Lista os níveis disponíveis
//	@Description	Retorna a lista de níveis de dificuldade e mundos suportados no jogo.
//	@Tags			levels
//	@Produce		json
//	@Success		200	{array}		exercise.LevelInfo
//	@Router			/api/v1/levels [get]
func handleListLevels(logger *slog.Logger, svc ExerciseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		levels := svc.ListLevels(r.Context())
		writeJSON(w, logger, http.StatusOK, levels)
	}
}

// handleListExercises godoc
//
//	@Summary		Lista exercícios com filtro opcional por nível
//	@Description	Retorna a lista de exercícios públicos do jogo, podendo filtrar por nível via query param.
//	@Tags			exercises
//	@Produce		json
//	@Param			level	query		string	false	"Filtro opcional por nível"	Enums(beginner, intermediate, advanced)
//	@Success		200		{array}		exercise.PublicExercise
//	@Failure		400		{object}	ErrorResponse
//	@Router			/api/v1/exercises [get]
func handleListExercises(logger *slog.Logger, svc ExerciseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		levelParam := strings.TrimSpace(r.URL.Query().Get("level"))
		if levelParam != "" {
			level := exercise.Level(levelParam)
			if !level.IsValid() {
				writeError(w, logger, http.StatusBadRequest, "nível inválido ou não suportado")
				return
			}
			exercises, err := svc.ListByLevel(r.Context(), level)
			if err != nil {
				writeError(w, logger, http.StatusNotFound, "nível não encontrado")
				return
			}
			writeJSON(w, logger, http.StatusOK, exercises)
			return
		}

		exercises, err := svc.ListAll(r.Context())
		if err != nil {
			logger.Error("falha ao listar todos os exercícios", slog.Any("error", err))
			writeError(w, logger, http.StatusInternalServerError, "erro interno do servidor")
			return
		}
		writeJSON(w, logger, http.StatusOK, exercises)
	}
}

// handleListExercisesByLevel godoc
//
//	@Summary		Lista os exercícios de um nível
//	@Description	Retorna a lista de exercícios públicos de um determinado nível. A resposta correta nunca é exposta.
//	@Tags			exercises
//	@Produce		json
//	@Param			level	path		string	true	"Nível (beginner, intermediate, advanced)"	Enums(beginner, intermediate, advanced)
//	@Success		200		{array}		exercise.PublicExercise
//	@Failure		404		{object}	ErrorResponse
//	@Router			/api/v1/levels/{level}/exercises [get]
func handleListExercisesByLevel(logger *slog.Logger, svc ExerciseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		levelStr := r.PathValue("level")
		level := exercise.Level(levelStr)
		if !level.IsValid() {
			writeError(w, logger, http.StatusNotFound, "nível não encontrado")
			return
		}

		exercises, err := svc.ListByLevel(r.Context(), level)
		if err != nil {
			if errors.Is(err, exercise.ErrLevelNotFound) {
				writeError(w, logger, http.StatusNotFound, "nível não encontrado")
				return
			}
			logger.Error("falha ao listar exercícios por nível", slog.String("level", levelStr), slog.Any("error", err))
			writeError(w, logger, http.StatusInternalServerError, "erro interno do servidor")
			return
		}

		writeJSON(w, logger, http.StatusOK, exercises)
	}
}

// handleGetExercise godoc
//
//	@Summary		Obtém um exercício por ID
//	@Description	Retorna os dados públicos de um exercício específico. A resposta correta nunca é exposta.
//	@Tags			exercises
//	@Produce		json
//	@Param			id	path		string	true	"ID do exercício (ex: beg-001)"
//	@Success		200	{object}	exercise.PublicExercise
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/api/v1/exercises/{id} [get]
func handleGetExercise(logger *slog.Logger, svc ExerciseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.PathValue("id"))
		if id == "" {
			writeError(w, logger, http.StatusBadRequest, "id do exercício não pode ser vazio")
			return
		}

		ex, err := svc.GetExerciseByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, exercise.ErrExerciseNotFound) {
				writeError(w, logger, http.StatusNotFound, "exercício não encontrado")
				return
			}
			logger.Error("falha ao obter exercício", slog.String("id", id), slog.Any("error", err))
			writeError(w, logger, http.StatusInternalServerError, "erro interno do servidor")
			return
		}

		writeJSON(w, logger, http.StatusOK, ex)
	}
}

// handleAnswerExercise godoc
//
//	@Summary		Valida a resposta de um exercício
//	@Description	Avalia se a resposta enviada pelo usuário está correta segundo regras fonéticas e normalização do domínio.
//	@Tags			exercises
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string			true	"ID do exercício"
//	@Param			request	body		AnswerRequest	true	"Resposta do usuário"
//	@Success		200		{object}	exercise.ValidationResult
//	@Failure		400		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Router			/api/v1/exercises/{id}/answer [post]
func handleAnswerExercise(logger *slog.Logger, svc ExerciseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.PathValue("id"))
		if id == "" {
			writeError(w, logger, http.StatusBadRequest, "id do exercício não pode ser vazio")
			return
		}

		var req AnswerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, logger, http.StatusBadRequest, "corpo da requisição inválido")
			return
		}

		if strings.TrimSpace(req.Answer) == "" {
			writeError(w, logger, http.StatusBadRequest, "a resposta não pode ser vazia")
			return
		}

		result, err := svc.ValidateAnswer(r.Context(), id, req.Answer)
		if err != nil {
			if errors.Is(err, exercise.ErrExerciseNotFound) {
				writeError(w, logger, http.StatusNotFound, "exercício não encontrado")
				return
			}
			if errors.Is(err, exercise.ErrEmptyAnswer) {
				writeError(w, logger, http.StatusBadRequest, "a resposta não pode ser vazia")
				return
			}
			logger.Error("falha ao validar resposta", slog.String("id", id), slog.Any("error", err))
			writeError(w, logger, http.StatusInternalServerError, "erro interno do servidor")
			return
		}

		writeJSON(w, logger, http.StatusOK, result)
	}
}
