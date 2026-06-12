# command-handler — Examples (Go + TypeScript)

> Illustrative — adapt to the stack. `PlaceOrder`: dedup at entry, one aggregate method, state + events ([[domain-event]]) committed through the outbox in a single transaction, ack returned.

## Go

```go
// application/place_order.go
type PlaceOrderCmd struct {
    CommandID  string // deterministic — enables dedup
    CustomerID string
    Items      []ItemDTO
}

type PlaceOrder struct {
    db     *sql.DB
    orders order.Repository // driven port
    outbox outbox.Store     // driven port
}

func (h *PlaceOrder) Handle(ctx context.Context, c PlaceOrderCmd) (string, error) {
    tx, err := h.db.BeginTx(ctx, nil)
    if err != nil { return "", err }
    defer tx.Rollback()

    // 1. Dedup at entry — unique constraint on processed_commands.command_id
    if dup, err := dedup.Claim(ctx, tx, c.CommandID); err != nil {
        return "", err
    } else if dup {
        return "", nil // already processed: ack again, no new effects
    }

    // 2. One aggregate method does the business work and records events
    o, err := order.New(order.CustomerID(c.CustomerID), toLines(c.Items))
    if err != nil { return "", err }       // domain error — do NOT retry
    if err := o.Place(); err != nil { return "", err }

    // 3. State + events in the SAME transaction (handler = tx boundary)
    if err := h.orders.SaveTx(ctx, tx, o); err != nil { return "", err } // OCC inside
    if err := h.outbox.Append(ctx, tx, o.PullEvents()); err != nil { return "", err }

    if err := tx.Commit(); err != nil { return "", err } // infra error — retryable
    return string(o.ID()), nil // ack: id only, never display data
}
```

## TypeScript

```typescript
// application/place-order.ts
interface PlaceOrderCmd {
  commandId: string; // deterministic — enables dedup
  customerId: string;
  items: ItemDto[];
}

export class PlaceOrder {
  constructor(
    private readonly uow: UnitOfWork, // wraps one DB transaction
    private readonly orders: OrderRepository, // driven port
    private readonly outbox: OutboxStore, // driven port
  ) {}

  async handle(c: PlaceOrderCmd): Promise<string> {
    return this.uow.transact(async (tx) => {
      // 1. Dedup at entry — unique constraint on processed_commands
      const duplicate = await claimCommand(tx, c.commandId);
      if (duplicate) return c.commandId; // ack again, no new effects

      // 2. One aggregate method; domain errors are NOT retried
      const order = Order.create(
        new CustomerId(c.customerId),
        toLines(c.items),
      );
      order.place(); // records OrderPlaced internally

      // 3. State + events in the SAME transaction
      await this.orders.save(tx, order); // OCC: WHERE version = expected
      await this.outbox.append(tx, order.pullEvents()); // relay publishes later

      return order.id.value; // ack: id only
    }); // commit failure = infra error — caller may retry safely
  }
}
```

Note in both: the handler contains zero business rules (those live in `Order.place()`), publishing never touches the broker (outbox only), and handling the same `commandId` twice commits nothing new — the test to write first.
