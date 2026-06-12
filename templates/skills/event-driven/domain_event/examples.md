# domain-event — Examples (Go + TypeScript)

> Illustrative — adapt to the stack. `OrderPlaced`: envelope + payload, raised by the aggregate, persisted to the outbox in the same transaction as state.

## Go

```go
// domain/order/events.go — envelope + a concrete fact
type Envelope struct {
    EventID       string    `json:"event_id"`       // unique per occurrence
    EventType     string    `json:"event_type"`     // "OrderPlaced"
    EventVersion  int       `json:"event_version"`  // 1
    AggregateID   string    `json:"aggregate_id"`
    OccurredAt    time.Time `json:"occurred_at"`
    CorrelationID string    `json:"correlation_id"` // whole flow
    CausationID   string    `json:"causation_id"`   // command/event that caused this
}

type OrderPlaced struct { // payload: the fact, not the aggregate dump
    Envelope
    CustomerID string      `json:"customer_id"`
    Lines      []LinePlaced `json:"lines"`
    TotalCents int64       `json:"total_cents"`
    Currency   string      `json:"currency"`
}

// domain/order/order.go — the aggregate records facts; it does not publish
func (o *Order) Place() error {
    if len(o.lines) == 0 { return ErrEmptyOrder } // invariant first
    o.status = StatusPlaced
    o.record(OrderPlaced{ // buffered on the aggregate, drained by the handler
        Envelope:   newEnvelope("OrderPlaced", o.id),
        CustomerID: o.customerID, Lines: toLinesPlaced(o.lines),
        TotalCents: o.total.Cents(), Currency: o.total.Currency(),
    })
    return nil
}

// infrastructure/outbox/store.go — same tx as aggregate state (see command-handler)
func (s *Store) Append(ctx context.Context, tx *sql.Tx, evs []order.Event) error {
    for _, e := range evs {
        payload, _ := json.Marshal(e)
        if _, err := tx.ExecContext(ctx,
            `INSERT INTO outbox (event_id, event_type, aggregate_id, payload, occurred_at)
             VALUES ($1,$2,$3,$4,$5)`,
            e.ID(), e.Type(), e.Aggregate(), payload, e.At()); err != nil {
            return err
        }
    }
    return nil // relay process publishes later; published_at stays NULL until then
}
```

## TypeScript

```typescript
// domain/order/events.ts — envelope + a concrete fact
interface Envelope {
  eventId: string; // unique per occurrence
  eventType: string; // "OrderPlaced"
  eventVersion: number; // 1
  aggregateId: string;
  occurredAt: string; // ISO-8601
  correlationId: string; // whole flow
  causationId: string; // command/event that caused this
}

export interface OrderPlaced extends Envelope {
  eventType: "OrderPlaced";
  customerId: string;
  lines: LinePlaced[];
  totalCents: number;
  currency: string;
}

// domain/order/order.ts — the aggregate records facts; it does not publish
export class Order {
  private pending: DomainEvent[] = [];

  place(): void {
    if (this.lines.length === 0) throw new EmptyOrderError(); // invariant first
    this.status = "placed";
    this.pending.push({
      ...newEnvelope("OrderPlaced", this.id.value),
      customerId: this.customerId.value,
      lines: this.lines.map(toLinePlaced),
      totalCents: this.total.cents,
      currency: this.total.currency,
    } satisfies OrderPlaced);
  }

  pullEvents(): DomainEvent[] {
    // drained by the command handler
    const evs = this.pending;
    this.pending = [];
    return evs;
  }
}

// infrastructure/outbox/store.ts — same tx as aggregate state
export class OutboxStore {
  async append(tx: Tx, events: DomainEvent[]): Promise<void> {
    for (const e of events) {
      await tx.query(
        `INSERT INTO outbox (event_id, event_type, aggregate_id, payload, occurred_at)
         VALUES ($1,$2,$3,$4,$5)`,
        [
          e.eventId,
          e.eventType,
          e.aggregateId,
          JSON.stringify(e),
          e.occurredAt,
        ],
      ); // relay publishes later
    }
  }
}
```

Note in both: the event name is past tense, the payload carries the fact (not the whole `Order`), the aggregate only _records_ — publishing happens via the outbox relay, never inline.
