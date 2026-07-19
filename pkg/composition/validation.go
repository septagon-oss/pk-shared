package composition

// validation.go owns structural validation and deterministic overlay
// application for application composition descriptors.
//
// Implements: REQ-002.
// Per: ADR-0005 (no silent failures), ADR-0029 (file purpose declaration); C-10 (shared builders return errors).
// Discipline: C-14.

import (
	"cmp"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

const (
	// SeverityError marks a validation issue that makes a descriptor invalid.
	SeverityError = "error"
	// SeverityWarning marks a non-fatal validation issue worth surfacing.
	SeverityWarning = "warning"
)

const (
	patchOpAdd     = "add"
	patchOpRemove  = "remove"
	patchOpReplace = "replace"
	patchOpMove    = "move"
	patchOpCopy    = "copy"
	patchOpTest    = "test"
)

// ValidationIssue describes a single problem found during validation.
type ValidationIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Module   string `json:"module,omitempty"`
	Message  string `json:"message"`
}

// ValidationReport summarizes the result of validating an Application.
type ValidationReport struct {
	Valid        bool              `json:"valid"`
	Errors       []ValidationIssue `json:"errors,omitempty"`
	Warnings     []ValidationIssue `json:"warnings,omitempty"`
	Suggestions  []string          `json:"suggestions,omitempty"`
	ModuleCount  int               `json:"moduleCount"`
	EnabledCount int               `json:"enabledCount"`
}

// ModuleCatalogEntry is the minimal catalog information needed to validate an
// application descriptor. The caller owns the catalog source.
type ModuleCatalogEntry struct {
	ID           string   `json:"id"`
	Tier         string   `json:"tier"`
	Domain       string   `json:"domain"`
	Dependencies []string `json:"dependencies,omitempty"`
	Presets      []string `json:"presets,omitempty"`
}

// Validate checks an Application against the provided catalog and returns a
// full validation report.
func Validate(app *Application, catalog []ModuleCatalogEntry) *ValidationReport {
	report := &ValidationReport{}
	if app == nil {
		report.addError("", "missing_application", "application descriptor must not be nil")
		report.Valid = false
		return report
	}

	catalogMap := make(map[string]ModuleCatalogEntry, len(catalog))
	for _, entry := range catalog {
		entry.ID = strings.TrimSpace(entry.ID)
		if entry.ID == "" {
			continue
		}
		catalogMap[entry.ID] = entry
	}

	if strings.TrimSpace(app.APIVersion) != APIVersionV1 {
		report.addError("", "invalid_api_version",
			fmt.Sprintf("apiVersion must be %q, got %q", APIVersionV1, app.APIVersion))
	}
	if strings.TrimSpace(app.Kind) != KindApplication {
		report.addError("", "invalid_kind",
			fmt.Sprintf("kind must be %q, got %q", KindApplication, app.Kind))
	}
	if strings.TrimSpace(app.Metadata.Name) == "" {
		report.addError("", "missing_name", "metadata.name must not be empty")
	}

	enabledSet := make(map[string]bool)
	seen := make(map[string]bool)
	for _, ref := range app.Spec.Modules {
		name := strings.TrimSpace(ref.Name)
		report.ModuleCount++
		if ref.Enabled {
			report.EnabledCount++
		}
		if name == "" {
			report.addError("", "missing_module_name", "module name must not be empty")
			continue
		}
		if seen[name] {
			report.addError(name, "duplicate_module", fmt.Sprintf("module %q appears more than once", name))
			continue
		}
		seen[name] = true
		if ref.Enabled {
			enabledSet[name] = true
		}
	}

	enabledNames := make([]string, 0, len(enabledSet))
	for name := range enabledSet {
		enabledNames = append(enabledNames, name)
	}
	slices.Sort(enabledNames)
	for _, name := range enabledNames {
		entry, exists := catalogMap[name]
		if !exists {
			report.addError(name, "unknown_module", fmt.Sprintf("module %q is not in the catalog", name))
			continue
		}
		for _, dep := range entry.Dependencies {
			dep = strings.TrimSpace(dep)
			if dep != "" && !enabledSet[dep] {
				report.addError(name, "missing_dependency",
					fmt.Sprintf("module %q requires %q, which is not enabled", name, dep))
			}
		}
		if strings.TrimSpace(entry.Tier) == "experimental" {
			report.Suggestions = append(report.Suggestions,
				fmt.Sprintf("module %q is experimental; review its stability before production use", name))
		}
	}
	slices.Sort(report.Suggestions)

	for _, overlay := range app.Spec.Overlays {
		target := strings.TrimSpace(overlay.Target)
		if target == "" {
			report.addError("", "missing_overlay_target",
				fmt.Sprintf("overlay %q must declare target or %q", overlay.Name, "*"))
		}
		if target != "" && target != "*" && !seen[target] {
			report.addWarning(target, "unknown_overlay_target",
				fmt.Sprintf("overlay %q targets module %q, which is not in the application", overlay.Name, target))
		}
		for i, patch := range overlay.Patches {
			for _, issue := range validatePatchShape(overlay.Name, target, i, patch) {
				report.addIssue(issue)
			}
		}
	}

	report.Valid = len(report.Errors) == 0
	return report
}

