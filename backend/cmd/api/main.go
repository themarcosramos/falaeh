// Package main inicializa a API HTTP do Fala Eh.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/themarcosramos/falaeh/backend/internal/httpapi"
)

const (
	defaultPort     = "8080"
	readTimeout     = 5 * time.Second
	writeTimeout    = 10 * time.Second
	idleTimeout     = 60 * time.Second
	shutdownTimeout = 10 * time.Second
)

// @title                     Fala Eh API
// @version                   0.1.0
// @description               API do jogo educativo de exercícios fonoaudiológicos. Não possui autenticação nem armazena dados pessoais.
// @license.name              MIT
// @license.url               https://github.com/themarcosramos/falaeh/blob/main/LICENSE
// @host                      localhost:8080
// @BasePath                  /
// @schemes                   http https
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("erro ao executar a aplicação", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	server := &http.Server{
		Addr:         ":" + port(),
		Handler:      httpapi.NewRouter(logger),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	serverErr := make(chan error, 1)

	go func() {
		logger.Info("servidor iniciado", slog.String("addr", server.Addr))

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("encerrando servidor")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

func port() string {
	if value := os.Getenv("APP_PORT"); value != "" {
		return value
	}

	return defaultPort
}
