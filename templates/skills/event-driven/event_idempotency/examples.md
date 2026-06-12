# event-idempotency — Examples (Go + TypeScript)

> Illustrative — adapt to the stack. Two patterns on `OrderPlaced` ([[domain-event]]): an **inbox-table consumer** (strategy A — dedup claim + effects in one transaction) and a **deterministic-key notification** (strategy B — the side effect itself is keyed by intent).

Inbox table:

```sql
CREATE TABLE processed_events (
  handler      TEXT NOT NULL,                  -- dedup is per handler
  event_id     TEXT NOT NULL,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (handler, event_id)
);

CREATE TABLE notifications (
  operation_id TEXT PRIMARY KEY,               -- e.g. notification:order-123:placed
  recipient    TEXT NOT NULL,
  status       TEXT NOT NULL DEFAULT 'pending' -- worker sends + marks 'sent'
);
```

## Go

```go
// consumer/order_placed.go — strategy A: inbox claim + effects, ONE transaction
func (c *Consumer) HandleOrderPlaced(ctx context.Context, e OrderPlaced) error {
    tx, err := c.db.BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()

    // 1. Atomic claim — NOT select-then-insert (that races)
    res, err := tx.ExecContext(ctx, `
        INSERT INTO processed_events (handler, event_id)
        VALUES ('loyalty-points', $1)
        ON CONFLICT DO NOTHING`, e.EventID)
    if err != nil { return err }
    if n, _ := res.RowsAffected(); n == 0 {
        return nil // duplicate delivery: ack to the broker, zero effects
    }

    // 2. Effects in the SAME transaction as the claim
    if _, err := tx.ExecContext(ctx, `
        UPDATE loyalty_accounts SET points = points + $1 WHERE customer_id = $2`,
        pointsFor(e.TotalCents), e.CustomerID); err != nil {
        return err
    }

    // 3. External side effect → strategy B: keyed record, sent by a worker
    //    with a downstream Idempotency-Key — never SMTP inside this tx
    opID := fmt.Sprintf("notification:%s:placed", e.AggregateID) // deterministic
    if _, err := tx.ExecContext(ctx, `
        INSERT INTO notifications (operation_id, recipient)
        VALUES ($1, $2) ON CONFLICT DO NOTHING`, opID, e.CustomerID); err != nil {
        return err
    }

    return tx.Commit() // crash before commit → redelivery replays this path safely
}

// Mandatory test: same event twice → points granted once, one notification row.
func TestHandleOrderPlaced_Twice(t *testing.T) {
    must(c.HandleOrderPlaced(ctx, ev))
    must(c.HandleOrderPlaced(ctx, ev)) // redelivery
    assertEqual(t, 1, countNotifications(t, "notification:order-123:placed"))
    assertEqual(t, pointsFor(ev.TotalCents), pointsOf(t, ev.CustomerID))
}
```

## TypeScript

```typescript
// consumer/order-placed.ts — strategy A: inbox claim + effects, ONE transaction
export class OrderPlacedConsumer {
  constructor(private readonly uow: UnitOfWork) {}

  async handle(e: OrderPlaced): Promise<void> {
    await this.uow.transact(async (tx) => {
      // 1. Atomic claim — NOT select-then-insert (that races)
      const claim = await tx.query(
        `INSERT INTO processed_events (handler, event_id)
         VALUES ('loyalty-points', $1)
         ON CONFLICT DO NOTHING`,
        [e.eventId],
      );
      if (claim.rowCount === 0) return; // duplicate: ack, zero effects

      // 2. Effects in the SAME transaction as the claim
      await tx.query(
        `UPDATE loyalty_accounts SET points = points + $1 WHERE customer_id = $2`,
        [pointsFor(e.totalCents), e.customerId],
      );

      // 3. External side effect → strategy B: deterministic operation key;
      //    a worker sends it with a downstream Idempotency-Key
      const opId = `notification:${e.aggregateId}:placed`; // keyed by intent
      await tx.query(
        `INSERT INTO notifications (operation_id, recipient)
         VALUES ($1, $2) ON CONFLICT DO NOTHING`,
        [opId, e.customerId],
      );
    }); // crash before commit → redelivery replays this path safely
  }
}

// Mandatory test: same event twice → points granted once, one notification row.
it("is idempotent under redelivery", async () => {
  await consumer.handle(ev);
  await consumer.handle(ev); // redelivery
  expect(await countNotifications("notification:order-123:placed")).toBe(1);
  expect(await pointsOf(ev.customerId)).toBe(pointsFor(ev.totalCents));
});
```

Note in both: the claim and the effects share one transaction (no crash window between them), the email never happens inline (only a keyed record the worker fulfills with a provider idempotency key), and the duplicate-delivery test is written first — it is the specification of this skill.
