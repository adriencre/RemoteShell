#!/bin/bash

# Script de démarrage pour RemoteShell en production
set -e

echo "🚀 Démarrage de RemoteShell en production..."

# Vérifier si .env existe
if [ ! -f .env ]; then
    echo "❌ Fichier .env non trouvé!"
    echo "📝 Création du fichier .env depuis env.prod.example..."
    cp env.prod.example .env
    echo ""
    echo "⚠️  IMPORTANT: Modifiez le fichier .env avec vos valeurs avant de continuer!"
    echo "   nano .env"
    echo ""
    read -p "Appuyez sur Entrée après avoir modifié .env..."
fi

# Vérifier que Docker est disponible
if ! command -v docker &> /dev/null; then
    echo "❌ Docker n'est pas installé ou n'est pas dans le PATH"
    exit 1
fi

# Vérifier que docker-compose est disponible
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "❌ docker-compose n'est pas installé ou n'est pas dans le PATH"
    exit 1
fi

# Utiliser docker compose (v2) ou docker-compose (v1)
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

echo "📦 Construction de l'image Docker..."
$DOCKER_COMPOSE -f docker-compose.prod.yml build

echo "🔄 Démarrage des services..."
$DOCKER_COMPOSE -f docker-compose.prod.yml up -d

echo "⏳ Attente du démarrage du serveur (10 secondes)..."
sleep 10

echo "🔍 Vérification de l'état du serveur..."
if $DOCKER_COMPOSE -f docker-compose.prod.yml ps | grep -q "Up"; then
    echo "✅ Serveur démarré avec succès!"
    echo ""
    echo "📊 Vérification du health check..."
    if curl -s http://localhost:8081/health > /dev/null; then
        echo "✅ Health check OK: http://localhost:8081/health"
    else
        echo "⚠️  Health check échoué, vérifiez les logs:"
        echo "   $DOCKER_COMPOSE -f docker-compose.prod.yml logs remoteshell-server"
    fi
    echo ""
    echo "📋 Commandes utiles:"
    echo "   Voir les logs: $DOCKER_COMPOSE -f docker-compose.prod.yml logs -f"
    echo "   Arrêter: $DOCKER_COMPOSE -f docker-compose.prod.yml down"
    echo "   Redémarrer: $DOCKER_COMPOSE -f docker-compose.prod.yml restart"
else
    echo "❌ Le serveur n'a pas démarré correctement"
    echo "📋 Vérifiez les logs:"
    echo "   $DOCKER_COMPOSE -f docker-compose.prod.yml logs remoteshell-server"
    exit 1
fi

