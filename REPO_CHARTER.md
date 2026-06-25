# pk-shared Charter

## Purpose

Shared vocabulary and small primitives used across PlatformKit OSS repos. Keeps cross-repo coupling explicit and minimal.

## In Scope

- Composition descriptors (`pkg/composition`): module capability declarations
- Contract identifiers (`pkg/contract`): type-safe ID + version pairs
- Flow definitions (`pkg/flowdef`): state machine specification format
- State machine (`pkg/statemachine`): generic FSM with Mermaid rendering

## Out of Scope

- Runtime execution or orchestration
- Business logic or domain types
- Repository, database, or network abstractions

## Dependencies

None (zero-dependency module).
