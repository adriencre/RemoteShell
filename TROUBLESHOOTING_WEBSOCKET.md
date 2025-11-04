# Dépannage des erreurs de connexion WebSocket

## Problème : Handshake error sur le port 443

### Cause probable

L'erreur "handshake error" sur le port 443 indique généralement que le reverse proxy (nginx) n'est pas correctement configuré pour transmettre les connexions WebSocket.

### Solution : Configuration nginx pour WebSockets

Votre configuration nginx doit inclure les headers WebSocket suivants :

```nginx
location /ws {
    proxy_pass http://localhost:8080;  # Port interne du serveur Go
    proxy_http_version 1.1;
    
    # Headers WebSocket essentiels
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    
    # Headers standards
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # Timeouts pour WebSocket
    proxy_connect_timeout 7d;
    proxy_send_timeout 7d;
    proxy_read_timeout 7d;
    
    # Buffer settings
    proxy_buffering off;
}
```

### Vérification de la configuration

1. **Vérifier que nginx écoute sur le port 443 :**
   ```bash
   sudo netstat -tlnp | grep :443
   ```

2. **Tester la connexion WebSocket manuellement :**
   ```bash
   curl -i -N -H "Connection: Upgrade" \
        -H "Upgrade: websocket" \
        -H "Sec-WebSocket-Version: 13" \
        -H "Sec-WebSocket-Key: test" \
        https://rms.lfgroup.fr/ws
   ```

3. **Vérifier les logs nginx :**
   ```bash
   sudo tail -f /var/log/nginx/error.log
   ```

### Alternative : Utiliser le port direct (si disponible)

Si vous ne pouvez pas configurer le reverse proxy, vous pouvez essayer de vous connecter directement au port du serveur Go (si exposé) :

- **Port 8080** : Connexion WebSocket non sécurisée (ws://)
- **Port 8081** : Si le serveur écoute sur ce port

### Test de connexion

Pour tester quelle configuration fonctionne :

1. **Port 443 avec TLS :**
   ```bash
   # Dans le script d'installation, utilisez :
   SERVER_HOST_PORT="rms.lfgroup.fr:443"
   # Avec --tls activé
   ```

2. **Port direct (si accessible) :**
   ```bash
   # Sans TLS
   SERVER_HOST_PORT="IP_SERVEUR:8080"
   ```

### Logs à vérifier

Sur le serveur RemoteShell :
```bash
# Logs du serveur
journalctl -u remoteshell-server -f

# Logs nginx
sudo tail -f /var/log/nginx/error.log
sudo tail -f /var/log/nginx/access.log
```

Sur l'agent :
```bash
# Logs de l'agent
journalctl -u remoteshell-agent -f
```

### Messages d'erreur courants

- **"handshake error"** : Reverse proxy mal configuré ou problème SSL
- **"connection refused"** : Port fermé ou service non démarré
- **"timeout"** : Firewall ou réseau bloquant

### Commande de test rapide

Pour tester la connexion WebSocket directement :
```bash
wscat -c wss://rms.lfgroup.fr/ws
# ou
wscat -c ws://rms.lfgroup.fr:8080/ws
```

