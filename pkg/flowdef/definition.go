// Package flowdef defines neutral, reusable flow contracts.
//
// A flow definition describes what user or API behavior exists and how it can
// be exercised across UI and API channels. Execution, browser automation,
// coverage reporting, and environment setup belong in downstream test kits.
package flowdef

// definition.go owns the neutral flow contract consumed by module authoring,
// UI/API verification, and downstream E2E/testkit bridges.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// ChannelKind identifies an execution channel that can satisfy a flow.
type ChannelKind string

const (
	// ChannelUI exercises a flow through a user interface.
	ChannelUI ChannelKind = "ui"
	// ChannelAPI exercises a flow through an API surface.
	ChannelAPI ChannelKind = "api"
)

// AuthMode describes how a flow channel authenticates.
type AuthMode string

const (
	// AuthModeNone performs the flow channel without authentication.
	AuthModeNone AuthMode = "none"
	// AuthModeSession authenticates the flow channel with a session cookie.
	AuthModeSession AuthMode = "session"
	// AuthModeBearer authenticates the flow channel with a bearer token.
	AuthModeBearer AuthMode = "bearer"
	// AuthModeAPIKey authenticates the flow channel with an API key.
	AuthModeAPIKey AuthMode = "api_key"
)

// Definition is the neutral, cross-layer description of a reusable business
// flow.
type Definition struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Module      string   `json:"module,omitempty"`
	Feature     string   `json:"feature,omitempty"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Provides    []string `json:"provides,omitempty"`
	Requires    []string `json:"requires,omitempty"`
	Components  []string `json:"components,omitempty"`
	UseCases    []string `json:"use_cases,omitempty"`
	Fulfills    []string `json:"fulfills,omitempty"`
	Channels    Channels `json:"channels"`
}

// Validate ensures the definition is structurally coherent.
func (d Definition) Validate() error {
	id := strings.TrimSpace(d.ID)
	if id == "" {
		return fmt.Errorf("flow definition id is required")
	}
	if strings.ContainsAny(id, " \t\n\r") {
		return fmt.Errorf("flow definition id %q must not contain whitespace", id)
	}
	module := strings.TrimSpace(d.Module)
	if module != "" && strings.ContainsAny(module, " \t\n\r") {
		return fmt.Errorf("flow definition %q module %q must not contain whitespace", id, module)
	}
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("flow definition %q name is required", id)
	}
	if strings.TrimSpace(d.Feature) != "" && module == "" {
		return fmt.Errorf("flow definition %q feature requires module", id)
	}
	if err := d.Channels.Validate(); err != nil {
		return fmt.Errorf("flow definition %q channels: %w", id, err)
	}
	return nil
}

// HasChannel reports whether the definition supports the requested channel.
func (d Definition) HasChannel(kind ChannelKind) bool {
	return d.Channels.Has(kind)
}

// ChannelKinds returns the configured channel kinds in deterministic order.
func (d Definition) ChannelKinds() []ChannelKind {
	return d.Channels.Kinds()
}

// Channels groups supported execution channels for a flow definition.
type Channels struct {
	UI  *UIChannel  `json:"ui,omitempty"`
	API *APIChannel `json:"api,omitempty"`
}

// Validate ensures the channel set is coherent.
func (c Channels) Validate() error {
	if c.UI == nil && c.API == nil {
		return fmt.Errorf("at least one channel is required")
	}
	if c.UI != nil {
		if err := c.UI.Validate(); err != nil {
			return fmt.Errorf("ui: %w", err)
		}
	}
	if c.API != nil {
		if err := c.API.Validate(); err != nil {
			return fmt.Errorf("api: %w", err)
		}
	}
	return nil
}

// Has reports whether the requested channel exists.
func (c Channels) Has(kind ChannelKind) bool {
	switch kind {
	case ChannelUI:
		return c.UI != nil
	case ChannelAPI:
		return c.API != nil
	default:
		return false
	}
}

// Kinds returns configured channel kinds in deterministic order.
func (c Channels) Kinds() []ChannelKind {
	kinds := make([]ChannelKind, 0, 2)
	if c.UI != nil {
		kinds = append(kinds, ChannelUI)
	}
	if c.API != nil {
		kinds = append(kinds, ChannelAPI)
	}
	return kinds
}

// UIChannel describes how a flow is exercised through a user interface.
type UIChannel struct {
	Route              string            `json:"route,omitempty"`
	Page               string            `json:"page,omitempty"`
	Action             string            `json:"action,omitempty"`
	FormFields         []string          `json:"form_fields,omitempty"`
	FormFieldSelectors map[string]string `json:"form_field_selectors,omitempty"`
}

// Validate ensures the UI channel carries enough information to be meaningful.
func (c UIChannel) Validate() error {
	if strings.TrimSpace(c.Route) == "" &&
		strings.TrimSpace(c.Page) == "" &&
		strings.TrimSpace(c.Action) == "" {
		return fmt.Errorf("route, page, or action is required")
	}
	return nil
}

// APIChannel describes how a flow is exercised through an API surface.
type APIChannel struct {
	Steps []APIStep `json:"steps,omitempty"`
}

// Validate ensures the API channel carries the minimum stable binding
// contract.
func (c APIChannel) Validate() error {
	steps := c.NormalizedSteps()
	if len(steps) == 0 {
		return fmt.Errorf("at least one api step is required")
	}
	for idx, step := range steps {
		if err := step.Validate(); err != nil {
			return fmt.Errorf("step %d: %w", idx, err)
		}
	}
	return nil
}

// NormalizedSteps returns cloned API steps in execution order.
func (c APIChannel) NormalizedSteps() []APIStep {
	if len(c.Steps) == 0 {
		return nil
	}
	steps := make([]APIStep, 0, len(c.Steps))
	for _, step := range c.Steps {
		steps = append(steps, cloneAPIStep(step))
	}
	return steps
}

// APIStep describes one concrete API operation within an API flow channel.
type APIStep struct {
	OperationID     string            `json:"operation_id"`
	Method          string            `json:"method"`
	Path            string            `json:"path"`
	Auth            AuthMode          `json:"auth,omitempty"`
	SuccessStatuses []int             `json:"success_statuses,omitempty"`
	PathParams      []string          `json:"path_params,omitempty"`
	QueryParams     []string          `json:"query_params,omitempty"`
	BodyFields      []string          `json:"body_fields,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
}

