// Package statemachine declares a portable, declarative state machine model.
//
// The package is intentionally runtime-free: it defines lifecycle contracts,
// validates their shape, exposes traversal helpers, and can render Mermaid
// diagrams. Execution, persistence, guard evaluation, and action side effects
// belong in downstream runtimes.
package statemachine

// definition.go owns the portable state-machine declaration types and
// defaults shared by modules, tooling, and downstream runtimes.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

// StateType classifies how a state node behaves within the chart.
type StateType string

const (
	// StateTypeAtomic is a leaf state with no children.
	StateTypeAtomic StateType = "atomic"

	// StateTypeCompound has children where exactly one child is active.
	StateTypeCompound StateType = "compound"

	// StateTypeParallel has children that are active at the same time.
	StateTypeParallel StateType = "parallel"

	// StateTypeFinal is terminal and cannot be exited.
	StateTypeFinal StateType = "final"
)

// ActionType classifies an action declared on state entry, state exit, or a
// transition.
type ActionType string

const (
	// ActionTypeEmitEvent emits an event through the hosting runtime.
	ActionTypeEmitEvent ActionType = "emit_event"

	// ActionTypeCallService invokes a service or port method in the hosting
	// runtime.
	ActionTypeCallService ActionType = "call_service"

	// ActionTypeSetField sets a field value on the governed entity.
	ActionTypeSetField ActionType = "set_field"

	// ActionTypeNotify sends a notification through the hosting runtime.
	ActionTypeNotify ActionType = "notify"
)

// StateMachineDefinition is the top-level declaration of an entity lifecycle.
type StateMachineDefinition struct {
	// ID is the unique machine identifier.
	ID string `json:"id"`

	// EntityType is the entity this machine governs.
	EntityType string `json:"entityType"`

	// Module is the owning module ID when the machine belongs to a modular
	// application.
	Module string `json:"module"`

	// Version is the machine contract version.
	Version string `json:"version"`

	// Description is an optional human-readable summary.
	Description string `json:"description,omitempty"`

	// InitialState is the state entered when a new entity is created.
	InitialState string `json:"initialState"`

	// StateField is the entity field that holds the current state value.
	// Defaults to "status" when empty.
	StateField string `json:"stateField"`

	// States is the top-level state list for this machine.
	States []StateDefinition `json:"states"`

	// Metadata holds arbitrary extension data such as documentation links or
	// UI hints.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// EffectiveStateField returns StateField if set, otherwise "status".
func (d *StateMachineDefinition) EffectiveStateField() string {
	if d == nil || d.StateField == "" {
		return "status"
	}
	return d.StateField
}

// StateDefinition describes a single state node within the chart.
type StateDefinition struct {
	// ID is the state identifier.
	ID string `json:"id"`

	// Type classifies the state. Defaults to StateTypeAtomic when empty.
	Type StateType `json:"type"`

	// Description is an optional human-readable summary.
	Description string `json:"description,omitempty"`

	// Transitions lists outbound transitions from this state.
	Transitions []TransitionDef `json:"transitions,omitempty"`

	// OnEntry lists actions executed when entering this state.
	OnEntry []ActionDef `json:"onEntry,omitempty"`

	// OnExit lists actions executed when leaving this state.
	OnExit []ActionDef `json:"onExit,omitempty"`

	// Children holds nested state definitions for compound and parallel states.
	Children []StateDefinition `json:"children,omitempty"`

	// Metadata holds UI hints and extension data.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// EffectiveType returns Type if set, otherwise StateTypeAtomic.
func (s *StateDefinition) EffectiveType() StateType {
	if s == nil || s.Type == "" {
		return StateTypeAtomic
	}
	return s.Type
}

// TransitionDef describes a state transition triggered by a named event.
type TransitionDef struct {
	// Event is the trigger event name.
	Event string `json:"event"`

	// Target is the destination state ID.
	Target string `json:"target"`

	// Guard is an optional expression evaluated by the hosting runtime.
	Guard string `json:"guard,omitempty"`

	// Description is an optional human-readable summary.
	Description string `json:"description,omitempty"`

	// Actions lists actions executed during the transition.
	Actions []ActionDef `json:"actions,omitempty"`

	// RequiresInput indicates whether this transition needs user input.
	RequiresInput bool `json:"requiresInput,omitempty"`

	// InputSchema is a JSON Schema describing required input.
	InputSchema map[string]any `json:"inputSchema,omitempty"`

	// RequiredPermissions lists permission identifiers the actor must hold.
	RequiredPermissions []string `json:"requiredPermissions,omitempty"`

	// Metadata holds UI hints and extension data.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ActionDef describes an action to execute during a transition or on state
// entry/exit.
type ActionDef struct {
	// Type classifies the action.
	Type ActionType `json:"type"`

	// Config holds type-specific configuration interpreted by the hosting
	// runtime.
	Config map[string]any `json:"config"`
}
