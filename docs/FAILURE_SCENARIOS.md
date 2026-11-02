# Failure scenarios

| Scenario | Expected behavior |
|---|---|
| Redis cache GET/SET fails on public read | log warning, query PostgreSQL, return the response |
| Redis rate limiter fails | fail closed with `500 internal_error` |
| Redis revocation check fails on admin call | fail closed with `500 internal_error` |
| PostgreSQL read fails/times out | `500 internal_error`; no stale-cache fallback is claimed |
| malformed/expired/wrong-algorithm JWT | `401 unauthorized` |
| token user inactive/deleted or version stale | `401 unauthorized` |
| authenticated non-admin user | `403 forbidden` |
| malformed UUID or version outside 1–8 | `422 validation_error` |
| concurrent publish/archive | row lock serializes transitions; each committed call writes its audit event |
| audit insert fails | transaction rolls back the place status update |
| post-commit admin read fails | mutation is committed but request returns an error; client retry is a repeat assignment and creates another audit event |
| public body exceeds configured limit | `413 payload_too_large` when a body is consumed; admin bodies are drained and bounded |
| panic in handler | recovered, logged with request ID, `500 internal_error` when possible |
| readiness dependency fails | `/v1/health/ready` returns `503`; liveness remains process-only |

The post-commit read case must be covered by operational guidance: investigate audit/state before manually replaying a write. It is not safe to infer “no write occurred” from a transport error.

