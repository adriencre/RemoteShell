# RemoteShell - Gestion des Serveurs d'Impression

RemoteShell est un système de gestion à distance des serveurs d'impression qui permet de surveiller, contrôler et administrer plusieurs serveurs d'impression depuis une interface web centralisée.

## 🚀 Fonctionnalités

- **Gestion multi-serveurs** : Surveillez et contrôlez plusieurs serveurs d'impression
- **Interface web moderne** : Dashboard React avec design responsive
- **Terminal interactif** : Exécution de commandes à distance via WebSocket avec shell persistant
- **Gestionnaire de fichiers** : Upload/download de fichiers bidirectionnel avec accès système complet
- **Gestion des services** : Contrôlez les services systemd et conteneurs Docker à distance
- **Visualisation des logs** : Consultez les logs de l'agent et du système (journalctl, /var/log/*)
- **Monitoring d'imprimantes** : Surveillance en temps réel des imprimantes (CUPS/Linux, WMI/Windows)
- **Authentification sécurisée** : JWT tokens avec support TLS/SSL
- **Multi-plateforme** : Support Linux, Windows et macOS
- **API REST** : Interface programmatique complète
- **Service systemd** : Installation automatique comme service Linux

## 🏗️ Architecture

Le système est composé de 3 composants principaux :

1. **Agent** (`rms-agent`) : Service déployé sur chaque serveur d'impression
2. **Serveur Central** (`remoteshell-server`) : Hub de gestion avec API REST et WebSocket
3. **Interface Web** : Dashboard React pour l'administration

## 📋 Prérequis

- **Go 1.21+** pour le backend
- **Node.js 18+** et **npm** pour l'interface web
- **CUPS** (Linux) ou **WMI** (Windows) pour le monitoring d'imprimantes

## 🛠️ Installation

### Installation rapide

```bash
# Cloner le repository
git clone https://github.com/votre-org/remoteshell.git
cd remoteshell

# Build complet
make build

# Ou utiliser le script de build
./scripts/build.sh
```

### Installation manuelle

#### 1. Backend (Go)

```bash
# Installer les dépendances
go mod download

# Build du serveur
go build -o remoteshell-server ./cmd/server

# Build de l'agent
go build -o rms-agent ./cmd/agent
```

#### 2. Interface Web (React)

```bash
cd web

# Installer les dépendances
npm install

# Build de production
npm run build
```

## 🚀 Utilisation

### Démarrage du serveur central

```bash
# Mode développement
./remoteshell-server

# Mode production avec TLS
./remoteshell-server --tls --cert server.crt --key server.key --port 443

# Avec base de données personnalisée
./remoteshell-server --db /var/lib/remoteshell/data.db
```

### Démarrage d'un agent

```bash
# Connexion au serveur central
./rms-agent --server 192.168.1.100:8080 --token YOUR_TOKEN

# Avec nom personnalisé
./rms-agent --server 192.168.1.100:8080 --token YOUR_TOKEN --name "Serveur-Impression-01"

# Avec TLS
./rms-agent --server 192.168.1.100:443 --token YOUR_TOKEN --tls
```

### Interface Web

1. Ouvrez votre navigateur sur `http://localhost:8080`
2. Connectez-vous avec les identifiants par défaut : `admin` / `admin`
3. Configurez vos agents et surveillez vos serveurs d'impression

### Nouvelles fonctionnalités 🎯

#### Gestion des services

L'interface web permet maintenant de gérer les services systemd et les conteneurs Docker à distance :

- 📋 **Liste des services** : Visualisez tous les services systemd et conteneurs Docker
- ▶️ **Démarrage/Arrêt** : Contrôlez les services en un clic
- 🔄 **Redémarrage** : Redémarrez les services rapidement
- 🔍 **Filtres** : Filtrez par type (systemd/docker) et recherchez par nom
- ⚡ **Statut en temps réel** : Mise à jour automatique du statut

Accès : Dashboard → Agent → **Gestion des services**

#### Visualisation des logs

Consultez les logs de l'agent et du système directement depuis l'interface :

