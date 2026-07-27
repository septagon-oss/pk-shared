# pk-shared Charter

## Purpose

Shared vocabulary and small primitives used across PlatformKit OSS repos. Keeps cross-repo coupling explicit and minimal.

## In Scope

- API wire contract (`pkg/apiwire`): canonical list-query parameters and response envelopes (REQ-021)
- Composition descriptors (`pkg/composition`): module capability declarations
- Canonical URL path segments (`pkg/pathsegment`): lossless, fail-closed transport for opaque entity IDs
- Flow definitions (`pkg/flowdef`): state machine specification format
- Permission-token grammar (`pkg/permissiontoken`): canonical, provider-neutral permission declarations

## Out of Scope

- Runtime execution or orchestration
- Business logic or domain types
- Repository, database, or network abstractions

## Dependencies

None (zero-dependency module).
