# event-projection — Reference

## Checkpoint storage options

| Where the checkpoint lives                      | Atomic with read model? | When to use                                                 |
| ----------------------------------------------- | ----------------------- | ----------------------------------------------------------- |
| Same DB, same transaction (a `checkpoints` row) | **Yes** — preferred     | Read model in a transactional store (Postgres, etc.)        |
| Broker consumer offset (Kafka commit)           | No                      | Only with idempotent applies; commit offset **after** apply |
| Event-store subscription position               | Depends on store        | Stores with built-in persistent subscriptions               |

Rule: if the checkpoint can commit without the data (or vice versa), you have either lost updates or double applies. The same-transaction `checkpoints(projection_name, position)` row is the boring, correct default.

**Position semantics**: store the _global_ stream position (or per-partition offsets), not a timestamp. Timestamps are not unique and not monotonic across producers.

## Delivery, ordering, partitioning

- Delivery is **at-least-once**; duplicates and (cross-partition) reordering are normal.
- Per-aggregate ordering is what you should design for: partition the stream by `aggregate_id` so all `Order:123` events arrive in order to one consumer instance.
- For events of _different_ aggregates, the projection must tolerate any interleaving.
- Out-of-order tolerant applies: prefer upserts keyed by aggregate ID + comparison on the event's per-aggregate sequence (`WHERE last_seq < $event_seq`) over blind updates.

## Catch-up vs live subscription

A projection has two modes and must handle the seam:

1. **Catch-up**: read historical events from the store/topic from the checkpoint forward, batched, as fast as possible.
2. **Live**: switch to the subscription once caught up.

Most brokers/stores hide the seam (a Kafka consumer just keeps polling; event stores offer catch-up subscriptions). If you implement it manually: keep applying from history until the history position passes the subscription's buffered start, then drain the buffer. Never serve "caught up" status before the seam is crossed if queries gate on freshness.

## Zero-downtime rebuild — blue/green projections

To fix a projection bug or change the schema without downtime:

1. Deploy the new projection version writing to a **new table set** (`order_summary_v2`) with its **own checkpoint**, starting from position 0.
2. Let it replay history while the old projection keeps serving.
3. When v2's lag ≈ live, flip the query handler to the v2 tables (config/feature flag).
4. Drop the v1 tables after a soak period.

This is why read models must be private and disposable — the flip is invisible to consumers because only the query handler knows the table names.

**Rebuild time** is your real SLO for this pattern: measure events/sec of replay regularly. If full replay takes days, introduce snapshots upstream or coarser source streams — _before_ the emergency.

## Eventual consistency — choosing a UX strategy

| Strategy                 | Cost     | Use for                                        |
| ------------------------ | -------- | ---------------------------------------------- |
| Optimistic UI            | Frontend | Most interactive flows                         |
| "Saved" ack + async view | Cheap    | Forms, submissions                             |
| Version-gated read       | Backend  | The few flows that truly need read-your-writes |
| "As of" timestamp        | Trivial  | Dashboards, reports                            |

Pick per use case. Blanket "wait for the projection" turns the async system back into a slow synchronous one.

## Tooling and tests

- **testcontainers** (Go `testcontainers-go`, TS `testcontainers`): Postgres for the read model + Redpanda/Kafka or RabbitMQ for the stream; run the projection binary against both.
- **Rebuild test (mandatory)**: fixture stream of events → run projection from zero → assert final read-model rows. This single test catches most apply bugs.
- **Idempotency test (mandatory)**: apply the same event twice (and a duplicate batch); assert no further change ([[event-idempotency]]).
- **Crash test**: stop the projection mid-batch, restart from checkpoint, assert no lost/doubled rows.
- Keep apply functions pure (event + current rows → new rows) so most tests run without containers.
