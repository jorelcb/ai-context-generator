# cqrs-command — Reference

## The CQRS spectrum (and our chosen point)

1. **Model separation** ← _this skill's scope._ Separate command/query handlers and DTOs over the **same** database. Cheap, high value.
2. Separate read models / projections over the same store. Optional optimization.
3. Separate write and read **stores** + event sourcing. **Out of scope here** — large operational cost; only for specific high-scale/audit needs. If a project genuinely needs it, treat it as a dedicated design effort, not a default.

Default to level 1. Do not introduce event sourcing because "CQRS".

## Direct invocation vs command bus

|        | Direct invocation                    | Command bus / mediator                  |
| ------ | ------------------------------------ | --------------------------------------- |
| Wiring | Primary adapter calls handler method | Adapter dispatches a command object     |
| When   | Default; few handlers                | Many handlers needing shared middleware |
| Pros   | Simple, explicit, easy to trace      | Centralized logging/retry/tx middleware |
| Cons   | Cross-cutting logic repeated         | Indirection, harder to navigate         |

Start direct. Add a bus only when cross-cutting concerns repeat across many handlers.

## Validation placement

- **Boundary (DTO)**: structural — required fields, formats, ranges. Reject malformed input early.
- **Domain**: business rules and invariants ([[ddd-entity]]) — uniqueness, state transitions, consistency.
  Never duplicate business rules at the boundary; the domain is the source of truth.

## Error strategy

- Expected business outcomes (not found, invalid transition, conflict) → **domain error / Result type** the handler returns; the primary adapter maps it to a transport status (404/409/422).
- Programmer errors / truly exceptional infra failures → panic (Go) / throw (TS).
- Keep error types in the domain so handlers and adapters share one vocabulary.

## How this maps to the layers

- A **command/query handler = Application Service** ([[clean-arch-layer]]).
- Its dependencies are **driven ports** ([[hexagonal-port]]); it never touches DB/HTTP directly (except a read port for queries).
- The handler is the driving-side entry; per the optional-interface rule, give it an interface only if more than one primary adapter needs it.

## Read side

Queries may use a dedicated **read port** returning DTOs directly (e.g. a `OrderReadModel` that issues a tailored SQL projection), bypassing aggregate reconstruction. This keeps reads fast without polluting the write model.
