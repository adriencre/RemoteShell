#!/bin/bash

# Script simplifié de déploiement de l agent RemoteShell avec installation comme service systemd
# Usage: ./deploy-agent-service.sh [SERVEUR_URL] [TOKEN] [AGENT_NAME] [AGENT_ID]

set -e

# Couleurs
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration par défaut (vos paramètres)
DEFAULT_HOST="10.0.0.72"
DEFAULT_USER="ServeurImpression"
REMOTE_HOME="/home/ServeurImpression"

# Paramètres
SERVER_URL="${1:-10.0.0.59:8080}"
TOKEN="${2:-test-token}"
AGENT_NAME="${3:-Serveur d impression principal}"
AGENT_ID="${4:-serveur-impression-01}"

echo -e "${BLUE}🚀 Déploiement de l agent RemoteShell comme service systemd${NC}"
echo -e "   Serveur: ${DEFAULT_USER}@${DEFAULT_HOST}"
echo -e "   Agent ID: ${AGENT_ID}"
echo -e "   Agent Name: ${AGENT_NAME}"
echo -e "   Server URL: ${SERVER_URL}"
echo ""

# Vérifier que l agent est compilé
if [ ! -f "build/remoteshell-agent" ]; then
    echo -e "${YELLOW}⚠️  L agent n est pas compilé. Compilation en cours...${NC}"
    make agent
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Erreur lors de la compilation${NC}"
        exit 1
    fi
fi

# Afficher la taille de l agent
SIZE=$(stat -c%s "build/remoteshell-agent" 2>/dev/null || stat -f%z "build/remoteshell-agent" 2>/dev/null)
echo -e "${BLUE}📦 Taille de l agent: $(numfmt --to=iec-i --suffix=B $SIZE 2>/dev/null || echo "${SIZE} bytes")${NC}"
echo ""

# Copier l agent vers le serveur distant
echo -e "${BLUE}📤 Copie de l agent vers ${DEFAULT_USER}@${DEFAULT_HOST}...${NC}"
scp ./build/remoteshell-agent ${DEFAULT_USER}@${DEFAULT_HOST}:${REMOTE_HOME}/remoteshell-agent

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Erreur lors de la copie de l agent${NC}"
    echo -e "${YELLOW}💡 Vérifiez votre connexion SSH et que l utilisateur ${DEFAULT_USER} peut se connecter à ${DEFAULT_HOST}${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Agent copié avec succès${NC}"
echo ""

# Installation comme service systemd
echo -e "${BLUE}🔧 Installation comme service systemd...${NC}"

# Utiliser une approche plus simple avec heredoc et variables
# Encoder les variables en base64 pour éviter les problèmes d'échappement
SSH_SERVER_URL=$(echo -n "${SERVER_URL}" | base64 -w 0)
SSH_AGENT_ID=$(echo -n "${AGENT_ID}" | base64 -w 0)
SSH_AGENT_NAME=$(echo -n "${AGENT_NAME}" | base64 -w 0)
SSH_TOKEN=$(echo -n "${TOKEN}" | base64 -w 0)

ssh ${DEFAULT_USER}@${DEFAULT_HOST} bash << ENDSSH
set -e

# Variables (décodées depuis base64)
SERVER_URL=\$(echo "${SSH_SERVER_URL}" | base64 -d)
AGENT_ID=\$(echo "${SSH_AGENT_ID}" | base64 -d)
AGENT_NAME=\$(echo "${SSH_AGENT_NAME}" | base64 -d)
TOKEN=\$(echo "${SSH_TOKEN}" | base64 -d)

# Rendre l agent exécutable
chmod +x ~/remoteshell-agent

# Arrêter le service systemd s il existe et est actif
if sudo systemctl is-active --quiet remoteshell-agent 2>/dev/null; then
    echo "🛑 Arrêt du service systemd existant..."
    sudo systemctl stop remoteshell-agent || true
fi

# Tuer tous les processus remoteshell-agent en cours d exécution
if pgrep -f remoteshell-agent > /dev/null 2>&1; then
    echo "🛑 Arrêt des processus remoteshell-agent en cours..."
    sudo pkill -9 -f remoteshell-agent || true
    sleep 1
fi

# Créer les répertoires nécessaires
sudo mkdir -p /usr/local/bin
sudo mkdir -p /etc/remoteshell

# Copier l agent dans /usr/local/bin
echo "📋 Copie du nouvel agent..."
sudo cp ~/remoteshell-agent /usr/local/bin/remoteshell-agent
sudo chmod +x /usr/local/bin/remoteshell-agent

