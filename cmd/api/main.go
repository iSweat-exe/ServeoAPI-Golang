package main

import (
	"log"
	"serveoapi/internal/core/config"
	"serveoapi/internal/core/database"
	"serveoapi/internal/core/server"
	"serveoapi/internal/modules/v2/auth"
	"serveoapi/internal/router"
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
	cfg := config.Load()

	// Initialize Database
	if err := database.InitDatabase("serveo.db"); err != nil {
		log.Fatalf("Impossible d'initialiser la base de données: %v", err)
	}
	if err := auth.MigrateDatabase(); err != nil {
		log.Fatalf("Impossible de migrer la base de données Auth: %v", err)
	}

	// Create HTTP router
	apiHandler := router.New()

	// Start server (blocking with graceful shutdown)
	server.Start(cfg, apiHandler)
}
