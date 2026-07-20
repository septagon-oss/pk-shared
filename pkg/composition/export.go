package composition

// Implements: REQ-002.
// Per: ADR-0017.
// Discipline: C-14.
// export.go owns derived map views used by deployment/rendering tools while
// keeping the composition descriptor as the source of truth.
//
// ADR: ADR-0005 (no silent failures), ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import "strings"

// ToHelmValues converts an Application descriptor into a map suitable for Helm
// values rendering. Only enabled modules are included.
func ToHelmValues(app *Application) map[string]any {
	modules := enabledModuleValues(app, true)
	name := ""
	if app != nil {
		name = app.Metadata.Name
	}
	return map[string]any{
		"app": map[string]any{
			"name":          name,
			"module_preset": "none",
		},
		"modules": modules,
	}
}

// ToConfigYAML converts an Application descriptor into a map matching
// PlatformKit config.yaml layout. Only enabled modules are included.
func ToConfigYAML(app *Application) map[string]any {
	name := ""
	if app != nil {
		name = app.Metadata.Name
	}
	return map[string]any{
		"app": map[string]any{
			"name":          name,
			"module_preset": "none",
		},
		"modules": enabledModuleValues(app, false),
	}
}

func enabledModuleValues(app *Application, includeConfig bool) map[string]any {
	modules := make(map[string]any)
	if app == nil {
		return modules
	}
	for _, ref := range app.Spec.Modules {
		if !ref.Enabled {
			continue
		}
		name := strings.TrimSpace(ref.Name)
		if name == "" {
			continue
		}
		entry := map[string]any{"enabled": true}
		if includeConfig && len(ref.Config) > 0 {
			entry["config"] = deepCopyMap(ref.Config)
		}
		modules[name] = entry
	}
	return modules
}
