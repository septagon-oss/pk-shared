package composition

// cell.go owns provider-neutral topology cells used by module, blueprint,
// catalog, component, and surface composition manifests.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import (
	"fmt"
	"slices"
	"strings"
)

// CellRole identifies a cell's responsibility in the platform topology.
type CellRole string

const (
	CellRoleControlPlane   CellRole = "control-plane"
	CellRoleSharedService  CellRole = "shared-service"
	CellRoleTenantService  CellRole = "tenant-service"
	CellRoleInfrastructure CellRole = "infrastructure"
	CellRoleWorkload       CellRole = "workload"
)

// CellScope identifies the routing and ownership scope of a cell.
type CellScope string

const (
	CellScopeGlobal      CellScope = "global"
	CellScopeEnvironment CellScope = "environment"
	CellScopeRegion      CellScope = "region"
	CellScopeTenant      CellScope = "tenant"
	CellScopeWorkload    CellScope = "workload"
)

// CellIsolation identifies a cell's isolation model.
type CellIsolation string

const (
	CellIsolationShared    CellIsolation = "shared"
	CellIsolationPooled    CellIsolation = "pooled"
	CellIsolationTenant    CellIsolation = "tenant"
	CellIsolationDedicated CellIsolation = "dedicated"
)

// CellDescriptor defines where a module, service, or blueprint lives in the
// platform topology and which failure/isolation domain it belongs to.
type CellDescriptor struct {
	ID            string        `yaml:"id" json:"id"`
	Role          CellRole      `yaml:"role" json:"role"`
	Scope         CellScope     `yaml:"scope" json:"scope"`
	Isolation     CellIsolation `yaml:"isolation" json:"isolation"`
	RoutingKeys   []string      `yaml:"routingKeys,omitempty" json:"routingKeys,omitempty"`
	FailureDomain string        `yaml:"failureDomain,omitempty" json:"failureDomain,omitempty"`
	Stateful      bool          `yaml:"stateful,omitempty" json:"stateful,omitempty"`
}

// Normalize trims descriptor fields and canonicalizes routing keys.
func (c CellDescriptor) Normalize() CellDescriptor {
	c.ID = strings.TrimSpace(c.ID)
	c.Role = CellRole(strings.TrimSpace(string(c.Role)))
	c.Scope = CellScope(strings.TrimSpace(string(c.Scope)))
	c.Isolation = CellIsolation(strings.TrimSpace(string(c.Isolation)))
	c.FailureDomain = strings.TrimSpace(c.FailureDomain)
	c.RoutingKeys = normalizeRoutingKeys(c.RoutingKeys)
	return c
}

// Validate checks that a cell descriptor is complete and uses known core
// vocabulary.
func (c CellDescriptor) Validate() error {
	c = c.Normalize()
	if c.ID == "" {
		return fmt.Errorf("cell id is required")
	}
	switch c.Role {
	case CellRoleControlPlane, CellRoleSharedService, CellRoleTenantService, CellRoleInfrastructure, CellRoleWorkload:
	default:
		return fmt.Errorf("cell %q has unsupported role %q", c.ID, c.Role)
	}
	switch c.Scope {
	case CellScopeGlobal, CellScopeEnvironment, CellScopeRegion, CellScopeTenant, CellScopeWorkload:
	default:
		return fmt.Errorf("cell %q has unsupported scope %q", c.ID, c.Scope)
	}
	switch c.Isolation {
	case CellIsolationShared, CellIsolationPooled, CellIsolationTenant, CellIsolationDedicated:
	default:
		return fmt.Errorf("cell %q has unsupported isolation %q", c.ID, c.Isolation)
	}
	if c.FailureDomain == "" {
		return fmt.Errorf("cell %q must declare a failure domain", c.ID)
	}
	if len(c.RoutingKeys) == 0 {
		return fmt.Errorf("cell %q must declare at least one routing key", c.ID)
	}
	return nil
}

// DefaultModuleCell returns the default topology cell for a backend module.
func DefaultModuleCell(moduleName, category, basePath string) CellDescriptor {
	moduleName = normalizeCellIDPart(moduleName)
	category = strings.TrimSpace(strings.ToLower(category))
	basePath = strings.TrimSpace(strings.ToLower(basePath))
	switch {
	case strings.HasPrefix(basePath, "/admin"), category == "admin", category == "core", category == "platform":
		return CellDescriptor{
			ID:            "control." + moduleName,
			Role:          CellRoleControlPlane,
			Scope:         CellScopeEnvironment,
			Isolation:     CellIsolationPooled,
			RoutingKeys:   []string{"environment"},
			FailureDomain: "environment",
		}
	case category == "monitoring", category == "governance":
		return CellDescriptor{
			ID:            "shared." + moduleName,
			Role:          CellRoleSharedService,
			Scope:         CellScopeEnvironment,
			Isolation:     CellIsolationPooled,
			RoutingKeys:   []string{"environment"},
			FailureDomain: "environment",
			Stateful:      true,
		}
	default:
		return CellDescriptor{
			ID:            "tenant." + moduleName,
			Role:          CellRoleTenantService,
			Scope:         CellScopeTenant,
			Isolation:     CellIsolationTenant,
			RoutingKeys:   []string{"tenant", "environment"},
			FailureDomain: "tenant",
			Stateful:      true,
		}
	}
}

