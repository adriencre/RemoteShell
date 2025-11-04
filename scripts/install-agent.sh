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
echo "Ce script va vous poser quelques questions pour configurer l'agent."
echo ""

# Demander l'URL du serveur (utilisée pour télécharger l'agent ET pour la connexion)
# Valeur par défaut
DEFAULT_SERVER_URL="rms.lfgroup.fr:8081"

while [ -z "$SERVER_URL" ]; do
    read -p "URL du serveur RemoteShell [défaut: $DEFAULT_SERVER_URL]: " SERVER_URL
    # Si vide, utiliser la valeur par défaut
    if [ -z "$SERVER_URL" ]; then
        SERVER_URL="$DEFAULT_SERVER_URL"
        echo "✅ Utilisation de l'URL par défaut: $SERVER_URL"
    fi
done
echo "ℹ️  Cette adresse sera utilisée pour télécharger l'agent et pour la connexion de l'agent au serveur."
echo ""

# Normaliser l'URL (ajouter http:// ou https:// si nécessaire)
if [[ "$SERVER_URL" == http://* ]] || [[ "$SERVER_URL" == https://* ]]; then
    DOWNLOAD_BASE="$SERVER_URL"
    # Extraire host:port pour la configuration
    SERVER_HOST_PORT="${SERVER_URL#http://}"
    SERVER_HOST_PORT="${SERVER_HOST_PORT#https://}"
else
    # Utiliser HTTPS pour rms.lfgroup.fr, HTTP pour les autres
    if [[ "$SERVER_URL" == *"rms.lfgroup.fr"* ]]; then
        DOWNLOAD_BASE="https://$SERVER_URL"
    else
        DOWNLOAD_BASE="http://$SERVER_URL"
    fi
    SERVER_HOST_PORT="$SERVER_URL"
fi

echo "📥 Téléchargement de l'agent depuis $DOWNLOAD_BASE/download/agent..."
echo ""

# Créer un répertoire temporaire
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Télécharger l'agent
echo "🔗 Connexion à $DOWNLOAD_BASE/download/agent..."
if command -v curl &> /dev/null; then
    if ! curl -f -s -o "$TMP_DIR/remoteshell-agent" "$DOWNLOAD_BASE/download/agent"; then
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
    if ! wget -q -O "$TMP_DIR/remoteshell-agent" "$DOWNLOAD_BASE/download/agent"; then
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

if [ ! -f "$TMP_DIR/remoteshell-agent" ] || [ ! -s "$TMP_DIR/remoteshell-agent" ]; then
    echo "❌ Erreur: Le fichier téléchargé est vide ou invalide"
    exit 1
fi

chmod +x "$TMP_DIR/remoteshell-agent"
echo "✅ Agent téléchargé avec succès"
echo ""

# Demander les paramètres de configuration
echo ""
echo "📋 Configuration de l'agent"
echo ""

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

# Créer les répertoires nécessaires
mkdir -p /opt/remoteshell
mkdir -p /etc/remoteshell

# Copier l'agent
echo "📋 Installation de l'agent vers /usr/local/bin/..."
cp "$TMP_DIR/remoteshell-agent" /usr/local/bin/remoteshell-agent
chmod +x /usr/local/bin/remoteshell-agent

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

# Créer le fichier de service systemd
echo "📄 Création du service systemd..."
cat > /etc/systemd/system/remoteshell-agent.service <<EOF
[Unit]
Description=RemoteShell Agent
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/remoteshell
ExecStart=/usr/local/bin/remoteshell-agent --server ${SERVER_HOST_PORT} --id "${AGENT_ID}" --name "${AGENT_NAME}" --token ${AUTH_TOKEN}
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
systemctl enable remoteshell-agent

# Démarrer le service
echo "▶️  Démarrage du service..."
systemctl start remoteshell-agent

# Attendre un peu pour que le service démarre
sleep 2

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  ✅ Installation terminée avec succès !"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "📊 Statut du service:"
systemctl status remoteshell-agent --no-pager || true
echo ""
echo "📋 Commandes utiles:"
echo "   • Voir les logs:      journalctl -u remoteshell-agent -f"
echo "   • Arrêter le service: systemctl stop remoteshell-agent"
echo "   • Démarrer le service: systemctl start remoteshell-agent"
echo "   • Redémarrer:         systemctl restart remoteshell-agent"
echo "   • Statut:             systemctl status remoteshell-agent"
echo ""
echo "📝 Configuration sauvegardée dans: /etc/remoteshell/agent.conf"
echo "🔗 L'agent devrait maintenant apparaître dans l'interface web"
echo ""

