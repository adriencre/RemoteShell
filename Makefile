# Makefile pour RemoteShell

# Variables
VERSION ?= 1.0.0
BUILD_DIR = build
DIST_DIR = dist
GO_VERSION = 1.21

# Couleurs
BLUE = \033[0;34m
GREEN = \033[0;32m
YELLOW = \033[1;33m
RED = \033[0;31m
NC = \033[0m # No Color

.PHONY: help build clean test lint web agent server install deps

# Aide par défaut
help: ## Afficher l'aide
	@echo "$(BLUE)RemoteShell - Makefile$(NC)"
	@echo ""
	@echo "Commandes disponibles:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Vérifier les dépendances
deps: ## Vérifier les dépendances
	@echo "$(BLUE)[INFO]$(NC) Vérification des dépendances..."
	@command -v go >/dev/null 2>&1 || { echo "$(RED)[ERROR]$(NC) Go n'est pas installé"; exit 1; }
	@command -v node >/dev/null 2>&1 || { echo "$(RED)[ERROR]$(NC) Node.js n'est pas installé"; exit 1; }
	@echo "$(GREEN)[SUCCESS]$(NC) Toutes les dépendances sont installées"

# Nettoyer
clean: ## Nettoyer les fichiers de build
	@echo "$(BLUE)[INFO]$(NC) Nettoyage..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@cd web && rm -rf node_modules dist
	@echo "$(GREEN)[SUCCESS]$(NC) Nettoyage terminé"

# Installer les dépendances Go
go-deps: ## Installer les dépendances Go
	@echo "$(BLUE)[INFO]$(NC) Installation des dépendances Go..."
	@go mod download
	@go mod tidy
	@echo "$(GREEN)[SUCCESS]$(NC) Dépendances Go installées"

# Installer les dépendances web
web-deps: ## Installer les dépendances web
	@echo "$(BLUE)[INFO]$(NC) Installation des dépendances web..."
	@cd web && npm install
	@echo "$(GREEN)[SUCCESS]$(NC) Dépendances web installées"

# Build de l'interface web
web: web-deps ## Build de l'interface web
	@echo "$(BLUE)[INFO]$(NC) Build de l'interface web..."
	@cd web && npm run build
	@mkdir -p $(BUILD_DIR)
	@cp -r web/dist $(BUILD_DIR)/web
	@echo "$(GREEN)[SUCCESS]$(NC) Interface web buildée"

# Build de l'agent
agent: go-deps ## Build de l'agent
	@echo "$(BLUE)[INFO]$(NC) Build de l'agent..."
	@go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/rms-agent ./cmd/agent
	@echo "$(GREEN)[SUCCESS]$(NC) Agent buildé"

# Build du serveur
server: go-deps ## Build du serveur
	@echo "$(BLUE)[INFO]$(NC) Build du serveur..."
	@go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/remoteshell-server ./cmd/server
	@echo "$(GREEN)[SUCCESS]$(NC) Serveur buildé"

# Build complet
build: clean deps web agent server ## Build complet
	@echo "$(GREEN)[SUCCESS]$(NC) Build complet terminé"

# Build multi-plateforme
build-all: clean deps web ## Build multi-plateforme
	@echo "$(BLUE)[INFO]$(NC) Build multi-plateforme..."
	@./scripts/build.sh
	@echo "$(GREEN)[SUCCESS]$(NC) Build multi-plateforme terminé"

# Tests
test: go-deps ## Exécuter les tests
	@echo "$(BLUE)[INFO]$(NC) Exécution des tests..."
	@go test ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Tests terminés"

# Linting
lint: ## Exécuter le linting
	@echo "$(BLUE)[INFO]$(NC) Linting..."
	@cd web && npm run lint
	@echo "$(GREEN)[SUCCESS]$(NC) Linting terminé"

# Démarrer le serveur en mode développement
dev-server: ## Démarrer le serveur en mode développement
	@echo "$(BLUE)[INFO]$(NC) Démarrage du serveur en mode développement..."
	@go run ./cmd/server

# Démarrer l'agent en mode développement
dev-agent: ## Démarrer l'agent en mode développement
	@echo "$(BLUE)[INFO]$(NC) Démarrage de l'agent en mode développement..."
	@go run ./cmd/agent --server localhost:8080 --token test-token

# Démarrer l'interface web en mode développement
dev-web: web-deps ## Démarrer l'interface web en mode développement
	@echo "$(BLUE)[INFO]$(NC) Démarrage de l'interface web en mode développement..."
	@cd web && npm run dev

# Installation
install: build ## Installer RemoteShell
	@echo "$(BLUE)[INFO]$(NC) Installation de RemoteShell..."
	@sudo cp $(BUILD_DIR)/remoteshell-server /usr/local/bin/
	@sudo cp $(BUILD_DIR)/rms-agent /usr/local/bin/
	@echo "$(GREEN)[SUCCESS]$(NC) RemoteShell installé"

# Désinstallation
uninstall: ## Désinstaller RemoteShell
	@echo "$(BLUE)[INFO]$(NC) Désinstallation de RemoteShell..."
	@sudo rm -f /usr/local/bin/remoteshell-server
	@sudo rm -f /usr/local/bin/rms-agent
	@echo "$(GREEN)[SUCCESS]$(NC) RemoteShell désinstallé"

# Docker
docker-build: ## Build de l'image Docker
	@echo "$(BLUE)[INFO]$(NC) Build de l'image Docker..."
	@docker build -t remoteshell:$(VERSION) .
	@echo "$(GREEN)[SUCCESS]$(NC) Image Docker buildée"

# Formatage du code
fmt: ## Formater le code Go
	@echo "$(BLUE)[INFO]$(NC) Formatage du code..."
	@go fmt ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Code formaté"

# Vérification des vulnérabilités
security: go-deps ## Vérifier les vulnérabilités
	@echo "$(BLUE)[INFO]$(NC) Vérification des vulnérabilités..."
	@go list -json -deps ./... | nancy sleuth
	@echo "$(GREEN)[SUCCESS]$(NC) Vérification terminée"

# Génération de la documentation
docs: ## Générer la documentation
	@echo "$(BLUE)[INFO]$(NC) Génération de la documentation..."
	@go doc -all ./... > docs/api.md
	@echo "$(GREEN)[SUCCESS]$(NC) Documentation générée"

# Version
version: ## Afficher la version
	@echo "$(BLUE)RemoteShell v$(VERSION)$(NC)"