// Validate ensures the API step carries the minimum stable binding contract.
func (s APIStep) Validate() error {
	if strings.TrimSpace(s.OperationID) == "" {
		return fmt.Errorf("operation_id is required")
	}
	method := strings.ToUpper(strings.TrimSpace(s.Method))
	if method == "" {
		return fmt.Errorf("method is required")
	}
	if !validHTTPMethod(method) {
		return fmt.Errorf("unsupported http method %q", s.Method)
	}
	path := strings.TrimSpace(s.Path)
	if path == "" {
		return fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path %q must start with /", s.Path)
	}
	auth := AuthMode(strings.TrimSpace(string(s.Auth)))
	if auth != "" && !validAuthMode(auth) {
		return fmt.Errorf("unsupported auth mode %q", s.Auth)
	}
	for _, status := range s.SuccessStatuses {
		if status < 100 || status > 599 {
			return fmt.Errorf("invalid success status %d", status)
		}
	}
	return nil
}

func validHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions:
		return true
	default:
		return false
	}
}

func validAuthMode(auth AuthMode) bool {
	switch auth {
	case AuthModeNone, AuthModeSession, AuthModeBearer, AuthModeAPIKey:
		return true
	default:
		return false
	}
}

func cloneAPIStep(step APIStep) APIStep {
	cloned := step
	cloned.SuccessStatuses = slices.Clone(step.SuccessStatuses)
	cloned.PathParams = slices.Clone(step.PathParams)
	cloned.QueryParams = slices.Clone(step.QueryParams)
	cloned.BodyFields = slices.Clone(step.BodyFields)
	cloned.Headers = cloneStringMap(step.Headers)
	return cloned
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
