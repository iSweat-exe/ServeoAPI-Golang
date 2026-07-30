package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/database"
	"serveoapi/internal/core/server"
	"serveoapi/internal/core/updater"
	"serveoapi/internal/modules/v2/apikeys"
	"serveoapi/internal/modules/v2/auth"
	"serveoapi/internal/modules/v2/metrics"
	"serveoapi/internal/router"

	"github.com/docker/docker/client"
)

// @title           ServeoAPI
// @version         2.0
// @description     This is the modular ServeoAPI management server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

func main() {
	// Configure le logger JSON par défaut
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Intercepteur CLI pour la mise à jour automatique
	if len(os.Args) > 1 && os.Args[1] == "update" {
		updater.RunCheckAndUpdate()
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := config.Load()

	// Initialisation de la base de données
	if err := database.InitDatabase("serveo.db"); err != nil {
		slog.Error("Impossible d'initialiser la base de données", "error", err)
		os.Exit(1)
	}
	if err := auth.MigrateDatabase(); err != nil {
		slog.Error("Impossible de migrer la base de données Auth", "error", err)
		os.Exit(1)
	}
	if err := apikeys.MigrateDatabase(); err != nil {
		slog.Error("Impossible de migrer la base de données ApiKeys", "error", err)
		os.Exit(1)
	}
	if err := metrics.MigrateDatabase(); err != nil {
		slog.Error("Impossible de migrer la base de données Metrics", "error", err)
		os.Exit(1)
	}

	// Initialisation du client Docker
	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("Impossible d'initialiser le client Docker", "error", err)
		os.Exit(1)
	}
	defer dockerCli.Close()

	// Démarrage des workers en arrière-plan
	metrics.StartMetricsWorker(ctx, 5*time.Minute, dockerCli)

	apiHandler := router.New(cfg, dockerCli)

	// Démarrage du serveur (bloquant, avec arrêt gracieux)
	server.Start(ctx, cfg, apiHandler)
}
