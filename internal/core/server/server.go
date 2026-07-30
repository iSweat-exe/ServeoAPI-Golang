package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"serveoapi/internal/core/config"
)

func Start(
	ctx context.Context,
	cfg *config.Config,
	handler http.Handler,
) {
	server := &http.Server{
		Addr:        fmt.Sprintf(":%s", cfg.Port),
		Handler:     handler,
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	go func() {
		slog.Info(
			"ServeoAPI is running",
			"version",
			config.AppVersion,
			"port",
			cfg.Port,
			"env",
			cfg.Env,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failure", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("Context cancelled, starting graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Forced shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped gracefully")
}