# Créer le fichier de configuration
echo "SERVER_URL=${SERVER_URL}" > /tmp/agent.conf
echo "AGENT_ID=${AGENT_ID}" >> /tmp/agent.conf
echo "AGENT_NAME=${AGENT_NAME}" >> /tmp/agent.conf
echo "AUTH_TOKEN=${TOKEN}" >> /tmp/agent.conf
sudo mv /tmp/agent.conf /etc/remoteshell/agent.conf

# Créer le service systemd avec les vraies valeurs (échapper les caractères spéciaux pour sed)
ESCAPED_SERVER_URL=$(echo "${SERVER_URL}" | sed 's/[[\.*^$()+?{|]/\\&/g')
ESCAPED_AGENT_ID=$(echo "${AGENT_ID}" | sed 's/[[\.*^$()+?{|]/\\&/g')
ESCAPED_AGENT_NAME=$(echo "${AGENT_NAME}" | sed 's/[[\.*^$()+?{|]/\\&/g')
ESCAPED_TOKEN=$(echo "${TOKEN}" | sed 's/[[\.*^$()+?{|]/\\&/g')

cat > /tmp/remoteshell-agent.service << 'SERVICE'
[Unit]
Description=RemoteShell Agent - Gestionnaire de serveurs d impression
Documentation=https://github.com/votre-org/remoteshell
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/remoteshell-agent --server SERVER_URL_PLACEHOLDER --id "AGENT_ID_PLACEHOLDER" --name "AGENT_NAME_PLACEHOLDER" --token "TOKEN_PLACEHOLDER"
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=remoteshell-agent

[Install]
WantedBy=multi-user.target
SERVICE

# Remplacer les placeholders par les vraies valeurs
sed -i "s|SERVER_URL_PLACEHOLDER|${ESCAPED_SERVER_URL}|g" /tmp/remoteshell-agent.service
sed -i "s|AGENT_ID_PLACEHOLDER|${ESCAPED_AGENT_ID}|g" /tmp/remoteshell-agent.service
sed -i "s|AGENT_NAME_PLACEHOLDER|${ESCAPED_AGENT_NAME}|g" /tmp/remoteshell-agent.service
sed -i "s|TOKEN_PLACEHOLDER|${ESCAPED_TOKEN}|g" /tmp/remoteshell-agent.service

sudo mv /tmp/remoteshell-agent.service /etc/systemd/system/remoteshell-agent.service

# Recharger systemd
echo "🔄 Rechargement de systemd..."
sudo systemctl daemon-reload

# Arrêter l ancien service s il existe
if systemctl is-active --quiet remoteshell-agent 2>/dev/null; then
    echo "🛑 Arrêt de l ancien service..."
    sudo systemctl stop remoteshell-agent
fi

# Désactiver l ancien service s il existe
if systemctl is-enabled --quiet remoteshell-agent 2>/dev/null; then
    sudo systemctl disable remoteshell-agent
fi

# Activer et démarrer le nouveau service
echo "🚀 Activation et démarrage du service..."
sudo systemctl enable remoteshell-agent
sudo systemctl start remoteshell-agent

# Attendre un peu pour vérifier que ça démarre
sleep 2

# Afficher le statut
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
sudo systemctl status remoteshell-agent --no-pager -l || true
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
ENDSSH

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✅ Service installé et démarré avec succès !${NC}"
    echo ""
    echo -e "${BLUE}📊 Commandes utiles:${NC}"
    echo "   Vérifier le statut:"
    echo "   ssh ${DEFAULT_USER}@${DEFAULT_HOST} \"sudo systemctl status remoteshell-agent\""
    echo ""
    echo "   Voir les logs en temps réel:"
    echo "   ssh ${DEFAULT_USER}@${DEFAULT_HOST} \"sudo journalctl -u remoteshell-agent -f\""
    echo ""
    echo "   Redémarrer le service:"
    echo "   ssh ${DEFAULT_USER}@${DEFAULT_HOST} \"sudo systemctl restart remoteshell-agent\""
    echo ""
    echo "   Arrêter le service:"
    echo "   ssh ${DEFAULT_USER}@${DEFAULT_HOST} \"sudo systemctl stop remoteshell-agent\""
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✅ Déploiement terminé !${NC}"
    echo ""
else
    echo -e "${RED}❌ Erreur lors de l installation du service${NC}"
    exit 1
fi

