#!/bin/bash

# Script d'installation automatique de l'agent RemoteShell
# Ce script télécharge l'agent et l'installe en service systemd
# Usage: curl -sSL http://VOTRE_SERVEUR:PORT/download/install-agent.sh | sudo bash
#
# Ce script est autonome et pose toutes les questions nécessaires.
# Il n'utilise pas de variables d'environnement.

set -e

# Définir un PATH complet pour assurer l'accès à tous les binaires système
# Nécessaire sur Raspberry Pi où certains binaires sont dans /sbin ou /usr/sbin
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH"

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

# Valeur par défaut avec port 443 pour HTTPS/WSS
DEFAULT_SERVER_URL="rms.lfgroup.fr:443"

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
# Extraire le domaine et le port
if [[ "$SERVER_URL" == http://* ]] || [[ "$SERVER_URL" == https://* ]]; then
    # URL avec protocole déjà spécifié
    PROTOCOL="${SERVER_URL%%://*}"
    HOST_PORT="${SERVER_URL#*://}"
else
    # URL sans protocole
    HOST_PORT="$SERVER_URL"
    # Déterminer le protocole selon le port ou TLS
    if [[ "$NEEDS_TLS" == true ]] || [[ "$HOST_PORT" == *:443 ]]; then
        PROTOCOL="https"
    else
        PROTOCOL="http"
    fi
fi

# Extraire le domaine et le port
if [[ "$HOST_PORT" == *":"* ]]; then
    DOMAIN_ONLY="${HOST_PORT%%:*}"
    PORT_PART="${HOST_PORT##*:}"
else
    DOMAIN_ONLY="$HOST_PORT"
    PORT_PART=""
fi

# Si le port est 443, utiliser HTTPS et enlever le port de l'URL de téléchargement
# (HTTPS utilise 443 par défaut, donc pas besoin de le spécifier)
if [[ "$PORT_PART" == "443" ]] || [[ "$NEEDS_TLS" == true ]]; then
    DOWNLOAD_BASE="https://$DOMAIN_ONLY"
    # Pour WebSocket, utiliser le domaine avec le port 443
    SERVER_HOST_PORT="$DOMAIN_ONLY:443"
elif [[ "$PORT_PART" != "" ]]; then
    # Autre port spécifié
    if [[ "$PROTOCOL" == "https" ]] || [[ "$NEEDS_TLS" == true ]]; then
        DOWNLOAD_BASE="https://$DOMAIN_ONLY:$PORT_PART"
    else
        DOWNLOAD_BASE="http://$DOMAIN_ONLY:$PORT_PART"
    fi
    SERVER_HOST_PORT="$DOMAIN_ONLY:$PORT_PART"
else
    # Pas de port spécifié
    if [[ "$PROTOCOL" == "https" ]] || [[ "$SERVER_URL" == *"rms.lfgroup.fr"* ]]; then
        DOWNLOAD_BASE="https://$DOMAIN_ONLY"
        SERVER_HOST_PORT="$DOMAIN_ONLY:8081"
    else
        DOWNLOAD_BASE="http://$DOMAIN_ONLY"
        SERVER_HOST_PORT="$DOMAIN_ONLY:8080"
    fi
fi

# Détecter l'OS et l'architecture
echo "🔍 Détection de l'OS et de l'architecture..."
OS=""
ARCH=""
EXT=""

# Détecter l'OS
case "$(uname -s)" in
    Linux*)
        OS="linux"
        ;;
    Darwin*)
        OS="darwin"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        OS="windows"
        EXT=".exe"
        ;;
    *)
        echo "❌ Erreur: OS non supporté: $(uname -s)"
        echo "   OS supportés: Linux, macOS (Darwin), Windows"
        exit 1
        ;;
esac

# Détecter l'architecture
case "$(uname -m)" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    armv7l)
        ARCH="armv7l"
        ;;
    armv6l)
        ARCH="armv6l"
        ;;
    arm*)
        # Autres variantes ARM 32-bit, utiliser armv7l par défaut
        ARCH="armv7l"
        ;;
    *)
        echo "⚠️  Architecture non reconnue: $(uname -m), tentative avec amd64..."
        ARCH="amd64"
        ;;
esac

echo "✅ OS détecté: $OS"
echo "✅ Architecture détectée: $ARCH"
echo ""

# Vérifier si l'OS est Windows (nécessite un script PowerShell)
if [ "$OS" = "windows" ]; then
    echo "❌ Erreur: Ce script est pour Linux/macOS."
    echo "   Pour Windows, utilisez le script PowerShell d'installation."
    echo "   Téléchargez-le depuis: $DOWNLOAD_BASE/download/install-agent.ps1"
    exit 1
fi

echo "📥 Téléchargement de l'agent depuis $DOWNLOAD_BASE/download/agent?os=$OS&arch=$ARCH..."
echo ""

