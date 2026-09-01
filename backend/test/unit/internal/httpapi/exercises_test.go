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
	"github.com/themarcosramos/falaeh/backend/internal/httpapi"
)

type mockExerciseService struct {
	listLevelsFn      func(ctx context.Context) []exercise.LevelInfo
	listAllFn         func(ctx context.Context) ([]exercise.PublicExercise, error)
	listByLevelFn     func(ctx context.Context, level exercise.Level) ([]exercise.PublicExercise, error)
	getExerciseByIDFn func(ctx context.Context, id string) (exercise.PublicExercise, error)
	validateAnswerFn  func(ctx context.Context, exerciseID string, answer string) (exercise.ValidationResult, error)
}

func (m *mockExerciseService) ListLevels(ctx context.Context) []exercise.LevelInfo {
	if m.listLevelsFn != nil {
		return m.listLevelsFn(ctx)
	}
	return exercise.AvailableLevels
}

func (m *mockExerciseService) ListAll(ctx context.Context) ([]exercise.PublicExercise, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx)
	}
	return nil, nil
}

func (m *mockExerciseService) ListByLevel(ctx context.Context, level exercise.Level) ([]exercise.PublicExercise, error) {
	if m.listByLevelFn != nil {
		return m.listByLevelFn(ctx, level)
	}
	return nil, nil
}

func (m *mockExerciseService) GetExerciseByID(ctx context.Context, id string) (exercise.PublicExercise, error) {
	if m.getExerciseByIDFn != nil {
		return m.getExerciseByIDFn(ctx, id)
	}
	return exercise.PublicExercise{}, nil
}

func (m *mockExerciseService) ValidateAnswer(ctx context.Context, exerciseID string, answer string) (exercise.ValidationResult, error) {
	if m.validateAnswerFn != nil {
		return m.validateAnswerFn(ctx, exerciseID, answer)
	}
	return exercise.ValidationResult{ExerciseID: exerciseID, Correct: true}, nil
}

func TestHTTP_ListLevels(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		listLevelsFn: func(ctx context.Context) []exercise.LevelInfo {
			return []exercise.LevelInfo{
				{ID: exercise.LevelBeginner, Name: "Iniciante", Description: "Mundo 1"},
			}
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/levels", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestHTTP_ListExercises_All(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		listAllFn: func(ctx context.Context) ([]exercise.PublicExercise, error) {
			return []exercise.PublicExercise{
				{ID: "ex-1", Level: exercise.LevelBeginner},
			}, nil
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestHTTP_ListExercises_WithValidLevel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		listByLevelFn: func(ctx context.Context, level exercise.Level) ([]exercise.PublicExercise, error) {
			return []exercise.PublicExercise{
				{ID: "ex-1", Level: level},
			}, nil
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises?level=beginner", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestHTTP_ListExercises_LevelNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		listByLevelFn: func(ctx context.Context, level exercise.Level) ([]exercise.PublicExercise, error) {
			return nil, exercise.ErrLevelNotFound
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises?level=beginner", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}

func TestHTTP_ListExercises_InvalidLevel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises?level=invalido", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_ListExercises_InternalError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		listAllFn: func(ctx context.Context) ([]exercise.PublicExercise, error) {
			return nil, errors.New("db error")
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHTTP_ListExercisesByLevel_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		listByLevelFn: func(ctx context.Context, level exercise.Level) ([]exercise.PublicExercise, error) {
			return []exercise.PublicExercise{{ID: "ex-1", Level: level}}, nil
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/levels/beginner/exercises", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestHTTP_ListExercisesByLevel_InvalidLevel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/levels/nao-existe/exercises", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}

func TestHTTP_ListExercisesByLevel_LevelNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		listByLevelFn: func(ctx context.Context, level exercise.Level) ([]exercise.PublicExercise, error) {
			return nil, exercise.ErrLevelNotFound
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/levels/beginner/exercises", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}

func TestHTTP_ListExercisesByLevel_InternalError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		listByLevelFn: func(ctx context.Context, level exercise.Level) ([]exercise.PublicExercise, error) {
			return nil, errors.New("unexpected error")
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/levels/beginner/exercises", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHTTP_GetExercise_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		getExerciseByIDFn: func(ctx context.Context, id string) (exercise.PublicExercise, error) {
			return exercise.PublicExercise{ID: id, Level: exercise.LevelBeginner}, nil
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/beg-001", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestHTTP_GetExercise_NotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		getExerciseByIDFn: func(ctx context.Context, id string) (exercise.PublicExercise, error) {
			return exercise.PublicExercise{}, exercise.ErrExerciseNotFound
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/inexistente", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}

func TestHTTP_GetExercise_InternalError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		getExerciseByIDFn: func(ctx context.Context, id string) (exercise.PublicExercise, error) {
			return exercise.PublicExercise{}, errors.New("unexpected error")
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/ex-1", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHTTP_AnswerExercise_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		validateAnswerFn: func(ctx context.Context, exerciseID string, answer string) (exercise.ValidationResult, error) {
			return exercise.ValidationResult{ExerciseID: exerciseID, Correct: true}, nil
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/beg-001/answer", bytes.NewBufferString(`{"answer":"pato"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestHTTP_AnswerExercise_MalformedBody(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/beg-001/answer", bytes.NewBufferString(`{invalido`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_AnswerExercise_BodyTooLarge(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{}

	body := `{"answer":"` + strings.Repeat("a", 8<<10) + `"}`

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/beg-001/answer", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_AnswerExercise_EmptyAnswer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/beg-001/answer", bytes.NewBufferString(`{"answer":""}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_AnswerExercise_ExerciseNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		validateAnswerFn: func(ctx context.Context, exerciseID string, answer string) (exercise.ValidationResult, error) {
			return exercise.ValidationResult{}, exercise.ErrExerciseNotFound
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/inexistente/answer", bytes.NewBufferString(`{"answer":"pato"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}

func TestHTTP_AnswerExercise_EmptyAnswerErrorFromService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		validateAnswerFn: func(ctx context.Context, exerciseID string, answer string) (exercise.ValidationResult, error) {
			return exercise.ValidationResult{}, exercise.ErrEmptyAnswer
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/beg-001/answer", bytes.NewBufferString(`{"answer":"a"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_AnswerExercise_InternalError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockExerciseService{
		validateAnswerFn: func(ctx context.Context, exerciseID string, answer string) (exercise.ValidationResult, error) {
			return exercise.ValidationResult{}, errors.New("unexpected error")
		},
	}

	router := httpapi.NewRouter(logger, mockSvc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/ex-1/answer", bytes.NewBufferString(`{"answer":"pato"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHTTP_NilExerciseService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := httpapi.NewRouter(logger, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, esperado %d", rec.Code, http.StatusOK)
	}
}
