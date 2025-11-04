#!/bin/bash

# Script de déploiement de l'agent RemoteShell avec shell persistant et privilèges root
# Usage: ./deploy-agent-root.sh [IP_SERVEUR] [UTILISATEUR] [--install-service]

set -e

# Configuration par défaut
DEFAULT_IP="10.0.0.72"
DEFAULT_USER="ServeurImpression"
AGENT_NAME="rms-agent-root"

# Paramètres
SERVER_IP=${1:-$DEFAULT_IP}
SERVER_USER=${2:-$DEFAULT_USER}
INSTALL_SERVICE=false

# Vérifier si --install-service est passé en paramètre
for arg in "$@"; do
    if [ "$arg" = "--install-service" ]; then
        INSTALL_SERVICE=true
    fi
done

echo "🚀 Déploiement de l'agent RemoteShell avec shell persistant"
echo "📡 Serveur: $SERVER_USER@$SERVER_IP"
echo ""

# Vérifier que l'agent existe
if [ ! -f "./build/rms-agent" ]; then
    echo "❌ Erreur: L'agent n'existe pas. Compilez d'abord avec 'make agent'"
    exit 1
fi

echo "📦 Copie de l'agent vers le serveur..."
scp ./build/rms-agent $SERVER_USER@$SERVER_IP:/tmp/$AGENT_NAME

if [ $? -eq 0 ]; then
    echo "✅ Agent copié avec succès"
else
    echo "❌ Erreur lors de la copie de l'agent"
    exit 1
fi

echo ""

if [ "$INSTALL_SERVICE" = true ]; then
    echo "🔧 Installation automatique du service systemd sur le serveur distant..."
    echo ""
    
    # Copier également le script d'installation
    echo "📦 Copie du script d'installation..."
    scp ./scripts/install-service.sh $SERVER_USER@$SERVER_IP:/tmp/
    
    # Demander les informations de configuration
    echo "📋 Configuration de l'agent..."
    read -p "URL du serveur (défaut: 10.0.0.59:8081): " CONFIG_SERVER_URL
    CONFIG_SERVER_URL=${CONFIG_SERVER_URL:-"10.0.0.59:8081"}
    
    read -p "ID de l'agent (défaut: serveur-impression-01): " CONFIG_AGENT_ID
    CONFIG_AGENT_ID=${CONFIG_AGENT_ID:-"serveur-impression-01"}
    
    read -p "Nom de l'agent (défaut: Serveur d'impression principal): " CONFIG_AGENT_NAME
    CONFIG_AGENT_NAME=${CONFIG_AGENT_NAME:-"Serveur d'impression principal"}
    
    read -p "Token d'authentification (défaut: test-token): " CONFIG_AUTH_TOKEN
    CONFIG_AUTH_TOKEN=${CONFIG_AUTH_TOKEN:-"test-token"}
    
    echo ""
    
    # Exécuter l'installation à distance
    echo "🚀 Exécution de l'installation sur le serveur distant..."
    ssh $SERVER_USER@$SERVER_IP << ENDSSH
cd /tmp
# Créer un répertoire temporaire avec tous les fichiers nécessaires
mkdir -p build
mv rms-agent-root build/rms-agent
chmod +x install-service.sh
# Exporter les variables pour l'installation non-interactive
export SERVER_URL="$CONFIG_SERVER_URL"
export AGENT_ID="$CONFIG_AGENT_ID"
export AGENT_NAME="$CONFIG_AGENT_NAME"
export AUTH_TOKEN="$CONFIG_AUTH_TOKEN"
sudo -E ./install-service.sh install
ENDSSH
    
    if [ $? -eq 0 ]; then
        echo ""
        echo "✅ Service installé avec succès !"
        echo ""
        echo "📊 Pour vérifier le statut du service:"
        echo "   ssh $SERVER_USER@$SERVER_IP 'sudo systemctl status rms-agent'"
        echo ""
        echo "📝 Pour voir les logs en temps réel:"
        echo "   ssh $SERVER_USER@$SERVER_IP 'sudo journalctl -u rms-agent -f'"
    else
        echo ""
        echo "❌ Erreur lors de l'installation du service"
        echo "Consultez les messages ci-dessus pour plus de détails"
    fi
else
    echo "🔧 Instructions pour le serveur d'impression:"
    echo "1. Connectez-vous au serveur: ssh $SERVER_USER@$SERVER_IP"
    echo "2. Arrêtez l'ancien agent (Ctrl+C si en cours)"
    echo "3. Copiez le nouvel agent:"
    echo "   sudo cp /tmp/$AGENT_NAME /home/$SERVER_USER/rms-agent"
    echo "   sudo chmod +x /home/$SERVER_USER/rms-agent"
    echo ""
    echo "4. Configurez sudo sans mot de passe (optionnel mais recommandé):"
    echo "   sudo visudo"
    echo "   Ajoutez: $SERVER_USER ALL=(ALL) NOPASSWD: ALL"
    echo ""
    echo "5. Lancez le nouvel agent:"
    echo "   ./rms-agent --server 10.0.0.59:8081 --id \"serveur-impression-01\" --name \"Serveur d'impression principal\" --token \"test-token\""
    echo ""
    echo "6. OU installez-le comme service systemd (recommandé):"
    echo "   ./deploy-agent-root.sh $SERVER_IP $SERVER_USER --install-service"
    echo ""
    echo "🎯 Nouvelles fonctionnalités:"
    echo "   ✅ Shell persistant (contexte conservé entre commandes)"
    echo "   ✅ Privilèges root automatiques"
    echo "   ✅ Commandes 'cd' fonctionnent et persistent"
    echo "   ✅ Variables d'environnement conservées"
    echo "   ✅ Gestion des services (systemd + Docker)"
    echo "   ✅ Visualisation des logs (agent + système)"
    echo ""
    echo "📝 Test recommandé:"
    echo "   cd /root"
    echo "   pwd"
    echo "   ls -la"
    echo "   (Le répertoire devrait rester /root pour les commandes suivantes)"
fi