# Créer un répertoire temporaire
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Télécharger l'agent avec les paramètres OS/arch
AGENT_URL="$DOWNLOAD_BASE/download/agent?os=$OS&arch=$ARCH"
echo "🔗 Connexion à $AGENT_URL..."

# Télécharger avec curl (afficher les erreurs HTTP et les headers)
if command -v curl &> /dev/null; then
    # Télécharger le fichier et récupérer le code HTTP et les headers
    HTTP_CODE=$(curl -s -o "$TMP_DIR/rms-agent" -w "%{http_code}" -D "$TMP_DIR/headers.txt" "$AGENT_URL" || echo "000")
    
    # Vérifier les headers pour confirmer quel binaire est servi
    if [ -f "$TMP_DIR/headers.txt" ]; then
        BINARY_HEADER=$(grep -i "X-Agent-Binary:" "$TMP_DIR/headers.txt" | cut -d' ' -f2- | tr -d '\r' || echo "")
        OS_HEADER=$(grep -i "X-Agent-OS:" "$TMP_DIR/headers.txt" | cut -d' ' -f2- | tr -d '\r' || echo "")
        ARCH_HEADER=$(grep -i "X-Agent-Arch:" "$TMP_DIR/headers.txt" | cut -d' ' -f2- | tr -d '\r' || echo "")
        
        if [ -n "$BINARY_HEADER" ]; then
            echo "ℹ️  Binaire servi par le serveur: $BINARY_HEADER"
            if [ -n "$OS_HEADER" ] && [ -n "$ARCH_HEADER" ]; then
                echo "   OS: $OS_HEADER, Architecture: $ARCH_HEADER"
                # Vérifier que le serveur a bien servi le bon binaire
                if [ "$OS_HEADER" != "$OS" ] || [ "$ARCH_HEADER" != "$ARCH" ]; then
                    echo ""
                    echo "⚠️  Attention: Le serveur a servi un binaire pour $OS_HEADER/$ARCH_HEADER"
                    echo "   mais vous avez demandé $OS/$ARCH"
                fi
            fi
        fi
    fi
    
    # Vérifier le code HTTP
    if [ "$HTTP_CODE" != "200" ]; then
        echo ""
        echo "❌ Erreur: Impossible de télécharger l'agent depuis $AGENT_URL"
        echo "   Code HTTP: $HTTP_CODE"
        
        # Si c'est une erreur 404, essayer de récupérer le message d'erreur du serveur
        if [ "$HTTP_CODE" = "404" ]; then
            ERROR_MSG=$(cat "$TMP_DIR/rms-agent" 2>/dev/null | grep -o '"hint":"[^"]*"' | sed 's/"hint":"\([^"]*\)"/\1/' || echo "")
            if [ -n "$ERROR_MSG" ]; then
                echo "   Message du serveur: $ERROR_MSG"
            fi
        fi
        
        echo ""
        echo "💡 Vérifications possibles:"
        echo "   1. Vérifiez que l'URL du serveur est correcte"
        echo "   2. Vérifiez la connectivité réseau: ping $(echo $SERVER_HOST_PORT | cut -d: -f1)"
        echo "   3. Vérifiez que le serveur est accessible: curl -I $DOWNLOAD_BASE/health"
        echo "   4. Vérifiez que le binaire pour $OS/$ARCH est disponible sur le serveur"
        echo "   5. Si le binaire n'existe pas, le serveur doit builder les binaires multi-plateformes:"
        echo "      ./scripts/build.sh ou make build-all"
        echo "   6. Essayez avec l'adresse IP directement au lieu du nom de domaine"
        exit 1
    fi
elif command -v wget &> /dev/null; then
    if ! wget -q -O "$TMP_DIR/rms-agent" "$AGENT_URL"; then
        echo ""
        echo "❌ Erreur: Impossible de télécharger l'agent depuis $AGENT_URL"
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

# Vérifier le type de fichier si la commande 'file' est disponible
if command -v file &> /dev/null; then
    FILE_INFO=$(file "$TMP_DIR/rms-agent" 2>/dev/null || echo "")
    echo "ℹ️  Type de fichier: $FILE_INFO"
    
    # Vérifier que c'est bien un binaire exécutable
    if ! echo "$FILE_INFO" | grep -qE "(ELF|executable|binary)"; then
        echo "⚠️  Attention: Le fichier ne semble pas être un binaire exécutable"
    fi
fi

