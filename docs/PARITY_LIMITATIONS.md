# Parity limitations

This candidate has not been connected to a staging or production Bazgain deployment.

Known validation boundaries:

1. The authoritative Flutter checkout named by the source repository was unavailable during implementation. The committed locked mobile snapshot and current admin/runtime implementation were used instead.
2. `api/openapi.yaml` is derived and intentionally limited to the five domain endpoints. It is not a claim that the whole Bazgain API was migrated.
3. The local schema is a compatibility baseline copied from the source migrations for disposable development databases. It must not be applied to an existing environment.
4. No production traffic, latency percentile, error-rate, or resource-utilization result is claimed.
5. Real rollout still requires dual-read comparison against the current service, staging verification, load testing at an agreed traffic model, and owner approval.
6. OpenAPI wire types are handwritten because the generation spike could not preserve all OpenAPI 3.1 nullable/admin runtime details without manual adaptation. The decision is recorded in ADR 0001.
7. The current NestJS publish rule accepts any gallery media, including a video-only gallery. The public list contract, however, requires `cover_image_url` to be a non-empty URI and the current fallback selects only images. This Go candidate deliberately preserves the NestJS publish behavior and therefore can return an empty cover URL for that edge case. `TestVideoOnlyGalleryPreservesCurrentNestJSPublishParity` records it. Fixing it requires a coordinated product/contract decision in both services; no Go-only error code or stricter behavior has been invented.

