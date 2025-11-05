# Guide de Déploiement Automatique

## Problème

Le répertoire `build/` est dans `.gitignore` car les binaires ne doivent pas être versionnés. Cela signifie qu'après chaque `git pull`, les binaires sont supprimés et doivent être rebuildés.

## Solution : Script de Déploiement Automatique

Le script `scripts/deploy-server.sh` automatise le processus de déploiement :

1. Met à jour le code depuis Git
2. Build les binaires multi-plateformes (nécessaires pour `/download/agent`)
3. Build le serveur
4. Redémarre le service systemd

### Utilisation Manuelle

```bash
cd /path/to/RemoteShell
./scripts/deploy-server.sh
```

### Configuration avec Git Hook

#### Option 1 : Hook post-receive (Bare Repository)

Si vous avez un dépôt bare sur le serveur :

```bash
# 1. Créer le hook
sudo nano /path/to/repo.git/hooks/post-receive

# 2. Copier le contenu de scripts/post-receive-hook.example
# 3. Modifier DEPLOY_DIR pour pointer vers votre répertoire de déploiement
# 4. Rendre exécutable
chmod +x /path/to/repo.git/hooks/post-receive
```

#### Option 2 : Webhook GitHub/GitLab

1. Créer un script webhook qui appelle `deploy-server.sh`
2. Configurer le webhook dans GitHub/GitLab pour pointer vers ce script

Exemple avec un simple serveur HTTP :

```bash
#!/bin/bash
# webhook-server.sh
while true; do
    echo -e "HTTP/1.1 200 OK\r\n\r\n" | nc -l -p 9000
    /path/to/RemoteShell/scripts/deploy-server.sh
done
```

#### Option 3 : Cron Job

Pour vérifier automatiquement les mises à jour :

```bash
# Ajouter au crontab (crontab -e)
# Vérifier toutes les 5 minutes
*/5 * * * * cd /path/to/RemoteShell && git fetch && git diff origin/main --quiet || ./scripts/deploy-server.sh
```

### Configuration avec systemd

Si votre serveur tourne comme service systemd, le script le redémarre automatiquement.

Vérifier le nom du service :
```bash
systemctl list-units | grep remoteshell
```

Le script détecte automatiquement `remoteshell-server` ou `remoteshell`.

## Vérification

Après le déploiement, vérifier que les binaires sont disponibles :

```bash
ls -lh build/agent-*
```

Tester l'endpoint de téléchargement :
```bash
curl -I http://localhost:8080/download/agent?os=linux&arch=arm64
```

## Dépannage

### Les binaires ne sont pas trouvés après git pull

Le script `deploy-server.sh` rebuild automatiquement les binaires. Si le problème persiste :

1. Vérifier que Go est installé : `go version`
2. Vérifier que Node.js est installé (pour le build web) : `node --version`
3. Exécuter manuellement : `./scripts/deploy-server.sh`

### Le serveur ne redémarre pas

Vérifier le nom du service :
```bash
systemctl list-units | grep remoteshell
```

Modifier le script `deploy-server.sh` si nécessaire pour utiliser le bon nom de service.

### Permissions

Assurez-vous que le script a les permissions d'exécution :
```bash
chmod +x scripts/deploy-server.sh
```

Si le script doit redémarrer un service systemd, il faudra peut-être utiliser `sudo` :
```bash
sudo ./scripts/deploy-server.sh
```

