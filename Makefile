# Makefile pour ServeoAPI

APP_NAME=serveoapi
BUILD_DIR=bin

.PHONY: help build build-linux run test clean

help:
	@echo "Commandes disponibles :"
	@echo "  make run         - Lance l'API en mode dev"
	@echo "  make build       - Compile le binaire pour l'OS hôte"
	@echo "  make build-linux - Cross-compile le binaire exécutable pour Linux (amd64)"
	@echo "  make test        - Exécute les tests unitaires"
	@echo "  make clean       - Nettoie les binaires compilés"

run:
	go run ./cmd/api

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/api

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME)-linux ./cmd/api

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)
