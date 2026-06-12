# hexagonal-port — Examples (Go + TypeScript)

> Illustrative: one driven port in the Domain, two swappable adapters, and the contract test both must pass.

## Go

```go
// domain/order/repository.go — DRIVEN PORT (domain types only)
package order
type Repository interface {
    Save(ctx context.Context, o *Order) error
    FindByID(ctx context.Context, id ID) (*Order, error)
}

// infrastructure/postgres/order_repo.go — ADAPTER A
type PostgresRepo struct{ db *sql.DB }
func (r *PostgresRepo) Save(ctx context.Context, o *order.Order) error { /* SQL */ return nil }
func (r *PostgresRepo) FindByID(ctx context.Context, id order.ID) (*order.Order, error) { /* SQL */ return nil, nil }

// infrastructure/memory/order_repo.go — ADAPTER B (for tests/dev)
type MemRepo struct{ m map[order.ID]*order.Order }
func (r *MemRepo) Save(ctx context.Context, o *order.Order) error { r.m[o.ID()] = o; return nil }
func (r *MemRepo) FindByID(ctx context.Context, id order.ID) (*order.Order, error) { return r.m[id], nil }
```

```go
// CONTRACT TEST — every adapter must pass this same suite.
func testRepositoryContract(t *testing.T, newRepo func() order.Repository) {
    repo := newRepo()
    o, _ := order.New("cust-1", lines)
    if err := repo.Save(context.Background(), o); err != nil { t.Fatal(err) }
    got, _ := repo.FindByID(context.Background(), o.ID())
    if got.ID() != o.ID() { t.Fatal("round-trip failed") }
}
func TestPostgres(t *testing.T) { testRepositoryContract(t, func() order.Repository { return newPostgresForTest(t) }) }
func TestMemory(t *testing.T)   { testRepositoryContract(t, func() order.Repository { return &MemRepo{m: map[order.ID]*order.Order{}} }) }
```

## TypeScript

```typescript
// domain/order/order-repository.ts — DRIVEN PORT
export interface OrderRepository {
  save(o: Order): Promise<void>;
  findById(id: OrderId): Promise<Order | null>;
}

// infrastructure/postgres/order-repository.ts — ADAPTER A
export class PostgresOrderRepository implements OrderRepository {
  constructor(private readonly db: Pool) {}
  async save(o: Order) {
    /* SQL */
  }
  async findById(id: OrderId) {
    /* SQL */ return null;
  }
}

// infrastructure/memory/order-repository.ts — ADAPTER B
export class InMemoryOrderRepository implements OrderRepository {
  private store = new Map<string, Order>();
  async save(o: Order) {
    this.store.set(o.id.value, o);
  }
  async findById(id: OrderId) {
    return this.store.get(id.value) ?? null;
  }
}
```

```typescript
// CONTRACT TEST — run for every adapter implementation.
export function orderRepositoryContract(
  name: string,
  make: () => OrderRepository,
) {
  describe(name, () => {
    it("round-trips an order", async () => {
      const repo = make();
      const order = Order.create(customerId, lines);
      await repo.save(order);
      expect((await repo.findById(order.id))?.id).toEqual(order.id);
    });
  });
}
orderRepositoryContract("InMemory", () => new InMemoryOrderRepository());
// orderRepositoryContract("Postgres", () => new PostgresOrderRepository(testPool));
```

The port names a domain capability; the adapters are interchangeable because the contract test forces them to behave identically.
