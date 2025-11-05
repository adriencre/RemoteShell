#!/bin/bash

# Script de déploiement automatique du serveur RemoteShell
# À utiliser avec un webhook Git ou un cron job

set -e

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Obtenir le répertoire du script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

log "Répertoire du projet: $PROJECT_DIR"

# Vérifier que Go est installé
if ! command -v go &> /dev/null; then
    error "Go n'est pas installé"
    exit 1
fi

# Vérifier que Node.js est installé (pour le build web)
if ! command -v node &> /dev/null; then
    warning "Node.js n'est pas installé, le build web sera ignoré"
    BUILD_WEB=false
else
    BUILD_WEB=true
fi

# 1. Mettre à jour depuis Git
log "Mise à jour depuis Git..."
if [ -d ".git" ]; then
    git fetch origin
    git pull origin main || git pull origin master
    success "Code mis à jour depuis Git"
else
    warning "Ce n'est pas un dépôt Git, saut de la mise à jour"
fi

# 2. Builder les binaires multi-plateformes (nécessaires pour l'API /download/agent)
log "Build des binaires multi-plateformes..."
if [ -f "scripts/build.sh" ]; then
    chmod +x scripts/build.sh
    ./scripts/build.sh
    success "Binaires multi-plateformes buildés"
else
    error "Script build.sh non trouvé"
    exit 1
fi

# 3. Vérifier que les binaires existent
log "Vérification des binaires..."
REQUIRED_BINARIES=(
    "build/agent-linux-amd64"
    "build/agent-linux-arm64"
)

MISSING_BINARIES=()
for binary in "${REQUIRED_BINARIES[@]}"; do
    if [ ! -f "$binary" ]; then
        MISSING_BINARIES+=("$binary")
    fi
done

if [ ${#MISSING_BINARIES[@]} -gt 0 ]; then
    error "Binaires manquants: ${MISSING_BINARIES[*]}"
    exit 1
fi

success "Tous les binaires requis sont présents"

# 4. Builder le serveur (si nécessaire)
log "Build du serveur..."
if [ ! -f "build/remoteshell-server" ] || [ "build/remoteshell-server" -ot "cmd/server/main.go" ]; then
    log "Compilation du serveur..."
    go build -ldflags "-X main.version=1.0.0" -o build/remoteshell-server ./cmd/server
    success "Serveur buildé"
else
    log "Serveur déjà à jour"
fi

# 5. Redémarrer le serveur (si c'est un service systemd)
if systemctl is-active --quiet remoteshell-server 2>/dev/null || systemctl is-active --quiet remoteshell 2>/dev/null; then
    SERVICE_NAME=""
    if systemctl is-active --quiet remoteshell-server 2>/dev/null; then
        SERVICE_NAME="remoteshell-server"
    elif systemctl is-active --quiet remoteshell 2>/dev/null; then
        SERVICE_NAME="remoteshell"
    fi
    
    if [ -n "$SERVICE_NAME" ]; then
        log "Redémarrage du service $SERVICE_NAME..."
        sudo systemctl restart "$SERVICE_NAME"
        sleep 2
        
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            success "Service $SERVICE_NAME redémarré avec succès"
        else
            error "Le service $SERVICE_NAME n'a pas démarré correctement"
            systemctl status "$SERVICE_NAME" --no-pager || true
            exit 1
        fi
    fi
else
    warning "Aucun service systemd actif trouvé"
    log "Pour démarrer le serveur manuellement:"
    log "  ./build/remoteshell-server"
fi

# 6. Afficher les binaires disponibles
log "Binaires disponibles pour téléchargement:"
ls -lh build/agent-* 2>/dev/null | awk '{print "  - " $9 " (" $5 ")"}'

success "Déploiement terminé avec succès !"

