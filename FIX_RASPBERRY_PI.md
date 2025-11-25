# Correction du problème d'installation sur Raspberry Pi

## Problème résolu
Sur Raspberry Pi (Raspberry Pi OS), le répertoire `/usr/local/bin` n'existe pas toujours par défaut. Certains systèmes utilisent uniquement `/usr/local/sbin`. Lors de l'exécution des scripts d'installation, si le script tente de copier l'agent vers `/usr/local/bin` sans vérifier son existence, l'installation échoue avec :

```
cp: cannot create regular file '/usr/local/bin/rms-agent': No such file or directory
```

## Solutions appliquées

### 1. Détection automatique bin vs sbin
Les scripts détectent maintenant automatiquement quel répertoire utiliser :
- Si `/usr/local/bin` existe → utiliser `/usr/local/bin`
- Sinon, si `/usr/local/sbin` existe → utiliser `/usr/local/sbin`
- Sinon → créer `/usr/local/bin` et l'utiliser

### 2. Valeurs par défaut facilitées
Pour simplifier l'installation :
- **URL par défaut** : `rms.lfgroup.fr:443` (avec TLS/WSS activé automatiquement)
- **Token par défaut** : `votre-token-securise-changez-moi` (à changer en production)

### 1. Scripts de déploiement locaux
J'ai modifié les scripts suivants pour inclure explicitement `/sbin` et `/usr/sbin` dans le PATH au début de l'exécution :

- ✅ `/home/dell02/Bureau/new-rms/RemoteShell/scripts/install-agent.sh` (ligne 13)
- ✅ `/home/dell02/Bureau/new-rms/RemoteShell/deploy-agent-service.sh` (ligne 9)
- ✅ `/home/dell02/Bureau/new-rms/RemoteShell/deploy-ssh.sh` (ligne 9)

**Modification appliquée :**
```bash
# Définir un PATH complet pour assurer l'accès à tous les binaires système
# Nécessaire sur Raspberry Pi où certains binaires sont dans /sbin ou /usr/sbin
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH"
```

### 2. Fichiers de service systemd
Les fichiers suivants ont déjà le PATH correct dans leur configuration :
- ✅ `systemd/rms-agent.service`
- ✅ `systemd/remoteshell-agent.service`
- ✅ `scripts/install-agent.sh` (génération du service à la ligne 489)

**Configuration systemd :**
```ini
Environment="PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
```

## Instructions pour tester sur Raspberry Pi

### Méthode 1 : Installation directe
```bash
# Sur votre machine de développement, créer une archive
cd /home/dell02/Bureau/new-rms/RemoteShell
make build-all  # Construire les binaires pour toutes les plateformes

# Sur le Raspberry Pi
curl -sSL https://VOTRE_SERVEUR/download/install-agent.sh -o install-agent.sh
chmod +x install-agent.sh
sudo ./install-agent.sh
```

### Méthode 2 : Déploiement via SSH
```bash
# Depuis votre machine de développement
./deploy-ssh.sh --host IP_RASPBERRY --user pi --server VOTRE_SERVEUR:8080 \
  --token VOTRE_TOKEN --name "Raspberry Pi Agent" --install-service
```

### Méthode 3 : Installation manuelle avec le script local
```bash
# Copier le script sur le Raspberry Pi
scp scripts/install-agent.sh pi@IP_RASPBERRY:/tmp/

# Sur le Raspberry Pi
cd /tmp
chmod +x install-agent.sh
sudo ./install-agent.sh
```

## Vérification post-installation

Sur le Raspberry Pi, vérifier que le service fonctionne :

```bash
# Vérifier le statut
sudo systemctl status rms-agent

# Voir les logs
sudo journalctl -u rms-agent -f

# Vérifier que le binaire est bien installé
ls -la /usr/local/bin/rms-agent

# Vérifier que le service peut accéder aux commandes système
sudo systemctl show rms-agent | grep PATH
```

## Diagnostic en cas de problème

Si vous rencontrez toujours des problèmes :

1. **Vérifier le PATH du service**
   ```bash
   sudo systemctl show rms-agent | grep PATH
   ```

2. **Vérifier les logs d'erreur**
   ```bash
   sudo journalctl -u rms-agent -n 50 --no-pager
   ```

3. **Tester manuellement l'agent**
   ```bash
   sudo /usr/local/bin/rms-agent --server VOTRE_SERVEUR:8080 \
     --id "test-raspberry" --name "Test Raspberry" --token "VOTRE_TOKEN"
   ```

4. **Vérifier l'emplacement des commandes système**
   ```bash
   which systemctl
   which lsof
   which ip
   ```

## Architecture Raspberry Pi

Le script détecte automatiquement l'architecture :
- **Raspberry Pi 3/4** : ARM64 (aarch64)
- **Raspberry Pi Zero/2** : ARMv6/v7 (arm)

Assurez-vous que les binaires pour ces architectures sont bien construits :
```bash
make build-all
# ou
./scripts/build.sh
```

Les binaires devraient se trouver dans :
- `build/agent-linux-arm64` (pour Raspberry Pi 3/4)
- `build/agent-linux-arm` (pour Raspberry Pi Zero/2, si supporté)
