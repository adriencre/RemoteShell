#!/bin/bash

# Script de déploiement de l'agent RemoteShell via SSH
# Usage: ./deploy-ssh.sh [HOST] [USER] [SERVER_URL] [TOKEN] [AGENT_NAME]

set -e

# Couleurs
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Fonction d'aide
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --host HOST         Adresse IP ou hostname du serveur distant (requis)"
    echo "  -u, --user USER         Nom d'utilisateur SSH (requis)"
    echo "  -s, --server URL        URL du serveur RemoteShell (ex: 10.0.0.59:8080)"
    echo "  -t, --token TOKEN       Token d'authentification"
    echo "  -n, --name NAME         Nom de l'agent"
    echo "  -i, --id ID             ID unique de l'agent"
    echo "  -p, --port PORT         Port SSH (défaut: 22)"
    echo "  --install-service       Installer comme service systemd"
    echo "  --help                  Afficher cette aide"
    echo ""
    echo "Exemples:"
    echo "  $0 -h 192.168.1.100 -u user -s localhost:8080 -t mytoken -n 'Mon Agent'"
    echo "  $0 --host 10.0.0.72 --user admin --server 10.0.0.59:8080 --token secret --install-service"
    exit 0
}

# Valeurs par défaut
HOST=""
USER=""
SERVER_URL=""
TOKEN=""
AGENT_NAME=""
AGENT_ID=""
SSH_PORT=22
INSTALL_SERVICE=false

# Parser les arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)
            HOST="$2"
            shift 2
            ;;
        -u|--user)
            USER="$2"
            shift 2
            ;;
        -s|--server)
            SERVER_URL="$2"
            shift 2
            ;;
        -t|--token)
            TOKEN="$2"
            shift 2
            ;;
        -n|--name)
            AGENT_NAME="$2"
            shift 2
            ;;
        -i|--id)
            AGENT_ID="$2"
            shift 2
            ;;
        -p|--port)
            SSH_PORT="$2"
            shift 2
            ;;
        --install-service)
            INSTALL_SERVICE=true
            shift
            ;;
        --help)
            show_help
            ;;
        *)
            echo -e "${RED}Erreur: Option inconnue '$1'${NC}"
            show_help
            ;;
    esac
done

# Vérifications
if [ -z "$HOST" ] || [ -z "$USER" ]; then
    echo -e "${RED}❌ Erreur: HOST et USER sont requis${NC}"
    echo ""
    show_help
fi

# Vérifier que l'agent est compilé
if [ ! -f "build/remoteshell-agent" ]; then
    echo -e "${YELLOW}⚠️  L'agent n'est pas compilé. Compilation en cours...${NC}"
    make agent
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Erreur lors de la compilation${NC}"
        exit 1
    fi
fi

echo -e "${BLUE}🚀 Déploiement de l'agent RemoteShell${NC}"
echo -e "   Host: ${USER}@${HOST}:${SSH_PORT}"
echo ""

# Si les informations sont manquantes, les demander
if [ -z "$SERVER_URL" ]; then
    read -p "URL du serveur RemoteShell (ex: localhost:8080): " SERVER_URL
fi

if [ -z "$TOKEN" ]; then
    read -p "Token d'authentification: " TOKEN
fi

if [ -z "$AGENT_NAME" ]; then
    read -p "Nom de l'agent: " AGENT_NAME
fi

if [ -z "$AGENT_ID" ]; then
    AGENT_ID="${AGENT_NAME// /-}"  # Remplacer espaces par tirets
    AGENT_ID=$(echo "$AGENT_ID" | tr '[:upper:]' '[:lower:]')
    echo "ID de l'agent généré: $AGENT_ID"
fi

echo ""
echo -e "${BLUE}📦 Copie de l'agent vers le serveur distant...${NC}"

# Créer un répertoire temporaire sur le serveur distant
REMOTE_DIR="/tmp/remoteshell-deploy-$$"
ssh -p "$SSH_PORT" "$USER@$HOST" "mkdir -p $REMOTE_DIR"

