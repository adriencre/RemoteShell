# Dockerfile pour RemoteShell
FROM golang:1.24-alpine AS builder

# Installer les dépendances système
RUN apk add --no-cache git ca-certificates tzdata

# Définir le répertoire de travail
WORKDIR /app

# Copier les fichiers go.mod et go.sum
COPY go.mod go.sum ./

# Télécharger les dépendances
RUN go mod download

# Copier le code source
COPY . .

# Build des binaires
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o remoteshell-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o remoteshell-agent ./cmd/agent

# Build de l'interface web
FROM node:18-alpine AS web-builder

WORKDIR /app/web

# Copier les fichiers package
COPY web/package*.json ./

# Installer les dépendances (nécessite aussi les devDependencies pour le build)
RUN npm ci

# Copier le code source web
COPY web/ .

# Build de production
RUN npm run build

# Image finale
FROM alpine:latest

# Installer les dépendances runtime
RUN apk --no-cache add ca-certificates tzdata

# Créer un utilisateur non-root
RUN addgroup -g 1001 -S remoteshell && \
    adduser -u 1001 -S remoteshell -G remoteshell

# Définir le répertoire de travail
WORKDIR /app

# Copier les binaires depuis le builder
COPY --from=builder /app/remoteshell-server .
COPY --from=builder /app/remoteshell-agent .

# Copier l'interface web depuis le web-builder
COPY --from=web-builder /app/web/dist ./web

# Copier les fichiers de configuration
COPY --from=builder /app/README.md .
COPY --from=builder /app/LICENSE* .

# Créer le répertoire de données (avant changement d'utilisateur)
RUN mkdir -p /app/data

# Changer les permissions
RUN chown -R remoteshell:remoteshell /app

# Passer à l'utilisateur non-root
USER remoteshell

# Exposer le port
EXPOSE 8080

# Variables d'environnement par défaut (peuvent être surchargées)
ENV REMOTESHELL_SERVER_HOST=0.0.0.0
ENV REMOTESHELL_SERVER_PORT=8080
ENV REMOTESHELL_DB_PATH=/app/data/remoteshell.db
ENV REMOTESHELL_SERVER_TLS=false
ENV REMOTESHELL_LOG_LEVEL=info

# Variables d'environnement optionnelles (à définir au runtime)
# REMOTESHELL_AUTH_TOKEN - Token d'authentification (requis en production)
# REMOTESHELL_CERT_FILE - Chemin du fichier certificat TLS
# REMOTESHELL_KEY_FILE - Chemin du fichier clé privée TLS
# REMOTESHELL_LOG_FILE - Chemin du fichier de log
# REMOTESHELL_RECONNECT_DELAY - Délai de reconnexion (format: 5s, 10m, etc.)
# REMOTESHELL_HEARTBEAT_INTERVAL - Intervalle de heartbeat (format: 30s, 1m, etc.)
# REMOTESHELL_MAX_FILE_SIZE - Taille maximale des fichiers en octets
# REMOTESHELL_CHUNK_SIZE - Taille des chunks pour transferts de fichiers en octets

# Configuration OAuth2/Authentik SSO
# REMOTESHELL_OAUTH2_ENABLED=true - Activer l'authentification OAuth2
# REMOTESHELL_OAUTH2_PROVIDER=authentik - Provider OAuth2 (authentik)
# REMOTESHELL_OAUTH2_CLIENT_ID - Client ID depuis Authentik
# REMOTESHELL_OAUTH2_CLIENT_SECRET - Client Secret depuis Authentik
# REMOTESHELL_OAUTH2_BASE_URL - URL de base d'Authentik (ex: https://auth.example.com)
# REMOTESHELL_OAUTH2_REDIRECT_URL - URL de callback (ex: https://remoteshell.example.com/api/auth/oauth2/callback)
# REMOTESHELL_OAUTH2_SCOPES - Scopes séparés par des virgules (défaut: openid,profile,email)

# Point d'entrée
ENTRYPOINT ["./remoteshell-server"]


