# Proposed rollback runbook

This runbook is unexecuted design documentation.

## Trigger examples

- contract mismatch or mobile-client regression;
- elevated error/latency beyond the approved gate;
- missing or incorrect audit events;
- authentication or authorization anomaly;
- PostgreSQL/Redis saturation attributable to the candidate.

## Procedure

1. Stop increasing traffic and identify the active candidate commit/image digest.
2. Route affected endpoints back to the NestJS service at the gateway/load balancer.
3. Disable Go admin mutation routing before public read routing if write correctness is in question.
4. Keep the Go instance available only for diagnostics if safe; otherwise scale it to zero.
5. Do not reverse committed place state automatically. Lifecycle writes use the shared source-of-truth schema and must be assessed from audit events.
6. Confirm mobile/admin success against the original service, then preserve logs, traces, and comparison artifacts.
7. Document cause, impact, and required fix before another canary.

Redis cache keys are namespaced/versioned and can expire naturally. Cache deletion is not required for routing rollback.

