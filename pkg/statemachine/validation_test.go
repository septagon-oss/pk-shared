package statemachine

// validation_test.go validates state-machine structural diagnostics and error
// formatting for invalid lifecycle definitions.
//
// ADR: ADR-0005 (no silent failures), ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import "testing"

func TestValidateValidDefinition(t *testing.T) {
	t.Parallel()

	if errs := Validate(bookingLifecycle()); len(errs) != 0 {
		t.Fatalf("Validate() returned errors: %#v", errs)
	}
}

func TestValidateStructuralErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		def  *StateMachineDefinition
		code string
	}{
		{
			name: "nil definition",
			def:  nil,
			code: CodeNilDefinition,
		},
		{
			name: "no states",
			def:  &StateMachineDefinition{ID: "empty", InitialState: "draft"},
			code: CodeNoStates,
		},
		{
			name: "missing initial",
			def: &StateMachineDefinition{
				ID:           "bad_initial",
				InitialState: "missing",
				States:       []StateDefinition{{ID: "draft"}},
			},
			code: CodeMissingInitialState,
		},
		{
			name: "duplicate state",
			def: &StateMachineDefinition{
				ID:           "dupes",
				InitialState: "draft",
				States:       []StateDefinition{{ID: "draft"}, {ID: "draft"}},
			},
			code: CodeDuplicateID,
		},
		{
			name: "duplicate state after trim",
			def: &StateMachineDefinition{
				ID:           "trimmed_dupes",
				InitialState: "draft",
				States:       []StateDefinition{{ID: "draft"}, {ID: " draft "}},
			},
			code: CodeDuplicateID,
		},
		{
			name: "missing transition target",
			def: &StateMachineDefinition{
				ID:           "bad_target",
				InitialState: "draft",
				States: []StateDefinition{{
					ID:          "draft",
					Transitions: []TransitionDef{{Event: "go", Target: "nowhere"}},
				}},
			},
			code: CodeMissingTarget,
		},
		{
			name: "final state has transitions",
			def: &StateMachineDefinition{
				ID:           "bad_final",
				InitialState: "done",
				States: []StateDefinition{{
					ID:          "done",
					Type:        StateTypeFinal,
					Transitions: []TransitionDef{{Event: "reopen", Target: "done"}},
				}},
			},
			code: CodeFinalHasTransitions,
		},
		{
			name: "compound no children",
			def: &StateMachineDefinition{
				ID:           "bad_compound",
				InitialState: "parent",
				States:       []StateDefinition{{ID: "parent", Type: StateTypeCompound}},
			},
			code: CodeCompoundNoChildren,
		},
		{
			name: "parallel no children",
			def: &StateMachineDefinition{
				ID:           "bad_parallel",
				InitialState: "parallel_state",
				States:       []StateDefinition{{ID: "parallel_state", Type: StateTypeParallel}},
			},
			code: CodeParallelNoChildren,
		},
		{
			name: "invalid state type",
			def: &StateMachineDefinition{
				ID:           "bad_type",
				InitialState: "draft",
				States:       []StateDefinition{{ID: "draft", Type: "waiting"}},
			},
			code: CodeInvalidStateType,
		},
		{
			name: "missing event",
			def: &StateMachineDefinition{
				ID:           "missing_event",
				InitialState: "draft",
				States: []StateDefinition{{
					ID:          "draft",
					Transitions: []TransitionDef{{Target: "draft"}},
				}},
			},
			code: CodeMissingEvent,
		},
		{
			name: "missing input schema",
			def: &StateMachineDefinition{
				ID:           "missing_input",
				InitialState: "draft",
				States: []StateDefinition{{
					ID:          "draft",
					Transitions: []TransitionDef{{Event: "submit", Target: "draft", RequiresInput: true}},
				}},
			},
			code: CodeMissingInputSchema,
		},
		{
			name: "invalid action type",
			def: &StateMachineDefinition{
				ID:           "invalid_action",
				InitialState: "draft",
				States:       []StateDefinition{{ID: "draft", OnEntry: []ActionDef{{Type: "shell"}}}},
			},
			code: CodeInvalidActionType,
		},
		{
			name: "missing action config",
			def: &StateMachineDefinition{
				ID:           "missing_action_config",
				InitialState: "draft",
				States:       []StateDefinition{{ID: "draft", OnEntry: []ActionDef{{Type: ActionTypeSetField}}}},
			},
			code: CodeMissingActionConfig,
		},
		{
			name: "missing permission",
			def: &StateMachineDefinition{
				ID:           "missing_permission",
				InitialState: "draft",
				States: []StateDefinition{{
					ID: "draft",
					Transitions: []TransitionDef{{
						Event:               "submit",
						Target:              "draft",
						RequiredPermissions: []string{" "},
					}},
				}},
			},
			code: CodeMissingPermission,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assertHasErrorCode(t, Validate(test.def), test.code)
		})
	}
}

func TestValidateNestedDuplicateStateID(t *testing.T) {
	t.Parallel()

	def := &StateMachineDefinition{
		ID:           "nested_dupes",
		InitialState: "parent",
		States: []StateDefinition{
			{
				ID:       "parent",
				Type:     StateTypeCompound,
				Children: []StateDefinition{{ID: "child"}},
			},
			{ID: "child"},
		},
	}

	assertHasErrorCode(t, Validate(def), CodeDuplicateID)
}

func TestValidateReportsMultipleErrors(t *testing.T) {
	t.Parallel()

	def := &StateMachineDefinition{
		ID:           "many_problems",
		InitialState: "missing",
		States: []StateDefinition{
			{ID: "a", Transitions: []TransitionDef{{Event: "go", Target: "ghost"}}},
			{ID: "a"},
			{ID: "done", Type: StateTypeFinal, Transitions: []TransitionDef{{Event: "nope", Target: "a"}}},
		},
	}
	errs := Validate(def)
	for _, code := range []string{CodeMissingInitialState, CodeDuplicateID, CodeMissingTarget, CodeFinalHasTransitions} {
		assertHasErrorCode(t, errs, code)
	}
}

func TestValidationErrorImplementsError(t *testing.T) {
	t.Parallel()

	err := error(ValidationError{Code: "test", Message: "something went wrong"})
	if err.Error() != "something went wrong" {
		t.Fatalf("Error() = %q", err.Error())
	}
}

func assertHasErrorCode(t *testing.T, errs []ValidationError, code string) {
	t.Helper()
	for _, err := range errs {
		if err.Code == code {
			return
		}
	}
	t.Fatalf("expected code %q in %#v", code, errs)
}
