# event-projection — Examples (Go + TypeScript)

> Illustrative — adapt to the stack. An `OrderSummary` read model folded from `OrderPlaced` and `OrderShipped` ([[domain-event]]), with the checkpoint committed in the same transaction as the rows.

Read model (one table, shaped by the "list my orders" query):

```sql
CREATE TABLE order_summary (
  order_id     TEXT PRIMARY KEY,
  customer_id  TEXT NOT NULL,
  status       TEXT NOT NULL,
  total_cents  BIGINT NOT NULL,
  placed_at    TIMESTAMPTZ NOT NULL,
  last_seq     BIGINT NOT NULL          -- per-aggregate sequence guard
);
CREATE INDEX ON order_summary (customer_id, placed_at DESC);

CREATE TABLE checkpoints (projection TEXT PRIMARY KEY, position BIGINT NOT NULL);
```

## Go

```go
// projection/order_summary.go — pure apply + atomic checkpoint
type OrderSummaryProjection struct{ db *sql.DB }

func (p *OrderSummaryProjection) Handle(ctx context.Context, e Envelope, pos int64) error {
    tx, err := p.db.BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()

    switch ev := e.Payload.(type) {
    case OrderPlaced: // idempotent upsert: replay-safe
        _, err = tx.ExecContext(ctx, `
            INSERT INTO order_summary (order_id, customer_id, status, total_cents, placed_at, last_seq)
            VALUES ($1,$2,'placed',$3,$4,$5)
            ON CONFLICT (order_id) DO NOTHING`,
            ev.AggregateID, ev.CustomerID, ev.TotalCents, ev.OccurredAt, ev.Seq)
    case OrderShipped: // sequence guard: duplicate or stale event changes nothing
        _, err = tx.ExecContext(ctx, `
            UPDATE order_summary SET status='shipped', last_seq=$2
            WHERE order_id=$1 AND last_seq < $2`,
            ev.AggregateID, ev.Seq)
    default:
        // unknown event types are skipped — but the checkpoint still advances
    }
    if err != nil { return err }

    // checkpoint in the SAME transaction as the row changes
    if _, err := tx.ExecContext(ctx, `
        UPDATE checkpoints SET position=$1 WHERE projection='order_summary'`,
        pos); err != nil { return err }
    return tx.Commit()
}
// NOTE: no commands dispatched, no emails sent — pure fold. Rebuild =
// TRUNCATE order_summary; UPDATE checkpoints SET position=0; replay.
```

## TypeScript

```typescript
// projection/order-summary.ts — pure apply + atomic checkpoint
export class OrderSummaryProjection {
  constructor(private readonly uow: UnitOfWork) {}

  async handle(e: EventEnvelope, position: bigint): Promise<void> {
    await this.uow.transact(async (tx) => {
      switch (e.eventType) {
        case "OrderPlaced": {
          const ev = e.payload as OrderPlaced;
          await tx.query(
            `INSERT INTO order_summary (order_id, customer_id, status, total_cents, placed_at, last_seq)
             VALUES ($1,$2,'placed',$3,$4,$5)
             ON CONFLICT (order_id) DO NOTHING`, // idempotent: replay-safe
            [
              ev.aggregateId,
              ev.customerId,
              ev.totalCents,
              ev.occurredAt,
              ev.seq,
            ],
          );
          break;
        }
        case "OrderShipped": {
          const ev = e.payload as OrderShipped;
          await tx.query(
            `UPDATE order_summary SET status='shipped', last_seq=$2
             WHERE order_id=$1 AND last_seq < $2`, // sequence guard
            [ev.aggregateId, ev.seq],
          );
          break;
        }
        // unknown events: skip, checkpoint still advances
      }

      // checkpoint in the SAME transaction as the row changes
      await tx.query(
        `UPDATE checkpoints SET position=$1 WHERE projection='order_summary'`,
        [position],
      );
    });
  }
}
```

Note in both: applies are **upserts/guarded updates** (consuming an event twice is a no-op), the checkpoint commits with the data, and the projection performs zero side effects — so truncate-and-replay rebuilds it exactly.
