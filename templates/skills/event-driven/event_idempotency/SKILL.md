# Event Idempotency Skill

> **Scope:** making every handler — event consumers, [[command-handler]]s, [[event-projection]] applies, [[saga-orchestrator]] steps — safe to execute more than once. This is the **non-negotiable property** of event-driven systems: the outbox guarantees at-least-once delivery ([[domain-event]]), so duplicates are not a failure mode, they are the contract.

## When to use

Adding a new event consumer, building a command handler, debugging duplicate side effects in production (double email, double charge), or auditing an existing handler for idempotency gaps.

## Why this is non-negotiable

- Brokers (Kafka, RabbitMQ, SQS, NATS) deliver **at-least-once**: retries, consumer restarts, rebalances, and outbox relay crashes all cause redelivery.
- "Exactly-once delivery" is a marketing term. What exists is at-least-once delivery + **idempotent processing**, which together produce **exactly-once _effect_** — and the second half is your code's job.
- Without it, every redelivery is a duplicate side effect: double charge, double email, corrupted counters. With it, retries become free — which is what makes the retry policies in [[command-handler]] and [[saga-orchestrator]] safe at all.

## Strategies (pick deliberately, per handler)

**A. Inbox / dedup table** — the general-purpose default for event consumers.
Insert the `event_id` into a `processed_events` table with a unique constraint, **in the same transaction** as the handler's effects. Conflict = already processed → ack and skip. (This is the inbox pattern — the consumer-side mirror of the outbox.)

**B. Deterministic operation IDs** — for side effects keyed by intent.
Derive the ID from what the operation _means_: `notification:order-123:placed`. However many times `OrderPlaced` is delivered, only one row with that key exists, so only one notification fires. Sagas use this shape for step commands: `(saga_id, step)`.

**C. Aggregate version (optimistic concurrency)** — for command processing.
The command targets `expected_version`; a duplicate arrives stale and is rejected with no effect. Natural in event-sourced aggregates (`AppendToStream` with expected version).

**D. Naturally idempotent operations** — when you can get them.
Set-to-fixed-value, upsert by deterministic key, guarded update (`WHERE last_seq < $seq`). Prefer designing operations into this shape — it removes bookkeeping entirely ([[event-projection]] applies are built this way).

## Where idempotency lives (the process)

1. **Identify every entry point** that consumes a message: event handlers, command handlers, saga steps.
2. **At entry, dedup**: check/claim the event or command ID _inside the handler's transaction_ — a check in one transaction and effects in another is a race, not a guarantee.
3. **For each external side effect** (email, payment API, third-party call): the DB dedup does not cover it if the process crashes between the call and the commit. Pass an **idempotency key** to the downstream API (Stripe-style `Idempotency-Key`); if the API has none, route the effect through a keyed task table (strategy B) so retries collapse.
4. **Test it explicitly** — see Verification. An untested idempotency claim is a wish.

## Know your duplicate sources

Duplicates do not only come from broker redelivery. Audit all of them: outbox relay crash-and-republish ([[domain-event]]), consumer rebalance mid-batch, client/API retries, [[saga-orchestrator]] re-dispatching a step command, and **operator-initiated replays** (projection rebuilds, backfills). The last one routinely outlives any dedup-table retention window — which is why [[event-projection]] applies must be naturally idempotent rather than inbox-dependent.

## Anti-patterns to avoid

- **"Redelivery is rare, it's fine"** — production guarantees you the rebalance during the deploy during the retry storm.
- **Trusting the broker's "exactly-once" mode** — it covers broker-internal hops at best; your DB write and your email API are outside it.
- **Read-then-write checks** (`SELECT` "already there?" then `INSERT`) — race-prone; use unique constraints / atomic claims.
- **Dedup in a different transaction** than the effects — crash between them and you get either lost or doubled processing.
- **Forgetting non-DB side effects** — the handler is "idempotent" but the email goes out twice.
- **Unbounded dedup tables** — define retention aligned with the broker's max redelivery window and clean up.

## Verification

- [ ] Processing the same message 1, 2, or N times yields identical observable state.
- [ ] Dedup claim commits in the same transaction as the effects.
- [ ] Every external side effect carries an idempotency key or is routed through a keyed record.
- [ ] A test delivers the same event twice and asserts effects happened exactly once.
- [ ] Dedup storage has a retention/cleanup policy.

## Going deeper

- **[reference.md](reference.md)** — inbox pattern in detail, delivery-semantics decision table (exactly-once effect vs at-least-once), strategy selection table, retention sizing, broker-specific notes, testcontainers test recipes.
- **[examples.md](examples.md)** — an inbox-table consumer (dedup + effect + checkpoint in one transaction) and a deterministic-key notification sender, in Go and TypeScript.
