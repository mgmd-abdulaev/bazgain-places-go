# ADR 0001: handwritten wire types

Status: accepted for this candidate.

The locked public contract uses OpenAPI 3.1 nullable unions, while the generated admin snapshot contains nullable object constructs that do not faithfully describe the runtime DTOs. To avoid silently changing nullability, this candidate keeps a derived five-operation OpenAPI document and handwritten Go wire types. Contract tests must validate those types against `api/openapi.yaml` and the original source when it becomes available.

Generated files are not claimed. The contract remains the boundary, and `internal/wire` must contain no persistence behavior.

The spike was reproduced with:

```bash
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1 \
  -generate types -package generated -o /dev/null api/openapi.yaml
```

The generator parsed the document but warned that OpenAPI 3.1 is unsupported and stopped on a nullable object union (`type: [object, "null"]`). This is why generated output is not committed.
