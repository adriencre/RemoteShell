#!/bin/bash

# Script d'installation du service systemd RemoteShell Agent
# Usage: ./install-service.sh [install|uninstall|update]

set -e

SERVICE_NAME="rms-agent"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
CONFIG_DIR="/etc/remoteshell"
CONFIG_FILE="${CONFIG_DIR}/agent.conf"
BIN_PATH="/usr/local/bin/rms-agent"
WORK_DIR="/opt/remoteshell"

# Couleurs pour l'affichage
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Vérifier les privilèges root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}❌ Ce script doit être exécuté en tant que root${NC}"
    exit 1
fi

# Fonction d'installation
install_service() {
    echo -e "${GREEN}📦 Installation du service RemoteShell Agent${NC}"
    echo ""
    
    # Créer le répertoire de configuration
    if [ ! -d "$CONFIG_DIR" ]; then
        echo "📁 Création du répertoire de configuration: $CONFIG_DIR"
        mkdir -p "$CONFIG_DIR"
    fi
    
    # Créer le répertoire de travail
    if [ ! -d "$WORK_DIR" ]; then
        echo "📁 Création du répertoire de travail: $WORK_DIR"
        mkdir -p "$WORK_DIR"
    fi
    
    # Vérifier si l'exécutable existe
    if [ ! -f "./build/rms-agent" ]; then
        echo -e "${RED}❌ L'exécutable n'existe pas. Compilez d'abord avec 'make agent'${NC}"
        exit 1
    fi
    
    # Copier l'exécutable
    echo "📋 Copie de l'exécutable vers $BIN_PATH"
    cp ./build/rms-agent "$BIN_PATH"
    chmod +x "$BIN_PATH"
    
    # Créer le fichier de configuration s'il n'existe pas
    if [ ! -f "$CONFIG_FILE" ]; then
        echo "📝 Création du fichier de configuration"
        
        # Demander les informations à l'utilisateur (ou utiliser les variables d'environnement)
        if [ -z "$SERVER_URL" ]; then
            read -p "URL du serveur (ex: 10.0.0.59:8081): " SERVER_URL
        fi
        if [ -z "$AGENT_ID" ]; then
            read -p "ID de l'agent (ex: serveur-01): " AGENT_ID
        fi
        if [ -z "$AGENT_NAME" ]; then
            read -p "Nom de l'agent (ex: Serveur principal): " AGENT_NAME
        fi
        if [ -z "$AUTH_TOKEN" ]; then
            read -p "Token d'authentification: " AUTH_TOKEN
        fi
        
        # Créer le fichier de configuration (avec échappement correct)
        cat > "$CONFIG_FILE" <<EOF
# Configuration de l'agent RemoteShell
SERVER_URL="${SERVER_URL}"
AGENT_ID="${AGENT_ID}"
AGENT_NAME="${AGENT_NAME}"
AUTH_TOKEN="${AUTH_TOKEN}"
EOF
        
        chmod 600 "$CONFIG_FILE"
        echo -e "${GREEN}✅ Configuration créée: $CONFIG_FILE${NC}"
    else
        echo -e "${YELLOW}⚠️  Le fichier de configuration existe déjà: $CONFIG_FILE${NC}"
    fi
    
    # Charger la configuration
    source "$CONFIG_FILE"
    
    # Créer le fichier de service systemd
    echo "📄 Création du fichier de service systemd"
    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=RemoteShell Agent
Documentation=https://github.com/votre-projet/remoteshell
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=${WORK_DIR}
ExecStart=${BIN_PATH} --server ${SERVER_URL} --id "${AGENT_ID}" --name "${AGENT_NAME}" --token ${AUTH_TOKEN}
Restart=always
RestartSec=30
StandardOutput=journal
StandardError=journal

# Sécurité
NoNewPrivileges=false
PrivateTmp=false

# Limites de ressources
LimitNOFILE=65536
LimitNPROC=4096

# Variables d'environnement
Environment="PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

[Install]
WantedBy=multi-user.target
EOF
    
    # Recharger systemd
    echo "🔄 Rechargement de systemd"
    systemctl daemon-reload
    
    # Activer le service
    echo "✅ Activation du service"
    systemctl enable "$SERVICE_NAME"
    
    # Démarrer le service
    echo "▶️  Démarrage du service"
    systemctl start "$SERVICE_NAME"
    
    # Afficher le statut
    echo ""
    echo -e "${GREEN}✅ Installation terminée avec succès !${NC}"
    echo ""
    echo "📊 Statut du service:"
    systemctl status "$SERVICE_NAME" --no-pager || true
    echo ""
    echo "Commandes utiles:"
    echo "  • Voir les logs:      journalctl -u $SERVICE_NAME -f"
    echo "  • Arrêter le service: systemctl stop $SERVICE_NAME"
    echo "  • Démarrer:          systemctl start $SERVICE_NAME"
    echo "  • Redémarrer:        systemctl restart $SERVICE_NAME"
    echo "  • Statut:            systemctl status $SERVICE_NAME"
}

