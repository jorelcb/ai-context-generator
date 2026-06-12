# CQRS Command / Query Skill

> **Scope:** simple read/write **model separation**. Event sourcing and separate write/read stores are **out of scope** — don't introduce them here. A command/query handler **is** an Application Service in [[clean-arch-layer]]: a thin orchestrator with no business logic.

## When to use

Adding a new use case, separating a read path from a write path, building a command or query handler, or designing the DTOs that cross the application boundary.

## Commands vs Queries (normative)

- **Command** — _changes state_, returns only a confirmation/identifier (or nothing). Named as an imperative: `PlaceOrder`, `UpdateProfile`.
- **Query** — _reads state_, returns data, **never mutates**. Named as a question/lookup: `GetOrderById`, `ListActiveUsers`.
- **Never mix**: a command must not return domain data for display; a query must not cause side effects.

## Command handler pattern

1. Define the **Command** (input DTO with primitive/transport fields).
2. Create the **handler** (= Application Service) with dependencies injected as **ports** ([[hexagonal-port]]).
3. Orchestrate: **validate input → map to domain → execute domain logic → persist via repository → return id/result**.
4. The handler is the **transaction boundary**.
5. Business rules stay in the domain ([[ddd-entity]]); the handler only coordinates.

## Query handler pattern

1. Define the **Query** (filter / pagination parameters).
2. Create the handler with **read-optimized** dependencies.
3. Return **read DTOs**, never domain entities.
4. Queries **may bypass the domain model** and read directly from a read source for performance — this is allowed (the one place infra-shaped reads are fine, behind a read port).

## DTO design

- **Input DTOs**: validate at the boundary; map to domain objects _inside_ the handler.
- **Output DTOs**: flatten domain complexity for the consumer.
- **Never expose domain entities** through DTOs (no leaking aggregates over the wire).

## Decisions this skill settles

- **Direct invocation by default**; introduce a command bus only when you need cross-cutting middleware (logging, retries, transactions) across many handlers — not pre-emptively.
- **Validation**: structural validation (required, format) at the DTO; business-rule validation in the domain.
- **Errors**: return a domain error / Result for expected business failures (not found, invalid transition); reserve panics/exceptions for truly exceptional cases.

## Anti-patterns to avoid

- Fat command doing many things — split into focused commands.
- Query with side effects.
- **Business logic in the handler** — push it into domain entities/services ([[ddd-entity]]).
- Returning domain entities from a command handler.
- A command bus / mediator added before there's a real cross-cutting need.

## Verification

- [ ] Commands return id/confirmation, not display data.
- [ ] Queries perform zero writes.
- [ ] Handlers are thin: no business rules, only orchestration.
- [ ] No domain entity crosses a DTO boundary.
- [ ] Each command handler owns one transaction.

## Examples

A command handler and a query handler (with DTOs and a read port), in Go and TypeScript: **[examples.md](examples.md)**.

## Reference

Command-bus trade-offs, validation placement, error/Result strategy, and the CQRS spectrum (and why event sourcing is out of scope here): **[reference.md](reference.md)**.
