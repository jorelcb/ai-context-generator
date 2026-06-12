# Saga Orchestrator Skill

> **Scope:** coordinating long-running flows that span multiple aggregates or services. A saga is a **stateful, reactive coordinator**: it subscribes to events ([[domain-event]]), dispatches commands to [[command-handler]]s, tracks which step it is on, and carries explicit **compensation** for every step with side effects. A saga is **NOT a distributed transaction** — there is no atomic commit across services, no global lock, no rollback. Intermediate states are visible and legitimate; the saga's job is to drive the flow to one of exactly two honest outcomes: **completed** or **fully compensated**.

## When to use

A flow needs multiple aggregates to change in coordination (order → stock → payment), an external service participates in the flow, or some steps must be undone when a later step fails. If the whole flow fits in one aggregate, you don't need a saga — use a single [[command-handler]].

## Orchestration vs choreography (decide first)

- **Orchestration** (this skill's default): one saga component owns the flow — it hears events and decides the next command. Use for flows with 3+ steps, branching, or compensation.
- **Choreography**: services react to each other's events with no central coordinator. Acceptable for 2-step linear flows; beyond that the flow logic is smeared across services and nobody can answer "where is order 123 stuck?". The decision table is in [reference.md](reference.md).

## Saga state (normative)

Persist state on **every transition** — a saga that keeps state in memory dies with the process:

- `saga_id` — derived deterministically from the originating event (e.g. `order-fulfillment:{order_id}`), which also dedups duplicate triggers.
- `current_step`, `completed_steps`, `compensation_history`, `started_at`, `updated_at`, `status` (`running | completed | compensating | compensated | failed_needs_attention`).
- Step transitions and dispatched-command records commit atomically (saga state table + outbox, one transaction).

## The step contract (the process)

For each step in the flow:

1. **Trigger**: which event advances the saga into this step (`OrderPlaced` → step `reserve-stock`).
2. **Command**: what the saga dispatches (`ReserveStock`), with a command ID derived from `(saga_id, step)` so retries dedup at the handler ([[event-idempotency]]).
3. **Success event**: what moves to the next step (`StockReserved`).
4. **Failure event**: what triggers compensation (`StockRejected`).
5. **Timeout**: what happens if neither arrives — sagas must own deadlines (schedule a timeout message when entering a step).
6. **Compensation**: the command that undoes this step's side effect (`ReleaseStock`).

## Compensation (normative)

- Every step with side effects gets a compensating action — defined **when the step is designed**, not after the first incident.
- Compensations run in **reverse order** of completed steps.
- Compensations are **semantic undo**, not perfect inverses: you can refund a payment; you cannot unsend an email — for those, design the forward action to be safe (send the email only after the point of no return) or compensate with a corrective action (apology + voucher).
- Compensations are first-class steps: persisted, idempotent, retried. A compensation that fails permanently moves the saga to `failed_needs_attention` and **alerts a human** — document this path; it is part of the design.

## Failure handling (normative)

- **Transient failure** of a step command → retry with backoff (idempotent handlers make this safe).
- **Business rejection** (e.g. `PaymentDeclined`) → start the compensation chain. Not an error — a designed outcome.
- **Timeout** → treat as failure of the step (after checking status if the action might have happened).
- **Compensation failure** → bounded retries, then human escalation. Never silently drop.

## Anti-patterns to avoid

- **Treating the saga as a distributed transaction** — expecting atomicity/isolation across services; intermediate states leak and that's by design.
- **Synchronous saga** — calling services in a blocking chain; if you can afford that coupling, you can probably afford one transaction and no saga.
- **Saga mutating aggregates directly** — it must dispatch commands; the aggregate's [[command-handler]] enforces invariants.
- **Hidden choreography** — command handlers dispatching "the next" command; promote the flow to an explicit saga.
- **"Forget and forgive"** — assuming a failed step will sort itself out without explicit compensation or escalation.
- **Business logic in the saga** — the saga decides _sequencing_; each aggregate decides _whether_ (rules stay in the domain).
- **In-memory saga state** or state updated outside the dispatch transaction.

## Verification

- [ ] Saga ID is deterministic from the trigger; duplicate triggers don't start a second saga.
- [ ] Every step declares trigger, command, success/failure events, timeout, and compensation.
- [ ] Redelivered events don't re-execute completed steps (state-based dedup).
- [ ] Compensation paths have tests, not just the happy path.
- [ ] Final state is always `completed` or `compensated` — or `failed_needs_attention` with an alert.

## Going deeper

- **[reference.md](reference.md)** — orchestration vs choreography decision table, compensation design patterns, timeouts and the verify-before-compensate rule, saga state storage, workflow-engine trade-offs, testing setups.
- **[examples.md](examples.md)** — an order-fulfillment saga (`OrderPlaced` → reserve stock → take payment, with compensation on `PaymentDeclined`), in Go and TypeScript.
