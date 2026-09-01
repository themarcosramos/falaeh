package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/game"
)

// maxAnswerBodyBytes limita o corpo das requisições de partida para evitar abuso de memória.
const maxAnswerBodyBytes = 4 << 10

// GameService define os casos de uso de partida consumidos pela API HTTP.
type GameService interface {
	Start(ctx context.Context, level exercise.Level, sessionID string) (game.StartResult, error)
	Answer(ctx context.Context, sessionID, exerciseID, answer string) (game.AnswerResult, error)
}

// StartGameRequest representa o corpo da requisição de início de fase.
type StartGameRequest struct {
	Level     string `json:"level" example:"beginner"`
	SessionID string `json:"sessionId,omitempty" example:"9f1c2b3a4d5e6f708192a3b4c5d6e7f8"`
}

// GameAnswerRequest representa o corpo da requisição de resposta dentro de uma partida.
type GameAnswerRequest struct {
	ExerciseID string `json:"exerciseId,omitempty" example:"beg-001"`
	Answer     string `json:"answer" example:"pato"`
}

// handleStartGame godoc
//
//	@Summary		Inicia uma fase da partida
//	@Description	Cria (ou retoma, informando sessionId) uma partida em memória e devolve o primeiro exercício do nível. Nenhum dado pessoal é armazenado.
//	@Tags			game
//	@Accept			json
//	@Produce		json
//	@Param			request	body		StartGameRequest	true	"Nível a iniciar e sessão opcional a retomar"
//	@Success		200		{object}	game.StartResult
//	@Failure		400		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		503		{object}	ErrorResponse
//	@Router			/api/v1/game/start [post]
func handleStartGame(logger *slog.Logger, svc GameService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req StartGameRequest
		if !decodeJSONBody(w, r, logger, &req) {
			return
		}

		level := exercise.Level(strings.TrimSpace(req.Level))
		if level == "" {
			level = exercise.LevelBeginner
		}
		if !level.IsValid() {
			writeError(w, logger, http.StatusBadRequest, "nível inválido ou não suportado")
			return
		}

		result, err := svc.Start(r.Context(), level, strings.TrimSpace(req.SessionID))
		if err != nil {
			writeGameError(w, logger, err, "falha ao iniciar partida")
			return
		}

		writeJSON(w, logger, http.StatusOK, result)
	}
}

// handleGameAnswer godoc
//
//	@Summary		Responde o exercício atual da partida
//	@Description	Valida a resposta no backend e devolve XP, streak, conquistas e o próximo exercício. A pontuação nunca é calculada pelo cliente.
//	@Tags			game
//	@Accept			json
//	@Produce		json
//	@Param			sessionId	path		string				true	"Identificador da partida"
//	@Param			request		body		GameAnswerRequest	true	"Resposta do usuário"
//	@Success		200			{object}	game.AnswerResult
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Router			/api/v1/game/{sessionId}/answer [post]
func handleGameAnswer(logger *slog.Logger, svc GameService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := strings.TrimSpace(r.PathValue("sessionId"))
		if sessionID == "" {
			writeError(w, logger, http.StatusBadRequest, "identificador da partida não pode ser vazio")
			return
		}

		var req GameAnswerRequest
		if !decodeJSONBody(w, r, logger, &req) {
			return
		}

		if strings.TrimSpace(req.Answer) == "" {
			writeError(w, logger, http.StatusBadRequest, "a resposta não pode ser vazia")
			return
		}

		result, err := svc.Answer(r.Context(), sessionID, strings.TrimSpace(req.ExerciseID), req.Answer)
		if err != nil {
			writeGameError(w, logger, err, "falha ao processar resposta da partida")
			return
		}

		writeJSON(w, logger, http.StatusOK, result)
	}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, logger *slog.Logger, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxAnswerBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, logger, http.StatusBadRequest, "corpo da requisição inválido")
		return false
	}
	return true
}

func writeGameError(w http.ResponseWriter, logger *slog.Logger, err error, logMsg string) {
	switch {
	case errors.Is(err, game.ErrSessionNotFound):
		writeError(w, logger, http.StatusNotFound, "partida não encontrada ou expirada")
	case errors.Is(err, exercise.ErrExerciseNotFound):
		writeError(w, logger, http.StatusNotFound, "exercício não encontrado")
	case errors.Is(err, exercise.ErrLevelNotFound), errors.Is(err, game.ErrNoExercises):
		writeError(w, logger, http.StatusNotFound, "nível não encontrado")
	case errors.Is(err, exercise.ErrInvalidLevel):
		writeError(w, logger, http.StatusBadRequest, "nível inválido ou não suportado")
	case errors.Is(err, exercise.ErrEmptyAnswer):
		writeError(w, logger, http.StatusBadRequest, "a resposta não pode ser vazia")
	case errors.Is(err, game.ErrLevelLocked):
		writeError(w, logger, http.StatusForbidden, "conclua o nível anterior para desbloquear este mundo")
	case errors.Is(err, game.ErrExerciseMismatch), errors.Is(err, game.ErrPhaseFinished):
		writeError(w, logger, http.StatusConflict, "a partida não está aguardando esta resposta")
	case errors.Is(err, game.ErrTooManySessions):
		writeError(w, logger, http.StatusServiceUnavailable, "servidor ocupado, tente novamente em instantes")
	default:
		logger.Error(logMsg, slog.Any("error", err))
		writeError(w, logger, http.StatusInternalServerError, "erro interno do servidor")
	}
}
