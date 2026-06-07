# pk-shared

[![Go Reference](https://pkg.go.dev/badge/github.com/septagon-oss/pk-shared.svg)](https://pkg.go.dev/github.com/septagon-oss/pk-shared)
[![CI](https://github.com/septagon-oss/pk-shared/actions/workflows/go.yml/badge.svg)](https://github.com/septagon-oss/pk-shared/actions/workflows/go.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

`pk-shared` is the deliberately small, provider-neutral contract library for the
OSS PlatformKit family (`pk-core`, `pk-design`, `pk-modules`, `pk-apps`). It owns
only the cross-repo vocabulary — application composition descriptors, stable
identifiers, neutral flow definitions, and a portable state-machine model — that
cannot cleanly belong to a single owning repo. If a contract has a natural home,
it lives there instead of becoming ambient shared state.

## Install

```bash
go get github.com/septagon-oss/pk-shared@v0.1.0
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/septagon-oss/pk-shared/pkg/composition"
)

func main() {
	app := &composition.Application{
		APIVersion: composition.APIVersionV1,
		Kind:       composition.KindApplication,
		Metadata:   composition.AppMetadata{Name: "demo"},
		Spec: composition.ApplicationSpec{
			Modules: []composition.ModuleRef{
				{Name: "user_management", Enabled: true},
			},
		},
	}

	catalog := []composition.ModuleCatalogEntry{{ID: "user_management"}}
	report := composition.Validate(app, catalog)
	fmt.Println("valid:", report.Valid)
	fmt.Println("enabled:", composition.EnabledModules(app))
}
```

## Current Surface

- `pkg/composition`: application, overlay, and topology-cell descriptors used to
  compose modules, surfaces, catalogs, and infrastructure blueprints, plus
  validation and Helm/config export helpers
- `pkg/contract`: stable module identifiers and semantic version vocabulary
- `pkg/flowdef`: neutral reusable flow definitions for UI/API coverage,
  authoring, and E2E/testkit bridges
- `pkg/statemachine`: declarative lifecycle definitions, structural validation,
  read-only traversal helpers, and Mermaid rendering

## Verify

```bash
make verify   # go test + go vet + staticcheck + race
```

## License

Apache-2.0. See [LICENSE](LICENSE).
