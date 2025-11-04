# Guide de déploiement avec Dockploy

## Configuration pour Dockploy

Dockploy peut utiliser directement le `docker-compose.yml` ou `docker-compose.prod.yml`. 

## 📋 Étapes de déploiement

### 1. Variables d'environnement à configurer dans Dockploy

Dans l'interface Dockploy, configurez ces variables d'environnement :

```bash
REMOTESHELL_SERVER_HOST=0.0.0.0
REMOTESHELL_SERVER_PORT=8081
REMOTESHELL_AUTH_TOKEN=votre-token-securise-changez-moi
REMOTESHELL_DB_PATH=/app/data/remoteshell.db
REMOTESHELL_LOG_LEVEL=info

# OAuth2 / Authentik
REMOTESHELL_OAUTH2_ENABLED=true
REMOTESHELL_OAUTH2_PROVIDER=authentik
REMOTESHELL_OAUTH2_CLIENT_ID=wLuJP9g1hn8HihS1LuGDu6TSfok6z00dWx6P3XhE
REMOTESHELL_OAUTH2_CLIENT_SECRET=OKe5eQhw7o6OjKqJ5bGSXIZY5gLfzzjvufP15dsk4MckretVFktmWfw7uIh7XDVfvEwBVR1OwngTkPq5skMiuRVJBCIynAadtOTc0QdLSfWkYqRHUF59nfmXCx95iAze
REMOTESHELL_OAUTH2_BASE_URL=https://auth.lfgroup.fr
REMOTESHELL_OAUTH2_REDIRECT_URL=https://rms.lfgroup.fr/api/auth/oauth2/callback
```

### 2. Configuration du port dans Dockploy

- **Port interne** : `8081`
- **Port exposé** : Dockploy peut mapper automatiquement ou vous pouvez le configurer

### 3. Fichier docker-compose à utiliser

**✅ Méthode recommandée : Renommer le fichier**

Dockploy utilise automatiquement `docker-compose.yml` par défaut. Pour utiliser la version optimisée :

```bash
# Option 1 : Utiliser le script automatique
./prepare-dockploy.sh

# Option 2 : Manuellement
mv docker-compose.yml docker-compose.yml.backup
mv docker-compose.dockploy.yml docker-compose.yml
```

**Alternative : Configuration dans l'interface Dockploy**

Si Dockploy permet de spécifier un fichier personnalisé :
- Cherchez "Compose file", "Docker Compose File", ou "Custom compose file"
- Spécifiez : `docker-compose.dockploy.yml`

**Important** : Si vous utilisez un reverse proxy externe (Nginx géré par Dockploy ou ailleurs), assurez-vous que :
- Le reverse proxy proxifie vers `remoteshell-server:8081` (si dans le même réseau Docker)
- Ou vers `127.0.0.1:8081` (si Nginx est sur l'hôte)

### 4. Volumes persistants

Le volume `remoteshell-data` sera créé automatiquement par Docker Compose. Il stocke :
- La base de données SQLite (`/app/data/remoteshell.db`)
- Les données persistantes de l'application

### 5. Health Check

Dockploy peut utiliser le healthcheck configuré dans docker-compose.yml :
```yaml
healthcheck:
  test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8081/health"]
```

## 🔧 Configuration spécifique Dockploy

### Si vous utilisez le reverse proxy de Dockploy

Si Dockploy gère automatiquement le reverse proxy (Traefik, Nginx, etc.), vous pouvez :

1. **Désactiver le port mappé** dans docker-compose.yml (si Dockploy gère les ports)
2. **Utiliser les labels** pour la configuration du reverse proxy (selon ce que Dockploy supporte)

### Exemple avec labels Traefik (si Dockploy utilise Traefik)

```yaml
services:
  remoteshell-server:
    # ... autres configs ...
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.remoteshell.rule=Host(`rms.lfgroup.fr`)"
      - "traefik.http.routers.remoteshell.entrypoints=websecure"
      - "traefik.http.routers.remoteshell.tls.certresolver=letsencrypt"
      - "traefik.http.services.remoteshell.loadbalancer.server.port=8081"
```

## ✅ Checklist de déploiement

- [ ] Toutes les variables d'environnement sont configurées dans Dockploy
- [ ] Le port `8081` est correctement mappé
- [ ] Le reverse proxy est configuré pour proxifier vers le bon port
- [ ] L'URL de redirection OAuth2 correspond à votre domaine
- [ ] Le token d'authentification est sécurisé (pas de valeur par défaut)
- [ ] Les certificats SSL sont configurés (si nécessaire)

## 🐛 Dépannage

### Si vous avez un 502 Bad Gateway

1. Vérifiez que le conteneur démarre :
   ```bash
   docker ps | grep remoteshell
   ```

2. Vérifiez les logs :
   ```bash
   docker logs remoteshell-server
   ```

3. Vérifiez que le serveur répond :
   ```bash
   curl http://localhost:8081/health
   ```

4. Si le serveur répond mais le reverse proxy ne fonctionne pas :
   - Vérifiez la configuration du reverse proxy dans Dockploy
   - Assurez-vous que le proxy pointe vers `remoteshell-server:8081` ou `127.0.0.1:8081`

## 📝 Notes importantes

- **Ne committez jamais** le fichier `.env` avec des secrets réels
- Le `REMOTESHELL_AUTH_TOKEN` doit être le même pour tous les agents qui se connectent
- L'URL `REMOTESHELL_OAUTH2_REDIRECT_URL` doit correspondre exactement à celle configurée dans Authentik

