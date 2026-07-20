// Package composition defines provider-neutral PlatformKit application
// composition descriptors.
package composition

// Implements: REQ-002.
// Per: ADR-0017.
// Discipline: C-14.
// application.go owns the stable application composition wire contract shared
// by OSS tools, modules, apps, and the private downstream.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

// APIVersionV1 is the current schema version for PlatformKit application
// descriptors.
const APIVersionV1 = "platformkit.dev/v1"

// KindApplication is the descriptor kind for a composed application.
const KindApplication = "Application"

// Application declares which modules, traits, and overlays compose a
// PlatformKit product.
type Application struct {
	APIVersion string          `yaml:"apiVersion" json:"apiVersion"`
	Kind       string          `yaml:"kind" json:"kind"`
	Metadata   AppMetadata     `yaml:"metadata" json:"metadata"`
	Spec       ApplicationSpec `yaml:"spec" json:"spec"`
}

// AppMetadata carries identity and labeling for an application descriptor.
type AppMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Version     string            `yaml:"version" json:"version"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// ApplicationSpec holds the composition body.
type ApplicationSpec struct {
	Modules    []ModuleRef  `yaml:"modules" json:"modules"`
	Traits     []TraitRef   `yaml:"traits,omitempty" json:"traits,omitempty"`
	Experience string       `yaml:"experience,omitempty" json:"experience,omitempty"`
	Overlays   []OverlayRef `yaml:"overlays,omitempty" json:"overlays,omitempty"`
}

// ModuleRef is a reference to a module in a PlatformKit catalog.
type ModuleRef struct {
	Name    string         `yaml:"name" json:"name"`
	Version string         `yaml:"version,omitempty" json:"version,omitempty"`
	Config  map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
	Enabled bool           `yaml:"enabled" json:"enabled"`
}

// TraitRef describes a cross-cutting capability applied to the application.
type TraitRef struct {
	Type   string         `yaml:"type" json:"type"`
	Config map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

// OverlayRef groups RFC 6902-style patch operations targeting a module or all
// modules with target "*".
type OverlayRef struct {
	Name    string         `yaml:"name" json:"name"`
	Target  string         `yaml:"target" json:"target"`
	Patches []OverlayPatch `yaml:"patches" json:"patches"`
}

// OverlayPatch is a single patch operation.
type OverlayPatch struct {
	Op    string `yaml:"op" json:"op"`
	Path  string `yaml:"path" json:"path"`
	From  string `yaml:"from,omitempty" json:"from,omitempty"`
	Value any    `yaml:"value,omitempty" json:"value,omitempty"`
}
