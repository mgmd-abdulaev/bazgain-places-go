# Contract parity

`api/openapi.yaml` is a derived, five-operation view of the locked mobile contract, admin contract snapshot, and current Places runtime DTOs/controllers. API JSON remains `snake_case`, public envelopes remain `collections`, `items`, and `place`, and errors retain `{ "error": { "code", "message", "fields?" } }`.

## Preserved behaviors

- public endpoints expose published records only;
- list ordering and nested relation ordering are deterministic;
- PostGIS geometry is returned as GeoJSON in longitude/latitude order;
- public GETs emit ETag and cache-control headers and honor `If-None-Match` with `304`;
- unknown or invalid detail slugs return the locked not-found envelope;
- admin IDs accept RFC UUID variants with versions 1 through 8 and are canonicalized to lowercase;
- publish requires gallery media and a navigation point;
- repeat publish/archive calls are safe lifecycle assignments and still produce an audit event;
- admin responses contain the full nested admin place view;
- incidental admin request bodies are ignored after the global size limit is enforced.

## Error mapping

| Situation | HTTP | Stable code |
|---|---:|---|
| invalid admin UUID | 422 | `validation_error` |
| missing/invalid/revoked/stale access token | 401 | `access_token_invalid` |
| current user is not admin | 403 | `forbidden` |
| place not found | 404 | `place_not_found` |
| publish without gallery | 409 | `cannot_publish_place_without_gallery` |
| publish without navigation point | 409 | `cannot_publish_place_without_navigation_point` |
| rate limit exceeded | 429 | `rate_limited` |
| dependency or unexpected failure | 500 | `internal_error` |

Automated contract and integration tests are expected to assert exact payloads, required-nullable fields, empty arrays, status codes, headers, and error envelopes.

