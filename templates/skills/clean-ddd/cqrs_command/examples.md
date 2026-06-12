# cqrs-command — Examples (Go + TypeScript)

> Illustrative: a command handler (writes, returns an id) and a query handler (reads via a read port, returns a DTO). Both are Application Services — thin orchestrators.

## Go

```go
// application/place_order.go — COMMAND handler (write; returns id only)
type PlaceOrderCmd struct{ CustomerID string; Items []ItemDTO } // input DTO
type PlaceOrder struct{ orders order.Repository }                // driven port
func (h *PlaceOrder) Handle(ctx context.Context, c PlaceOrderCmd) (string, error) {
    if len(c.Items) == 0 { return "", ErrNoItems }     // structural validation
    o, err := order.New(order.CustomerID(c.CustomerID), toLines(c.Items)) // domain enforces rules
    if err != nil { return "", err }                   // business error bubbles up
    if err := h.orders.Save(ctx, o); err != nil { return "", err } // tx boundary
    return string(o.ID()), nil                          // confirmation, not display data
}

// application/get_order.go — QUERY handler (read; returns DTO via read port)
type GetOrderQuery struct{ ID string }
type OrderView struct{ ID, Status string; Total int64 } // output DTO (flattened)
type ReadModel interface { OrderByID(ctx context.Context, id string) (*OrderView, error) }
type GetOrder struct{ read ReadModel }
func (h *GetOrder) Handle(ctx context.Context, q GetOrderQuery) (*OrderView, error) {
    return h.read.OrderByID(ctx, q.ID) // no writes, no aggregate reconstruction
}
```

## TypeScript

```typescript
// application/place-order.ts — COMMAND handler
interface PlaceOrderCmd {
  customerId: string;
  items: ItemDto[];
} // input DTO
export class PlaceOrder {
  constructor(private readonly orders: OrderRepository) {} // driven port
  async handle(c: PlaceOrderCmd): Promise<string> {
    if (c.items.length === 0) throw new NoItemsError(); // structural validation
    const order = Order.create(new CustomerId(c.customerId), toLines(c.items)); // domain rules
    await this.orders.save(order); // tx boundary
    return order.id.value; // confirmation only
  }
}

// application/get-order.ts — QUERY handler
interface OrderView {
  id: string;
  status: string;
  total: number;
} // output DTO
interface OrderReadModel {
  byId(id: string): Promise<OrderView | null>;
}
export class GetOrder {
  constructor(private readonly read: OrderReadModel) {}
  async handle(q: { id: string }): Promise<OrderView | null> {
    return this.read.byId(q.id); // pure read, returns DTO
  }
}
```

Note in both: the command returns an id (never the `Order` entity), the query returns a flat DTO via a read port, and neither handler contains business rules — those live in `Order`.
