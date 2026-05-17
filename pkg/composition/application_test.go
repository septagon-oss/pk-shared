package composition

// application_test.go validates the public composition descriptor, validation,
// overlay, and export contracts before they are consumed by Pro adapters.
//
// ADR: ADR-0005 (no silent failures), ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import (
	"slices"
	"testing"
)

func testCatalog() []ModuleCatalogEntry {
	return []ModuleCatalogEntry{
		{ID: "user_management", Tier: "core-certified", Domain: "identity-access", Presets: []string{"minimal", "core", "default"}},
		{ID: "auth_management", Tier: "core-certified", Domain: "identity-access", Dependencies: []string{"user_management"}, Presets: []string{"minimal", "core", "default"}},
		{ID: "booking_management", Tier: "supported", Domain: "workspace", Dependencies: []string{"user_management"}, Presets: []string{"default"}},
		{ID: "beta_feature", Tier: "experimental", Domain: "platform"},
	}
}

func validApp() *Application {
	return &Application{
		APIVersion: APIVersionV1,
		Kind:       KindApplication,
		Metadata:   AppMetadata{Name: "test-app", Version: "1.0.0"},
		Spec: ApplicationSpec{
			Modules: []ModuleRef{
				{Name: "user_management", Enabled: true},
				{Name: "auth_management", Enabled: true},
				{Name: "booking_management", Enabled: true},
			},
		},
	}
}

func TestValidateApplication(t *testing.T) {
	report := Validate(validApp(), testCatalog())
	if !report.Valid {
		t.Fatalf("expected valid report, got errors: %v", report.Errors)
	}
	if report.ModuleCount != 3 || report.EnabledCount != 3 {
		t.Fatalf("counts = %d/%d; want 3/3", report.ModuleCount, report.EnabledCount)
	}
}

func TestValidateRejectsInvalidApplication(t *testing.T) {
	tests := []struct {
		name string
		app  *Application
		code string
	}{
		{name: "nil", code: "missing_application"},
		{name: "unknown", app: &Application{APIVersion: APIVersionV1, Kind: KindApplication, Metadata: AppMetadata{Name: "app"}, Spec: ApplicationSpec{Modules: []ModuleRef{{Name: "missing", Enabled: true}}}}, code: "unknown_module"},
		{name: "duplicate", app: &Application{APIVersion: APIVersionV1, Kind: KindApplication, Metadata: AppMetadata{Name: "app"}, Spec: ApplicationSpec{Modules: []ModuleRef{{Name: "user_management", Enabled: true}, {Name: "user_management", Enabled: true}}}}, code: "duplicate_module"},
		{name: "dependency", app: &Application{APIVersion: APIVersionV1, Kind: KindApplication, Metadata: AppMetadata{Name: "app"}, Spec: ApplicationSpec{Modules: []ModuleRef{{Name: "auth_management", Enabled: true}}}}, code: "missing_dependency"},
		{name: "patch", app: &Application{APIVersion: APIVersionV1, Kind: KindApplication, Metadata: AppMetadata{Name: "app"}, Spec: ApplicationSpec{Overlays: []OverlayRef{{Name: "bad", Target: "*", Patches: []OverlayPatch{{Op: "destroy", Path: "/name"}}}}}}, code: "invalid_patch_op"},
		{name: "patch path", app: &Application{APIVersion: APIVersionV1, Kind: KindApplication, Metadata: AppMetadata{Name: "app"}, Spec: ApplicationSpec{Overlays: []OverlayRef{{Name: "bad", Target: "*", Patches: []OverlayPatch{{Op: "add", Path: "name"}}}}}}, code: "invalid_patch_path"},
		{name: "patch from", app: &Application{APIVersion: APIVersionV1, Kind: KindApplication, Metadata: AppMetadata{Name: "app"}, Spec: ApplicationSpec{Overlays: []OverlayRef{{Name: "bad", Target: "*", Patches: []OverlayPatch{{Op: "copy", Path: "/name"}}}}}}, code: "invalid_patch_from"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := Validate(tt.app, testCatalog())
			if report.Valid {
				t.Fatal("expected invalid report")
			}
			if !hasIssueCode(report.Errors, tt.code) {
				t.Fatalf("errors = %#v; want code %q", report.Errors, tt.code)
			}
		})
	}
}

