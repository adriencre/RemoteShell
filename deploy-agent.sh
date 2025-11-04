#!/bin/bash

# Script pour déployer la nouvelle version de l'agent sur le serveur web

echo "=== Déploiement de l'agent RemoteShell ==="
echo ""

# Vérifier que l'agent existe
if [ ! -f "build/rms-agent" ]; then
    echo "❌ L'agent n'existe pas. Compilation nécessaire."
    echo "🔧 Compilation de l'agent..."
    make agent
    if [ $? -ne 0 ]; then
        echo "❌ Erreur lors de la compilation"
        exit 1
    fi
fi

echo "✅ Agent trouvé: build/rms-agent"
echo ""

# Vérifier la taille du fichier
SIZE=$(stat -c%s "build/rms-agent")
echo "📊 Taille de l'agent: $SIZE bytes"
echo ""

# Copier l'agent vers le répertoire web pour qu'il soit accessible via HTTP
echo "🚀 Déploiement de l'agent sur le serveur web..."

# Créer le répertoire web s'il n'existe pas
mkdir -p web/public

# Copier l'agent
cp build/rms-agent web/public/rms-agent

echo "✅ Agent déployé dans web/public/rms-agent"
echo ""

# Vérifier que le serveur web est en cours d'exécution
echo "🔍 Vérification du serveur web..."
if curl -s -o /dev/null -w "%{http_code}" http://10.0.0.59:8082/rms-agent | grep -q "200"; then
    echo "✅ Serveur web accessible sur http://10.0.0.59:8082/rms-agent"
else
    echo "⚠️  Serveur web non accessible. Assurez-vous que le serveur RemoteShell est démarré."
    echo "   Vous pouvez le démarrer avec: make dev-server"
fi

echo ""
echo "📋 Instructions pour le serveur d'impression:"
echo "   1. Arrêter l'ancien agent (Ctrl+C)"
echo "   2. Télécharger la nouvelle version:"
echo "      wget http://10.0.0.59:8082/rms-agent -O rms-agent-new"
echo "   3. Rendre exécutable:"
echo "      chmod +x rms-agent-new"
echo "   4. Démarrer avec la nouvelle version:"
echo "      ./rms-agent-new --server 10.0.0.59:8081 --id \"serveur-impression-01\" --name \"Serveur d'impression principal\" --token \"test-token\""
echo ""
echo "🎯 La nouvelle version inclut:"
echo "   - Gestion correcte des messages file_list"
echo "   - Logs de débogage détaillés"
echo "   - Support de différents formats de données"
echo ""
