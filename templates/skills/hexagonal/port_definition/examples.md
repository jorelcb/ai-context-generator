# port-definition — Examples (Go + TypeScript)

> Illustrative — adapt to the stack. Driven ports for `Order` plus the driving entry point `PlaceOrder`. Adapters implementing these ports live in [[adapter-pattern]].

## Go

```go
// domain/order/order.go — the aggregate the ports speak about (abridged)
package order

type ID string

var (
    ErrNotFound  = errors.New("order: not found")
    ErrDuplicate = errors.New("order: duplicate")
)

// domain/order/repository.go — DRIVEN PORT (persistence capability)
// Implementations must be safe for concurrent use.
type Repository interface {
    Save(ctx context.Context, o *Order) error                 // ErrDuplicate on ID collision
    FindByID(ctx context.Context, id ID) (*Order, error)      // ErrNotFound when absent
}

// domain/order/notifier.go — DRIVEN PORT (notification capability, separate
// port: different reason to change than persistence)
type PlacedNotifier interface {
    OrderPlaced(ctx context.Context, o *Order) error
}
```

```go
// application/placeorder/service.go — DRIVING ENTRY POINT.
// Concrete service: NO driving interface by default — primary adapters
// (HTTP handler, CLI) call *PlaceOrder directly.
package placeorder

type PlaceOrder struct {
    repo     order.Repository     // driven ports injected as interfaces
    notifier order.PlacedNotifier
}

func New(r order.Repository, n order.PlacedNotifier) *PlaceOrder {
    return &PlaceOrder{repo: r, notifier: n}
}

func (s *PlaceOrder) Execute(ctx context.Context, cmd Command) (order.ID, error) {
    o, err := order.New(cmd.CustomerID, cmd.Lines)
    if err != nil {
        return "", err
    }
    if err := s.repo.Save(ctx, o); err != nil {
        return "", err
    }
    return o.ID(), s.notifier.OrderPlaced(ctx, o)
}

// OPTIONAL driving interface — only if HTTP + CLI + scheduler all need the
// same abstraction, or you contract-test primary adapters against a double:
// type UseCase interface {
//     Execute(ctx context.Context, cmd Command) (order.ID, error)
// }
```

## TypeScript

```typescript
// domain/order/order.ts — domain types the ports speak (abridged)
export class OrderId {
  constructor(readonly value: string) {}
}
export class OrderNotFoundError extends Error {}
export class DuplicateOrderError extends Error {}

// domain/order/order-repository.ts — DRIVEN PORT
export interface OrderRepository {
  /** Rejects with DuplicateOrderError on ID collision. */
  save(order: Order): Promise<void>;
  /** Resolves null when absent (or reject with OrderNotFoundError — pick ONE
      convention and let the contract test enforce it). */
  findById(id: OrderId): Promise<Order | null>;
}

// domain/order/placed-notifier.ts — DRIVEN PORT (separate capability)
export interface OrderPlacedNotifier {
  orderPlaced(order: Order): Promise<void>;
}
```

```typescript
// application/place-order.ts — DRIVING ENTRY POINT (concrete class,
// no driving interface by default).
export class PlaceOrder {
  constructor(
    private readonly repo: OrderRepository,
    private readonly notifier: OrderPlacedNotifier,
  ) {}

  async execute(cmd: PlaceOrderCommand): Promise<OrderId> {
    const order = Order.create(cmd.customerId, cmd.lines);
    await this.repo.save(order);
    await this.notifier.orderPlaced(order);
    return order.id;
  }
}

// OPTIONAL driving interface — only with >1 primary adapter or for
// contract-testing the adapter against a double:
// export interface PlaceOrderUseCase {
//   execute(cmd: PlaceOrderCommand): Promise<OrderId>;
// }
```

Both ports are named by capability, signatures carry only domain types, and the error contract is part of the port — exactly what the contract tests in [[hex-integration-test]] pin down.
