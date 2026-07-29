# ServeoAPI 🚀

API développée en Go (Golang), conçue avec une architecture propre et prête pour le déploiement sous Linux.

## 📁 Structure du projet

```
ServeoAPI V2/
├── cmd/
│   └── api/
│       └── main.go       # Point d'entrée de l'application
├── internal/
│   ├── config/           # Gestion des configurations d'environnement
│   ├── handler/          # Handlers HTTP (Health, Ping, etc.)
│   └── router/           # Configuration des routes HTTP
├── .gitignore            # Fichiers ignorés par Git
├── Dockerfile            # Multi-stage build pour conteneur Linux
├── Makefile              # Raccourcis de compilation et exécution
├── go.mod                # Dépendances et nom du module Go
└── README.md             # Documentation
```

## 🛠️ Utilisation en développement

### Exécuter l'API localement
```bash
go run ./cmd/api
```
L'API sera accessible sur `http://localhost:8080`.

### Endpoints disponibles
- `GET /health` : Vérification de santé et informations système (OS, Arch, version Go)
- `GET /api/v1/ping` : Endpoint de test rapide (retourne `{"message": "pong"}`)

### Lancer les tests unitaires
```bash
go test -v ./...
```

---

## 🐧 Compilation pour Linux (Cross-compilation)

Pour générer un binaire exécutable nativement sous Linux depuis Windows :

### Via PowerShell (Windows)
```powershell
$env:CGO_ENABLED="0"
$env:GOOS="linux"
$env:GOARCH="amd64"
go build -ldflags="-w -s" -o bin/serveoapi-linux ./cmd/api
```

### Via Make (si `make` est installé)
```bash
make build-linux
```

---

## 🐳 Déploiement avec Docker (Linux Container)

### Construire l'image Docker
```bash
docker build -t serveoapi:latest .
```

### Lancer le conteneur Linux
```bash
docker run -d -p 8080:8080 --name serveoapi-app serveoapi:latest
```