func TestValidateWarnsForUnknownOverlayTarget(t *testing.T) {
	app := validApp()
	app.Spec.Overlays = []OverlayRef{{Name: "stray", Target: "ghost_module", Patches: []OverlayPatch{{Op: "add", Path: "/x", Value: 1}}}}
	report := Validate(app, testCatalog())
	if !hasIssueCode(report.Warnings, "unknown_overlay_target") {
		t.Fatalf("warnings = %#v; want unknown overlay target", report.Warnings)
	}
}

func TestExperimentalModuleSuggestion(t *testing.T) {
	app := &Application{
		APIVersion: APIVersionV1,
		Kind:       KindApplication,
		Metadata:   AppMetadata{Name: "test-app"},
		Spec:       ApplicationSpec{Modules: []ModuleRef{{Name: "beta_feature", Enabled: true}}},
	}
	report := Validate(app, testCatalog())
	if len(report.Suggestions) == 0 {
		t.Fatal("expected experimental module suggestion")
	}
}

func TestEnabledModules(t *testing.T) {
	app := &Application{Spec: ApplicationSpec{Modules: []ModuleRef{
		{Name: "z_module", Enabled: true},
		{Name: "a_module", Enabled: true},
		{Name: "disabled"},
		{Name: "m_module", Enabled: true},
		{Name: " a_module ", Enabled: true},
		{Name: " ", Enabled: true},
	}}}
	got := EnabledModules(app)
	want := []string{"a_module", "m_module", "z_module"}
	if !slices.Equal(got, want) {
		t.Fatalf("EnabledModules = %v; want %v", got, want)
	}
}

func TestNewApplicationFromPreset(t *testing.T) {
	app := NewApplicationFromPreset("my-app", "minimal", testCatalog())
	enabled := EnabledModules(app)
	want := []string{"auth_management", "user_management"}
	if !slices.Equal(enabled, want) {
		t.Fatalf("EnabledModules = %v; want %v", enabled, want)
	}
}

func TestMergeOverlays(t *testing.T) {
	base := map[string]any{"title": "Hello", "count": 42, "remove": "gone"}
	overlays := []OverlayRef{
		{Name: "customize", Target: "my_module", Patches: []OverlayPatch{
			{Op: "replace", Path: "/title", Value: "Updated"},
			{Op: "add", Path: "/nested", Value: map[string]any{}},
			{Op: "add", Path: "/nested/value", Value: "added"},
			{Op: "remove", Path: "/remove"},
		}},
		{Name: "other", Target: "other_module", Patches: []OverlayPatch{{Op: "replace", Path: "/title", Value: "Wrong"}}},
	}
	result := MergeOverlays(base, overlays, "my_module")
	if result["title"] != "Updated" || result["count"] != 42 {
		t.Fatalf("merged result = %#v", result)
	}
	if _, exists := result["remove"]; exists {
		t.Fatal("remove patch should delete key")
	}
	nested, ok := result["nested"].(map[string]any)
	if !ok || nested["value"] != "added" {
		t.Fatalf("nested result = %#v", result["nested"])
	}
	if base["title"] != "Hello" {
		t.Fatal("base map was mutated")
	}
}

