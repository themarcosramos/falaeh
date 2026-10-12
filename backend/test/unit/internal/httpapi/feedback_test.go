package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/feedback"
	"github.com/themarcosramos/falaeh/backend/internal/httpapi"
)

type mockFeedbackService struct {
	submittedRating  string
	submittedComment string
	errToReturn      error
}

func (m *mockFeedbackService) Submit(_ context.Context, rawRating, rawComment string) error {
	m.submittedRating = rawRating
	m.submittedComment = rawComment
	return m.errToReturn
}

func TestHandleFeedback(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("sucesso ao enviar avaliação válida com comentário", func(t *testing.T) {
		mockSvc := &mockFeedbackService{}
		router := httpapi.NewRouter(logger, nil, nil, mockSvc)

		body := map[string]string{
			"rating":  "gostei_muito",
			"comment": "jogo incrível!",
		}
		payload, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("código esperado %d, obtido %d", http.StatusOK, rec.Code)
		}

		var resp httpapi.FeedbackResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("falha ao decodificar resposta: %v", err)
		}
		if resp.Status != "ok" {
			t.Errorf("status esperado ok, obtido %s", resp.Status)
		}
		if mockSvc.submittedRating != "gostei_muito" {
			t.Errorf("rating esperado gostei_muito, obtido %s", mockSvc.submittedRating)
		}
		if mockSvc.submittedComment != "jogo incrível!" {
			t.Errorf("comment esperado 'jogo incrível!', obtido %s", mockSvc.submittedComment)
		}
	})

	t.Run("rejeição com 400 em avaliação vazia", func(t *testing.T) {
		mockSvc := &mockFeedbackService{errToReturn: feedback.ErrEmptyRating}
		router := httpapi.NewRouter(logger, nil, nil, mockSvc)

		body := map[string]string{"rating": ""}
		payload, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("código esperado %d, obtido %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("rejeição com 400 em avaliação desconhecida", func(t *testing.T) {
		mockSvc := &mockFeedbackService{errToReturn: feedback.ErrInvalidRating}
		router := httpapi.NewRouter(logger, nil, nil, mockSvc)

		body := map[string]string{"rating": "nao_sei"}
		payload, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("código esperado %d, obtido %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("rejeição com 400 em corpo JSON inválido", func(t *testing.T) {
		mockSvc := &mockFeedbackService{}
		router := httpapi.NewRouter(logger, nil, nil, mockSvc)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader([]byte("{invalid-json")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("código esperado %d, obtido %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("falha em provedor externo não bloqueia e retorna 200", func(t *testing.T) {
		mockSvc := &mockFeedbackService{errToReturn: errors.New("timeout no google forms")}
		router := httpapi.NewRouter(logger, nil, nil, mockSvc)

		body := map[string]string{"rating": "gostei"}
		payload, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("código esperado %d mesmo com falha externa, obtido %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("serviço nil retorna 503", func(t *testing.T) {
		router := httpapi.NewRouter(logger, nil, nil)

		body := map[string]string{"rating": "gostei"}
		payload, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		// Sem o serviço configurado no NewRouter, o endpoint sequer é registrado (404)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("código esperado %d, obtido %d", http.StatusNotFound, rec.Code)
		}
	})
}
