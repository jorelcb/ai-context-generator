# saga-orchestrator — Reference

## Orchestration vs choreography — decision table

| Question                                       | Choreography (events only)              | Orchestration (central saga)                                   |
| ---------------------------------------------- | --------------------------------------- | -------------------------------------------------------------- |
| Steps in the flow                              | 2, linear                               | 3+ or branching                                                |
| Compensation needed?                           | No / trivial                            | Yes                                                            |
| "Where is order 123 stuck?" must be answerable | Hard — state is implicit                | One row in the saga table                                      |
| Flow changes often                             | Every service changes                   | One component changes                                          |
| Coupling concern                               | Services coupled to each other's events | Services coupled only to commands/events of the orchestrator   |
| Risk                                           | Hidden cycles, emergent behavior        | Orchestrator becomes a god-object if it absorbs business rules |

Default: choreography for trivial reactive pairs, orchestration as soon as compensation or branching appears. The orchestrator owns **sequencing only** — the moment it starts deciding _whether_ (not _when_), move that rule into the relevant aggregate.

## Why a saga is not a distributed transaction (and what you give up)

| 2PC / distributed tx promises | Saga reality                                                  |
| ----------------------------- | ------------------------------------------------------------- |
| Atomicity (all or nothing)    | Steps commit independently; undo = compensation               |
| Isolation                     | **None** — intermediate states are visible to everyone        |
| Rollback                      | Compensation: a _new forward action_ that semantically undoes |

The missing isolation is the sharp edge: another process can observe (and act on) `stock reserved, payment pending`. Countermeasures: **semantic locks** (mark the reservation as `pending-saga`), **commutative design** (steps safe in any interleaving), or **rereading state before compensating**.

## Compensation design patterns

- **Perfect inverse**: `ReserveStock` ↔ `ReleaseStock`. Prefer steps shaped this way.
- **Corrective action**: can't unsend an email → send a correction; can't uncapture some payments → refund as a new transaction.
- **Reorder for compensability**: do the hardest-to-undo step **last** (classic: send the confirmation email only after payment succeeded — then it never needs compensation).
- **Pivot step**: the step after which the saga can only go forward (retry until success) because earlier steps are no longer compensable. Identify it explicitly in the design.
- Compensations must themselves be **idempotent commands** with IDs derived from `(saga_id, step, "compensate")` — they get retried too ([[event-idempotency]]).

## Timeouts — the verify-before-compensate rule

A timeout means "no answer", not "it failed". If the timed-out action might have happened (a payment!), the saga must first dispatch a **status query / verification command** and only compensate on confirmed failure or confirmed-unknowable. Compensating a payment that actually succeeded = double refund risk; assuming it failed and retrying = double charge risk. For actions that are cheap to redo and idempotent, skip verification and just retry.

Implement timeouts as **scheduled messages** (broker delayed delivery, or a `saga_timeouts` table polled by the relay) recorded in the same transaction as the step transition — never as in-process timers.

## Saga state storage

- A `sagas` table (id, type, status, current_step, state JSON, version, timestamps) + append-only `saga_transitions` for audit.
- **OCC on the saga row** (`WHERE version = $expected`): two concurrent events targeting the same saga serialize; the loser re-reads and usually finds its step already done.
- Saga transition + outbox command dispatch = **one transaction** (the saga is itself a command producer and must not dual-write — [[domain-event]] outbox rules apply).

## Hand-rolled vs workflow engine

| Hand-rolled saga (this skill)                 | Workflow engine (Temporal, Camunda, AWS Step Functions)        |
| --------------------------------------------- | -------------------------------------------------------------- |
| No new infra; full control; transparent state | Durable timers, retries, history, versioning built in          |
| You implement timeouts, retries, audit        | Operational + conceptual surface; vendor coupling              |
| Fine for a handful of saga types              | Worth it when sagas are numerous, long (days+), or audit-heavy |

The step contract in the SKILL is engine-agnostic — adopting an engine later changes the runtime, not the design.

## Testing

- **Unit**: the saga decision function is pure — `(state, event) → (newState, commands)`. Table-test every transition, including failure events and timeouts. No infra needed.
- **Compensation tests are mandatory**: assert reverse order, assert the pivot step never compensates.
- **Redelivery test**: feed the same success event twice; assert the next command is dispatched once.
- **Integration (testcontainers)**: Postgres + broker; run the full flow with a stubbed payment service that declines, assert stock ends released and saga ends `compensated`.
