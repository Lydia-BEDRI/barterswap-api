# BarterSwap - API d'échange de compétences

BarterSwap est une API REST qui permet à des particuliers d'échanger des
compétences sans argent. Le temps est représenté par des crédits : rendre un
service permet de recevoir des crédits, puis de les utiliser pour demander un
autre service.

Le projet utilise uniquement Go, la bibliothèque standard pour HTTP, JSON,
tests et context, ainsi que `database/sql` avec le driver MySQL.

## Installation

```bash
git clone https://github.com/Lydia-BEDRI/barterswap-api.git
cd barterswap-api
go mod tidy
```

Copier le fichier d'environnement :

```bash
cp .env.example .env
```

Variables principales :

| Variable | Description |
| --- | --- |
| `APP_PORT` | Port HTTP de l'API, par défaut `8080` |
| `DB_HOST` | Hôte MySQL utilisé par l'API |
| `DB_PORT` | Port MySQL exposé sur la machine |
| `DB_USER` | Utilisateur MySQL |
| `DB_PASSWORD` | Mot de passe MySQL |
| `DB_NAME` | Base de données |

## Lancement

Avec Docker :

```bash
docker compose up --build
```

L'API écoute sur le port défini par `APP_PORT`.

```bash
curl http://localhost:8080/healthz
```

Sans Docker :

```bash
mysql -u root -p < initdb.d/01-schema.sql
mysql -u root -p barterswap < initdb.d/02-seed.sql
go run .
```

`DB_DSN` doit être défini si l'application est lancée sans Docker :

```bash
export DB_DSN='barter:barter@tcp(127.0.0.1:3306)/barterswap?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci'
go run .
```

## Authentification

L'authentification est volontairement simple. Les routes protégées utilisent le
header suivant :

```text
X-UserID: 1
```

Il n'y a pas de JWT, session, mot de passe ou système avancé.

## Endpoints

### Santé

| Méthode | Route | Description | Auth |
| --- | --- | --- | --- |
| `GET` | `/healthz` | Vérifier que l'API répond | Non |

### Utilisateurs

| Méthode | Route | Description | Auth |
| --- | --- | --- | --- |
| `POST` | `/api/users` | Créer un utilisateur avec 10 crédits de bienvenue | Non |
| `GET` | `/api/users/{id}` | Lire le profil public d'un utilisateur | Non |
| `PUT` | `/api/users/{id}` | Modifier son profil | Oui |
| `GET` | `/api/users/{id}/skills` | Lister les compétences d'un utilisateur | Non |
| `PUT` | `/api/users/{id}/skills` | Remplacer toutes ses compétences | Oui |
| `GET` | `/api/users/{id}/stats` | Lire les statistiques d'un utilisateur | Non |
| `GET` | `/api/users/{id}/reviews` | Lister les avis reçus par un utilisateur | Non |

### Services

| Méthode | Route | Description | Auth |
| --- | --- | --- | --- |
| `GET` | `/api/services` | Lister les services actifs | Non |
| `POST` | `/api/services` | Créer une annonce de service | Oui |
| `GET` | `/api/services/{id}` | Lire le détail d'un service | Non |
| `PUT` | `/api/services/{id}` | Modifier son annonce | Oui |
| `DELETE` | `/api/services/{id}` | Désactiver son annonce | Oui |
| `GET` | `/api/services/{id}/reviews` | Lister les avis liés à un service | Non |

Filtres disponibles sur `GET /api/services` :

| Paramètre | Exemple |
| --- | --- |
| `categorie` | `/api/services?categorie=Informatique` |
| `ville` | `/api/services?ville=Paris` |
| `search` | `/api/services?search=photo` |

Catégories autorisées : `Informatique`, `Jardinage`, `Bricolage`, `Cuisine`,
`Musique`, `Langues`, `Sport`, `Tutorat`, `Déménagement`, `Photographie`,
`Animalier`, `Couture`, `Autre`.

### Échanges

| Méthode | Route | Description | Auth |
| --- | --- | --- | --- |
| `POST` | `/api/exchanges` | Créer une demande d'échange | Oui |
| `GET` | `/api/exchanges` | Lister les échanges de l'utilisateur connecté | Oui |
| `GET` | `/api/exchanges/{id}` | Lire un échange dont on est participant | Oui |
| `PUT` | `/api/exchanges/{id}/accept` | Accepter une demande | Oui |
| `PUT` | `/api/exchanges/{id}/reject` | Refuser une demande | Oui |
| `PUT` | `/api/exchanges/{id}/complete` | Marquer un échange comme terminé | Oui |
| `PUT` | `/api/exchanges/{id}/cancel` | Annuler une demande ou un échange accepté | Oui |

Filtre disponible :

| Paramètre | Exemple |
| --- | --- |
| `status` | `/api/exchanges?status=pending` |

