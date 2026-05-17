package composition

// cell_test.go validates topology-cell normalization and defaults used by
// composable modules, blueprints, catalogs, components, and surfaces.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import "testing"

func TestCellDescriptorValidate(t *testing.T) {
	t.Parallel()

	cell := CellDescriptor{
		ID:            "tenant.visit-management",
		Role:          CellRoleTenantService,
		Scope:         CellScopeTenant,
		Isolation:     CellIsolationTenant,
		RoutingKeys:   []string{"tenant", "environment", "tenant"},
		FailureDomain: "tenant",
		Stateful:      true,
	}

	if err := cell.Validate(); err != nil {
		t.Fatalf("Validate error = %v", err)
	}
	normalized := cell.Normalize()
	if len(normalized.RoutingKeys) != 2 {
		t.Fatalf("routing keys = %v; want deduped keys", normalized.RoutingKeys)
	}
}

func TestDefaultCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cell CellDescriptor
		role CellRole
	}{
		{"admin module", DefaultModuleCell("admin_management", "admin", "/admin"), CellRoleControlPlane},
		{"foundation blueprint", DefaultBlueprintCell("aws_client_foundation_eks", "foundation"), CellRoleInfrastructure},
		{"component", DefaultComponentCell("theme_management"), CellRoleSharedService},
		{"catalog", DefaultCatalogCell("lean", "module-catalog"), CellRoleControlPlane},
		{"admin surface", DefaultSurfaceCell("starter-admin", "admin"), CellRoleControlPlane},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.cell.Role != tt.role {
				t.Fatalf("Role = %q; want %q", tt.cell.Role, tt.role)
			}
			if err := tt.cell.Validate(); err != nil {
				t.Fatalf("Validate error = %v", err)
			}
		})
	}
}
