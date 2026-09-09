package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/feedback"
	"github.com/themarcosramos/falaeh/backend/internal/httpapi"
)

func TestFeedbackAcceptance_Scenarios(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	feedbackService := feedback.NewService(nil, logger)
	router := httpapi.NewRouter(logger, nil, nil, feedbackService)

	t.Run("Critério 1: Envio anônimo para cada uma das quatro opções fechadas", func(t *testing.T) {
		options := []string{"gostei_muito", "gostei", "mais_ou_menos", "nao_gostei"}

		for _, opt := range options {
			payload, _ := json.Marshal(map[string]string{"rating": opt})
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/feedback", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("opção %q falhou com código %d, esperado %d", opt, rec.Code, http.StatusOK)
			}

			var resp httpapi.FeedbackResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("falha ao decodificar resposta JSON para %q: %v", opt, err)
			}
			if resp.Status != "ok" {
				t.Errorf("status inesperado para %q: %s", opt, resp.Status)
			}
		}
	})

	t.Run("Critério 2: Rejeição amigável de respostas abertas ou opções inválidas", func(t *testing.T) {
		invalidInputs := []string{"excelente", "pessimo", "texto livre qualquer", "123", ""}

		for _, input := range invalidInputs {
			payload, _ := json.Marshal(map[string]string{"rating": input})
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/feedback", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("entrada inválida %q deveria retornar 400, retornou %d", input, rec.Code)
			}
		}
	})

	t.Run("Critério 3: Isolamento total — nenhum dado pessoal ou da partida é exigido", func(t *testing.T) {
		// Envia exclusivamente a chave "rating"
		payload, _ := json.Marshal(map[string]string{
			"rating": "gostei_muito",
		})
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/feedback", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("esperado 200 OK sem dados da partida, obtido %d", rec.Code)
		}
	})

	t.Run("Critério 4: Suporte a comentário descritivo opcional", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]string{
			"rating":  "gostei",
			"comment": "Adorei os efeitos visuais e o planeta dos sons!",
		})
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/feedback", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("esperado 200 OK com comentário opcional, obtido %d", rec.Code)
		}
	})
}
