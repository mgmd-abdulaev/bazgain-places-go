# Contract provenance

This derived five-operation contract was prepared from the read-only Bazgain backend checkout:

- mobile snapshot: `contracts/mobile-openapi/locked/places.openapi.yaml`
- admin snapshot: `docs/api/admin/places.openapi.yaml`
- runtime DTOs/controllers under `src/modules/places`

The authoritative Flutter checkout referenced by the backend's `AGENTS.md` was not available locally. Before publication or integration, re-check this derived contract against that checkout and obtain permission to publish contract fragments and branding.

No full source snapshot is copied into this repository while publication permission remains unconfirmed.

Inspection baseline:

- backend commit: `736c7a95ebff735d6e440347799d5c9c989c2207`
- mobile snapshot SHA-256: `1ac09ba086f58f48bf9139427f7bea161f9ce40f41da734fb02b9163a9970041`
- admin snapshot SHA-256: `b1ddeb8c0ecccd11f7e32ebf31d12ecf5895d4b8e637a51c4c0c29833f338d5e`

The source checkout contained a pre-existing uncommitted change outside this candidate repository. The hashes above identify the two contract files directly; the commit alone must not be treated as a clean-worktree attestation.
