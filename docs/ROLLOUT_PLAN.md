# Proposed rollout plan

This is a design for a future deployment, not evidence that a deployment occurred.

## Preconditions

- confirm publication and schema-use permission;
- compare the derived contract with the authoritative Flutter checkout;
- run unit, integration, contract, race, and load tests against an isolated environment;
- verify indexes and query plans against a production-like dataset;
- provision independent credentials, dashboards, alerts, and an immutable image tag;
- record the exact candidate commit and image digest in the change ticket.

## Staging

1. Deploy the Go service with no public routing.
2. Run migrations only if an approved, environment-specific migration review says they are needed; do not use `db/local`.
3. Replay or mirror sanitized requests to both services.
4. Compare status, headers, JSON after normalizing timestamps/order where explicitly allowed, and database side effects for admin calls.
5. Exercise Redis loss, PostgreSQL failover, token revocation, duplicate lifecycle calls, and rollback.

## Production canary

1. Start with one instance and zero user traffic; validate readiness and dependency connectivity.
2. Route an agreed small percentage of only the three public GETs.
3. Compare error rate, latency, saturation, cache behavior, and contract mismatches against pre-agreed thresholds.
4. Increase public traffic in explicit gates.
5. Enable admin mutations separately only after write-path parity, audit verification, and rollback rehearsal.
6. Record every gate decision and the responsible reviewer.

Do not describe this plan as completed until deployment records and monitoring evidence exist.