// EnabledModules returns the sorted list of enabled module names.
func EnabledModules(app *Application) []string {
	if app == nil {
		return nil
	}
	var names []string
	seen := map[string]struct{}{}
	for _, ref := range app.Spec.Modules {
		name := strings.TrimSpace(ref.Name)
		if ref.Enabled && name != "" {
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// NewApplicationFromPreset creates an Application enabling every module whose
// Presets include preset.
func NewApplicationFromPreset(name, preset string, catalog []ModuleCatalogEntry) *Application {
	app := &Application{
		APIVersion: APIVersionV1,
		Kind:       KindApplication,
		Metadata:   AppMetadata{Name: strings.TrimSpace(name)},
	}
	seen := map[string]struct{}{}
	for _, entry := range catalog {
		id := strings.TrimSpace(entry.ID)
		if id != "" && containsPreset(entry.Presets, preset) {
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			app.Spec.Modules = append(app.Spec.Modules, ModuleRef{Name: id, Enabled: true})
		}
	}
	slices.SortStableFunc(app.Spec.Modules, func(a, b ModuleRef) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return app
}

func (r *ValidationReport) addError(module, code, message string) {
	r.Errors = append(r.Errors, ValidationIssue{
		Severity: SeverityError,
		Code:     code,
		Module:   module,
		Message:  message,
	})
}

func (r *ValidationReport) addWarning(module, code, message string) {
	r.Warnings = append(r.Warnings, ValidationIssue{
		Severity: SeverityWarning,
		Code:     code,
		Module:   module,
		Message:  message,
	})
}

func containsPreset(presets []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, preset := range presets {
		if strings.TrimSpace(preset) == target {
			return true
		}
	}
	return false
}

func isValidPatchOp(op string) bool {
	switch op {
	case patchOpAdd, patchOpRemove, patchOpReplace, patchOpMove, patchOpCopy, patchOpTest:
		return true
	default:
		return false
	}
}

func validatePatchShape(overlayName, target string, index int, patch OverlayPatch) []ValidationIssue {
	op := strings.TrimSpace(patch.Op)
	if !isValidPatchOp(op) {
		return []ValidationIssue{patchIssue(overlayName, target, index, "invalid_patch_op",
			fmt.Sprintf("has invalid op %q", patch.Op))}
	}
	if _, err := parsePointer(patch.Path); err != nil {
		return []ValidationIssue{patchIssue(overlayName, target, index, "invalid_patch_path", err.Error())}
	}
	switch op {
	case patchOpMove, patchOpCopy:
		if _, err := parsePointer(patch.From); err != nil {
			return []ValidationIssue{patchIssue(overlayName, target, index, "invalid_patch_from", err.Error())}
		}
	}
	return nil
}

func patchIssue(overlayName, target string, index int, code, message string) ValidationIssue {
	module := target
	if module == "*" {
		module = ""
	}
	return ValidationIssue{
		Severity: SeverityError,
		Code:     code,
		Module:   module,
		Message:  fmt.Sprintf("overlay %q patch[%d] %s", overlayName, index, message),
	}
}

func (r *ValidationReport) addIssue(issue ValidationIssue) {
	if issue.Severity == SeverityWarning {
		r.Warnings = append(r.Warnings, issue)
		return
	}
	if issue.Severity == "" {
		issue.Severity = SeverityError
	}
	r.Errors = append(r.Errors, issue)
}

// ApplyOverlays applies RFC 6902-style map/list patches whose target matches
// target or "*". It returns structured issues for invalid or failed patch
// operations so callers do not have to discover overlay drift at runtime.
func ApplyOverlays(base map[string]any, overlays []OverlayRef, target string) (map[string]any, []ValidationIssue) {
	result := deepCopyMap(base)
	target = strings.TrimSpace(target)
	var issues []ValidationIssue
	for _, overlay := range overlays {
		overlayTarget := strings.TrimSpace(overlay.Target)
		if overlayTarget == "" {
			issues = append(issues, ValidationIssue{
				Severity: SeverityError,
				Code:     "missing_overlay_target",
				Message:  fmt.Sprintf("overlay %q must declare target or %q", overlay.Name, "*"),
			})
			continue
		}
		if overlayTarget != target && overlayTarget != "*" {
			continue
		}
		for i, patch := range overlay.Patches {
			shapeIssues := validatePatchShape(overlay.Name, overlayTarget, i, patch)
			if len(shapeIssues) > 0 {
				issues = append(issues, shapeIssues...)
				continue
			}
			candidate := deepCopyMap(result)
			next, err := applyOverlayPatch(candidate, patch)
			if err != nil {
				issues = append(issues, patchIssue(overlay.Name, overlayTarget, i, "patch_apply_failed", err.Error()))
				continue
			}
			result = next
		}
	}
	return result, issues
}

func applyOverlayPatch(root map[string]any, patch OverlayPatch) (map[string]any, error) {
	op := strings.TrimSpace(patch.Op)
	path, err := parsePointer(patch.Path)
	if err != nil {
		return nil, err
	}
	switch op {
	case patchOpAdd:
		updated, err := setValue(root, path, deepCopyAny(patch.Value), setModeAdd)
		return asRootMap(updated, err)
	case patchOpReplace:
		updated, err := setValue(root, path, deepCopyAny(patch.Value), setModeReplace)
		return asRootMap(updated, err)
	case patchOpRemove:
		updated, err := removeValue(root, path)
		return asRootMap(updated, err)
	case patchOpCopy:
		from, err := parsePointer(patch.From)
		if err != nil {
			return nil, err
		}
		value, ok := getValue(root, from)
		if !ok {
			return nil, fmt.Errorf("source path %q was not found", patch.From)
		}
		updated, err := setValue(root, path, deepCopyAny(value), setModeAdd)
		return asRootMap(updated, err)
	case patchOpMove:
		from, err := parsePointer(patch.From)
		if err != nil {
			return nil, err
		}
		if pathDescendsFrom(path, from) {
			return nil, fmt.Errorf("destination path %q cannot be inside source path %q", patch.Path, patch.From)
		}
		value, ok := getValue(root, from)
		if !ok {
			return nil, fmt.Errorf("source path %q was not found", patch.From)
		}
		withoutSource, err := removeValue(root, from)
		if err != nil {
			return nil, err
		}
		updated, err := setValue(withoutSource, path, deepCopyAny(value), setModeAdd)
		return asRootMap(updated, err)
	case patchOpTest:
		value, ok := getValue(root, path)
		if !ok {
			return nil, fmt.Errorf("path %q was not found", patch.Path)
		}
		if !reflect.DeepEqual(value, patch.Value) {
			return nil, fmt.Errorf("path %q value did not match test value", patch.Path)
		}
		return root, nil
	default:
		return nil, fmt.Errorf("unsupported patch op %q", patch.Op)
	}
}

func pathDescendsFrom(path, from []string) bool {
	if len(path) <= len(from) {
		return false
	}
	for i := range from {
		if path[i] != from[i] {
			return false
		}
	}
	return true
}

func parsePointer(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("path %q must begin with /", path)
	}
	rawParts := strings.Split(path[1:], "/")
	parts := make([]string, 0, len(rawParts))
	for _, raw := range rawParts {
		if raw == "" {
			return nil, fmt.Errorf("path %q contains an empty segment", path)
		}
		part, err := decodePointerSegment(raw)
		if err != nil {
			return nil, fmt.Errorf("path %q has invalid segment %q: %w", path, raw, err)
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func decodePointerSegment(segment string) (string, error) {
	if !strings.Contains(segment, "~") {
		return segment, nil
	}
	var builder strings.Builder
	builder.Grow(len(segment))
	for i := 0; i < len(segment); i++ {
		if segment[i] != '~' {
			builder.WriteByte(segment[i])
			continue
		}
		if i+1 >= len(segment) {
			return "", fmt.Errorf("dangling escape")
		}
		switch segment[i+1] {
		case '0':
			builder.WriteByte('~')
		case '1':
			builder.WriteByte('/')
		default:
			return "", fmt.Errorf("unsupported escape ~%c", segment[i+1])
		}
		i++
	}
	return builder.String(), nil
}

type patchSetMode int

const (
	setModeAdd patchSetMode = iota
	setModeReplace
)

func setValue(node any, parts []string, value any, mode patchSetMode) (any, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("root replacement is not supported")
	}
	switch typed := node.(type) {
	case map[string]any:
		key := parts[0]
		if len(parts) == 1 {
			if mode == setModeReplace {
				if _, exists := typed[key]; !exists {
					return nil, fmt.Errorf("path %q was not found", "/"+strings.Join(parts, "/"))
				}
			}
			typed[key] = value
			return typed, nil
		}
		child, exists := typed[key]
		if !exists || child == nil {
			return nil, fmt.Errorf("path parent %q was not found", key)
		}
		updated, err := setValue(child, parts[1:], value, mode)
		if err != nil {
			return nil, err
		}
		typed[key] = updated
		return typed, nil
	case []any:
		if len(parts) == 1 {
			switch mode {
			case setModeAdd:
				index, err := arrayIndex(parts[0], len(typed), true)
				if err != nil {
					return nil, err
				}
				if index == len(typed) {
					return append(typed, value), nil
				}
				typed = append(typed, nil)
				copy(typed[index+1:], typed[index:])
				typed[index] = value
				return typed, nil
			case setModeReplace:
				index, err := arrayIndex(parts[0], len(typed), false)
				if err != nil {
					return nil, err
				}
				typed[index] = value
				return typed, nil
			default:
				return nil, fmt.Errorf("unsupported set mode %d", mode)
			}
		}
		index, err := arrayIndex(parts[0], len(typed), false)
		if err != nil {
			return nil, err
		}
		updated, err := setValue(typed[index], parts[1:], value, mode)
		if err != nil {
			return nil, err
		}
		typed[index] = updated
		return typed, nil
	default:
		return nil, fmt.Errorf("path parent %q is not an object or array", parts[0])
	}
}

func removeValue(node any, parts []string) (any, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("root removal is not supported")
	}
	switch typed := node.(type) {
	case map[string]any:
		key := parts[0]
		if len(parts) == 1 {
			if _, exists := typed[key]; !exists {
				return nil, fmt.Errorf("path %q was not found", "/"+key)
			}
			delete(typed, key)
			return typed, nil
		}
		child, exists := typed[key]
		if !exists {
			return nil, fmt.Errorf("path parent %q was not found", key)
		}
		updated, err := removeValue(child, parts[1:])
		if err != nil {
			return nil, err
		}
		typed[key] = updated
		return typed, nil
	case []any:
		index, err := arrayIndex(parts[0], len(typed), false)
		if err != nil {
			return nil, err
		}
		if len(parts) == 1 {
			return append(typed[:index], typed[index+1:]...), nil
		}
		updated, err := removeValue(typed[index], parts[1:])
		if err != nil {
			return nil, err
		}
		typed[index] = updated
		return typed, nil
	default:
		return nil, fmt.Errorf("path parent %q is not an object or array", parts[0])
	}
}

func getValue(node any, parts []string) (any, bool) {
	if len(parts) == 0 {
		return node, true
	}
	switch typed := node.(type) {
	case map[string]any:
		child, ok := typed[parts[0]]
		if !ok {
			return nil, false
		}
		return getValue(child, parts[1:])
	case []any:
		index, err := arrayIndex(parts[0], len(typed), false)
		if err != nil {
			return nil, false
		}
		return getValue(typed[index], parts[1:])
	default:
		return nil, false
	}
}

func arrayIndex(part string, length int, allowEnd bool) (int, error) {
	if part == "-" {
		if allowEnd {
			return length, nil
		}
		return 0, fmt.Errorf("array index - is only valid for add")
	}
	index, err := strconv.Atoi(part)
	if err != nil {
		return 0, fmt.Errorf("array index %q is not numeric", part)
	}
	if index < 0 {
		return 0, fmt.Errorf("array index %d is negative", index)
	}
	if allowEnd {
		if index > length {
			return 0, fmt.Errorf("array index %d exceeds length %d", index, length)
		}
		return index, nil
	}
	if index >= length {
		return 0, fmt.Errorf("array index %d exceeds length %d", index, length)
	}
	return index, nil
}

func asRootMap(node any, err error) (map[string]any, error) {
	if err != nil {
		return nil, err
	}
	root, ok := node.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("root value is not an object")
	}
	return root, nil
}

func deepCopyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for key, value := range m {
		out[key] = deepCopyAny(value)
	}
	return out
}

func deepCopyAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return deepCopyMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = deepCopyAny(item)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(typed))
		maps.Copy(out, typed)
		return out
	case []string:
		return slices.Clone(typed)
	default:
		return value
	}
}
