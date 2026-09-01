package httpapi_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/game"
	"github.com/themarcosramos/falaeh/backend/internal/httpapi"
)

type mockGameService struct {
	startFn  func(ctx context.Context, level exercise.Level, sessionID string) (game.StartResult, error)
	answerFn func(ctx context.Context, sessionID, exerciseID, answer string) (game.AnswerResult, error)
}

func (m *mockGameService) Start(ctx context.Context, level exercise.Level, sessionID string) (game.StartResult, error) {
	if m.startFn != nil {
		return m.startFn(ctx, level, sessionID)
	}
	return game.StartResult{SessionID: "sess-1", Level: level}, nil
}

func (m *mockGameService) Answer(ctx context.Context, sessionID, exerciseID, answer string) (game.AnswerResult, error) {
	if m.answerFn != nil {
		return m.answerFn(ctx, sessionID, exerciseID, answer)
	}
	return game.AnswerResult{ExerciseID: exerciseID, Correct: true}, nil
}

func newGameRouter(svc httpapi.GameService) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httpapi.NewRouter(logger, nil, svc)
}

func postJSON(t *testing.T, router http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	return rec
}

func TestHTTP_StartGame_Success(t *testing.T) {
	var gotLevel exercise.Level
	router := newGameRouter(&mockGameService{
		startFn: func(ctx context.Context, level exercise.Level, sessionID string) (game.StartResult, error) {
			gotLevel = level
			return game.StartResult{SessionID: "sess-1", Level: level}, nil
		},
	})

	rec := postJSON(t, router, "/api/v1/game/start", `{"level":"intermediate"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}
	if gotLevel != exercise.LevelIntermediate {
		t.Errorf("level = %q, esperado intermediate", gotLevel)
	}
}

func TestHTTP_StartGame_DefaultsToBeginner(t *testing.T) {
	var gotLevel exercise.Level
	router := newGameRouter(&mockGameService{
		startFn: func(ctx context.Context, level exercise.Level, sessionID string) (game.StartResult, error) {
			gotLevel = level
			return game.StartResult{Level: level}, nil
		},
	})

	if rec := postJSON(t, router, "/api/v1/game/start", `{}`); rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}
	if gotLevel != exercise.LevelBeginner {
		t.Errorf("level = %q, esperado beginner", gotLevel)
	}
}

func TestHTTP_StartGame_InvalidLevel(t *testing.T) {
	router := newGameRouter(&mockGameService{})

	if rec := postJSON(t, router, "/api/v1/game/start", `{"level":"marte"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_StartGame_MalformedBody(t *testing.T) {
	router := newGameRouter(&mockGameService{})

	if rec := postJSON(t, router, "/api/v1/game/start", `{invalido`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_StartGame_BodyTooLarge(t *testing.T) {
	router := newGameRouter(&mockGameService{})

	body := `{"level":"beginner","sessionId":"` + strings.Repeat("a", 8<<10) + `"}`
	if rec := postJSON(t, router, "/api/v1/game/start", body); rec.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_StartGame_ErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{name: "nível bloqueado", err: game.ErrLevelLocked, expected: http.StatusForbidden},
		{name: "nível sem exercícios", err: game.ErrNoExercises, expected: http.StatusNotFound},
		{name: "nível não encontrado", err: exercise.ErrLevelNotFound, expected: http.StatusNotFound},
		{name: "limite de sessões", err: game.ErrTooManySessions, expected: http.StatusServiceUnavailable},
		{name: "erro inesperado", err: errors.New("boom"), expected: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newGameRouter(&mockGameService{
				startFn: func(ctx context.Context, level exercise.Level, sessionID string) (game.StartResult, error) {
					return game.StartResult{}, tt.err
				},
			})

			if rec := postJSON(t, router, "/api/v1/game/start", `{"level":"beginner"}`); rec.Code != tt.expected {
				t.Fatalf("status code = %d, esperado %d", rec.Code, tt.expected)
			}
		})
	}
}

func TestHTTP_GameAnswer_Success(t *testing.T) {
	var gotSession, gotExercise, gotAnswer string
	router := newGameRouter(&mockGameService{
		answerFn: func(ctx context.Context, sessionID, exerciseID, answer string) (game.AnswerResult, error) {
			gotSession, gotExercise, gotAnswer = sessionID, exerciseID, answer
			return game.AnswerResult{ExerciseID: exerciseID, Correct: true, EarnedXP: 150}, nil
		},
	})

	rec := postJSON(t, router, "/api/v1/game/sess-1/answer", `{"exerciseId":"beg-001","answer":"pato"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}
	if gotSession != "sess-1" || gotExercise != "beg-001" || gotAnswer != "pato" {
		t.Errorf("argumentos = %q/%q/%q", gotSession, gotExercise, gotAnswer)
	}
}

func TestHTTP_GameAnswer_EmptyAnswer(t *testing.T) {
	router := newGameRouter(&mockGameService{})

	if rec := postJSON(t, router, "/api/v1/game/sess-1/answer", `{"answer":"   "}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_GameAnswer_MalformedBody(t *testing.T) {
	router := newGameRouter(&mockGameService{})

	if rec := postJSON(t, router, "/api/v1/game/sess-1/answer", `{invalido`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_GameAnswer_ErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{name: "partida inexistente", err: game.ErrSessionNotFound, expected: http.StatusNotFound},
		{name: "exercício inexistente", err: exercise.ErrExerciseNotFound, expected: http.StatusNotFound},
		{name: "exercício divergente", err: game.ErrExerciseMismatch, expected: http.StatusConflict},
		{name: "fase concluída", err: game.ErrPhaseFinished, expected: http.StatusConflict},
		{name: "resposta vazia no domínio", err: exercise.ErrEmptyAnswer, expected: http.StatusBadRequest},
		{name: "erro inesperado", err: errors.New("boom"), expected: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newGameRouter(&mockGameService{
				answerFn: func(ctx context.Context, sessionID, exerciseID, answer string) (game.AnswerResult, error) {
					return game.AnswerResult{}, tt.err
				},
			})

			if rec := postJSON(t, router, "/api/v1/game/sess-1/answer", `{"answer":"pato"}`); rec.Code != tt.expected {
				t.Fatalf("status code = %d, esperado %d", rec.Code, tt.expected)
			}
		})
	}
}

func TestHTTP_NilGameService(t *testing.T) {
	router := newGameRouter(nil)

	if rec := postJSON(t, router, "/api/v1/game/start", `{"level":"beginner"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}
