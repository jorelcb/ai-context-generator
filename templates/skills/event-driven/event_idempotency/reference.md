# event-idempotency — Reference

## Delivery semantics — what you can actually buy

| Semantics               | Who provides it                                | Reality check                                                       |
| ----------------------- | ---------------------------------------------- | ------------------------------------------------------------------- |
| At-most-once            | Fire-and-forget publish                        | Loses messages on any failure — almost never acceptable             |
| At-least-once           | Outbox + acked consumption (the default)       | Duplicates are guaranteed _by design_; this is what you build on    |
| "Exactly-once delivery" | Nobody, end to end                             | Physically impossible across independent systems (FLP/Two Generals) |
| Exactly-once **effect** | **Your handler** (at-least-once + idempotency) | The achievable goal — and it's application code, not broker config  |

Kafka's `exactly_once_v2`/transactions cover Kafka-to-Kafka hops (consume→produce atomically). The moment a handler touches Postgres, Stripe, or SMTP, you're back to at-least-once + idempotent processing.

## Inbox pattern in detail

Consumer-side mirror of the outbox ([[domain-event]]):

```sql
CREATE TABLE processed_events (
  handler   TEXT NOT NULL,   -- dedup is PER HANDLER, not global:
  event_id  TEXT NOT NULL,   -- two consumers of the same event dedup independently
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (handler, event_id)
);
```

Flow inside ONE transaction: `INSERT INTO processed_events ... ON CONFLICT DO NOTHING` → if 0 rows inserted, ack and return (duplicate); else perform the effects → commit → ack the broker. Crash before commit = redelivery hits the same path; crash after commit but before ack = redelivery dedups on the inserted row. No window leaks.

**Variant — inbox-then-process**: insert the full message into an `inbox` table in a fast transaction, ack immediately, process asynchronously from the inbox. Decouples broker ack latency from processing time; the unique constraint still dedups.

## Strategy selection table

| Handler type                                 | First choice                              | Why                                          |
| -------------------------------------------- | ----------------------------------------- | -------------------------------------------- |
| Event consumer with DB effects               | A — inbox table                           | General, transactional, mechanical           |
| Event consumer triggering email/notification | B — deterministic operation ID            | Keys the _side effect_, not just the message |
| [[command-handler]] (state mutation)         | C — aggregate version (OCC) + command-ID  | Covers duplicates AND concurrent writers     |
| [[event-projection]] apply                   | D — natural (upsert / `WHERE last_seq <`) | No bookkeeping; replay-safe by construction  |
| [[saga-orchestrator]] step                   | State guard + B for dispatched commands   | Saga state is itself the dedup record        |
| Call to external API                         | Downstream idempotency key (Stripe-style) | DB transaction can't protect a remote call   |

Defense in depth is normal: an inbox row **and** a guarded update costs little and covers operator replays, not just broker redelivery.

## External side effects — the crash window

The hard case: handler sends the email, then crashes before commit. The inbox row never lands; redelivery re-sends the email.

1. **Downstream idempotency key** (best): `Idempotency-Key: notification:order-123:placed` — the provider dedups across your crashes.
2. **Keyed task table**: the handler only inserts `(operation_id, payload)` with a unique key (transactional, safe); a separate worker performs the send and marks done. The send itself can still double on worker crash — combine with (1) when the provider allows.
3. **Accept-and-bound**: for low-stakes effects (metrics, cache warm), document that rare duplicates are acceptable. Never acceptable for money.

## Retention and cleanup

- Size the dedup window ≥ the maximum redelivery horizon: broker retention + max consumer downtime + operational replay window (be generous: days–weeks, not minutes).
- Cleanup: `DELETE FROM processed_events WHERE processed_at < now() - interval '30 days'` on a schedule, or native TTLs (DynamoDB TTL, Redis `EXPIRE` — Redis only if losing the dedup set is tolerable).
- If you replay history older than retention on purpose (projection rebuilds), the inbox won't protect you — that's why [[event-projection]] applies must be naturally idempotent (strategy D), not inbox-dependent.

## Testing recipes

- **Unit (mandatory)**: call `handle(event)` twice; assert effects exactly once (one row, one fake-email).
- **Crash-window test**: fake clock/failpoint between effect and commit; assert redelivery converges to exactly-once effect.
- **Concurrent duplicate test**: two goroutines/promises handle the same event simultaneously; the unique constraint must serialize them (this is the test read-then-write checks fail).
- **Integration (testcontainers)**: Postgres + Redpanda/RabbitMQ; publish, `kill -9` the consumer mid-handle, restart, assert single effect. Broker redelivery is simulated honestly by the real broker.
- Make "deliver everything twice" a mode in your test harness — run the whole suite under it once per CI pipeline.
