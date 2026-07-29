package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"serveoapi/internal/core/config"
)

func Start(ctx context.Context, cfg *config.Config, handler http.Handler) {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 ServeoAPI %s is running on port %s [%s]", config.AppVersion, cfg.Port, cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("❌ Server failure: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("⚠️ Context cancelled, starting graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("❌ Forced shutdown error: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}
