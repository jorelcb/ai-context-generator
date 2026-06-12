# hex-integration-test — Examples (Go + TypeScript)

> Illustrative — adapt to the stack. One contract suite for the `OrderRepository` driven port ([[port-definition]]), parameterized by an adapter factory, run against Postgres and the in-memory fake.

## Go

```go
// domain/order/repository_contract_test.go (or a shared testkit package)
// THE CONTRACT — written once, against the port type only.
func RepositoryContract(t *testing.T, newRepo func(t *testing.T) order.Repository) {
    ctx := context.Background()

    t.Run("save then find round-trips the order", func(t *testing.T) {
        repo := newRepo(t)
        o := mustNewOrder(t, "cust-1")
        if err := repo.Save(ctx, o); err != nil {
            t.Fatalf("save: %v", err)
        }
        got, err := repo.FindByID(ctx, o.ID())
        if err != nil {
            t.Fatalf("find: %v", err)
        }
        if got.ID() != o.ID() {
            t.Fatal("round-trip failed")
        }
    })

    t.Run("find missing id returns ErrNotFound", func(t *testing.T) {
        repo := newRepo(t)
        _, err := repo.FindByID(ctx, order.ID("missing"))
        if !errors.Is(err, order.ErrNotFound) { // DOMAIN error, regardless of tech
            t.Fatalf("want ErrNotFound, got %v", err)
        }
    })

    t.Run("saving the same id twice returns ErrDuplicate", func(t *testing.T) {
        repo := newRepo(t)
        o := mustNewOrder(t, "cust-1")
        _ = repo.Save(ctx, o)
        if err := repo.Save(ctx, o); !errors.Is(err, order.ErrDuplicate) {
            t.Fatalf("want ErrDuplicate, got %v", err)
        }
    })
}
```

```go
// infrastructure/postgres/order_repository_test.go — real infra via testcontainers
func TestPostgresOrderRepository(t *testing.T) {
    order_test.RepositoryContract(t, func(t *testing.T) order.Repository {
        db := startPostgresContainer(t) // fresh schema per test → clean state
        return postgres.NewOrderRepository(db)
    })
}

// infrastructure/memory/order_repository_test.go — the fake runs the SAME suite
func TestInMemoryOrderRepository(t *testing.T) {
    order_test.RepositoryContract(t, func(t *testing.T) order.Repository {
        return memory.NewOrderRepository()
    })
}
```

## TypeScript

```typescript
// domain/order/order-repository.contract.ts
// THE CONTRACT — written once, against the port type only.
export function orderRepositoryContract(
  name: string,
  makeRepo: () => Promise<OrderRepository>,
) {
  describe(`OrderRepository contract: ${name}`, () => {
    it("save then findById round-trips the order", async () => {
      const repo = await makeRepo();
      const order = Order.create(customerId("cust-1"), someLines());
      await repo.save(order);
      const found = await repo.findById(order.id);
      expect(found?.id).toEqual(order.id);
    });

    it("findById on a missing id resolves null", async () => {
      const repo = await makeRepo();
      expect(await repo.findById(new OrderId("missing"))).toBeNull();
    });

    it("saving the same id twice rejects with DuplicateOrderError", async () => {
      const repo = await makeRepo();
      const order = Order.create(customerId("cust-1"), someLines());
      await repo.save(order);
      await expect(repo.save(order)).rejects.toBeInstanceOf(
        DuplicateOrderError,
      );
    });
  });
}
```

```typescript
// infrastructure/postgres/order-repository.itest.ts — testcontainers
orderRepositoryContract("Postgres", async () => {
  const pool = await startPostgresContainer(); // fresh schema → clean state
  return new PostgresOrderRepository(pool);
});

// infrastructure/memory/order-repository.test.ts — the fake runs the SAME suite
orderRepositoryContract("InMemory", async () => new InMemoryOrderRepository());
```

Notes that make the suite a real contract:

- Test bodies never mention SQL, pools, or maps — only port methods and domain errors. The technology appears only in each factory.
- The in-memory fake passes the identical suite, so use-case unit tests built on it ([[dependency-inversion]]) reflect production behavior.
- If a future `MongoOrderRepository` is added ([[adapter-pattern]]), it ships with one new factory line — the contract is already written.