// DefaultBlueprintCell returns the default topology cell for an infrastructure
// blueprint.
func DefaultBlueprintCell(blueprintID, category string) CellDescriptor {
	blueprintID = normalizeCellIDPart(blueprintID)
	category = strings.TrimSpace(strings.ToLower(category))
	switch {
	case category == "foundation",
		strings.Contains(blueprintID, "bootstrap"),
		strings.Contains(blueprintID, "landing-zone"),
		strings.Contains(blueprintID, "landing_zone"),
		strings.Contains(blueprintID, "foundation"):
		return CellDescriptor{
			ID:            "infra." + blueprintID,
			Role:          CellRoleInfrastructure,
			Scope:         CellScopeRegion,
			Isolation:     CellIsolationPooled,
			RoutingKeys:   []string{"region", "environment"},
			FailureDomain: "region",
			Stateful:      true,
		}
	default:
		return CellDescriptor{
			ID:            "workload." + blueprintID,
			Role:          CellRoleWorkload,
			Scope:         CellScopeTenant,
			Isolation:     CellIsolationDedicated,
			RoutingKeys:   []string{"tenant", "environment", "region"},
			FailureDomain: "tenant",
			Stateful:      true,
		}
	}
}

// DefaultComponentCell returns the default topology cell for a UI component
// module.
func DefaultComponentCell(moduleName string) CellDescriptor {
	moduleName = normalizeCellIDPart(moduleName)
	if moduleName == "" {
		moduleName = "core"
	}
	return CellDescriptor{
		ID:            "ui." + moduleName,
		Role:          CellRoleSharedService,
		Scope:         CellScopeGlobal,
		Isolation:     CellIsolationShared,
		RoutingKeys:   []string{"build", "global", "ui"},
		FailureDomain: "build",
	}
}

// DefaultCatalogCell returns the default topology cell for a registry catalog.
func DefaultCatalogCell(catalogID, kind string) CellDescriptor {
	catalogID = normalizeCellIDPart(catalogID)
	kind = normalizeCellIDPart(kind)
	if catalogID == "" {
		catalogID = "default"
	}
	switch kind {
	case "component-catalog", "design-catalog":
		return CellDescriptor{
			ID:            "catalog." + catalogID,
			Role:          CellRoleSharedService,
			Scope:         CellScopeGlobal,
			Isolation:     CellIsolationShared,
			RoutingKeys:   []string{"build", "catalog", "global"},
			FailureDomain: "build",
		}
	default:
		return CellDescriptor{
			ID:            "catalog." + catalogID,
			Role:          CellRoleControlPlane,
			Scope:         CellScopeEnvironment,
			Isolation:     CellIsolationPooled,
			RoutingKeys:   []string{"catalog", "environment"},
			FailureDomain: "build",
		}
	}
}

// DefaultSurfaceCell returns the default topology cell for a rendered surface.
func DefaultSurfaceCell(surfaceID, shellProfile string) CellDescriptor {
	surfaceID = normalizeCellIDPart(surfaceID)
	shellProfile = normalizeCellIDPart(shellProfile)
	if surfaceID == "" {
		surfaceID = "default"
	}
	switch shellProfile {
	case "admin":
		return CellDescriptor{
			ID:            "surface." + surfaceID,
			Role:          CellRoleControlPlane,
			Scope:         CellScopeEnvironment,
			Isolation:     CellIsolationPooled,
			RoutingKeys:   []string{"environment", "surface"},
			FailureDomain: "environment",
		}
	default:
		return CellDescriptor{
			ID:            "surface." + surfaceID,
			Role:          CellRoleWorkload,
			Scope:         CellScopeTenant,
			Isolation:     CellIsolationPooled,
			RoutingKeys:   []string{"environment", "surface", "tenant"},
			FailureDomain: "tenant",
		}
	}
}

func normalizeRoutingKeys(keys []string) []string {
	if len(keys) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(strings.ToLower(key))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

func normalizeCellIDPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "_", "-")
	return value
}
