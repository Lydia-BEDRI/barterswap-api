# BarterSwap API

**BarterSwap** est une plateforme qui permet à des particuliers d'échanger leurs compétences sans transaction monétaire. Le système fonctionne avec un **crédit-temps** : chaque heure de service rendue donne droit à une heure de service reçue.

## Lancer avec Docker
Le fichier `.env.example` fournit `UID` et `GID` pour eviter les soucis de permissions.
```bash
docker compose up --build
```

Test rapide :
```bash
curl http://localhost:8080/healthz
```

## Lancer sans Docker
1. Avoir MySQL disponible en local.
2. Exporter `DB_DSN` puis lancer l'app :

```bash
export DB_DSN="barter:barter@tcp(127.0.0.1:3306)/barterswap?parseTime=true"
go mod tidy
go run .
```