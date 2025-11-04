#!/bin/bash

# Script pour démarrer l'agent RemoteShell

# Aller dans le répertoire du script
cd "$(dirname "$0")"

# Paramètres par défaut
SERVER="${1:-localhost:8080}"
TOKEN="${2:-test-token}"
NAME="${3:-Test Agent}"

echo "🚀 Démarrage de l'agent RemoteShell..."
echo "   Serveur: $SERVER"
echo "   Token: $TOKEN"
echo "   Nom: $NAME"
echo ""

if [ ! -f "build/rms-agent" ]; then
    echo "❌ L'agent n'est pas compilé. Exécutez d'abord: make build"
    exit 1
fi

./build/rms-agent --server "$SERVER" --token "$TOKEN" --name "$NAME"



