# Leboncoin Ads API

A Go HTTP API for classified ads, built for the Leboncoin backend hiring assessment. The implementation changes are limited to `/ads` and `/common`; the provided `/users` code remains untouched.

See [INSTRUCTIONS.md](INSTRUCTIONS.md) for the assessment requirements.

## Requirements

- Docker Desktop with Docker Compose
- Go 1.26 or later to run the test suite locally

## Run locally on Windows

Copy `.env.example` to `.env` in the repository root, open PowerShell in the repository directory, and make sure Docker Desktop is running. The `.env` file contains local Docker settings and is ignored by Git. Its `ADS_`-prefixed variables avoid collisions with environment variables from other projects. Start the API, PostgreSQL, and the identity service:

```powershell
Copy-Item .env.example .env
docker compose up --build -d
docker compose ps
```

Wait for the services to become healthy, then check the API:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

The database is initialized with sample ads and users. Fetch a seeded ad (the response includes its owner's username and the decimal `price` field):

```powershell
Invoke-RestMethod http://localhost:8080/ads/1
```

List ads with optional filters:

```powershell
Invoke-RestMethod 'http://localhost:8080/ads?title=bike&min_price=10.00'
```

Create an ad. The `Authorization` value must be a user ID known to the identity service:

```powershell
$headers = @{ Authorization = "00000000-0000-0000-0000-000000000001" }
$body = @{
  title = "Mountain bike"
  price = 120.50
  photo_url = "https://example.com/bike.jpg"
} | ConvertTo-Json

$created = Invoke-RestMethod -Method Post `
  -Uri http://localhost:8080/ads `
  -Headers $headers `
  -ContentType "application/json" `
  -Body $body
$created
```

Delete that ad as its owner:

```powershell
Invoke-WebRequest -Method Delete `
  -Uri "http://localhost:8080/ads/$($created.id)" `
  -Headers $headers
```

Bulk import is atomic and requires a unique `Idempotency-Key` for each batch. Retry the same payload with the same key to receive the original IDs:

```powershell
$headers = @{
  Authorization = "00000000-0000-0000-0000-000000000001"
  "Idempotency-Key" = "import-batch-001"
}
$body = @{
  ads = @(
    @{ title = "Desk"; price = 45.00; photo_url = "https://example.com/desk.jpg" }
  )
} | ConvertTo-Json -Depth 5

Invoke-RestMethod -Method Post `
  -Uri http://localhost:8080/ads/import `
  -Headers $headers `
  -ContentType "application/json" `
  -Body $body
```

Stop the services while retaining database data:

```powershell
docker compose down
```

### Existing database volume

PostgreSQL runs initialization SQL only when its data volume is first created. If you started this project before migrations `003`, `004`, or `005` were added, apply each missing migration once:

```powershell
docker compose exec -T ads-database psql -U ads -d ads -f /docker-entrypoint-initdb.d/003_create_ad_import_requests.sql
docker compose exec -T ads-database psql -U ads -d ads -f /docker-entrypoint-initdb.d/004_create_ad_creation_events.sql
docker compose exec -T ads-database psql -U ads -d ads -f /docker-entrypoint-initdb.d/005_add_ads_read_indexes.sql
```

## Postman collection

There is no web frontend in this project; it exposes an HTTP API. Import [the Postman collection](postman/leboncoin-ads-api.postman_collection.json) to try the local endpoints. Start the Docker Compose stack first, then use the collection variables `baseUrl`, `userId`, `adId`, and `importKey`. Run **Create ad** before **Get created ad** or **Delete created ad**; the create request saves its ID automatically. The delete request removes data. For a different import payload, change `importKey` first.

## API

| Method | Path | Auth | Description |
|---|---|---:|---|
| `GET` | `/health` | No | Health check |
| `GET` | `/ads` | No | List ads; filters plus `limit` (default 50, max 100) and `offset` pagination |
| `GET` | `/ads/{id}` | No | Get an ad and its owner's username |
| `POST` | `/ads` | Yes | Create an ad; maximum five publications per rolling 24 hours |
| `POST` | `/ads/import` | Yes | Atomically import ads using an `Idempotency-Key` header |
| `DELETE` | `/ads/{id}` | Yes | Delete an ad owned by the caller |

## Tests

```powershell
go test ./...
```

## Design notes

- Prices remain stored as integer cents in PostgreSQL. The API accepts and returns `price` as a decimal number; requests with more than two decimal places are rejected. Renaming the public field is a breaking API change, so a production rollout should version the API or provide a deprecation window.
- A single-ad creation checks the rolling 24-hour quota under a per-owner PostgreSQL advisory transaction lock. Publication events are stored separately from ads, so deleting an ad does not reset the quota. Bulk imports bypass this quota.
- Bulk imports and their response IDs commit together. A stored payload hash lets identical retries replay the original IDs and rejects reuse of a key with a different payload.
- Ad reads use a short-lived username cache to limit calls to the identity service.

# API notes

`GET /ads` supports `limit` (default 50, maximum 100) and `offset` (default 0). Results are ordered newest first. Database indexes cover owner/date, price, and date ordering. Title substring search uses `ILIKE` and may need a trigram index if production query metrics show it is a bottleneck.

Ad titles must contain 1–200 characters and `photo_url` must be an absolute HTTP(S) URL no longer than 2,048 characters. Imports accept at most 500 ads and 2 MiB per request. A quota response (`429`) includes `Retry-After` in seconds; the value is the configured 24-hour quota window.

Prices use a JSON number to preserve the assessment API contract. The HTTP layer validates at most two decimal places and converts the value to integer cents immediately; persistence and calculations use cents, avoiding floating-point money arithmetic. A string decimal is an exact alternative for a new/versioned API, but changing the current field would break clients expecting a JSON number.
