# BarterSwap API

**BarterSwap** est une plateforme qui permet à des particuliers d'échanger leurs compétences sans transaction monétaire. Le système fonctionne avec un **crédit-temps** : chaque heure de service rendue donne droit à une heure de service reçue.

## Configuration

### Fichier .env
Copier `.env.example` en `.env` pour configurer :
```bash
cp .env.example .env
```

Variables disponibles :
- `APP_ENV` : `development` ou `production`
- `APP_PORT` : port API (défaut 8080)
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` : paramètres MySQL
- `LOG_LEVEL` : niveau de log

## Lancer avec Docker

Le fichier `docker-compose.yml` crée 2 services :
- **api** : serveur Go sur port 8080
- **db** : MySQL 8.4 sur port 3306

Les fichiers SQL sont automatiquement exécutés :
- `initdb.d/01-schema.sql` : structure tables
- `initdb.d/02-seed.sql` : données de test (5 users, 10 services)

```bash
docker compose up --build
```

Test rapide :
```bash
curl http://localhost:8080/healthz
```

## Vérifier les données en base

Se connecter à MySQL depuis Docker :
```bash
docker exec barterswap-api-db-1 mysql -u barter -pbarter barterswap -e "SELECT * FROM users LIMIT 3;"
```

Ou avec une session interactive :
```bash
docker exec -it barterswap-api-db-1 mysql -u barter -pbarter barterswap
```

Requêtes utiles :
```sql
-- Voir tous les utilisateurs
SELECT id, pseudo, credit_balance, created_at FROM users;

-- Voir les services actifs
SELECT id, titre, provider_id, credits, ville FROM services WHERE actif = true;

-- Voir les échanges
SELECT id, service_id, requester_id, owner_id, status FROM exchanges;

-- Voir les critiques
SELECT id, exchange_id, author_id, note, commentaire FROM reviews;
```

## Lancer sans Docker

1. Installer MySQL 8.4 localement
2. Créer la base et les tables :
```bash
mysql -u root -p < initdb.d/01-schema.sql
mysql -u root -p barterswap < initdb.d/02-seed.sql
```

3. Configurer `.env` avec accès local
4. Lancer l'app :
```bash
go mod tidy
go run .
```