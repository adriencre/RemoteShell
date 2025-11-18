#!/bin/bash

# Script pour builder l'agent RemoteShell pour armv7l (ARM 32-bit)
set -e

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="build"
OUTPUT_FILE="$BUILD_DIR/agent-linux-armv7l"

echo -e "${BLUE}[INFO]${NC} Build de l'agent RemoteShell pour armv7l..."

# Vérifier que Go est installé
if ! command -v go &> /dev/null; then
    echo -e "${RED}[ERROR]${NC} Go n'est pas installé. Veuillez installer Go 1.21 ou plus récent."
    exit 1
fi

# Créer le répertoire de build
mkdir -p "$BUILD_DIR"

# Build de l'agent pour armv7l
echo -e "${BLUE}[INFO]${NC} Compilation de l'agent pour linux/armv7l..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build \
    -a -installsuffix cgo \
    -ldflags "-X main.version=$VERSION -extldflags '-static'" \
    -o "$OUTPUT_FILE" \
    ./cmd/agent

if [ -f "$OUTPUT_FILE" ]; then
    FILE_SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
    echo -e "${GREEN}[SUCCESS]${NC} Agent buildé avec succès!"
    echo -e "${GREEN}[SUCCESS]${NC} Fichier: $OUTPUT_FILE (${FILE_SIZE})"
    echo ""
    echo "Pour utiliser l'agent:"
    echo "  ./$OUTPUT_FILE --server HOST:PORT --token YOUR_TOKEN"
else
    echo -e "${RED}[ERROR]${NC} Le build a échoué"
    exit 1
fi


