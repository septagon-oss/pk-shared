package statemachine_test

// example_test.go provides runnable godoc examples for the statemachine
// package's declaration, validation, and traversal surface.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import (
	"fmt"

	"github.com/septagon-oss/pk-shared/pkg/statemachine"
)

// Example declares a minimal two-state lifecycle, validates it, and lists its
// events.
func Example() {
	def := &statemachine.StateMachineDefinition{
		ID:           "door",
		EntityType:   "Door",
		Version:      "1.0.0",
		InitialState: "open",
		States: []statemachine.StateDefinition{
			{
				ID:   "open",
				Type: statemachine.StateTypeAtomic,
				Transitions: []statemachine.TransitionDef{
					{Event: "close", Target: "closed"},
				},
			},
			{
				ID:   "closed",
				Type: statemachine.StateTypeFinal,
			},
		},
	}

	errs := statemachine.Validate(def)
	fmt.Println("valid:", len(errs) == 0)
	fmt.Println("events:", statemachine.AllEvents(def))
	// Output:
	// valid: true
	// events: [close]
}
