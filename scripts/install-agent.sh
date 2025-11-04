#!/bin/bash

# Script d'installation automatique de l'agent RemoteShell
# Ce script télécharge l'agent et l'installe en service systemd
# Usage: curl -sSL http://VOTRE_SERVEUR:PORT/download/install-agent.sh | sudo bash
#
# Ce script est autonome et pose toutes les questions nécessaires.
# Il n'utilise pas de variables d'environnement.

set -e

echo "═══════════════════════════════════════════════════════════════"
echo "  Installation automatique de l'agent RemoteShell"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Vérifier les privilèges root
if [ "$EUID" -ne 0 ]; then 
    echo "❌ Ce script nécessite les privilèges root (sudo)"
    echo "   Exécutez: curl -sSL http://VOTRE_SERVEUR:PORT/download/install-agent.sh | sudo bash"
    exit 1
fi

# Demander toutes les informations nécessaires
echo "📋 Configuration de l'agent RemoteShell"
echo ""

# Valeur par défaut (sans port pour HTTPS)
DEFAULT_SERVER_URL="rms.lfgroup.fr"

# Vérifier si stdin est disponible (tty) pour poser des questions interactives
if [ ! -t 0 ]; then
    # Mode non-interactif détecté - refuser l'exécution
    echo "❌ Erreur: Ce script nécessite un mode interactif."
    echo ""
    echo "Le script doit être exécuté de manière interactive pour poser les questions de configuration."
    echo ""
    echo "💡 Solution:"
    echo "   1. Téléchargez d'abord le script:"
    echo "      curl -O https://rms.lfgroup.fr/download/install-agent.sh"
    echo ""
    echo "   2. Rendez-le exécutable:"
    echo "      chmod +x install-agent.sh"
    echo ""
    echo "   3. Exécutez-le de manière interactive:"
    echo "      sudo ./install-agent.sh"
    echo ""
    exit 1
fi

# Mode interactif - poser des questions
echo "Ce script va vous poser quelques questions pour configurer l'agent."
echo ""

# Demander l'URL du serveur
while [ -z "$SERVER_URL" ]; do
    read -p "URL du serveur RemoteShell [défaut: $DEFAULT_SERVER_URL]: " SERVER_URL
    # Si vide, utiliser la valeur par défaut
    if [ -z "$SERVER_URL" ]; then
        SERVER_URL="$DEFAULT_SERVER_URL"
        echo "✅ Utilisation de l'URL par défaut: $SERVER_URL"
    fi
done

# Extraire le port de l'URL pour vérifier si TLS est nécessaire
EXTRACTED_PORT=""
if [[ "$SERVER_URL" == *":"* ]]; then
    EXTRACTED_PORT="${SERVER_URL##*:}"
fi

# Vérifier si le port 443 est utilisé (nécessite TLS)
NEEDS_TLS=false
if [[ "$EXTRACTED_PORT" == "443" ]] || [[ "$SERVER_URL" == *"https://"* ]]; then
    NEEDS_TLS=true
    echo ""
    echo "ℹ️  Le port 443 nécessite TLS/WSS. TLS sera activé automatiquement."
    USE_TLS_OPTION="--tls"
    echo "✅ TLS/WSS activé (requis pour le port 443)"
else
    # Demander si on veut utiliser TLS (pour tester)
    echo ""
    read -p "Utiliser TLS/WSS pour la connexion WebSocket ? [O/n]: " USE_TLS_INPUT
    USE_TLS_OPTION=""
    if [[ ! "$USE_TLS_INPUT" =~ ^[Nn]$ ]]; then
        USE_TLS_OPTION="--tls"
        echo "✅ TLS/WSS activé"
    else
        echo "⚠️  TLS/WSS désactivé (connexion non sécurisée)"
    fi
fi

echo "ℹ️  Cette adresse sera utilisée pour télécharger l'agent et pour la connexion de l'agent au serveur."
echo ""

