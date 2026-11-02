# Security deployment requirements

These are prerequisites for a future staging or production deployment. They are not evidence that this candidate has been deployed.

## HTTP and network boundary

- Terminate TLS at an approved Selectel load balancer or hardened reverse proxy and redirect HTTP to HTTPS there.
- Keep the Go application listener on a private network. Security groups must accept its port only from the load balancer and operational health-check sources. Direct Internet access to port 8080 is unsupported.
- Configure `TRUSTED_PROXY_CIDRS` with only the exact proxy networks. The service ignores forwarded client addresses from every other peer.
- Preserve the configured header/body/time limits at the edge. The application independently enforces `HTTP_MAX_HEADER_BYTES`, `REQUEST_BODY_MAX_BYTES`, and request timeouts.

## PostgreSQL and Redis transport

- Across any trust or host boundary, use PostgreSQL TLS with certificate and hostname verification (`sslmode=verify-full`) and a managed CA/root certificate. `sslmode=disable` is local development only.
- Across any trust or host boundary, use Redis TLS (`rediss://`) with certificate verification. If the managed Redis plan does not offer verified TLS, place the connection entirely inside the approved private network and record that exception in the deployment review.
- Restrict PostgreSQL and Redis security groups to the application workload and approved administration sources. Never publish their service ports to the Internet.
- Rotate independent application credentials through the deployment secret manager. Do not bake credentials into an image, Compose file used outside local development, or repository.

## Least-privilege identities

The runtime identity must not own the database, create extensions/schemas, bypass row-level security, or receive superuser/replication privileges. Provision schema and migrations with a separate migration identity. After reviewing the exact schema, grant the runtime role only:

- `CONNECT` on the application database and `USAGE` on its schema;
- `SELECT` on place read tables and the current `users` table used for authorization;
- the minimum `UPDATE` on lifecycle columns of `places`;
- `INSERT` on `audit_events`;
- sequence privileges only where an actually used sequence requires them.

Redis must use a dedicated ACL user limited to the key prefixes and commands needed for cache values, fixed-window counters, health checks, and session-revocation lookups. It must not have administrative, scripting-management, persistence, or cross-application key access. The fixed rate-limit Lua script requires the reviewed `EVALSHA`/script execution capability.

Validate the effective grants and Redis ACL in staging, then store the evidence with the deployment change. The local `bazgain` Compose identity is intentionally convenient and must not be reused outside disposable development.
