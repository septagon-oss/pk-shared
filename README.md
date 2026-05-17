# pk-shared

Small shared contract package for the OSS PlatformKit repos.

This repo should stay deliberately small. It owns only cross-repo vocabulary
that cannot clearly belong to `pk-core`, `pk-design`, `pk-modules`, or
`pk-apps`. If a contract has a natural owner, it should live there instead of
becoming ambient shared state.

## Current Surface

- `pkg/composition`: application, overlay, and topology-cell descriptors used
  to compose modules, surfaces, catalogs, and infrastructure blueprints
- `pkg/contract`: stable identifiers and semantic version vocabulary
- `pkg/flowdef`: neutral reusable flow definitions for UI/API coverage,
  authoring, and E2E/testkit bridges
- `pkg/statemachine`: declarative lifecycle definitions, structural
  validation, traversal helpers, and Mermaid rendering

## Verify

```bash
make verify
make staticcheck
```
