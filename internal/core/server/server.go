package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"serveoapi/internal/core/config"
)

func Start(cfg *config.Config, handler http.Handler) {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 ServeoAPI %s is running on port %s [%s]", config.AppVersion, cfg.Port, cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("❌ Server failure: %v", err)
		}
	}()

	sig := <-shutdownChan
	log.Printf("⚠️ Signal received (%s), starting graceful shutdown...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Forced shutdown error: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}
