# Problèmes identifiés qui interrompent le shell persistant

## Problème 1 : Répertoire initial vide

**Ligne 58 dans client.go :**
```go
executor := NewExecutor("")  // workingDir = ""
```

**Ligne 96 dans executor.go :**
```go
if e.workingDir != "" {  // Cette condition est FALSE !
    e.shellIn.Write([]byte(fmt.Sprintf("cd %s\n", e.workingDir)))
}
```

**Conséquence :** Le shell bash démarre depuis le répertoire de travail du processus parent (`/opt/remoteshell` où l'agent s'exécute), pas depuis un répertoire défini.

## Problème 2 : Bash en mode non-interactif

**Ligne 64 dans executor.go :**
```go
cmd = exec.Command("sudo", "-n", "bash")
```

Bash en mode non-interactif peut avoir des comportements différents :
- Il peut lire un fichier `.bashrc` qui pourrait changer le répertoire
- Il peut réinitialiser certaines variables d'environnement
- Le répertoire de travail peut être réinitialisé dans certains cas

## Problème 3 : Le scanner consomme peut-être trop de données

**Ligne 115-128 dans executor.go :**
```go
scanner := bufio.NewScanner(e.shellOut)
for scanner.Scan() {
    line := scanner.Text()
    if strings.Contains(line, marker) {
        break
    }
    output.WriteString(line + "\n")
}
```

Si le scanner lit toutes les données disponibles mais ne trouve pas le marqueur (par exemple, si le prompt bash est lu), il pourrait bloquer ou consommer des données qui interfèrent.

## Problème 4 : Le mutex bloque pendant toute l'exécution

**Ligne 163-164 dans executor.go :**
```go
e.shellMutex.Lock()
defer e.shellMutex.Unlock()
```

Le mutex est verrouillé pendant toute l'exécution de la commande, y compris la lecture de la sortie. Si plusieurs commandes arrivent rapidement, elles attendent, mais cela ne devrait pas affecter la persistance du shell.

## Problème 5 : Le répertoire de travail du processus bash

Quand bash démarre, il hérite du répertoire de travail du processus parent. Si l'agent s'exécute depuis `/opt/remoteshell`, bash démarre aussi depuis là. Le `cd /home` dans le shell change le répertoire courant du shell, mais le répertoire de travail du processus bash reste `/opt/remoteshell`.

## Solution probable

Le problème principal est que **bash en mode non-interactif peut réinitialiser le répertoire de travail** ou que le shell n'est pas vraiment "persistant" dans le sens où chaque commande est traitée de manière isolée.

**Solution :** Utiliser `bash -i` (mode interactif) ou s'assurer que le répertoire de travail est explicitement défini avant chaque commande si nécessaire.

