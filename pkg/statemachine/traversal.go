package statemachine

// traversal.go owns read-only traversal and Mermaid projection helpers for
// declarative state-machine definitions.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

var mermaidIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// AllStates returns a flat list of all state IDs, including nested children,
// in depth-first order.
func AllStates(def *StateMachineDefinition) []string {
	if def == nil {
		return nil
	}
	ids := []string{}
	collectStateIDs(def.States, &ids)
	return ids
}

func collectStateIDs(states []StateDefinition, ids *[]string) {
	for _, state := range states {
		*ids = append(*ids, state.ID)
		if len(state.Children) > 0 {
			collectStateIDs(state.Children, ids)
		}
	}
}

// AllEvents returns a deduplicated list of event names across all transitions,
// in the order first encountered.
func AllEvents(def *StateMachineDefinition) []string {
	if def == nil {
		return nil
	}
	events := []string{}
	seen := map[string]struct{}{}
	collectEvents(def.States, seen, &events)
	return events
}

func collectEvents(states []StateDefinition, seen map[string]struct{}, events *[]string) {
	for _, state := range states {
		for _, transition := range state.Transitions {
			event := strings.TrimSpace(transition.Event)
			if event == "" {
				continue
			}
			if _, ok := seen[event]; ok {
				continue
			}
			seen[event] = struct{}{}
			*events = append(*events, event)
		}
		if len(state.Children) > 0 {
			collectEvents(state.Children, seen, events)
		}
	}
}

// FindState searches for a state with the given ID, including nested children.
// It returns nil when the definition or state is absent.
func FindState(def *StateMachineDefinition, stateID string) *StateDefinition {
	if def == nil {
		return nil
	}
	return findStateIn(def.States, stateID)
}

func findStateIn(states []StateDefinition, stateID string) *StateDefinition {
	stateID = strings.TrimSpace(stateID)
	for i := range states {
		if strings.TrimSpace(states[i].ID) == stateID {
			state := cloneState(states[i])
			return &state
		}
		if found := findStateIn(states[i].Children, stateID); found != nil {
			return found
		}
	}
	return nil
}

// AvailableTransitions returns the transitions defined on the given state.
// It returns nil when the definition or state is absent.
func AvailableTransitions(def *StateMachineDefinition, currentState string) []TransitionDef {
	state := FindState(def, currentState)
	if state == nil {
		return nil
	}
	return cloneTransitions(state.Transitions)
}

func cloneState(state StateDefinition) StateDefinition {
	state.Transitions = cloneTransitions(state.Transitions)
	state.OnEntry = cloneActions(state.OnEntry)
	state.OnExit = cloneActions(state.OnExit)
	state.Children = cloneStates(state.Children)
	state.Metadata = deepCopyMap(state.Metadata)
	return state
}

func cloneStates(states []StateDefinition) []StateDefinition {
	if len(states) == 0 {
		return nil
	}
	out := make([]StateDefinition, len(states))
	for i, state := range states {
		out[i] = cloneState(state)
	}
	return out
}

func cloneTransitions(transitions []TransitionDef) []TransitionDef {
	if len(transitions) == 0 {
		return nil
	}
	out := make([]TransitionDef, len(transitions))
	for i, transition := range transitions {
		out[i] = transition
		out[i].Actions = cloneActions(transition.Actions)
		out[i].InputSchema = deepCopyMap(transition.InputSchema)
		out[i].RequiredPermissions = slices.Clone(transition.RequiredPermissions)
		out[i].Metadata = deepCopyMap(transition.Metadata)
	}
	return out
}

func cloneActions(actions []ActionDef) []ActionDef {
	if len(actions) == 0 {
		return nil
	}
	out := make([]ActionDef, len(actions))
	for i, action := range actions {
		out[i] = action
		out[i].Config = deepCopyMap(action.Config)
	}
	return out
}

func deepCopyMap(m map[string]any) map[string]any {
	if len(m) == 0 {
		return nil
	}
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
	case map[string]string:
		return maps.Clone(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = deepCopyAny(item)
		}
		return out
	case []string:
		return slices.Clone(typed)
	default:
		return value
	}
}

// ToMermaid generates a Mermaid state diagram. State IDs that are already valid
// Mermaid identifiers are emitted as-is; other IDs are assigned stable aliases.
func ToMermaid(def *StateMachineDefinition) string {
	var b strings.Builder
	b.WriteString("stateDiagram-v2\n")
	if def == nil {
		return b.String()
	}

	aliases := mermaidAliases(AllStates(def))
	writeMermaidAliases(&b, aliases)

	if def.InitialState != "" {
		fmt.Fprintf(&b, "    [*] --> %s\n", mermaidRef(def.InitialState, aliases))
	}
	writeMermaidTransitions(&b, def.States, aliases)
	writeMermaidFinals(&b, def.States, aliases)
	return b.String()
}

func mermaidAliases(ids []string) map[string]string {
	aliases := map[string]string{}
	used := map[string]struct{}{}
	for _, id := range ids {
		if id == "" || mermaidIdentifierPattern.MatchString(id) {
			used[id] = struct{}{}
			continue
		}
		base := sanitizeMermaidID(id)
		alias := base
		for i := 2; ; i++ {
			if _, exists := used[alias]; !exists {
				break
			}
			alias = fmt.Sprintf("%s_%d", base, i)
		}
		aliases[id] = alias
		used[alias] = struct{}{}
	}
	return aliases
}

func sanitizeMermaidID(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "state"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "state_" + out
	}
	return out
}

func writeMermaidAliases(b *strings.Builder, aliases map[string]string) {
	rawIDs := slices.Sorted(maps.Keys(aliases))
	for _, raw := range rawIDs {
		alias := aliases[raw]
		fmt.Fprintf(b, "    state %q as %s\n", raw, alias)
	}
}

func writeMermaidTransitions(b *strings.Builder, states []StateDefinition, aliases map[string]string) {
	for _, state := range states {
		for _, transition := range state.Transitions {
			fmt.Fprintf(b, "    %s --> %s : %s\n", mermaidRef(state.ID, aliases), mermaidRef(transition.Target, aliases), transition.Event)
		}
		if len(state.Children) > 0 {
			writeMermaidTransitions(b, state.Children, aliases)
		}
	}
}

func writeMermaidFinals(b *strings.Builder, states []StateDefinition, aliases map[string]string) {
	for _, state := range states {
		if state.EffectiveType() == StateTypeFinal {
			fmt.Fprintf(b, "    %s --> [*]\n", mermaidRef(state.ID, aliases))
		}
		if len(state.Children) > 0 {
			writeMermaidFinals(b, state.Children, aliases)
		}
	}
}

func mermaidRef(id string, aliases map[string]string) string {
	if alias, ok := aliases[id]; ok {
		return alias
	}
	return id
}
