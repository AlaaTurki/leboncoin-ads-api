# Way of working

This repository is read-only, click on "Use this template" => "Create a new repository" and work on your copy just as you would on a live one. \
Please make sure the first commit on your repo is the fresh copy of the original repository, please start your work from the second commit only. \
Once you are satisfied with your work, please send your public repository url to the hiring manager.

# Hiring instructions

Please refer to [these instructions](INSTRUCTIONS.md).

# Ads API

A small Go HTTP API around a single domain: *ads* (classified ads).

The API needs an up-and-running postgres database, which can be launched via the provided Docker Compose file.

## Requirements

- Go 1.26 or later
- Docker with the Compose plugin

## Launching the stack

```bash
docker compose up -d
```

This starts PostgreSQL with the schema and seed data, the identity service, and the `ads` API.

## Model

There are 2 entities in this sample:
- ad (title, price, etc...)
- user (id and nickname only)

## API

| Method | Path          | Auth | Description                                                                       |
|--------|---------------|------|-----------------------------------------------------------------------------------|
| `POST` | `/ads`        | yes | Create an ad, subject to the rolling quota |
| `GET` | `/ads/{id}`   | no | Get an ad and its owner's nickname |
| `GET` | `/ads`        | no | List ads (filterable by `title`, `user_id`, `min_price`, `max_price`) |
| `POST` | `/ads/import` | yes | Atomically import ads (requires an idempotency key) |
| `DELETE` | `/ads/{id}` | yes | Delete an ad owned by the caller |

```bash
# Health check
curl -s -v http://localhost:8080/health

# List ads (with optional filters)
curl -s 'http://localhost:8080/ads?title=bike&min_price=10.00'

# Get a single ad
curl -s http://localhost:8080/ads/1

# Create an ad (price is in currency units; requires Authorization header)
curl -s -X POST http://localhost:8080/ads \
  -H 'Authorization: 00000000-0000-0000-0000-000000000001' \
  -H 'Content-Type: application/json' \
  -d '{"title":"Mountain bike","price":120.00,"photo_url":"https://example.com/photos/bike.jpg"}'
```

# Import ads with stable retry behavior (requires a unique key per batch)
```bash
curl -s -X POST http://localhost:8080/ads/import \
  -H 'Authorization: 00000000-0000-0000-0000-000000000001' \
  -H 'Idempotency-Key: import-2026-11-16-001' \
  -H 'Content-Type: application/json' \
  -d '{"ads":[{"title":"Mountain bike","price":120.00,"photo_url":"https://example.com/photos/bike.jpg"}]}'
```

## Authentication

To pass authentication information to a call, simply add the `Authorization` header with the user ID in plain text.

It will be extracted by the service.

```
Authorization: 00000000-0000-0000-0000-000000000001
```

## Tests

To launch all the tests:

```shell
go test ./...
```

## Design notes

- Prices remain stored as integer cents in PostgreSQL. The API accepts and returns `price` as a decimal number; requests with more than two decimal places are rejected. This avoids a database migration while preserving exact stored amounts. Renaming the public field is a breaking API change; a production rollout should publish it in a new API version or provide a deprecation window before removing the old field.
- A single ad creation checks the rolling 24-hour quota under a per-owner PostgreSQL advisory transaction lock, so concurrent requests cannot exceed five; deleting an ad does not reset this counter. Publication events are recorded separately from ads. Bulk import is atomic: either the whole batch commits or none of it does.
- Bulk import requires an `Idempotency-Key`. Ads and the response IDs are stored atomically with a payload hash. Reusing the same key and payload returns the original IDs; reusing it with a different payload is rejected.
- `DELETE /ads/{id}` requires authentication and only deletes the caller's ad.