Statuts : `pending`, `accepted`, `rejected`, `cancelled`, `completed`.

Règles principales :

- un utilisateur ne peut pas demander son propre service ;
- un service ne peut avoir qu'un échange `pending` ou `accepted` à la fois ;
- à l'acceptation, les crédits sont bloqués chez le demandeur ;
- à la completion, les crédits sont transférés au prestataire ;
- en cas de refus ou annulation, les crédits bloqués sont remboursés.

### Avis

| Méthode | Route | Description | Auth |
| --- | --- | --- | --- |
| `POST` | `/api/exchanges/{id}/review` | Noter l'autre participant d'un échange terminé | Oui |
| `GET` | `/api/users/{id}/reviews` | Lister les avis reçus par un utilisateur | Non |
| `GET` | `/api/services/{id}/reviews` | Lister les avis liés à un service | Non |

La note doit être comprise entre 1 et 5. Chaque participant ne peut publier
qu'un seul avis par échange.

## Exemples curl

Créer un utilisateur :

```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"pseudo":"lina","bio":"Développeuse Go","ville":"Paris"}'
```

Définir ses compétences :

```bash
curl -X PUT http://localhost:8080/api/users/1/skills \
  -H "X-UserID: 1" \
  -H "Content-Type: application/json" \
  -d '[{"nom":"Go","niveau":"intermédiaire"},{"nom":"Cuisine","niveau":"débutant"}]'
```

Créer un service :

```bash
curl -X POST http://localhost:8080/api/services \
  -H "X-UserID: 1" \
  -H "Content-Type: application/json" \
  -d '{"titre":"Aide Go","description":"Bases du langage","categorie":"Informatique","duree_minutes":90,"credits":4,"ville":"Paris"}'
```

Filtrer les services :

```bash
curl "http://localhost:8080/api/services?categorie=Informatique&ville=Paris"
```

Créer et accepter un échange :

```bash
curl -X POST http://localhost:8080/api/exchanges \
  -H "X-UserID: 2" \
  -H "Content-Type: application/json" \
  -d '{"service_id":1}'

curl -X PUT http://localhost:8080/api/exchanges/1/accept \
  -H "X-UserID: 1"
```

Terminer un échange puis ajouter un avis :

```bash
curl -X PUT http://localhost:8080/api/exchanges/1/complete \
  -H "X-UserID: 2"

curl -X POST http://localhost:8080/api/exchanges/1/review \
  -H "X-UserID: 2" \
  -H "Content-Type: application/json" \
  -d '{"note":5,"commentaire":"Très bon échange"}'
```

Lire les statistiques :

```bash
curl http://localhost:8080/api/users/1/stats
```

Exemple d'erreur d'authentification :

```bash
curl -X POST http://localhost:8080/api/services \
  -H "Content-Type: application/json" \
  -d '{"titre":"Aide Go","categorie":"Informatique","duree_minutes":60,"credits":2}'
```

Réponse attendue :

```json
{"error":"forbidden"}
```

## Tests

Tests unitaires et tests API sans base :

```bash
go test -v ./...
go test -v -cover ./...
go vet ./...
```

Tests d'intégration avec MySQL Docker :

```bash
docker compose up --build -d
TEST_DB_DSN='barter:barter@tcp(127.0.0.1:3306)/barterswap?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci' go test -v ./...
TEST_DB_DSN='barter:barter@tcp(127.0.0.1:3306)/barterswap?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci' go test -v -cover ./...
```

Les tests couvrent notamment :

- validations métier table-driven ;
- handlers HTTP avec `httptest` ;
- cycle de vie des crédits ;
- création, acceptation, completion et annulation d'échanges ;
- avis et statistiques utilisateur.

## Architecture

Le code reste en `package main`, avec une séparation des responsabilités par
fichiers :

| Fichier | Role |
| --- | --- |
| `main.go` | Démarrage, connexion DB, serveur HTTP |
| `models.go` | Structures JSON principales |
| `http.go` | Router et helpers HTTP communs |
| `middleware.go` | Logging, CORS, auth `X-UserID`, recovery |
| `*_http.go` | Handlers HTTP par domaine |
| `*_store.go` | Accès SQL et logique métier par domaine |
| `*_test.go` | Tests unitaires, API et intégration |

Les handlers HTTP restent fins : ils lisent la requête, appellent le store et
écrivent la réponse. Les validations, transitions de statuts et mouvements de
crédits sont isolés dans les fonctions métier et SQL.

## Base de données

Le schéma est initialisé par :

```text
initdb.d/01-schema.sql
initdb.d/02-seed.sql
```

Tables principales :

- `users`
- `user_skills`
- `services`
- `exchanges`
- `credit_transactions`
- `reviews`

Requête rapide :

```bash
docker exec barterswap-api-db-1 mysql -u barter -pbarter barterswap -e "SELECT id, pseudo, credit_balance FROM users;"
```
