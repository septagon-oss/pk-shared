package statemachine

// validation.go owns structural diagnostics for portable state-machine
// definitions without executing guards, actions, or persistence behavior.
//
// ADR: ADR-0005 (no silent failures), ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import (
	"fmt"
	"strings"
)

const (
	CodeNilDefinition       = "nil_definition"
	CodeNoStates            = "no_states"
	CodeMissingInitialState = "missing_initial_state"
	CodeMissingStateID      = "missing_state_id"
	CodeDuplicateID         = "duplicate_id"
	CodeInvalidStateType    = "invalid_state_type"
	CodeMissingTarget       = "missing_target"
	CodeMissingEvent        = "missing_event"
	CodeFinalHasTransitions = "final_has_transitions"
	CodeCompoundNoChildren  = "compound_no_children"
	CodeParallelNoChildren  = "parallel_no_children"
	CodeInvalidActionType   = "invalid_action_type"
	CodeMissingInputSchema  = "missing_input_schema"
	CodeMissingPermission   = "missing_permission"
	CodeMissingActionConfig = "missing_action_config"
)

// ValidationError describes one structural problem in a state machine
// definition.
type ValidationError struct {
	// StateID identifies the state where the error was found, if applicable.
	StateID string `json:"stateId,omitempty"`

	// TransitionEvent identifies the transition event, if applicable.
	TransitionEvent string `json:"transitionEvent,omitempty"`

	// Code is a stable machine-readable error classification.
	Code string `json:"code"`

	// Message is a human-readable description of the problem.
	Message string `json:"message"`
}

// Error implements error for a single validation error.
func (e ValidationError) Error() string {
	return e.Message
}

// Validate checks a StateMachineDefinition for structural correctness and
// returns every error found. An empty slice means the definition is valid.
func Validate(def *StateMachineDefinition) []ValidationError {
	if def == nil {
		return []ValidationError{{Code: CodeNilDefinition, Message: "state machine definition is nil"}}
	}
	if len(def.States) == 0 {
		return []ValidationError{{Code: CodeNoStates, Message: "state machine must have at least one state"}}
	}

	errs := []ValidationError{}
	stateIDs := AllStates(def)
	seen := make(map[string]bool, len(stateIDs))
	for _, id := range stateIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			errs = append(errs, ValidationError{
				Code:    CodeMissingStateID,
				Message: "state ID is required",
			})
			continue
		}
		if seen[trimmed] {
			errs = append(errs, ValidationError{
				StateID: trimmed,
				Code:    CodeDuplicateID,
				Message: fmt.Sprintf("duplicate state ID %q", trimmed),
			})
		}
		seen[trimmed] = true
	}

	initialState := strings.TrimSpace(def.InitialState)
	if !seen[initialState] {
		errs = append(errs, ValidationError{
			Code:    CodeMissingInitialState,
			Message: fmt.Sprintf("initial state %q does not exist", def.InitialState),
		})
	}

	return validateStates(def.States, seen, errs)
}

func validateStates(states []StateDefinition, validIDs map[string]bool, errs []ValidationError) []ValidationError {
	for _, state := range states {
		stateID := strings.TrimSpace(state.ID)
		stateType := state.EffectiveType()
		if !validStateType(stateType) {
			errs = append(errs, ValidationError{
				StateID: stateID,
				Code:    CodeInvalidStateType,
				Message: fmt.Sprintf("state %q has invalid type %q", stateID, stateType),
			})
		}

		switch stateType {
		case StateTypeFinal:
			if len(state.Transitions) > 0 {
				errs = append(errs, ValidationError{
					StateID: stateID,
					Code:    CodeFinalHasTransitions,
					Message: fmt.Sprintf("final state %q must not have outbound transitions", stateID),
				})
			}
		case StateTypeCompound:
			if len(state.Children) == 0 {
				errs = append(errs, ValidationError{
					StateID: stateID,
					Code:    CodeCompoundNoChildren,
					Message: fmt.Sprintf("compound state %q must have at least one child", stateID),
				})
			}
		case StateTypeParallel:
			if len(state.Children) == 0 {
				errs = append(errs, ValidationError{
					StateID: stateID,
					Code:    CodeParallelNoChildren,
					Message: fmt.Sprintf("parallel state %q must have at least one child", stateID),
				})
			}
		}

		errs = validateActions(stateID, "", "onEntry", state.OnEntry, errs)
		errs = validateActions(stateID, "", "onExit", state.OnExit, errs)

		for _, transition := range state.Transitions {
			errs = validateTransition(stateID, transition, validIDs, errs)
		}

		if len(state.Children) > 0 {
			errs = validateStates(state.Children, validIDs, errs)
		}
	}
	return errs
}

func validateTransition(stateID string, transition TransitionDef, validIDs map[string]bool, errs []ValidationError) []ValidationError {
	event := strings.TrimSpace(transition.Event)
	if event == "" {
		errs = append(errs, ValidationError{
			StateID: stateID,
			Code:    CodeMissingEvent,
			Message: fmt.Sprintf("transition in state %q is missing an event", stateID),
		})
	}

	target := strings.TrimSpace(transition.Target)
	if target == "" || !validIDs[target] {
		errs = append(errs, ValidationError{
			StateID:         stateID,
			TransitionEvent: transition.Event,
			Code:            CodeMissingTarget,
			Message:         fmt.Sprintf("transition %q in state %q targets non-existent state %q", transition.Event, stateID, transition.Target),
		})
	}

	if transition.RequiresInput && len(transition.InputSchema) == 0 {
		errs = append(errs, ValidationError{
			StateID:         stateID,
			TransitionEvent: transition.Event,
			Code:            CodeMissingInputSchema,
			Message:         fmt.Sprintf("transition %q in state %q requires input but has no input schema", transition.Event, stateID),
		})
	}

	for _, permission := range transition.RequiredPermissions {
		if strings.TrimSpace(permission) == "" {
			errs = append(errs, ValidationError{
				StateID:         stateID,
				TransitionEvent: transition.Event,
				Code:            CodeMissingPermission,
				Message:         fmt.Sprintf("transition %q in state %q contains an empty required permission", transition.Event, stateID),
			})
		}
	}

	return validateActions(stateID, transition.Event, "transition", transition.Actions, errs)
}

func validateActions(stateID, event, context string, actions []ActionDef, errs []ValidationError) []ValidationError {
	for _, action := range actions {
		if !validActionType(action.Type) {
			errs = append(errs, ValidationError{
				StateID:         stateID,
				TransitionEvent: event,
				Code:            CodeInvalidActionType,
				Message:         fmt.Sprintf("%s action in state %q has invalid type %q", context, stateID, action.Type),
			})
			continue
		}
		if action.Type != "" && action.Config == nil {
			errs = append(errs, ValidationError{
				StateID:         stateID,
				TransitionEvent: event,
				Code:            CodeMissingActionConfig,
				Message:         fmt.Sprintf("%s action %q in state %q is missing config", context, action.Type, stateID),
			})
		}
	}
	return errs
}

func validStateType(stateType StateType) bool {
	switch stateType {
	case StateTypeAtomic, StateTypeCompound, StateTypeParallel, StateTypeFinal:
		return true
	default:
		return false
	}
}

func validActionType(actionType ActionType) bool {
	switch actionType {
	case ActionTypeEmitEvent, ActionTypeCallService, ActionTypeSetField, ActionTypeNotify:
		return true
	default:
		return false
	}
}
