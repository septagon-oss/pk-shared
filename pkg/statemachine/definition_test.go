package statemachine

// definition_test.go validates state-machine defaults, traversal helpers,
// Mermaid rendering, and aliasing behavior.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import (
	"strings"
	"testing"
)

func bookingLifecycle() *StateMachineDefinition {
	return &StateMachineDefinition{
		ID:           "booking_lifecycle",
		EntityType:   "Booking",
		Module:       "booking_management",
		Version:      "1.0.0",
		Description:  "Standard booking lifecycle",
		InitialState: "draft",
		StateField:   "status",
		States: []StateDefinition{
			{
				ID:          "draft",
				Type:        StateTypeAtomic,
				Description: "Newly created booking",
				Transitions: []TransitionDef{
					{Event: "request", Target: "requested", Description: "Submit booking for approval"},
					{Event: "cancel", Target: "cancelled", Description: "Cancel before submission"},
				},
				OnEntry: []ActionDef{
					{Type: ActionTypeSetField, Config: map[string]any{"field": "createdAt", "value": "now()"}},
				},
			},
			{
				ID:          "requested",
				Type:        StateTypeAtomic,
				Description: "Awaiting confirmation",
				Transitions: []TransitionDef{
					{
						Event:               "confirm",
						Target:              "confirmed",
						Guard:               "entity.totalAmount > 0",
						RequiredPermissions: []string{"booking:confirm"},
						Actions: []ActionDef{
							{Type: ActionTypeEmitEvent, Config: map[string]any{"event": "booking.confirmed"}},
							{Type: ActionTypeNotify, Config: map[string]any{"template": "booking_confirmed", "channels": []string{"email"}}},
						},
						Metadata: map[string]any{"variant": "primary", "label": "Confirm"},
					},
					{Event: "cancel", Target: "cancelled"},
				},
			},
			{
				ID:          "confirmed",
				Type:        StateTypeAtomic,
				Description: "Booking confirmed",
				Transitions: []TransitionDef{
					{
						Event:         "complete",
						Target:        "completed",
						RequiresInput: true,
						InputSchema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"notes": map[string]any{"type": "string"},
							},
						},
					},
					{Event: "cancel", Target: "cancelled"},
				},
			},
			{ID: "completed", Type: StateTypeFinal},
			{ID: "cancelled", Type: StateTypeFinal},
		},
	}
}

func TestEffectiveStateField(t *testing.T) {
	t.Parallel()

	def := &StateMachineDefinition{StateField: "phase"}
	if got := def.EffectiveStateField(); got != "phase" {
		t.Fatalf("EffectiveStateField() = %q", got)
	}

	def.StateField = ""
	if got := def.EffectiveStateField(); got != "status" {
		t.Fatalf("EffectiveStateField() = %q", got)
	}

	var nilDefinition *StateMachineDefinition
	if got := nilDefinition.EffectiveStateField(); got != "status" {
		t.Fatalf("nil EffectiveStateField() = %q", got)
	}
}

func TestEffectiveType(t *testing.T) {
	t.Parallel()

	state := StateDefinition{ID: "a"}
	if got := state.EffectiveType(); got != StateTypeAtomic {
		t.Fatalf("EffectiveType() = %q", got)
	}

	state.Type = StateTypeCompound
	if got := state.EffectiveType(); got != StateTypeCompound {
		t.Fatalf("EffectiveType() = %q", got)
	}
}

func TestTraversal(t *testing.T) {
	t.Parallel()

	def := bookingLifecycle()
	def.States[1].Transitions[0].Metadata["labels"] = map[string]string{"tone": "primary"}
	assertStrings(t, AllStates(def), []string{"draft", "requested", "confirmed", "completed", "cancelled"})
	assertStrings(t, AllEvents(def), []string{"request", "cancel", "confirm", "complete"})

	state := FindState(def, "confirmed")
	if state == nil || state.ID != "confirmed" {
		t.Fatalf("FindState(confirmed) = %#v", state)
	}

	transitions := AvailableTransitions(def, "draft")
	if len(transitions) != 2 || transitions[0].Event != "request" || transitions[1].Event != "cancel" {
		t.Fatalf("AvailableTransitions(draft) = %#v", transitions)
	}
	transitions[0].Event = "mutated"
	if def.States[0].Transitions[0].Event != "request" {
		t.Fatalf("AvailableTransitions returned aliased transitions: %#v", def.States[0].Transitions)
	}
	state.Transitions[0].Event = "mutated"
	if def.States[2].Transitions[0].Event != "complete" {
		t.Fatalf("FindState returned aliased state: %#v", def.States[2].Transitions)
	}
	if got := AvailableTransitions(def, "nonexistent"); got != nil {
		t.Fatalf("AvailableTransitions(nonexistent) = %#v", got)
	}

	requested := FindState(def, "requested")
	requested.Transitions[0].Metadata["labels"].(map[string]string)["tone"] = "mutated"
	if def.States[1].Transitions[0].Metadata["labels"].(map[string]string)["tone"] != "primary" {
		t.Fatalf("FindState returned aliased nested metadata: %#v", def.States[1].Transitions[0].Metadata)
	}
}

func TestTraversalNested(t *testing.T) {
	t.Parallel()

	def := &StateMachineDefinition{
		ID:           "nested",
		InitialState: "active",
		States: []StateDefinition{
			{
				ID:   "active",
				Type: StateTypeCompound,
				Children: []StateDefinition{
					{ID: "sub_a"},
					{ID: "sub_b"},
				},
			},
			{ID: "done", Type: StateTypeFinal},
		},
	}

	assertStrings(t, AllStates(def), []string{"active", "sub_a", "sub_b", "done"})
	if state := FindState(def, " sub_b "); state == nil || state.ID != "sub_b" {
		t.Fatalf("FindState(sub_b) = %#v", state)
	}
}

func TestToMermaid(t *testing.T) {
	t.Parallel()

	mermaid := ToMermaid(bookingLifecycle())
	for _, want := range []string{
		"stateDiagram-v2",
		"[*] --> draft",
		"draft --> requested : request",
		"draft --> cancelled : cancel",
		"requested --> confirmed : confirm",
		"requested --> cancelled : cancel",
		"confirmed --> completed : complete",
		"confirmed --> cancelled : cancel",
		"completed --> [*]",
		"cancelled --> [*]",
	} {
		if !containsLine(mermaid, want) {
			t.Fatalf("ToMermaid() missing %q\n%s", want, mermaid)
		}
	}
}

func TestToMermaidAliasesUnsafeStateIDs(t *testing.T) {
	t.Parallel()

	def := &StateMachineDefinition{
		InitialState: "in review",
		States: []StateDefinition{
			{ID: "in review", Transitions: []TransitionDef{{Event: "approve", Target: "done-state"}}},
			{ID: "done-state", Type: StateTypeFinal},
		},
	}

	mermaid := ToMermaid(def)
	for _, want := range []string{
		`state "in review" as in_review`,
		`state "done-state" as done_state`,
		"[*] --> in_review",
		"in_review --> done_state : approve",
		"done_state --> [*]",
	} {
		if !strings.Contains(mermaid, want) {
			t.Fatalf("ToMermaid() missing %q\n%s", want, mermaid)
		}
	}
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func containsLine(output, want string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}
