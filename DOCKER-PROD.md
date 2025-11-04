# Guide de déploiement Docker en production

## ⚠️ Problème "Bad Gateway 502" ?

**Si vous avez un 502 Bad Gateway, c'est probablement parce que le serveur Docker n'est pas démarré !**

Vérifiez avec :
```bash
docker ps | grep remoteshell
```

Si aucun conteneur n'apparaît, suivez les étapes ci-dessous pour démarrer le serveur.

## 🚀 Démarrage rapide (méthode recommandée)

### Utiliser le script de démarrage automatique :

```bash
./start-prod.sh
```

Le script va :
1. Créer le fichier `.env` si nécessaire
2. Construire l'image Docker
3. Démarrer le serveur
4. Vérifier que tout fonctionne

## Configuration manuelle

### 1. Créer un fichier `.env` avec vos variables

```bash
cp env.prod.example .env
nano .env
```

Modifiez les valeurs dans `.env` :
```bash
REMOTESHELL_SERVER_HOST=0.0.0.0
REMOTESHELL_SERVER_PORT=8081
REMOTESHELL_AUTH_TOKEN=votre-token-securise
REMOTESHELL_OAUTH2_ENABLED=true
REMOTESHELL_OAUTH2_PROVIDER=authentik
REMOTESHELL_OAUTH2_CLIENT_ID=wLuJP9g1hn8HihS1LuGDu6TSfok6z00dWx6P3XhE
REMOTESHELL_OAUTH2_CLIENT_SECRET=OKe5eQhw7o6OjKqJ5bGSXIZY5gLfzzjvufP15dsk4MckretVFktmWfw7uIh7XDVfvEwBVR1OwngTkPq5skMiuRVJBCIynAadtOTc0QdLSfWkYqRHUF59nfmXCx95iAze
REMOTESHELL_OAUTH2_BASE_URL=https://auth.lfgroup.fr
REMOTESHELL_OAUTH2_REDIRECT_URL=https://rms.lfgroup.fr/api/auth/oauth2/callback
```

### 2. Démarrer le serveur

```bash
docker-compose -f docker-compose.prod.yml up -d
```

### 3. Vérifier que le serveur démarre correctement

```bash
# Voir les logs
docker-compose -f docker-compose.prod.yml logs -f remoteshell-server

# Vérifier le health check
curl http://localhost:8081/health

# Vérifier que le conteneur est en cours d'exécution
docker ps | grep remoteshell-server
```

### 4. Si vous utilisez un reverse proxy externe (Nginx hors Docker)

Si vous avez un Nginx qui tourne sur l'hôte (pas dans Docker), configurez-le pour proxifier vers `localhost:8081` :

```nginx
location / {
    proxy_pass http://127.0.0.1:8081;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}

location /ws {
    proxy_pass http://127.0.0.1:8081;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 86400;
}
```

## 🔍 Diagnostic du problème "Bad Gateway 502"

### Étape 1 : Vérifier si le serveur Docker tourne

```bash
# Vérifier les conteneurs en cours d'exécution
docker ps | grep remoteshell

# Si aucun conteneur n'apparaît, le serveur n'est pas démarré !
# Démarrez-le avec :
docker-compose -f docker-compose.prod.yml up -d
# ou
./start-prod.sh
```

### Étape 2 : Vérifier les logs du serveur

```bash
docker-compose -f docker-compose.prod.yml logs remoteshell-server
```

**Cherchez les erreurs de démarrage :**
- Variables d'environnement manquantes
- Port déjà utilisé
- Erreurs de permissions

### Vérifications à faire :

1. **Le serveur démarre-t-il ?**
   ```bash
   docker-compose -f docker-compose.prod.yml logs remoteshell-server
   ```
   Cherchez les erreurs de démarrage.

2. **Le serveur écoute-t-il sur le bon port ?**
   ```bash
   docker-compose -f docker-compose.prod.yml exec remoteshell-server wget -qO- http://localhost:8081/health
   ```
   Devrait retourner `{"status":"ok"}`

3. **Le reverse proxy peut-il joindre le serveur ?**
   - Si Nginx est dans Docker : vérifiez qu'il est sur le même réseau (`remoteshell-network`)
   - Si Nginx est sur l'hôte : vérifiez qu'il proxifie vers `127.0.0.1:8081` (pas `localhost:8081`)

4. **Les variables d'environnement sont-elles correctes ?**
   ```bash
   docker-compose -f docker-compose.prod.yml exec remoteshell-server env | grep REMOTESHELL
   ```

## Problèmes courants

### Bad Gateway 502 - Solution complète

**Causes possibles (dans l'ordre de probabilité) :**

1. **Le serveur Docker n'est pas démarré** ⚠️ (Cause la plus fréquente)
   ```bash
   # Vérifier
   docker ps | grep remoteshell
   
   # Si vide, démarrer :
   docker-compose -f docker-compose.prod.yml up -d
   # ou
   ./start-prod.sh
   ```

2. **Le serveur n'a pas démarré correctement**
   ```bash
   # Vérifier les logs
   docker-compose -f docker-compose.prod.yml logs remoteshell-server
   ```

3. **Le port est incorrect dans le reverse proxy**
   - Le reverse proxy doit proxifier vers `http://127.0.0.1:8081` (pas `rms.lfgroup.fr:8081`)
   - Vérifiez votre configuration Nginx

4. **Le reverse proxy ne peut pas joindre le conteneur**
   - Si Nginx est sur l'hôte : utilisez `127.0.0.1:8081` (pas `localhost`)
   - Si Nginx est dans Docker : utilisez `remoteshell-server:8081` sur le même réseau

**Solution étape par étape :**
```bash
# 1. Vérifier que le conteneur tourne
docker ps | grep remoteshell-server

# 2. Vérifier que le serveur répond depuis l'hôte
curl http://localhost:8081/health
# Devrait retourner: {"status":"ok"}

# 3. Si le serveur répond, le problème vient du reverse proxy
# Vérifiez votre configuration Nginx - elle doit proxifier vers 127.0.0.1:8081
```

### Le serveur ne démarre pas

**Vérifiez les logs :**
```bash
docker-compose -f docker-compose.prod.yml logs remoteshell-server
```

**Causes courantes :**
- Variables d'environnement manquantes (notamment `REMOTESHELL_AUTH_TOKEN`)
- Port déjà utilisé
- Problème de permissions sur `/app/data`

## Commandes utiles

```bash
# Redémarrer le serveur
docker-compose -f docker-compose.prod.yml restart remoteshell-server

# Reconstruire l'image
docker-compose -f docker-compose.prod.yml build --no-cache

# Arrêter tout
docker-compose -f docker-compose.prod.yml down

# Voir les logs en temps réel
docker-compose -f docker-compose.prod.yml logs -f
```

