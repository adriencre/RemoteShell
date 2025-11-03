#!/bin/bash

# Script pour démarrer le serveur RemoteShell

# Aller dans le répertoire du script
cd "$(dirname "$0")"

echo "🚀 Démarrage du serveur RemoteShell..."
echo ""

if [ ! -f "build/remoteshell-server" ]; then
    echo "❌ Le serveur n'est pas compilé. Exécutez d'abord: make build"
    exit 1
fi

./build/remoteshell-server