- 📝 **Logs de l'agent** : Historique des actions de l'agent RemoteShell
- 🖥️ **Logs système** : Accès à journalctl pour les logs systemd
- 📂 **Fichiers logs** : Lecture des fichiers dans /var/log/*
- 🔍 **Filtres avancés** : Filtrez par niveau (error, warning, info), service, date
- 🔄 **Mode streaming** : Rafraîchissement automatique en temps réel
- 💾 **Export** : Téléchargez les logs pour analyse

Accès : Dashboard → Agent → **Visualisation des logs**

#### Terminal persistant

Le terminal a été amélioré avec un shell persistant :

- 🔒 **Contexte conservé** : Les commandes `cd`, variables d'environnement, etc. persistent
- 👑 **Privilèges root** : Exécution automatique avec privilèges élevés
- ⚡ **Performances** : Pas besoin de réinitialiser l'environnement à chaque commande
- 📜 **Historique** : Navigation dans l'historique des commandes

#### Gestionnaire de fichiers complet

Accès système complet avec le nouveau gestionnaire de fichiers :

- 🌍 **Accès root** : Naviguez dans tout le système de fichiers (/)
- 📁 **Opérations complètes** : Créer, supprimer, télécharger, uploader
- 🔐 **Permissions** : Affichage des permissions Unix
- 📊 **Informations détaillées** : Taille, date de modification, type

## ⚙️ Configuration

### Variables d'environnement

#### Serveur
- `REMOTESHELL_SERVER_HOST` : Adresse d'écoute (défaut: 0.0.0.0)
- `REMOTESHELL_SERVER_PORT` : Port d'écoute (défaut: 8080)
- `REMOTESHELL_SERVER_TLS` : Activer TLS (défaut: false)
- `REMOTESHELL_CERT_FILE` : Fichier de certificat TLS
- `REMOTESHELL_KEY_FILE` : Fichier de clé privée TLS
- `REMOTESHELL_DB_PATH` : Chemin de la base de données SQLite (défaut: remoteshell.db)

#### Base de données MySQL
- `REMOTESHELL_MYSQL_ENABLED` : Activer MySQL (défaut: false, mettre à "true" pour activer)
- `REMOTESHELL_MYSQL_HOST` : Adresse du serveur MySQL
- `REMOTESHELL_MYSQL_PORT` : Port du serveur MySQL (défaut: 3306)
- `REMOTESHELL_MYSQL_USER` : Nom d'utilisateur MySQL
- `REMOTESHELL_MYSQL_PASSWORD` : Mot de passe MySQL
- `REMOTESHELL_MYSQL_DATABASE` : Nom de la base de données

**Note**: Toutes les tables sont préfixées par `rms_` pour éviter les conflits (ex: `rms_users`, `rms_agents`, etc.)

#### Agent
- `REMOTESHELL_AGENT_ID` : ID unique de l'agent
- `REMOTESHELL_AGENT_NAME` : Nom de l'agent
- `REMOTESHELL_AUTH_TOKEN` : Token d'authentification
- `REMOTESHELL_SERVER_HOST` : Adresse du serveur central
- `REMOTESHELL_SERVER_PORT` : Port du serveur central

### Fichiers de configuration

Créez un fichier `config.yaml` pour une configuration avancée :

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  tls:
    enabled: true
    cert_file: "server.crt"
    key_file: "server.key"
  database:
    path: "remoteshell.db"

agent:
  id: "agent-001"
  name: "Serveur Impression Principal"
  server:
    host: "192.168.1.100"
    port: 8080
    tls: true
  auth:
    token: "your-secret-token"
```

## 🔧 Installation comme service Linux (systemd)

### Installation automatique de l'agent

Le moyen le plus simple d'installer l'agent comme service systemd est d'utiliser le script d'installation automatique :

```bash
# Compiler l'agent
make agent

# Installer le service (en mode interactif)
sudo ./scripts/install-service.sh install

# Ou depuis un serveur distant
./deploy-agent-root.sh 10.0.0.72 ServeurImpression --install-service
```

Le script d'installation vous demandera :
- L'URL du serveur central (ex: `10.0.0.59:8081`)
- L'ID de l'agent (ex: `serveur-impression-01`)
- Le nom de l'agent (ex: `Serveur d'impression principal`)
- Le token d'authentification

### Gestion du service

```bash
# Vérifier le statut
sudo systemctl status rms-agent

# Voir les logs en temps réel
sudo journalctl -u rms-agent -f

# Redémarrer le service
sudo systemctl restart rms-agent

# Arrêter le service
sudo systemctl stop rms-agent

# Désactiver le démarrage automatique
sudo systemctl disable rms-agent
```

### Mise à jour de l'agent

```bash
# Recompiler l'agent
make agent

# Mettre à jour le service
sudo ./scripts/install-service.sh update
```

### Désinstallation

```bash
# Désinstaller complètement le service
sudo ./scripts/install-service.sh uninstall
```

### Installation manuelle (avancé)

Si vous préférez installer manuellement :

#### Serveur

```bash
# Copier l'exécutable
sudo cp ./build/remoteshell-server /usr/local/bin/

# Créer le fichier de service
sudo nano /etc/systemd/system/remoteshell-server.service

# Recharger systemd
sudo systemctl daemon-reload

# Activer et démarrer
sudo systemctl enable remoteshell-server
sudo systemctl start remoteshell-server
```

#### Agent

```bash
# Copier l'exécutable
sudo cp ./build/rms-agent /usr/local/bin/

# Créer la configuration
sudo mkdir -p /etc/remoteshell
sudo nano /etc/remoteshell/agent.conf

# Créer le fichier de service en utilisant le template
sudo cp ./systemd/rms-agent.service /etc/systemd/system/
sudo nano /etc/systemd/system/rms-agent.service

# Recharger systemd
sudo systemctl daemon-reload

# Activer et démarrer
sudo systemctl enable rms-agent
sudo systemctl start rms-agent
```

### Localisation des fichiers

- **Binaires** : `/usr/local/bin/rms-agent`
- **Configuration** : `/etc/remoteshell/agent.conf`
- **Service systemd** : `/etc/systemd/system/rms-agent.service`
- **Logs** : `journalctl -u rms-agent`
- **Répertoire de travail** : `/opt/remoteshell`

### Windows (Service)

```cmd
# Installer le service serveur
sc create RemoteShellServer binPath= "C:\Program Files\RemoteShell\remoteshell-server.exe" start= auto

# Démarrer le service
sc start RemoteShellServer

# Installer le service agent
sc create RemoteShellAgent binPath= "C:\Program Files\RemoteShell\rms-agent.exe --server localhost:8080 --token YOUR_TOKEN" start= auto

# Démarrer le service
sc start RemoteShellAgent
```

## 🔒 Sécurité

### Authentification

- Utilisez des tokens JWT forts
- Changez les identifiants par défaut
- Activez TLS en production

### Génération de certificats

```bash
# Certificat auto-signé pour le développement
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes

# Certificat Let's Encrypt pour la production
certbot certonly --standalone -d votre-domaine.com
```

### Firewall

```bash
# Ouvrir le port du serveur central
sudo ufw allow 8080/tcp

# Ou pour HTTPS
sudo ufw allow 443/tcp
```

## 📊 Monitoring

### Logs

Les logs sont disponibles via :

- **Serveur** : `journalctl -u remoteshell-server -f`
- **Agent** : `journalctl -u rms-agent -f`

### Métriques

L'API expose des métriques de santé :

```bash
# Vérifier l'état du serveur
curl http://localhost:8080/health

# Statistiques des agents
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/agents
```

## 🐳 Docker

### Build de l'image

```bash
# Build de l'image
docker build -t remoteshell:latest .

# Ou utiliser le Makefile
make docker-build
```

### Démarrage avec Docker Compose

```yaml
version: '3.8'
services:
  remoteshell-server:
    image: remoteshell:latest
    ports:
      - "8080:8080"
    environment:
      - REMOTESHELL_SERVER_HOST=0.0.0.0
      - REMOTESHELL_SERVER_PORT=8080
    volumes:
      - ./data:/app/data
    command: ["./remoteshell-server"]
```

## 🧪 Tests

```bash
# Tests Go
make test

# Tests de l'interface web
cd web && npm test

# Tests d'intégration
make test-integration
```

## 📈 Performance

### Optimisations recommandées

- Utilisez un reverse proxy (nginx) pour l'interface web
- Configurez un load balancer pour plusieurs serveurs centraux
- Utilisez une base de données PostgreSQL pour de gros volumes
- Activez la compression gzip

### Limites

- **Agents simultanés** : 1000+ (selon les ressources)
- **Taille des fichiers** : 100MB par défaut
- **Connexions WebSocket** : 10000+ (selon les ressources)

## 🤝 Contribution

1. Fork le projet
2. Créez une branche feature (`git checkout -b feature/AmazingFeature`)
3. Committez vos changements (`git commit -m 'Add some AmazingFeature'`)
4. Push vers la branche (`git push origin feature/AmazingFeature`)
5. Ouvrez une Pull Request

## 📝 Licence

Ce projet est sous licence MIT. Voir le fichier [LICENSE](LICENSE) pour plus de détails.

## 🆘 Support

- **Documentation** : [Wiki du projet](https://github.com/votre-org/remoteshell/wiki)
- **Issues** : [GitHub Issues](https://github.com/votre-org/remoteshell/issues)
- **Discussions** : [GitHub Discussions](https://github.com/votre-org/remoteshell/discussions)

## 🗺️ Roadmap

### ✅ Récemment implémenté

- [x] Gestion des services systemd et Docker
- [x] Visualisation des logs (agent, système, fichiers)
- [x] Installation automatique comme service systemd
- [x] Terminal avec shell persistant
- [x] Gestionnaire de fichiers avec accès root complet

### 🔜 À venir

- [ ] Support des notifications push
- [ ] Intégration avec des systèmes de monitoring (Prometheus, Grafana)
- [ ] Support des plugins personnalisés
- [ ] Interface mobile (React Native)
- [ ] Support des imprimantes 3D
- [ ] Intégration avec Active Directory/LDAP
- [ ] Gestion avancée des conteneurs (logs, stats, exec)
- [ ] Éditeur de fichiers intégré
- [ ] Planification de tâches (cron jobs)

---

**RemoteShell** - Simplifiez la gestion de vos serveurs d'impression ! 🖨️
