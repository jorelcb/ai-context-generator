# dependency-inversion — Examples (Go + TypeScript)

> Illustrative — adapt to the stack. The same refactor in both languages: a `PlaceOrder` use case directly coupled to SQL, inverted into a driven port + adapter + composition-root wiring.

## Go

### Before — core depends on the detail

```go
// application/placeorder/service.go — VIOLATION: imports database/sql
package placeorder

import "database/sql" // ← core knows the storage technology

type PlaceOrder struct{ db *sql.DB }

func (s *PlaceOrder) Execute(ctx context.Context, cmd Command) error {
    o, err := order.New(cmd.CustomerID, cmd.Lines)
    if err != nil {
        return err
    }
    _, err = s.db.ExecContext(ctx, // SQL inside the use case
        `INSERT INTO orders (id, customer_id) VALUES ($1, $2)`,
        string(o.ID()), o.CustomerID())
    return err // sql errors leak to callers; tests need a real DB
}
```

### After — both sides depend on the abstraction

```go
// domain/order/repository.go — DRIVEN PORT, owned by the Domain
package order

type Repository interface {
    Save(ctx context.Context, o *Order) error // ErrDuplicate on collision
}

// application/placeorder/service.go — depends only on the port
package placeorder // imports domain/order ONLY

type PlaceOrder struct{ repo order.Repository }

func New(r order.Repository) *PlaceOrder { return &PlaceOrder{repo: r} }

func (s *PlaceOrder) Execute(ctx context.Context, cmd Command) error {
    o, err := order.New(cmd.CustomerID, cmd.Lines)
    if err != nil {
        return err
    }
    return s.repo.Save(ctx, o)
}

// infrastructure/postgres/order_repository.go — DETAIL depends on abstraction
// (full adapter, with error translation, in [[adapter-pattern]])

// cmd/api/main.go — COMPOSITION ROOT: the only meeting point
svc := placeorder.New(postgres.NewOrderRepository(db))
```

```go
// Unit test — the payoff: no DB, no flakiness
func TestPlaceOrder(t *testing.T) {
    svc := placeorder.New(&memory.OrderRepository{}) // fake implements the port
    err := svc.Execute(context.Background(), validCommand())
    if err != nil { t.Fatal(err) }
}
```

## TypeScript

### Before — core depends on the detail

```typescript
// application/place-order.ts — VIOLATION: imports the driver
import { Pool } from "pg"; // ← core knows the storage technology

export class PlaceOrder {
  constructor(private readonly pool: Pool) {}

  async execute(cmd: PlaceOrderCommand): Promise<void> {
    const order = Order.create(cmd.customerId, cmd.lines);
    await this.pool.query(
      // SQL inside the use case
      "INSERT INTO orders (id, customer_id) VALUES ($1, $2)",
      [order.id.value, order.customerId.value],
    ); // pg errors leak; tests need a running Postgres
  }
}
```

### After — both sides depend on the abstraction

```typescript
// domain/order/order-repository.ts — DRIVEN PORT, owned by the Domain
export interface OrderRepository {
  save(order: Order): Promise<void>; // DuplicateOrderError on collision
}

// application/place-order.ts — depends only on the port
export class PlaceOrder {
  constructor(private readonly repo: OrderRepository) {}

  async execute(cmd: PlaceOrderCommand): Promise<void> {
    const order = Order.create(cmd.customerId, cmd.lines);
    await this.repo.save(order);
  }
}

// infrastructure/postgres/order-repository.ts — DETAIL implements the port
// (full adapter, with error translation, in [[adapter-pattern]])

// main.ts — COMPOSITION ROOT
const placeOrder = new PlaceOrder(new PostgresOrderRepository(pool));
```

```typescript
// Unit test — no DB, runs in milliseconds
it("places an order", async () => {
  const svc = new PlaceOrder(new InMemoryOrderRepository()); // fake = port impl
  await expect(svc.execute(validCommand())).resolves.toBeUndefined();
});
```

The arrows flipped: before, `application → pg`; after, `application → domain ← infrastructure`. The in-memory fake and the Postgres adapter stay interchangeable because both pass the same contract suite ([[hex-integration-test]]).
