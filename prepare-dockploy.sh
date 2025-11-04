#!/bin/bash
# Script pour préparer la configuration Dockploy

echo "🔧 Préparation de la configuration Dockploy..."

# Sauvegarder docker-compose.yml si nécessaire
if [ -f docker-compose.yml ] && [ ! -f docker-compose.yml.backup ]; then
    echo "📦 Sauvegarde de docker-compose.yml..."
    cp docker-compose.yml docker-compose.yml.backup
    echo "✅ Sauvegardé dans docker-compose.yml.backup"
fi

# Copier docker-compose.dockploy.yml vers docker-compose.yml
if [ -f docker-compose.dockploy.yml ]; then
    echo "📋 Copie de docker-compose.dockploy.yml vers docker-compose.yml..."
    cp docker-compose.dockploy.yml docker-compose.yml
    echo "✅ Configuration Dockploy prête !"
    echo ""
    echo "📝 Dockploy utilisera maintenant docker-compose.yml (version optimisée)"
else
    echo "❌ Erreur: docker-compose.dockploy.yml non trouvé !"
    exit 1
fi
