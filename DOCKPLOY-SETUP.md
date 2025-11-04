# Configuration Dockploy - Utiliser docker-compose.dockploy.yml

## Méthode 1 : Renommer le fichier (✅ Recommandé)

Dockploy utilise automatiquement `docker-compose.yml` par défaut. La solution la plus simple est de renommer les fichiers :

```bash
# Sauvegarder l'ancien docker-compose.yml
mv docker-compose.yml docker-compose.yml.backup

# Renommer docker-compose.dockploy.yml en docker-compose.yml
mv docker-compose.dockploy.yml docker-compose.yml
```

Maintenant Dockploy utilisera automatiquement votre configuration optimisée !

## Méthode 2 : Configuration dans l'interface Dockploy

Si Dockploy vous permet de spécifier un fichier docker-compose personnalisé :

1. **Dans l'interface Dockploy**, cherchez une option comme :
   - "Compose file" ou "Docker Compose File"
   - "Configuration file"
   - "Custom compose file"
   - "Docker Compose Path"

2. **Spécifiez le chemin** : `docker-compose.dockploy.yml`

3. **Ou utilisez la variable d'environnement** (si supportée) :
   ```
   COMPOSE_FILE=docker-compose.dockploy.yml
   ```

## Méthode 3 : Script de préparation

Vous pouvez créer un script qui prépare les fichiers avant le déploiement :

```bash
#!/bin/bash
# prepare-dockploy.sh

# Sauvegarder docker-compose.yml si nécessaire
if [ -f docker-compose.yml ] && [ ! -f docker-compose.yml.backup ]; then
    cp docker-compose.yml docker-compose.yml.backup
fi

# Copier docker-compose.dockploy.yml vers docker-compose.yml
cp docker-compose.dockploy.yml docker-compose.yml

echo "✅ Configuration Dockploy prête !"
```

## Vérification

Après configuration, vérifiez que Dockploy utilise le bon fichier en regardant les logs de déploiement ou en inspectant le conteneur créé.

## Retour à la configuration par défaut

Si vous voulez revenir à l'ancien fichier :

```bash
mv docker-compose.yml docker-compose.dockploy.yml
mv docker-compose.yml.backup docker-compose.yml
```