# Vérifier la compatibilité de l'architecture du binaire
echo "🔍 Vérification de la compatibilité du binaire..."
BINARY_ARCH=""
if command -v readelf &> /dev/null && [ "$OS" = "linux" ]; then
    ARCH_INFO=$(readelf -h "$TMP_DIR/rms-agent" 2>/dev/null | grep "Machine:" || echo "")
    echo "   Architecture du binaire téléchargé: $ARCH_INFO"
    
    # Détecter l'architecture du binaire
    if echo "$ARCH_INFO" | grep -qi "x86-64\|Advanced Micro Devices X86-64"; then
        BINARY_ARCH="amd64"
    elif echo "$ARCH_INFO" | grep -qi "AArch64\|ARM aarch64"; then
        BINARY_ARCH="arm64"
    elif echo "$ARCH_INFO" | grep -qi "ARM"; then
        BINARY_ARCH="arm"
    fi
    
    # Vérifier si l'architecture correspond (arm est compatible avec armv7l/armv6l)
    ARCH_MATCH=false
    if [ -n "$BINARY_ARCH" ]; then
        if [ "$BINARY_ARCH" = "$ARCH" ]; then
            ARCH_MATCH=true
        elif [ "$BINARY_ARCH" = "arm" ] && [ "$ARCH" = "armv7l" ]; then
            ARCH_MATCH=true
        elif [ "$BINARY_ARCH" = "arm" ] && [ "$ARCH" = "armv6l" ]; then
            ARCH_MATCH=true
        fi
    fi
    
    if [ "$ARCH_MATCH" = false ] && [ -n "$BINARY_ARCH" ]; then
        echo ""
        echo "❌ ERREUR: Incompatibilité d'architecture détectée !"
        echo "   Architecture demandée: $ARCH ($(uname -m))"
        echo "   Architecture du binaire téléchargé: $BINARY_ARCH"
        echo ""
        echo "💡 Le serveur n'a probablement pas le binaire pour $OS/$ARCH"
        echo "   Le serveur doit builder les binaires multi-plateformes:"
        echo "   ./scripts/build.sh ou make build-all"
        echo ""
        echo "   Ou vérifiez que l'URL de téléchargement est correcte:"
        echo "   $AGENT_URL"
        echo ""
        exit 1
    fi
elif [ "$OS" = "linux" ] && command -v file &> /dev/null; then
    # Fallback avec file si readelf n'est pas disponible
    if echo "$FILE_INFO" | grep -qi "x86-64\|x86_64" && [ "$ARCH" = "arm64" ]; then
        echo ""
        echo "❌ ERREUR: Le binaire téléchargé est pour x86-64 mais vous êtes sur ARM64"
        echo "   Le serveur doit builder le binaire pour linux/arm64"
        echo ""
        exit 1
    fi
fi

# Tester l'exécution du binaire (version --help devrait fonctionner)
if ! "$TMP_DIR/rms-agent" --help &>/dev/null; then
    echo "⚠️  Le binaire ne répond pas à --help, mais cela peut être normal"
fi

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

# Demander le token d'authentification avec valeur par défaut
DEFAULT_TOKEN="votre-token-securise-changez-moi"
read -sp "Token d'authentification [défaut: $DEFAULT_TOKEN]: " AUTH_TOKEN
echo ""
if [ -z "$AUTH_TOKEN" ]; then
    AUTH_TOKEN="$DEFAULT_TOKEN"
    echo "✅ Utilisation du token par défaut (pensez à le changer en production !)"
fi

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

# Vérifier si le fichier existe et est en cours d'utilisation (dans bin ou sbin)
for OLD_AGENT_PATH in /usr/local/bin/rms-agent /usr/local/sbin/rms-agent; do
    if [ -f "$OLD_AGENT_PATH" ]; then
        if command -v lsof >/dev/null 2>&1 && lsof "$OLD_AGENT_PATH" >/dev/null 2>&1; then
            echo "⚠️  Le fichier agent est en cours d'utilisation, arrêt forcé..."
            systemctl stop rms-agent 2>/dev/null || true
            sleep 2
        fi
        echo "🗑️  Suppression de l'ancien agent ($OLD_AGENT_PATH)..."
    fi
done

# Créer les répertoires nécessaires
mkdir -p /opt/remoteshell
mkdir -p /etc/remoteshell

# Détecter si le système utilise /usr/local/bin ou /usr/local/sbin
# Sur certains systèmes (comme Raspberry Pi), seul /usr/local/sbin existe
INSTALL_DIR=""
if [ -d /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
elif [ -d /usr/local/sbin ]; then
    INSTALL_DIR="/usr/local/sbin"
    echo "ℹ️  Utilisation de /usr/local/sbin (détecté sur ce système)"
else
    # Si aucun n'existe, créer /usr/local/bin par défaut
    mkdir -p /usr/local/bin
    INSTALL_DIR="/usr/local/bin"
fi

# Copier l'agent (supprimer l'ancien si nécessaire)
echo "📋 Installation de l'agent vers $INSTALL_DIR/..."
if [ -f $INSTALL_DIR/rms-agent ]; then
    rm -f $INSTALL_DIR/rms-agent
fi
cp "$TMP_DIR/rms-agent" $INSTALL_DIR/rms-agent
chmod +x $INSTALL_DIR/rms-agent

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
ExecStart=${INSTALL_DIR}/rms-agent --server ${SERVER_HOST_PORT} --id "${AGENT_ID}" --name "${AGENT_NAME}" --token ${AUTH_TOKEN} ${USE_TLS}
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

