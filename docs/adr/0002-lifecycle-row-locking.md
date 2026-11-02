# ADR 0002: lifecycle row locking

Status: accepted.

Publish and archive select the place `FOR UPDATE` and write the lifecycle audit event in the same transaction. The NestJS implementation performs some publish guards before opening the transaction. The Go candidate intentionally narrows that race window while preserving the HTTP contract. Concurrent lifecycle tests must prove serialization and audit atomicity.