# Normaliser l'URL
# Pour rms.lfgroup.fr : HTTPS sans port pour téléchargement, mais ajouter le port pour WebSocket
if [[ "$SERVER_URL" == http://* ]] || [[ "$SERVER_URL" == https://* ]]; then
    # URL avec protocole déjà spécifié
    DOWNLOAD_BASE="$SERVER_URL"
    # Extraire host:port pour la configuration WebSocket
    SERVER_HOST_PORT="${SERVER_URL#http://}"
    SERVER_HOST_PORT="${SERVER_HOST_PORT#https://}"
    
    # Si c'est HTTPS avec rms.lfgroup.fr, enlever le port de DOWNLOAD_BASE
    if [[ "$DOWNLOAD_BASE" == https://rms.lfgroup.fr:* ]]; then
        DOWNLOAD_BASE="https://rms.lfgroup.fr"
    elif [[ "$DOWNLOAD_BASE" == https://rms.lfgroup.fr ]]; then
        # HTTPS avec rms.lfgroup.fr mais sans port - ajouter le port pour WebSocket
        SERVER_HOST_PORT="rms.lfgroup.fr:8081"
    fi
else
    # URL sans protocole
    if [[ "$SERVER_URL" == *"rms.lfgroup.fr"* ]]; then
        # Pour rms.lfgroup.fr, utiliser HTTPS sans port pour téléchargement
        if [[ "$SERVER_URL" == *:* ]]; then
            # Extraire le domaine sans le port pour HTTPS
            DOMAIN_ONLY="${SERVER_URL%%:*}"
            DOWNLOAD_BASE="https://$DOMAIN_ONLY"
            # Garder le port pour la connexion WebSocket
            SERVER_HOST_PORT="$SERVER_URL"
        else
            # Pas de port spécifié - utiliser HTTPS sans port pour téléchargement
            # Mais ajouter le port 8081 pour la connexion WebSocket
            DOWNLOAD_BASE="https://$SERVER_URL"
            SERVER_HOST_PORT="${SERVER_URL}:8081"
        fi
    else
        # Pour les autres domaines, utiliser HTTP et garder le port
        DOWNLOAD_BASE="http://$SERVER_URL"
        SERVER_HOST_PORT="$SERVER_URL"
    fi
fi

echo "📥 Téléchargement de l'agent depuis $DOWNLOAD_BASE/download/agent..."
echo ""

# Créer un répertoire temporaire
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Télécharger l'agent
echo "🔗 Connexion à $DOWNLOAD_BASE/download/agent..."
if command -v curl &> /dev/null; then
    if ! curl -f -s -o "$TMP_DIR/rms-agent" "$DOWNLOAD_BASE/download/agent"; then
        echo ""
        echo "❌ Erreur: Impossible de télécharger l'agent depuis $DOWNLOAD_BASE/download/agent"
        echo ""
        echo "💡 Vérifications possibles:"
        echo "   1. Vérifiez que l'URL du serveur est correcte"
        echo "   2. Vérifiez la connectivité réseau: ping $(echo $SERVER_HOST_PORT | cut -d: -f1)"
        echo "   3. Vérifiez que le serveur est accessible: curl -I $DOWNLOAD_BASE/health"
        echo "   4. Essayez avec l'adresse IP directement au lieu du nom de domaine"
        exit 1
    fi
elif command -v wget &> /dev/null; then
    if ! wget -q -O "$TMP_DIR/rms-agent" "$DOWNLOAD_BASE/download/agent"; then
        echo ""
        echo "❌ Erreur: Impossible de télécharger l'agent depuis $DOWNLOAD_BASE/download/agent"
        echo ""
        echo "💡 Vérifications possibles:"
        echo "   1. Vérifiez que l'URL du serveur est correcte"
        echo "   2. Vérifiez la connectivité réseau"
        echo "   3. Essayez avec l'adresse IP directement au lieu du nom de domaine"
        exit 1
    fi
else
    echo "❌ Erreur: curl ou wget est requis pour télécharger l'agent"
    exit 1
fi

if [ ! -f "$TMP_DIR/rms-agent" ] || [ ! -s "$TMP_DIR/rms-agent" ]; then
    echo "❌ Erreur: Le fichier téléchargé est vide ou invalide"
    exit 1
fi

chmod +x "$TMP_DIR/rms-agent"
echo "✅ Agent téléchargé avec succès"
echo ""

# Demander les paramètres de configuration
echo ""
echo "📋 Configuration de l'agent"
echo ""

# Mode interactif obligatoire - poser des questions
# Demander l'ID de l'agent
while [ -z "$AGENT_ID" ]; do
    read -p "ID de l'agent (ex: serveur-impression-01): " AGENT_ID
    if [ -z "$AGENT_ID" ]; then
        echo "⚠️  L'ID de l'agent ne peut pas être vide. Veuillez réessayer."
    fi
done

# Demander le nom de l'agent
while [ -z "$AGENT_NAME" ]; do
    read -p "Nom de l'agent (ex: Serveur d'impression principal): " AGENT_NAME
    if [ -z "$AGENT_NAME" ]; then
        echo "⚠️  Le nom de l'agent ne peut pas être vide. Veuillez réessayer."
    fi
done

# Demander le token d'authentification
while [ -z "$AUTH_TOKEN" ]; do
    read -sp "Token d'authentification: " AUTH_TOKEN
    echo ""
    if [ -z "$AUTH_TOKEN" ]; then
        echo "⚠️  Le token d'authentification ne peut pas être vide. Veuillez réessayer."
    fi
done

echo ""
echo "🔧 Installation en cours..."
echo ""

# Vérifier si l'agent est déjà installé
if systemctl list-unit-files | grep -q "rms-agent.service"; then
    echo "⚠️  L'agent RemoteShell est déjà installé."
    
    # Arrêter le service s'il est actif
    if systemctl is-active --quiet rms-agent 2>/dev/null; then
        echo "🛑 Arrêt du service..."
        systemctl stop rms-agent
        sleep 1
    fi
    
    # Désactiver le service (pour le réactiver après)
    if systemctl is-enabled --quiet rms-agent 2>/dev/null; then
        echo "🔌 Désactivation temporaire du service..."
        systemctl disable rms-agent 2>/dev/null || true
    fi
fi

# Vérifier si le fichier existe et est en cours d'utilisation
if [ -f /usr/local/bin/rms-agent ]; then
    if lsof /usr/local/bin/rms-agent >/dev/null 2>&1; then
        echo "⚠️  Le fichier agent est en cours d'utilisation, arrêt forcé..."
        systemctl stop rms-agent 2>/dev/null || true
        sleep 2
    fi
    echo "🗑️  Suppression de l'ancien agent..."
fi

# Créer les répertoires nécessaires
mkdir -p /opt/remoteshell
mkdir -p /etc/remoteshell

# Copier l'agent (supprimer l'ancien si nécessaire)
echo "📋 Installation de l'agent vers /usr/local/bin/..."
if [ -f /usr/local/bin/rms-agent ]; then
    rm -f /usr/local/bin/rms-agent
fi
cp "$TMP_DIR/rms-agent" /usr/local/bin/rms-agent
chmod +x /usr/local/bin/rms-agent

# Créer le fichier de configuration
echo "📝 Création du fichier de configuration..."
cat > /etc/remoteshell/agent.conf <<EOF
# Configuration de l'agent RemoteShell
SERVER_URL="${SERVER_HOST_PORT}"
AGENT_ID="${AGENT_ID}"
AGENT_NAME="${AGENT_NAME}"
AUTH_TOKEN="${AUTH_TOKEN}"
EOF
chmod 600 /etc/remoteshell/agent.conf

# Normaliser l'URL et déterminer le port final
if [[ -z "$USE_TLS_OPTION" ]]; then
    # Pas de TLS - utiliser le port tel quel ou le port 8081 par défaut
    # Si l'utilisateur a entré le port 443, on ne peut pas continuer sans TLS
    if [[ "$SERVER_HOST_PORT" == *":443" ]]; then
        echo "❌ Erreur: Le port 443 nécessite TLS/WSS."
        echo "   Le script va utiliser TLS automatiquement."
        USE_TLS_OPTION="--tls"
        USE_TLS="$USE_TLS_OPTION"
    elif [[ "$SERVER_HOST_PORT" == "rms.lfgroup.fr" ]]; then
        SERVER_HOST_PORT="rms.lfgroup.fr:8081"
        echo "ℹ️  Connexion WS (non sécurisée) sur le port 8081"
        USE_TLS=""
    else
        USE_TLS=""
    fi
else
    # TLS activé
    if [[ "$SERVER_HOST_PORT" == *"rms.lfgroup.fr"* ]]; then
        # Si l'utilisateur a spécifié un port autre que 443, utiliser 443 pour WSS
        if [[ "$SERVER_HOST_PORT" == *":8081" ]] || [[ "$SERVER_HOST_PORT" == "rms.lfgroup.fr" ]]; then
            SERVER_HOST_PORT="rms.lfgroup.fr:443"
            echo "ℹ️  Utilisation du port 443 (WSS) via le reverse proxy pour rms.lfgroup.fr"
        fi
        echo "ℹ️  Configuration: WSS (WebSocket Secure) sur $SERVER_HOST_PORT"
        echo "⚠️  IMPORTANT: Assurez-vous que votre reverse proxy (nginx) est configuré pour les WebSockets !"
        echo "   Voir TROUBLESHOOTING_WEBSOCKET.md pour la configuration nginx requise."
    fi
    USE_TLS="$USE_TLS_OPTION"
fi

# Créer le fichier de service systemd
echo "📄 Création du service systemd..."
cat > /etc/systemd/system/rms-agent.service <<EOF
[Unit]
Description=RemoteShell Agent
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/remoteshell
ExecStart=/usr/local/bin/rms-agent --server ${SERVER_HOST_PORT} --id "${AGENT_ID}" --name "${AGENT_NAME}" --token ${AUTH_TOKEN} ${USE_TLS}
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Limites de ressources
LimitNOFILE=65536
LimitNPROC=4096

Environment="PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

[Install]
WantedBy=multi-user.target
EOF

# Recharger systemd
echo "🔄 Rechargement de systemd..."
systemctl daemon-reload

# Activer le service
echo "✅ Activation du service..."
systemctl enable rms-agent

# Démarrer le service
echo "▶️  Démarrage du service..."
systemctl start rms-agent

# Attendre un peu pour que le service démarre
sleep 2

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  ✅ Installation terminée avec succès !"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "📊 Statut du service:"
systemctl status rms-agent --no-pager || true
echo ""
echo "📋 Commandes utiles:"
echo "   • Voir les logs:      journalctl -u rms-agent -f"
echo "   • Arrêter le service: systemctl stop rms-agent"
echo "   • Démarrer le service: systemctl start rms-agent"
echo "   • Redémarrer:         systemctl restart rms-agent"
echo "   • Statut:             systemctl status rms-agent"
echo ""
echo "📝 Configuration sauvegardée dans: /etc/remoteshell/agent.conf"
echo "🔗 L'agent devrait maintenant apparaître dans l'interface web"
echo ""

