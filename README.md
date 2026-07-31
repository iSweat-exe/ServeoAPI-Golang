# ServeoAPI V2

API Go qui pilote le panneau Serveo : gestion Docker (conteneurs, images, réseaux, volumes,
terminal interactif), fichiers et sauvegardes, métriques système, comptes utilisateurs avec
RBAC, clés d'API, OVHcloud et un serveur MCP pour les agents.

Stack : `net/http` (routeur standard Go 1.22+), GORM + SQLite, JWT, WebSocket, Prometheus, Swagger.

## Démarrage rapide

```bash
# Dépendances
go mod download

# Variables minimales (voir plus bas pour la liste complète)
export JWT_SECRET="$(openssl rand -hex 32)"
export ADMIN_PASSWORD="un-mot-de-passe-fort"

go run ./cmd/api
```

L'API écoute sur `http://localhost:8080`. Au premier démarrage, un compte `admin` est créé.
Si `ADMIN_PASSWORD` n'est pas défini, un mot de passe aléatoire est généré et affiché **une
seule fois** dans les logs.

## Configuration

Toute la configuration passe par des variables d'environnement.

| Variable | Défaut | Description |
| --- | --- | --- |
| `ENV` | `development` | `production` active les contrôles stricts (voir Sécurité) |
| `PORT` | `8080` | Port d'écoute HTTP |
| `JWT_SECRET` | secret de dev | **Obligatoire en production**, sinon le serveur refuse de démarrer |
| `ADMIN_PASSWORD` | aléatoire | Mot de passe du compte `admin` créé à la première migration |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | Origines CORS autorisées, séparées par des virgules (comparaison stricte) |
| `TRUST_PROXY` | `false` | Autorise la lecture de `X-Forwarded-For` pour identifier le client |
| `API_RATE_LIMIT` | `10` | Requêtes par seconde et par IP (burst = 2x) |
| `SQLITE_PATH` | `serveo.db` | Chemin du fichier SQLite |
| `ALLOWED_MOUNT_ROOT` | `/var/serveoapi/data/` | Racine autorisée pour les bind mounts Docker |
| `TEMPLATES_PATH` | `./data/templates` | Répertoire des templates de déploiement |
| `OVH_ENDPOINT`, `OVH_APP_KEY`, `OVH_APP_SECRET`, `OVH_CONSUMER_KEY` | vide | Identifiants OVHcloud |

## Sécurité

- **JWT** : chaque requête authentifiée porte un `Authorization: Bearer <token>`. Le claim
  `token_version` est comparé à celui stocké en base : un logout, un changement de mot de passe
  ou de permissions invalide immédiatement les jetons émis auparavant.
- **RBAC** : les permissions sont une liste séparée par des virgules (`docker.containers.read`),
  avec `*` comme joker global et `prefixe.*` comme joker de portée.
- **CORS** : comparaison exacte avec `ALLOWED_ORIGINS`, en-tête `Vary: Origin` systématique.
- **WebSocket** : le terminal exige un ticket à usage unique de 30 s obtenu via
  `POST /v2/auth/ticket`, et vérifie l'origine avec la même liste que CORS.
- **Bind mounts** : les chemins sont normalisés avant vérification, ce qui bloque les `../`.
- **En production** (`ENV=production`) : `JWT_SECRET` est obligatoire et `/swagger/` comme
  `/LLMs/` passent derrière l'authentification.
- **Mises à jour** : `serveoapi update` vérifie la signature minisign de la release avant de
  remplacer le binaire.

## Releases (signature minisign)

Le workflow `.github/workflows/release.yml` (tags `v*`) signe chaque binaire. Sans les
secrets suivants, le job **Sign Binary** échoue :

| Secret GitHub | Contenu |
| --- | --- |
| `MINISIGN_PRIVATE_KEY` | Contenu de `secrets/minisign.key` (fichier complet **ou** ligne base64 seule) |
| `MINISIGN_PUBLIC_KEY` | Clé publique **sur une seule ligne** (préfixe `RW…`) |

Génération locale (une seule fois) :

```bash
go run ./scripts/generate-minisign-keys
```

Puis collez les valeurs affichées dans
Settings → Secrets and variables → Actions du dépôt. Ne committez jamais `secrets/`.

## Structure

```
cmd/api/                  Point d'entrée : config, migrations, workers, serveur HTTP
internal/core/            Briques transverses
  config/                 Chargement et validation de la configuration
  contextkeys/            Clés de contexte typées (user ID, permissions)
  database/               Connexion SQLite, pool et PRAGMA
  middleware/             JWT, RBAC, CORS, rate limit, logger, métriques, recover
  response/               Helpers de réponse JSON uniformes
  server/                 Démarrage et arrêt gracieux
  stream/, updater/, validation/
internal/modules/v2/      Un package par domaine métier
  auth/ users/ apikeys/   Authentification, comptes, clés d'API
  docker/ files/ backups/ Docker, système de fichiers conteneur, sauvegardes
  metrics/ system/ health/ Observabilité
  ovh/ templates/ mcp/ metadata/
internal/router/          Assemblage des routes et de la chaîne de middlewares
internal/testutil/        Helpers de test (requêtes authentifiées)
docs/                     Spécification Swagger générée
LLMs/                     Documentation destinée aux agents
```

## Conventions

- Les réponses passent par `internal/core/response` (`SendJSON` / `SendError`), jamais par
  `http.Error` ou `json.NewEncoder` directement.
- Chaque module expose `RegisterRoutes(mux, authMiddleware, db, ...)` et déclare lui-même les
  permissions requises via `middleware.RequirePermission`.
- Les erreurs des opérations base de données sont toujours vérifiées et loguées via `slog`.

## Endpoints principaux

| Route | Description |
| --- | --- |
| `POST /v2/auth/login` | Connexion, retourne un JWT |
| `POST /v2/auth/logout` | Invalide les jetons de l'utilisateur |
| `POST /v2/auth/ticket` | Ticket court pour ouvrir un WebSocket |
| `GET /v2/docker/containers` | Liste des conteneurs |
| `GET /v2/docker/containers/{id}/exec` | Terminal interactif (WebSocket) |
| `GET /v2/docker/images` | Liste des images |
| `GET /v2/system/` | Informations système |
| `GET /v2/metrics/history/system` | Historique des métriques |
| `GET /v2/users` | Gestion des comptes (permission `users.manage`) |
| `PUT /v2/users/{id}/password` | Réinitialisation de mot de passe par un administrateur |
| `GET /health` | Sonde de santé (API, base, Docker) |
| `GET /prometheus` | Métriques Prometheus (authentifié) |
| `GET /swagger/` | Documentation interactive (authentifiée en production) |

La liste exhaustive est disponible dans Swagger.

## Développement

```bash
go test ./...            # Tests unitaires
go vet ./...             # Analyse statique
gofmt -l .               # Vérification du formatage
swag init -g cmd/api/main.go -o docs   # Régénérer la documentation Swagger
```

## Compilation et déploiement

```bash
# Binaire Linux depuis n'importe quelle plateforme
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/serveoapi ./cmd/api

# Image Docker
docker build -t serveoapi:latest .
docker run -d -p 8080:8080 \
  -e JWT_SECRET="$(openssl rand -hex 32)" \
  -e ALLOWED_ORIGINS="https://panel.example.com" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --name serveoapi serveoapi:latest
```

Le conteneur a besoin de l'accès au socket Docker pour piloter les conteneurs de l'hôte.