# Fonction de désinstallation
uninstall_service() {
    echo -e "${YELLOW}🗑️  Désinstallation du service RemoteShell Agent${NC}"
    echo ""
    
    # Arrêter le service
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo "⏹️  Arrêt du service"
        systemctl stop "$SERVICE_NAME"
    fi
    
    # Désactiver le service
    if systemctl is-enabled --quiet "$SERVICE_NAME"; then
        echo "❌ Désactivation du service"
        systemctl disable "$SERVICE_NAME"
    fi
    
    # Supprimer le fichier de service
    if [ -f "$SERVICE_FILE" ]; then
        echo "🗑️  Suppression du fichier de service"
        rm "$SERVICE_FILE"
    fi
    
    # Recharger systemd
    echo "🔄 Rechargement de systemd"
    systemctl daemon-reload
    
    # Supprimer l'exécutable
    if [ -f "$BIN_PATH" ]; then
        echo "🗑️  Suppression de l'exécutable"
        rm "$BIN_PATH"
    fi
    
    echo ""
    read -p "Supprimer également la configuration ($CONFIG_DIR) ? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$CONFIG_DIR"
        echo -e "${GREEN}✅ Configuration supprimée${NC}"
    fi
    
    echo ""
    echo -e "${GREEN}✅ Désinstallation terminée${NC}"
}

# Fonction de mise à jour
update_service() {
    echo -e "${GREEN}🔄 Mise à jour du service RemoteShell Agent${NC}"
    echo ""
    
    # Vérifier si l'exécutable existe
    if [ ! -f "./build/rms-agent" ]; then
        echo -e "${RED}❌ L'exécutable n'existe pas. Compilez d'abord avec 'make agent'${NC}"
        exit 1
    fi
    
    # Arrêter le service
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo "⏹️  Arrêt du service"
        systemctl stop "$SERVICE_NAME"
    fi
    
    # Copier le nouveau binaire
    echo "📋 Copie du nouvel exécutable"
    cp ./build/rms-agent "$BIN_PATH"
    chmod +x "$BIN_PATH"
    
    # Démarrer le service
    echo "▶️  Démarrage du service"
    systemctl start "$SERVICE_NAME"
    
    # Afficher le statut
    echo ""
    echo -e "${GREEN}✅ Mise à jour terminée avec succès !${NC}"
    echo ""
    systemctl status "$SERVICE_NAME" --no-pager || true
}

# Menu principal
case "${1:-install}" in
    install)
        install_service
        ;;
    uninstall)
        uninstall_service
        ;;
    update)
        update_service
        ;;
    *)
        echo "Usage: $0 {install|uninstall|update}"
        echo ""
        echo "Commandes:"
        echo "  install   - Installer le service (défaut)"
        echo "  uninstall - Désinstaller le service"
        echo "  update    - Mettre à jour le binaire et redémarrer"
        exit 1
        ;;
esac
