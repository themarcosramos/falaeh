package httpapi_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/httpapi"
)

func TestRouter_Health(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := httpapi.NewRouter(logger)

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

	// Verifica headers de segurança
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("header X-Content-Type-Options ausente ou incorreto")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("header X-Frame-Options ausente ou incorreto")
	}
}

func TestRouter_Docs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := httpapi.NewRouter(logger)

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
