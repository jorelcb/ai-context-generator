# Command Handler Skill

> **Scope:** command handlers in an **event-driven** architecture — handlers load an aggregate, the aggregate emits domain events ([[domain-event]]), and state + events are persisted through the transactional outbox, with idempotency mandatory ([[event-idempotency]]). If your project is simple CQRS without events or event sourcing, this is the wrong skill: use the analogous `cqrs-command` skill from the clean-ddd pack instead — the two architecture packs are alternatives, not layers.

## When to use

Adding a new use case that mutates state, refactoring a CRUD endpoint into an intent-revealing command, or splitting a service that mixes commands and queries.

## Command naming (normative)

- **Imperative, intent-revealing:** `PlaceOrder`, `CancelOrder`, `ApprovePayment`.
- NEVER `UpdateOrder` / `ChangeOrder` — CRUD verbs hide intent and produce meaningless events downstream.
- One command = one intent. Two intents → two commands.
- A command is a **request** — it can be rejected. Its successful outcome is a fact in past tense: `PlaceOrder` → `OrderPlaced`.

## Handler responsibilities (the process)

1. **Dedup at entry**: check the command ID (or rely on aggregate version) — see [[event-idempotency]].
2. **Validate preconditions**: structure was checked at the transport boundary; here check business preconditions only.
3. **Load the aggregate** from the repository (by ID, at a known version).
4. **Invoke one aggregate method** — on success it records events (e.g. `OrderPlaced`); on rule violation it returns a domain error.
5. **Persist atomically**: aggregate state + drained events into the outbox **in one transaction** (the handler is the transaction boundary).
6. **Return acknowledgement** — success/failure and the aggregate ID. NOT data; reads belong to [[event-projection]] queries.

## What a command handler does NOT do

- Does NOT query read models or projections (stale by design — never a basis for writes).
- Does NOT return domain entities or display DTOs.
- Does NOT publish events directly to the broker — only the outbox relay publishes ([[domain-event]]).
- Does NOT call other use cases or services in sequence — multi-step flows are a [[saga-orchestrator]].
- Does NOT span multiple aggregates in one transaction — the consistency boundary is a single aggregate; cross-aggregate consistency is eventual, via events.

## Idempotency (mandatory)

Commands arrive more than once: client retries, broker redelivery, saga re-dispatch. Pick at least one strategy:

- **Command ID dedup**: deterministic command ID, unique-constraint insert at handler entry; conflict → return previous result.
- **Aggregate version (optimistic concurrency)**: command targets `expected_version`; stale command → rejected, no duplicate effect.
- **Natural invariants**: the aggregate itself refuses a repeat (`Place()` on an already-placed Order is a no-op or domain error).

Details and a decision table in [[event-idempotency]].

## Failure handling (normative)

- **Business rule violation** → domain error, do NOT retry: retrying won't make an empty order valid.
- **Infrastructure failure** (DB down, lock timeout) → safe to retry; idempotency makes the retry harmless.
- **Concurrency conflict** (version mismatch) → reload and re-decide, or surface to the caller; never blind-merge.

## Anti-patterns to avoid

- Query logic inside a command handler (mixing the read path back in).
- Handler chaining — a handler dispatching the "next" command; that's hidden choreography, promote it to a [[saga-orchestrator]].
- Long-running operations (external HTTP calls, waits) inside the handler transaction — offload to a saga step.
- Multi-aggregate transactions "just this once".
- Dual write: publishing to the broker inside the handler instead of via outbox.
- Returning read data from a command "to save a round-trip".

## Verification

- [ ] Command name is imperative and intent-revealing; resulting event is past tense.
- [ ] Handler is thin: business rules live in the aggregate, not the handler.
- [ ] State + events committed in one transaction through the outbox.
- [ ] Replaying the same command produces no duplicate effects (test it).
- [ ] Domain errors are not retried; infra errors are retryable.

## Going deeper

- **[reference.md](reference.md)** — idempotency strategy decision table, optimistic concurrency mechanics, retry/backoff semantics, direct dispatch vs command bus, testing with testcontainers.
- **[examples.md](examples.md)** — `PlaceOrder` handler with dedup, aggregate invocation, and single-transaction outbox write, in Go and TypeScript.
