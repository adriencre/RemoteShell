#!/bin/bash

# Script de build multi-plateforme pour RemoteShell
set -e

# Couleurs pour les messages
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="remoteshell"
VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="build"
DIST_DIR="dist"

# Plateformes supportées
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "windows/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

# Fonction d'affichage des messages
log() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Vérifier que Go est installé
check_go() {
    if ! command -v go &> /dev/null; then
        error "Go n'est pas installé. Veuillez installer Go 1.21 ou plus récent."
        exit 1
    fi
    
    GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
    log "Go version détectée: $GO_VERSION"
}

# Vérifier que Node.js est installé
check_node() {
    if ! command -v node &> /dev/null; then
        error "Node.js n'est pas installé. Veuillez installer Node.js 18 ou plus récent."
        exit 1
    fi
    
    NODE_VERSION=$(node --version)
    log "Node.js version détectée: $NODE_VERSION"
}

# Nettoyer les anciens builds
clean() {
    log "Nettoyage des anciens builds..."
    rm -rf $BUILD_DIR
    rm -rf $DIST_DIR
    mkdir -p $BUILD_DIR
    mkdir -p $DIST_DIR
}

# Build de l'interface web
build_web() {
    log "Build de l'interface web..."
    
    cd web
    
    # Installer les dépendances si nécessaire
    if [ ! -d "node_modules" ]; then
        log "Installation des dépendances npm..."
        npm install
    fi
    
    # Build de production
    log "Build de production React..."
    npm run build
    
    # Vérifier que le répertoire dist existe et contient des fichiers
    if [ ! -d "dist" ] || [ -z "$(ls -A dist 2>/dev/null)" ]; then
        error "Le build web a échoué : le répertoire dist est vide ou n'existe pas"
        cd ..
        exit 1
    fi
    
    # Créer le répertoire de destination s'il n'existe pas
    mkdir -p ../$BUILD_DIR/web
    
    # Copier les fichiers buildés
    cp -r dist/* ../$BUILD_DIR/web/
    
    cd ..
    success "Interface web buildée avec succès"
}

# Build des binaires Go
build_binaries() {
    log "Build des binaires Go pour toutes les plateformes..."
    
    for platform in "${PLATFORMS[@]}"; do
        IFS='/' read -r os arch <<< "$platform"
        
        log "Build pour $os/$arch..."
        
        # Nom du fichier de sortie
        if [ "$os" = "windows" ]; then
            EXT=".exe"
        else
            EXT=""
        fi
        
        # Build du serveur (statique, sans CGO pour compatibilité maximale)
        SERVER_OUTPUT="$BUILD_DIR/server-${os}-${arch}${EXT}"
        log "  - Serveur: $SERVER_OUTPUT"
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -a -installsuffix cgo -ldflags "-X main.version=$VERSION -extldflags '-static'" -o "$SERVER_OUTPUT" ./cmd/server
        
        # Build de l'agent (statique, sans CGO pour compatibilité maximale)
        AGENT_OUTPUT="$BUILD_DIR/agent-${os}-${arch}${EXT}"
        log "  - Agent: $AGENT_OUTPUT"
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -a -installsuffix cgo -ldflags "-X main.version=$VERSION -extldflags '-static'" -o "$AGENT_OUTPUT" ./cmd/agent
        
        success "Build terminé pour $os/$arch"
    done
}

# Créer les packages de distribution
create_packages() {
    log "Création des packages de distribution..."
    
    for platform in "${PLATFORMS[@]}"; do
        IFS='/' read -r os arch <<< "$platform"
        
        if [ "$os" = "windows" ]; then
            EXT=".exe"
            PACKAGE_EXT=".zip"
        else
            EXT=""
            PACKAGE_EXT=".tar.gz"
        fi
        
        PACKAGE_NAME="${PROJECT_NAME}-${VERSION}-${os}-${arch}"
        PACKAGE_DIR="$DIST_DIR/$PACKAGE_NAME"
        
        log "Création du package $PACKAGE_NAME..."
        
        # Créer le répertoire du package
        mkdir -p "$PACKAGE_DIR"
        
        # Copier les binaires
        cp "$BUILD_DIR/server-${os}-${arch}${EXT}" "$PACKAGE_DIR/remoteshell-server${EXT}"
        cp "$BUILD_DIR/agent-${os}-${arch}${EXT}" "$PACKAGE_DIR/rms-agent${EXT}"
        
        # Copier l'interface web
        cp -r "$BUILD_DIR/web" "$PACKAGE_DIR/"
        
        # Copier les fichiers de configuration
        cp README.md "$PACKAGE_DIR/"
        cp LICENSE "$PACKAGE_DIR/" 2>/dev/null || true
        
        # Créer les fichiers de service
        create_service_files "$PACKAGE_DIR" "$os"
        
        # Créer l'archive
        cd "$DIST_DIR"
        if [ "$os" = "windows" ]; then
            zip -r "${PACKAGE_NAME}${PACKAGE_EXT}" "$PACKAGE_NAME"
        else
            tar -czf "${PACKAGE_NAME}${PACKAGE_EXT}" "$PACKAGE_NAME"
        fi
        cd ..
        
        # Nettoyer le répertoire temporaire
        rm -rf "$PACKAGE_DIR"
        
        success "Package créé: ${PACKAGE_NAME}${PACKAGE_EXT}"
    done
}

# Créer les fichiers de service
create_service_files() {
    local package_dir="$1"
    local os="$2"
    
    if [ "$os" = "linux" ] || [ "$os" = "darwin" ]; then
        # Service systemd pour Linux
        cat > "$package_dir/remoteshell-server.service" << EOF
[Unit]
Description=RemoteShell Server
After=network.target

[Service]
Type=simple
User=remoteshell
Group=remoteshell
WorkingDirectory=/opt/remoteshell
ExecStart=/opt/remoteshell/remoteshell-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

        cat > "$package_dir/rms-agent.service" << EOF
[Unit]
Description=RemoteShell Agent
After=network.target

[Service]
Type=simple
User=remoteshell
Group=remoteshell
WorkingDirectory=/opt/remoteshell
ExecStart=/opt/remoteshell/rms-agent --server localhost:8080 --token YOUR_TOKEN_HERE
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

        # Script d'installation
        cat > "$package_dir/install.sh" << 'EOF'
#!/bin/bash
set -e

# Script d'installation pour RemoteShell

echo "Installation de RemoteShell..."

# Créer l'utilisateur
if ! id "remoteshell" &>/dev/null; then
    useradd -r -s /bin/false remoteshell
fi

# Créer le répertoire d'installation
mkdir -p /opt/remoteshell
cp remoteshell-server /opt/remoteshell/
cp rms-agent /opt/remoteshell/
cp -r web /opt/remoteshell/
chown -R remoteshell:remoteshell /opt/remoteshell

# Installer les services systemd
cp remoteshell-server.service /etc/systemd/system/
cp rms-agent.service /etc/systemd/system/
systemctl daemon-reload

echo "Installation terminée!"
echo "Pour démarrer le serveur: systemctl start remoteshell-server"
echo "Pour démarrer l'agent: systemctl start rms-agent"
echo "N'oubliez pas de configurer le token dans le service agent!"
EOF
        chmod +x "$package_dir/install.sh"

    elif [ "$os" = "windows" ]; then
        # Service Windows
        cat > "$package_dir/install.bat" << 'EOF'
@echo off
echo Installation de RemoteShell...

REM Créer le répertoire d'installation
if not exist "C:\Program Files\RemoteShell" mkdir "C:\Program Files\RemoteShell"
copy remoteshell-server.exe "C:\Program Files\RemoteShell\"
copy rms-agent.exe "C:\Program Files\RemoteShell\"
xcopy /E /I web "C:\Program Files\RemoteShell\web"

echo Installation terminée!
echo Pour installer le service, utilisez: sc create RemoteShellServer binPath= "C:\Program Files\RemoteShell\remoteshell-server.exe"
echo Pour démarrer le service: sc start RemoteShellServer
EOF
    fi
}

# Afficher les statistiques de build
show_stats() {
    log "Statistiques de build:"
    echo "  - Version: $VERSION"
    echo "  - Plateformes: ${#PLATFORMS[@]}"
    echo "  - Packages créés: $(ls -1 $DIST_DIR/*.tar.gz $DIST_DIR/*.zip 2>/dev/null | wc -l)"
    echo "  - Taille totale: $(du -sh $DIST_DIR | cut -f1)"
}

# Fonction principale
main() {
    log "Démarrage du build RemoteShell v$VERSION"
    
    check_go
    check_node
    clean
    build_web
    build_binaries
    create_packages
    show_stats
    
    success "Build terminé avec succès!"
    log "Les packages sont disponibles dans le répertoire $DIST_DIR"
}

# Exécuter le script
main "$@"


