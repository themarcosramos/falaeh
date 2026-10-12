package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/game"
	"github.com/themarcosramos/falaeh/backend/internal/gamification"
	"github.com/themarcosramos/falaeh/backend/internal/httpapi"
)

func acceptanceRepository(t *testing.T) *exercise.JSONRepository {
	t.Helper()

	candidates := []string{"../../../../data", "../../../data", "../../data", "../data", "data"}
	var dirFS fs.FS
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "beginner.json")); err == nil {
			dirFS = os.DirFS(c)
			break
		}
	}
	if dirFS == nil {
		t.Fatalf("diretório de dados (backend/data) não encontrado: %v", candidates)
	}

	repo, err := exercise.NewJSONRepository(dirFS)
	if err != nil {
		t.Fatalf("falha ao instanciar repositório de dados: %v", err)
	}

	return repo
}

func newAcceptanceRouter(t *testing.T) http.Handler {
	t.Helper()

	svc := exercise.NewService(acceptanceRepository(t))
	manager := game.NewManager(svc, gamification.DefaultRules(), game.Config{})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httpapi.NewRouter(logger, svc, manager)
}

func TestRouter_Health(t *testing.T) {
	router := newAcceptanceRouter(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}

	var res httpapi.HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("erro ao decodificar json: %v", err)
	}

	if res.Status != "ok" {
		t.Errorf("status = %q, esperado 'ok'", res.Status)
	}

	// Testa também rota com prefixo /api/health
	recAPI := httptest.NewRecorder()
	reqAPI := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	router.ServeHTTP(recAPI, reqAPI)
	if recAPI.Code != http.StatusOK {
		t.Fatalf("status code /api/health = %d, esperado %d", recAPI.Code, http.StatusOK)
	}

	// Verifica headers de segurança
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("header X-Content-Type-Options ausente ou incorreto")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("header X-Frame-Options ausente ou incorreto")
	}
}

func TestRouter_Docs(t *testing.T) {
	router := newAcceptanceRouter(t)

	t.Run("Swagger UI", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/docs", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("Swagger Spec", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/swagger.yaml", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
		}
	})
}

func TestRouter_Levels(t *testing.T) {
	router := newAcceptanceRouter(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/levels", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}

	var levels []exercise.LevelInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &levels); err != nil {
		t.Fatalf("falha ao decodificar resposta de níveis: %v", err)
	}

	if len(levels) != 3 {
		t.Fatalf("esperava 3 níveis, obteve %d", len(levels))
	}
	if levels[0].ID != exercise.LevelBeginner || levels[0].Name == "" {
		t.Errorf("nível iniciante inválido: %+v", levels[0])
	}
}

func TestRouter_ListExercisesByLevel(t *testing.T) {
	router := newAcceptanceRouter(t)

	t.Run("nível válido iniciante", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/levels/beginner/exercises", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
		}

		var exercises []exercise.PublicExercise
		if err := json.Unmarshal(rec.Body.Bytes(), &exercises); err != nil {
			t.Fatalf("falha ao decodificar exercícios: %v", err)
		}

		if len(exercises) == 0 {
			t.Fatalf("esperava exercícios no nível iniciante")
		}

		// Garante que o campo correctAnswer não existe no payload retornado
		var rawList []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &rawList); err != nil {
			t.Fatalf("falha ao decodificar raw json: %v", err)
		}
		for _, raw := range rawList {
			if _, exists := raw["correctAnswer"]; exists {
				t.Errorf("segurança violada: correctAnswer vazou no payload público: %+v", raw)
			}
		}
	})

	t.Run("nível inválido ou inexistente", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/levels/inexistente/exercises", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusNotFound)
		}

		var errRes httpapi.ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &errRes); err != nil {
			t.Fatalf("falha ao decodificar resposta de erro: %v", err)
		}
		if errRes.Error != "nível não encontrado" {
			t.Errorf("mensagem de erro = %q, esperava 'nível não encontrado'", errRes.Error)
		}
	})
}

func TestRouter_ListExercises(t *testing.T) {
	router := newAcceptanceRouter(t)

	t.Run("listar todos os exercícios", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
		}

		var list []exercise.PublicExercise
		if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
			t.Fatalf("falha ao decodificar lista: %v", err)
		}
		if len(list) == 0 {
			t.Fatalf("esperava lista não-vazia de exercícios")
		}
	})

	t.Run("listar com query param de nível válido", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises?level=beginner", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
		}

		var list []exercise.PublicExercise
		if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
			t.Fatalf("falha ao decodificar lista: %v", err)
		}
		for _, ex := range list {
			if ex.Level != exercise.LevelBeginner {
				t.Errorf("exercício com nível inesperado: %s", ex.Level)
			}
		}
	})

	t.Run("listar com query param de nível inválido", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises?level=invalido", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestRouter_GetExercise(t *testing.T) {
	router := newAcceptanceRouter(t)

	t.Run("exercício existente", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/beg-001", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
		}

		var ex exercise.PublicExercise
		if err := json.Unmarshal(rec.Body.Bytes(), &ex); err != nil {
			t.Fatalf("falha ao decodificar exercício: %v", err)
		}
		if ex.ID != "beg-001" {
			t.Errorf("ID esperado beg-001, obteve %s", ex.ID)
		}

		// Checa que correctAnswer não está presente
		var raw map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
			t.Fatalf("falha ao decodificar raw json: %v", err)
		}
		if _, exists := raw["correctAnswer"]; exists {
			t.Errorf("correctAnswer não deve estar presente no endpoint público")
		}
	})

	t.Run("exercício inexistente", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/inexistente-999", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusNotFound)
		}
	})
}

func TestRouter_AnswerExercise(t *testing.T) {
	router := newAcceptanceRouter(t)

	t.Run("resposta correta", func(t *testing.T) {
		body := bytes.NewBufferString(`{"answer":"Pato"}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/beg-001/answer", body)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
		}

		var res exercise.ValidationResult
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("falha ao decodificar resposta: %v", err)
		}
		if !res.Correct || res.ExerciseID != "beg-001" {
			t.Errorf("resultado inesperado: %+v", res)
		}
	})

	t.Run("resposta incorreta", func(t *testing.T) {
		body := bytes.NewBufferString(`{"answer":"Gato"}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/beg-001/answer", body)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
		}

		var res exercise.ValidationResult
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("falha ao decodificar resposta: %v", err)
		}
		if res.Correct {
			t.Errorf("esperava resposta incorreta")
		}
	})

	t.Run("resposta com espaços e pontuação normalizada", func(t *testing.T) {
		body := bytes.NewBufferString(`{"answer":"  pato! "}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/beg-001/answer", body)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
		}

		var res exercise.ValidationResult
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("falha ao decodificar resposta: %v", err)
		}
		if !res.Correct {
			t.Errorf("esperava resposta aceita com normalização")
		}
	})

	t.Run("resposta vazia", func(t *testing.T) {
		body := bytes.NewBufferString(`{"answer":"   "}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/beg-001/answer", body)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("corpo JSON inválido", func(t *testing.T) {
		body := bytes.NewBufferString(`{invalido`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/beg-001/answer", body)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("exercício inexistente", func(t *testing.T) {
		body := bytes.NewBufferString(`{"answer":"Pato"}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/inexistente-999/answer", body)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusNotFound)
		}
	})
}