# Copier l'agent
scp -P "$SSH_PORT" build/remoteshell-agent "$USER@$HOST:$REMOTE_DIR/remoteshell-agent"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Agent copié avec succès${NC}"
else
    echo -e "${RED}❌ Erreur lors de la copie${NC}"
    exit 1
fi

echo ""

# Installer comme service ou lancer directement
if [ "$INSTALL_SERVICE" = true ]; then
    echo -e "${BLUE}🔧 Installation comme service systemd...${NC}"
    
    # Copier le script d'installation si disponible
    if [ -f "scripts/install-service.sh" ]; then
        scp -P "$SSH_PORT" scripts/install-service.sh "$USER@$HOST:$REMOTE_DIR/"
    fi
    
    # Créer le fichier de service systemd
    ssh -p "$SSH_PORT" "$USER@$HOST" << EOF
sudo mkdir -p /usr/local/bin /etc/remoteshell
sudo cp $REMOTE_DIR/remoteshell-agent /usr/local/bin/remoteshell-agent
sudo chmod +x /usr/local/bin/remoteshell-agent

# Créer la configuration
sudo tee /etc/remoteshell/agent.conf > /dev/null << CONF
SERVER_URL=$SERVER_URL
AGENT_ID=$AGENT_ID
AGENT_NAME=$AGENT_NAME
AUTH_TOKEN=$TOKEN
CONF

# Créer le service systemd
sudo tee /etc/systemd/system/remoteshell-agent.service > /dev/null << SERVICE
[Unit]
Description=RemoteShell Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/remoteshell-agent --server $SERVER_URL --id "$AGENT_ID" --name "$AGENT_NAME" --token "$TOKEN"
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
SERVICE

# Recharger systemd et démarrer le service
sudo systemctl daemon-reload
sudo systemctl enable remoteshell-agent
sudo systemctl restart remoteshell-agent
sudo systemctl status remoteshell-agent --no-pager

# Nettoyer
rm -rf $REMOTE_DIR
EOF

    if [ $? -eq 0 ]; then
        echo ""
        echo -e "${GREEN}✅ Service installé et démarré avec succès !${NC}"
        echo ""
        echo "📊 Commandes utiles:"
        echo "   Vérifier le statut: ssh -p $SSH_PORT $USER@$HOST 'sudo systemctl status remoteshell-agent'"
        echo "   Voir les logs: ssh -p $SSH_PORT $USER@$HOST 'sudo journalctl -u remoteshell-agent -f'"
        echo "   Redémarrer: ssh -p $SSH_PORT $USER@$HOST 'sudo systemctl restart remoteshell-agent'"
    else
        echo -e "${RED}❌ Erreur lors de l'installation du service${NC}"
        exit 1
    fi
else
    echo -e "${BLUE}🚀 Démarrage de l'agent...${NC}"
    
    # Créer un script de démarrage sur le serveur distant
    ssh -p "$SSH_PORT" "$USER@$HOST" << EOF
cd $REMOTE_DIR
chmod +x remoteshell-agent

# Tuer l'ancien agent s'il existe
pkill -f remoteshell-agent || true

# Démarrer le nouvel agent en arrière-plan
nohup ./remoteshell-agent --server "$SERVER_URL" --id "$AGENT_ID" --name "$AGENT_NAME" --token "$TOKEN" > agent.log 2>&1 &

sleep 2
if pgrep -f remoteshell-agent > /dev/null; then
    echo "✅ Agent démarré avec succès (PID: \$(pgrep -f remoteshell-agent))"
    echo "📝 Logs disponibles dans: $REMOTE_DIR/agent.log"
else
    echo "❌ Erreur lors du démarrage de l'agent"
    echo "Consultez les logs: cat $REMOTE_DIR/agent.log"
fi
EOF

    echo ""
    echo -e "${GREEN}✅ Agent déployé et démarré${NC}"
    echo ""
    echo "📝 Pour voir les logs en temps réel:"
    echo "   ssh -p $SSH_PORT $USER@$HOST 'tail -f $REMOTE_DIR/agent.log'"
    echo ""
    echo "🛑 Pour arrêter l'agent:"
    echo "   ssh -p $SSH_PORT $USER@$HOST 'pkill -f remoteshell-agent'"
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Déploiement terminé !${NC}"
echo ""



