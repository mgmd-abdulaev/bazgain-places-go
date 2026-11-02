# Bazgain Places Go migration candidate

This repository is a standalone Go implementation of five Bazgain Tourism Places endpoints. It is a **migration candidate** for contract-parity and operational evaluation. It has **not been deployed to staging or production**, and this repository contains no production metrics or deployment evidence.

Implemented domain endpoints:

- `GET /v1/place-collections`
- `GET /v1/places`
- `GET /v1/places/{placeSlug}`
- `POST /v1/admin/places/{id}/publish`
- `POST /v1/admin/places/{id}/archive`

The service also exposes `GET /v1/health/live` and `GET /v1/health/ready` for local orchestration.

## What this demonstrates

- Go 1.26.10 with `net/http` routing and explicit dependency wiring
- PostgreSQL 16, PostGIS 3.4, `pgxpool`, and normalized relation loading
- Redis response caching, ETags/conditional GET, distributed rate limits, and session-revocation checks
- HS256 access-token verification plus current user state, role, and token-version checks
- row-locked lifecycle transitions with atomic audit events and UUIDv7 audit identifiers
- locked `snake_case` JSON and `{ "error": { ... } }` error envelopes
- local Docker Compose, derived OpenAPI, CI, tests, and rollout/rollback design

## Run locally

Requirements: Docker with Compose, or Go 1.26.10 plus local PostgreSQL/PostGIS and Redis.

```bash
docker compose up --build
curl -i http://localhost:8080/v1/place-collections
curl -i http://localhost:8080/v1/places
curl -i http://localhost:8080/v1/places/sulakskiy-kanon
```

The Compose database is initialized from `db/local`. Those SQL files are a **local compatibility baseline**, not a production migration set. The seed uses synthetic data only.

For a host-run service:

```bash
cp .env.example .env
set -a; source .env; set +a
go run ./cmd/api
```

The example URLs in `.env.example` target the dependency ports published by Docker Compose (`5433`/`6380`).

## Verification

```bash
make fmt
make vet
make test
make race
make build
docker compose config --quiet
```

CI repeats formatting, vet, build, tests, race tests, and Compose configuration validation. Contract provenance and known parity limits are documented under `api/` and `docs/`.
The required CI integration job starts real PostGIS and Redis containers; dependency vulnerability scanning and a container build are separate required jobs.

## Runtime behavior

Public reads are cache-aside. PostgreSQL supplies a timestamp version that is included in the Redis key and ETag. A matching `If-None-Match` receives `304` with no JSON body. Redis cache reads and writes fail open; PostgreSQL remains authoritative. Redis failures in distributed rate limiting or admin session validation fail closed as `500 internal_error`.

Admin mutations require an HS256 access JWT and a current active admin row. Publish and archive lock the target place, update it, and write the audit event in one PostgreSQL transaction. Publish rejects a place without gallery media or a navigation point.

See [architecture](docs/ARCHITECTURE.md), [contract parity](docs/CONTRACT_PARITY.md), and [failure scenarios](docs/FAILURE_SCENARIOS.md).

Production network and credential requirements are in [security deployment](docs/SECURITY_DEPLOYMENT.md). Direct public exposure of the application listener is unsupported.

## Publication and ownership

No license is included because publication permission and licensing must be confirmed by the project owner before a public GitHub release. The checklist is in [PUBLICATION_CHECKLIST.md](docs/PUBLICATION_CHECKLIST.md).

