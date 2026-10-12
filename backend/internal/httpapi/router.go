// Package httpapi expõe a camada de entrega HTTP da aplicação.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// NewRouter monta as rotas da API. Novos domínios devem registrar suas rotas aqui.
func NewRouter(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth(logger))
	mux.HandleFunc("GET /docs", handleSwaggerUI(logger))
	mux.HandleFunc("GET /swagger.yaml", handleSwaggerSpec(logger))

	return securityHeaders(mux)
}

// HealthResponse informa a disponibilidade do serviço.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// handleHealth godoc
//
//	@Summary		Verifica se a API está disponível
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Router			/health [get]
func handleHealth(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, logger, http.StatusOK, HealthResponse{Status: "ok"})
	}
}

func writeJSON(w http.ResponseWriter, logger *slog.Logger, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logger.Error("falha ao escrever resposta JSON", slog.Any("error", err))
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")

		next.ServeHTTP(w, r)
	})
}
