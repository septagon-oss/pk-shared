# pk-shared

Small shared contract package for the OSS PlatformKit repos.

This repo should stay boring and small. It owns only cross-repo vocabulary that
cannot clearly belong to `pk-core`, `pk-design`,
`pk-modules`, or `pk-apps`.

## Current Surface

- `pkg/contract`: stable identifiers and semantic version vocabulary

## Verify

```bash
go test ./...
```
