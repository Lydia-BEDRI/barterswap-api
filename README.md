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

## Système de crédits

Le solde disponible est stocké sur l'utilisateur et chaque mouvement est
tracé dans `credit_transactions`, conformément aux types du sujet :

- `spend` (montant négatif) : blocage des crédits lors de l'acceptation ;
- `earn` (montant positif) : transfert à l'offreur lorsque l'échange est terminé ;
- `refund` (montant positif) : restitution au demandeur lors d'une annulation.

Le changement de statut, la mise à jour du solde et l'écriture du journal sont
effectués dans une même transaction SQL. Le verrouillage de la ligne utilisateur
empêche deux acceptations concurrentes de dépenser les mêmes crédits.

## Lancer avec Docker

Le fichier `docker-compose.yml` crée 2 services :
- **api** : serveur Go sur le port défini par `APP_PORT`
- **db** : MySQL 8.4 sur port 3306

Les fichiers SQL sont automatiquement exécutés :
- `initdb.d/01-schema.sql` : structure tables
- `initdb.d/02-seed.sql` : données de test (5 users, 10 services)

```bash
docker compose up --build
```

Test rapide :
```bash
curl http://localhost:15001/healthz
```

Pour une base existante créée avant le système de crédits, appliquer une seule
fois `migrations/001_credit_system.sql`. Une base Docker neuve utilise déjà le
schéma à jour présent dans `initdb.d/01-schema.sql`.

## Tests

Les tests unitaires ne nécessitent pas de base de données :

```bash
go test -v -cover ./...
```

Les tests d'intégration couvrent le blocage, le transfert, le remboursement et
deux acceptations concurrentes. Ils s'activent avec une base MySQL de test :

```powershell
$env:TEST_DB_DSN = "barter:barter@tcp(127.0.0.1:3307)/barterswap?parseTime=true&charset=utf8mb4"
go test -v -run Integration ./...
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
