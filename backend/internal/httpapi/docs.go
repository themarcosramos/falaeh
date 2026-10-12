package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/themarcosramos/falaeh/backend/docs"
)

func handleSwaggerSpec(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		writeBytes(w, logger, docs.Spec)
	}
}

func handleSwaggerUI(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		writeBytes(w, logger, docs.UI)
	}
}

func writeBytes(w http.ResponseWriter, logger *slog.Logger, content []byte) {
	if _, err := w.Write(content); err != nil {
		logger.Error("falha ao escrever resposta", slog.Any("error", err))
	}
}
