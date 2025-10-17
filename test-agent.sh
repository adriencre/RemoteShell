#!/bin/bash

# Script de test pour l'agent RemoteShell avec logs de débogage

echo "=== Test de l'agent RemoteShell avec corrections ==="
echo ""

# Vérifier que l'agent existe
if [ ! -f "build/remoteshell-agent" ]; then
    echo "❌ L'agent n'existe pas. Compilation nécessaire."
    exit 1
fi

echo "✅ Agent trouvé: build/remoteshell-agent"
echo ""

# Paramètres de test
SERVER="10.0.0.59:8081"
AGENT_ID="serveur-impression-01"
AGENT_NAME="Serveur d'impression principal"
TOKEN="test-token"

echo "🔧 Paramètres de test:"
echo "   Serveur: $SERVER"
echo "   Agent ID: $AGENT_ID"
echo "   Agent Name: $AGENT_NAME"
echo "   Token: $TOKEN"
echo ""

echo "🚀 Démarrage de l'agent avec logs de débogage..."
echo "   (Appuyez sur Ctrl+C pour arrêter)"
echo ""

# Démarrer l'agent avec les logs de débogage
./build/remoteshell-agent \
    --server "$SERVER" \
    --id "$AGENT_ID" \
    --name "$AGENT_NAME" \
    --token "$TOKEN"
