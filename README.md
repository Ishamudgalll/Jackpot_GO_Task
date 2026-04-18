# Admin Statistics API (Go + MongoDB)

This is a simple Go backend project that reads casino transaction data from MongoDB and gives admin stats.

If you know Java, think of this as:
- `cmd/api/main.go` = your `main` class + app bootstrap
- `internal/service` = service layer
- `internal/http/handlers` = controller layer
- `internal/store` = data access setup

## What this project does

- Checks `Authorization` header for all API calls.
- Validates `from` and `to` query params.
- Runs Mongo aggregation queries for stats.
- Caches API responses (in-memory by default, Redis optional).
- Seeds a very large dataset (2,000,000+ rounds).

## Main files

- `cmd/api/main.go` - starts the HTTP server.
- `cmd/seed/main.go` - generates random transactions and inserts in Mongo.
- `internal/service/stats_service.go` - all aggregation logic.
- `internal/http/handlers/stats_handler.go` - endpoint handlers.
- `internal/http/middleware/auth.go` - auth check.
- `internal/cache/cache.go` - in-memory cache.
- `internal/cache/redis.go` - Redis cache (optional).
- `scripts/seed.sh` - quick seed script.
- `postman/admin-stats.postman_collection.json` - ready Postman collection.

## Prerequisites

- Go 1.22+
- Docker (for MongoDB, and Redis if you want Redis cache)

## Quick start

1) Start local services:

```bash
docker compose up -d
```

2) Create local config file:

```bash
cp .env.example .env
```

You can edit `.env` if you want different values (for example, change port).

3) Download Go dependencies:

```bash
go mod tidy
```

4) Seed sample data (default is 2,000,000 rounds):

```bash
chmod +x scripts/seed.sh
./scripts/seed.sh
```

5) Run the API:

```bash
chmod +x scripts/run-api.sh
./scripts/run-api.sh
```

6) Check health:

```bash
curl -s -i http://localhost:8081/healthz \
  -H "Authorization: Bearer secret-admin-token"
```

7) Run tests:

```bash
go test ./...
```

## API routes

All routes need:
- Header: `Authorization: Bearer secret-admin-token`
- Query: `from` and `to` (`RFC3339` or `YYYY-MM-DD`)

### `GET /gross_gaming_rev`

```bash
curl -s "http://localhost:8080/gross_gaming_rev?from=2026-01-01&to=2026-03-01" \
  -H "Authorization: Bearer secret-admin-token" | jq
```

### `GET /daily_wager_volume`

```bash
curl -s "http://localhost:8080/daily_wager_volume?from=2026-01-01&to=2026-01-10" \
  -H "Authorization: Bearer secret-admin-token" | jq
```

### `GET /user/{user_id}/wager_percentile`

```bash
curl -s "http://localhost:8080/user/65d7665fcb31ee98fb584e4a/wager_percentile?from=2026-01-01&to=2026-03-01" \
  -H "Authorization: Bearer secret-admin-token" | jq
```

## Redis cache (optional)

By default, the app uses in-memory cache.

To use Redis cache, set this in `.env`:

```bash
USE_REDIS_CACHE=true
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

If Redis is not reachable, the app logs a warning and falls back to in-memory cache.

## Postman

Import this file:
- `postman/admin-stats.postman_collection.json`

Update variables (`baseUrl`, `token`, `from`, `to`, `user_id`) and run requests.

## Notes

- Transactions are stored with decimal values; API returns decimal values as strings to keep precision.
- Indexes are created automatically on startup for faster queries.
- App and scripts auto-load `.env` when present.

# Jackpot_GO_Task
Develop a GoLang REST API that demonstrates proficiency in modern backend development and advanced MongoDB queries. Create an admin API that aggregates useful data from a large MongoDB collection of user transactions.
