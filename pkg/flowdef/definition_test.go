package flowdef

// definition_test.go validates reusable flow definitions, channel validation,
// canonical JSON shape, and defensive copies.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import (
	"encoding/json"
	"testing"
)

func TestDefinitionValidateAcceptsDualPathDefinition(t *testing.T) {
	t.Parallel()

	def := Definition{
		ID:          "auth_management.authentication.login",
		Name:        "Authenticate user",
		Description: "Signs a user in through either UI or API.",
		Module:      "auth_management",
		Feature:     "authentication",
		Channels: Channels{
			UI: &UIChannel{
				Route:      "/login",
				Page:       `[data-page="login"]`,
				Action:     `[data-action="login"]`,
				FormFields: []string{"email", "password"},
				FormFieldSelectors: map[string]string{
					"email":    `input[name="email"]`,
					"password": `input[name="password"]`,
				},
			},
			API: &APIChannel{
				Steps: []APIStep{{
					OperationID:     "auth.login",
					Method:          "POST",
					Path:            "/api/v1/auth/login",
					Auth:            AuthModeNone,
					SuccessStatuses: []int{200, 201},
					BodyFields:      []string{"email", "password"},
				}},
			},
		},
	}

	if err := def.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !def.HasChannel(ChannelUI) || !def.HasChannel(ChannelAPI) {
		t.Fatalf("expected both UI and API channels")
	}
	assertChannelKinds(t, def.ChannelKinds(), []ChannelKind{ChannelUI, ChannelAPI})
}

func TestDefinitionValidateAcceptsMultiStepAPIChannel(t *testing.T) {
	t.Parallel()

	def := Definition{
		ID:      "admin_management.settings.update_settings",
		Name:    "Update Settings",
		Module:  "admin_management",
		Feature: "settings",
		Channels: Channels{
			UI: &UIChannel{Route: "/admin/settings"},
			API: &APIChannel{Steps: []APIStep{
				{OperationID: "updateGeneralSettings", Method: "POST", Path: "/admin/api/settings/general"},
				{OperationID: "updateEmailSettings", Method: "POST", Path: "/admin/api/settings/email"},
			}},
		},
	}

	if err := def.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	steps := def.Channels.API.NormalizedSteps()
	if len(steps) != 2 || steps[1].OperationID != "updateEmailSettings" {
		t.Fatalf("NormalizedSteps() = %#v", steps)
	}
}

func TestDefinitionValidateRejectsInvalidDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		def  Definition
	}{
		{
			name: "missing id",
			def: Definition{
				Name:     "Missing id",
				Channels: Channels{UI: &UIChannel{Route: "/login"}},
			},
		},
		{
			name: "feature without module",
			def: Definition{
				ID:       "auth.login",
				Name:     "Login",
				Feature:  "authentication",
				Channels: Channels{UI: &UIChannel{Route: "/login"}},
			},
		},
		{
			name: "id with whitespace",
			def: Definition{
				ID:       "auth login",
				Name:     "Login",
				Channels: Channels{UI: &UIChannel{Route: "/login"}},
			},
		},
		{
			name: "module with whitespace",
			def: Definition{
				ID:       "auth.login",
				Name:     "Login",
				Module:   "auth management",
				Channels: Channels{UI: &UIChannel{Route: "/login"}},
			},
		},
		{
			name: "no channels",
			def:  Definition{ID: "auth.login", Name: "Login"},
		},
		{
			name: "ui without route page action",
			def: Definition{
				ID:       "auth.login",
				Name:     "Login",
				Channels: Channels{UI: &UIChannel{}},
			},
		},
		{
			name: "api without operation id",
			def: Definition{
				ID:   "auth.login",
				Name: "Login",
				Channels: Channels{API: &APIChannel{Steps: []APIStep{{
					Method: "POST",
					Path:   "/api/v1/auth/login",
				}}}},
			},
		},
		{
			name: "api with unsupported method",
			def: Definition{
				ID:   "auth.login",
				Name: "Login",
				Channels: Channels{API: &APIChannel{Steps: []APIStep{{
					OperationID: "auth.login",
					Method:      "TRACE",
					Path:        "/api/v1/auth/login",
				}}}},
			},
		},
		{
			name: "api with relative path",
			def: Definition{
				ID:   "auth.login",
				Name: "Login",
				Channels: Channels{API: &APIChannel{Steps: []APIStep{{
					OperationID: "auth.login",
					Method:      "POST",
					Path:        "api/v1/auth/login",
				}}}},
			},
		},
		{
			name: "api with invalid auth mode",
			def: Definition{
				ID:   "auth.login",
				Name: "Login",
				Channels: Channels{API: &APIChannel{Steps: []APIStep{{
					OperationID: "auth.login",
					Method:      "POST",
					Path:        "/api/v1/auth/login",
					Auth:        "cookie",
				}}}},
			},
		},
		{
			name: "api with invalid status",
			def: Definition{
				ID:   "auth.login",
				Name: "Login",
				Channels: Channels{API: &APIChannel{Steps: []APIStep{{
					OperationID:     "auth.login",
					Method:          "POST",
					Path:            "/api/v1/auth/login",
					SuccessStatuses: []int{99},
				}}}},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.def.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestAPIChannelJSONUsesCanonicalStepShapeOnly(t *testing.T) {
	t.Parallel()

	channel := APIChannel{
		Steps: []APIStep{{
			OperationID:     "auth.login",
			Method:          "POST",
			Path:            "/api/v1/auth/login",
			Auth:            AuthModeNone,
			SuccessStatuses: []int{200, 201},
			BodyFields:      []string{"email", "password"},
		}},
	}

	data, err := json.Marshal(channel)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if _, ok := payload["steps"]; !ok {
		t.Fatalf("expected serialized api channel to include steps, got %#v", payload)
	}
	for _, forbidden := range []string{"operation_id", "method", "path", "auth", "success_statuses", "path_params", "query_params", "body_fields", "headers"} {
		if _, ok := payload[forbidden]; ok {
			t.Fatalf("expected serialized api channel to avoid top-level field %q, got %#v", forbidden, payload)
		}
	}
}

func TestNormalizedStepsReturnsDeepCopy(t *testing.T) {
	t.Parallel()

	channel := APIChannel{Steps: []APIStep{{
		OperationID:     "auth.login",
		Method:          "POST",
		Path:            "/api/v1/auth/login",
		SuccessStatuses: []int{200},
		Headers:         map[string]string{"X-Test": "1"},
	}}}

	steps := channel.NormalizedSteps()
	steps[0].SuccessStatuses[0] = 201
	steps[0].Headers["X-Test"] = "2"

	if channel.Steps[0].SuccessStatuses[0] != 200 {
		t.Fatalf("NormalizedSteps mutated SuccessStatuses: %#v", channel.Steps[0].SuccessStatuses)
	}
	if channel.Steps[0].Headers["X-Test"] != "1" {
		t.Fatalf("NormalizedSteps mutated Headers: %#v", channel.Steps[0].Headers)
	}
}

func assertChannelKinds(t *testing.T, got, want []ChannelKind) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d = %q, want %q", i, got[i], want[i])
		}
	}
}
