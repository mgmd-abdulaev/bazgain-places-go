# Architecture

The executable in `cmd/api` loads validated configuration and delegates startup to `internal/app`. Dependencies are wired explicitly: `pgxpool`, Redis, repositories, cache/rate-limit adapters, application service, authentication, health checks, and the `net/http` transport.

## Request flow

```text
client
  -> request ID / recovery / body limit / structured access log
  -> Redis fixed-window rate limit
  -> public handler OR JWT + Redis revocation + PostgreSQL user/role check
  -> places service
  -> PostgreSQL/PostGIS repository and Redis cache
  -> snake_case wire model
```

Controllers are represented by small HTTP handlers. Domain orchestration is in `internal/places`; SQL and relation hydration are isolated in `internal/postgres`; JSON contracts are in `internal/wire`.

## Public reads

Each public read first asks PostgreSQL for the relevant maximum `updated_at`. The timestamp becomes both the quoted ETag and part of the Redis cache key. Cache misses are rebuilt from PostgreSQL. Detail reads reject UUID-shaped or invalid slugs as not found, matching the source behavior.

Redis response-cache failures are logged and treated as misses. This makes PostgreSQL the source of truth, while cache entries expire after 60 seconds by default. Old versioned keys may remain until TTL expiry but are no longer selected after the PostgreSQL version changes.

## Lifecycle writes

Publish/archive use a row lock to serialize lifecycle changes. The status update and audit insert are one transaction. Publish validates gallery and navigation-point prerequisites. PostgreSQL timestamps are used for status and audit times. A full admin view is read after commit.

The row lock is deliberate concurrency hardening; see `docs/adr/0002-lifecycle-row-locking.md`.

## Authentication and authorization

Admin mutations accept only HS256 JWTs with the configured issuer/audience, expiration, `typ=access`, user `sub`, session `sid`, and token version `ver`. The session must not be revoked in Redis. The database user must still be active, have the same token version, and currently hold the `admin` role. The role claim is not trusted for authorization.

## Operational boundaries

- PostgreSQL connections are configured for UTC.
- Request/query/transaction/shutdown timeouts are bounded and configurable.
- Only explicitly trusted proxy CIDRs may supply `X-Forwarded-For`.
- Health liveness is process-only; readiness checks both PostgreSQL and Redis.
- Public lists have explicit hard caps and log when approaching 80% of a cap.
- The service emits structured JSON logs, but this candidate claims no observed production SLOs.
- Internet-facing TLS must terminate at an approved load balancer or reverse proxy. The application listener is private-only and direct public exposure of port 8080 is unsupported.
- PostgreSQL and Redis transport security and least-privilege requirements are specified in `docs/SECURITY_DEPLOYMENT.md`.