func TestApplyOverlaysSupportsCopyMoveTestAndArrays(t *testing.T) {
	base := map[string]any{
		"nested": map[string]any{"from": "copied", "move": "moved"},
		"items":  []any{"first"},
	}
	overlays := []OverlayRef{{
		Name:   "advanced",
		Target: "my_module",
		Patches: []OverlayPatch{
			{Op: "test", Path: "/nested/from", Value: "copied"},
			{Op: "copy", From: "/nested/from", Path: "/nested/copied"},
			{Op: "move", From: "/nested/move", Path: "/nested/moved"},
			{Op: "add", Path: "/items/-", Value: "second"},
		},
	}}

	result, issues := ApplyOverlays(base, overlays, "my_module")
	if len(issues) != 0 {
		t.Fatalf("ApplyOverlays issues = %#v", issues)
	}
	nested := result["nested"].(map[string]any)
	if nested["copied"] != "copied" || nested["moved"] != "moved" {
		t.Fatalf("nested result = %#v", nested)
	}
	if _, exists := nested["move"]; exists {
		t.Fatalf("move source still exists in %#v", nested)
	}
	items := result["items"].([]any)
	if !slices.Equal(items, []any{"first", "second"}) {
		t.Fatalf("items = %#v", items)
	}
	originalNested := base["nested"].(map[string]any)
	if _, exists := originalNested["copied"]; exists {
		t.Fatalf("base nested map was mutated: %#v", originalNested)
	}
	originalItems := base["items"].([]any)
	if !slices.Equal(originalItems, []any{"first"}) {
		t.Fatalf("base slice was mutated: %#v", originalItems)
	}
}

func TestApplyOverlaysReportsFailuresWithoutMutatingResult(t *testing.T) {
	base := map[string]any{"title": "Hello"}
	overlays := []OverlayRef{
		{
			Name: "missing-target",
			Patches: []OverlayPatch{
				{Op: "replace", Path: "/title", Value: "Ignored"},
			},
		},
		{
			Name:   "bad",
			Target: "*",
			Patches: []OverlayPatch{
				{Op: "test", Path: "/title", Value: "Wrong"},
				{Op: "replace", Path: "/missing", Value: "nope"},
				{Op: "add", Path: "/missing/child", Value: "nope"},
			},
		},
	}

	result, issues := ApplyOverlays(base, overlays, "my_module")
	if len(issues) != 4 {
		t.Fatalf("ApplyOverlays issues = %#v; want 4", issues)
	}
	if result["title"] != "Hello" {
		t.Fatalf("result mutated after failed patches: %#v", result)
	}
	if base["title"] != "Hello" {
		t.Fatalf("base mutated after failed patches: %#v", base)
	}
}

func TestExports(t *testing.T) {
	app := &Application{
		Metadata: AppMetadata{Name: "helm-app"},
		Spec: ApplicationSpec{Modules: []ModuleRef{
			{Name: "booking_management", Enabled: true, Config: map[string]any{"maxBookings": 100}},
			{Name: " payment_management ", Enabled: true},
			{Name: " ", Enabled: true},
			{Name: "disabled_module"},
		}},
	}
	values := ToHelmValues(app)
	modules := values["modules"].(map[string]any)
	if _, exists := modules["disabled_module"]; exists {
		t.Fatal("disabled module should not appear in helm values")
	}
	if _, exists := modules[""]; exists {
		t.Fatal("empty module name should not appear in helm values")
	}
	if _, exists := modules["payment_management"]; !exists {
		t.Fatal("trimmed payment module should appear in helm values")
	}
	booking := modules["booking_management"].(map[string]any)
	cfg := booking["config"].(map[string]any)
	if cfg["maxBookings"] != 100 {
		t.Fatalf("config = %#v", cfg)
	}
	cfg["maxBookings"] = 1
	if app.Spec.Modules[0].Config["maxBookings"] != 100 {
		t.Fatalf("exported config mutation changed application config: %#v", app.Spec.Modules[0].Config)
	}
	cfgValues := ToConfigYAML(app)
	cfgModules := cfgValues["modules"].(map[string]any)
	if cfgModules["booking_management"].(map[string]any)["enabled"] != true {
		t.Fatalf("config modules = %#v", cfgModules)
	}
}

func hasIssueCode(issues []ValidationIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
