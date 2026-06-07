package composition_test

// example_test.go provides runnable godoc examples for the composition
// package's application descriptor and validation surface.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import (
	"fmt"

	"github.com/septagon-oss/pk-shared/pkg/composition"
)

// Example builds an application descriptor and validates it against a catalog.
func Example() {
	catalog := []composition.ModuleCatalogEntry{
		{ID: "user_management", Tier: "core-certified"},
		{ID: "auth_management", Tier: "core-certified", Dependencies: []string{"user_management"}},
	}

	app := &composition.Application{
		APIVersion: composition.APIVersionV1,
		Kind:       composition.KindApplication,
		Metadata:   composition.AppMetadata{Name: "demo"},
		Spec: composition.ApplicationSpec{
			Modules: []composition.ModuleRef{
				{Name: "user_management", Enabled: true},
				{Name: "auth_management", Enabled: true},
			},
		},
	}

	report := composition.Validate(app, catalog)
	fmt.Println("valid:", report.Valid)
	fmt.Println("enabled:", composition.EnabledModules(app))
	// Output:
	// valid: true
	// enabled: [auth_management user_management]
}
