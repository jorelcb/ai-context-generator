# command-handler — Reference

## Idempotency strategy — which one for this handler?

| Strategy                | Mechanics                                                 | Best when                                  | Watch out for                                  |
| ----------------------- | --------------------------------------------------------- | ------------------------------------------ | ---------------------------------------------- |
| Command ID dedup        | Unique insert of `command_id` in the handler tx           | Client/saga retries with a stable ID       | Needs the caller to send a deterministic ID    |
| Aggregate version (OCC) | `UPDATE ... WHERE version = $expected`; 0 rows = conflict | Concurrent writers on the same aggregate   | Caller must handle the conflict path           |
| Natural invariant       | Aggregate method is a no-op / errors on repeat            | State machines (`Place()` on placed order) | Only covers repeats, not concurrent interleave |

Defense in depth: OCC on the aggregate **plus** command-ID dedup for externally-originated commands is the common production combination. Full treatment in [[event-idempotency]].

## Optimistic concurrency mechanics

1. Repository load returns the aggregate **and** its `version`.
2. The save runs `UPDATE orders SET ..., version = version + 1 WHERE id = $1 AND version = $2`.
3. Zero rows affected = someone else committed first → return a conflict error.
4. Caller policy: **reload-and-retry** for commutative commands (safe to re-decide), **surface to user** for conflicting intent (two people editing the same thing).

With an event store instead of state rows, the same idea is `AppendToStream(streamID, expectedVersion, events)` — the store rejects on mismatch.

## Retry and backoff semantics

| Failure class              | Retry? | How                                                     |
| -------------------------- | ------ | ------------------------------------------------------- |
| Domain error (rule broken) | **No** | Return to caller; it's a valid business answer          |
| Version conflict           | Maybe  | Reload + re-decide a bounded number of times            |
| Infra transient (DB, lock) | Yes    | Exponential backoff + jitter, bounded attempts          |
| Infra persistent           | Stop   | Dead-letter / alert; human or saga compensation decides |

Because the handler is idempotent, retries are always safe — that is the whole point of making idempotency non-negotiable.

## Direct dispatch vs command bus

|        | Direct invocation                      | Command bus                                           |
| ------ | -------------------------------------- | ----------------------------------------------------- |
| Wiring | Transport adapter calls handler method | Adapter dispatches a command object to a dispatcher   |
| When   | Default; few handlers                  | Many handlers sharing middleware (dedup, tx, logging) |
| Pros   | Explicit, traceable                    | Idempotency/transaction middleware written once       |
| Cons   | Cross-cutting code repeated            | Indirection; harder to navigate                       |

In event-driven systems the bus earns its keep earlier than in plain CQRS, because dedup + outbox-transaction wrapping is identical for every handler — but still: start direct, extract middleware when the third handler repeats it.

## Commands from sagas vs commands from users

- **User-originated**: command ID generated at the edge (HTTP idempotency key); reply synchronously with ack/conflict.
- **Saga-originated** ([[saga-orchestrator]]): command ID derived deterministically from `(sagaID, step)` so saga retries dedup naturally; replies come back as events, not return values.
- Both paths converge on the same handler — the handler must not care who sent the command.

## Testing

- **Unit**: aggregate method emits the right events / errors given state — no infra.
- **Handler unit**: fake repository + fake outbox; assert one transaction, ack returned, no data leaked.
- **Idempotency test (mandatory)**: handle the same command twice; assert single outbox row, single state change.
- **Integration (testcontainers)**: real Postgres; run handler concurrently with the same expected version; assert exactly one commit wins and the loser gets a conflict.
