# Dockerfile pour RemoteShell
FROM golang:1.21-alpine AS builder

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

# Installer les dépendances
RUN npm ci --only=production

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

# Changer les permissions
RUN chown -R remoteshell:remoteshell /app

# Passer à l'utilisateur non-root
USER remoteshell

# Exposer le port
EXPOSE 8080

# Variables d'environnement par défaut
ENV REMOTESHELL_SERVER_HOST=0.0.0.0
ENV REMOTESHELL_SERVER_PORT=8080
ENV REMOTESHELL_DB_PATH=/app/data/remoteshell.db

# Créer le répertoire de données
RUN mkdir -p /app/data

# Point d'entrée
ENTRYPOINT ["./remoteshell-server"]


