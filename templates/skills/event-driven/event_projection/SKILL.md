# Event Projection Skill

> **Scope:** the read side — projections that consume domain events ([[domain-event]]) and maintain **read models** optimized for specific queries. A projection is a pure consumer: it never issues commands (that's a [[saga-orchestrator]]) and never feeds writes (the [[command-handler]] never reads from it).

## When to use

Creating a new query view, optimizing a query that hits the write model, adding a screen-shaped read model, or migrating from a shared read/write schema to separated read models.

## Projection concept (normative)

A projection subscribes to one or more event streams and folds events into a read model tailored to **one query pattern**. Each projection **owns its data exclusively** — no shared schema with other projections or with the write model. Read models are **derived and disposable**: the event stream is the source of truth, the read model can always be rebuilt from it.

## Responsibilities (the process)

1. **Consume** events from the broker (live) or the event store (replay) — at-least-once delivery is the contract.
2. **Apply** each event to the read model: insert/update the denormalized rows the query needs.
3. **Checkpoint**: persist the last position consumed **atomically with the read-model update** (same transaction). Checkpoint apart from data = lost or doubled updates on crash.
4. **Be idempotent**: re-consuming an event must not change the read model further ([[event-idempotency]]) — the checkpoint-in-same-tx pattern gives you this for free against redelivery-from-restart; position comparison covers the rest.
5. **Resume** from the checkpoint on restart; a brand-new projection replays from the beginning of the stream — no shortcuts, no "seed from the write tables".

## Read model design (normative)

- **Shaped by the query, not by normalization** — denormalize freely; one table per screen/endpoint is fine.
- **Pre-compute** aggregations, counts, and joins at write time; the query becomes a primary-key or single-index lookup.
- **Index** for the access pattern; nothing else.
- **Private**: only this projection writes it; only its query handler reads it. Another consumer wants the data? It builds its own projection from the same events.

## Eventual consistency

Read models **lag** the write model — this is structural, not a bug. Make it a product decision:

- **Read-your-own-writes UX**: optimistic UI updates, or "your change was saved" confirmation while the view catches up.
- **Version-aware reads**: client passes the version it wrote; query waits/retries until the projection reaches it (use sparingly).
- **Honest staleness**: show "as of" timestamps on dashboards.
- Never "fix" lag by making the command handler write the read model synchronously — that recouples the paths you separated.

## Rebuild (the superpower — keep it usable)

Because the read model is derived: truncate it, reset the checkpoint to zero, replay all events. This is how you fix projection bugs, add columns, and bootstrap new views. Protect this capability: no out-of-band writes to read models, no projection logic that depends on wall-clock "now", no side effects in apply functions (an email inside a projection re-fires on every rebuild — side effects belong to dedicated handlers behind [[event-idempotency]]).

## Anti-patterns to avoid

- Projection **querying another projection** — build a third one from the events instead.
- Projection **issuing commands** — reacting-with-commands is a [[saga-orchestrator]].
- **Shared read models** between projections — independence is the point.
- **Seeding from current state** instead of replaying events — silently diverges and kills rebuildability.
- **Checkpoint in a different transaction** than the read-model update.
- **Side effects** (emails, API calls) inside apply functions — they re-fire on rebuild.
- Storing the **entire event history** in the projection — it's a view, not a second event store.

## Verification

- [ ] Empty read model + full replay = current state (rebuild test).
- [ ] Applying the same event twice changes nothing (idempotency test).
- [ ] Checkpoint and read-model update commit in one transaction.
- [ ] Only this projection writes its tables; only its queries read them.
- [ ] No side effects in apply functions.

## Going deeper

- **[reference.md](reference.md)** — checkpoint storage options, zero-downtime rebuild (blue/green projections), ordering and partitioning, catch-up vs live subscription, testcontainers setups.
- **[examples.md](examples.md)** — an `OrderSummary` projection folding `OrderPlaced`/`OrderShipped` with atomic checkpointing, in Go and TypeScript.
