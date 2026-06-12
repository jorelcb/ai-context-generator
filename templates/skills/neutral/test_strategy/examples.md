# test-strategy — Examples (Go + TypeScript)

> Illustrative — adapt to the project's stack. One example per pyramid level, showing the rules from the skill (behavior-focused, table-driven, independent, deterministic).

## Unit — table-driven, behavior-focused

```go
// Go — discount.go's public API only; no internals asserted.
func TestApplyCoupon(t *testing.T) {
    t.Parallel()
    now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) // injected clock
    cases := []struct {
        name    string
        coupon  Coupon
        total   Money
        want    Money
        wantErr error
    }{
        {"valid 10% coupon", coupon("SAVE10", 10, now.Add(24*time.Hour)), money(100_00), money(90_00), nil},
        {"expired coupon rejected", coupon("OLD", 10, now.Add(-time.Hour)), money(100_00), Money{}, ErrCouponExpired},
        {"zero total stays zero", coupon("SAVE10", 10, now.Add(24*time.Hour)), money(0), money(0), nil},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got, err := ApplyCoupon(tc.total, tc.coupon, now)
            if !errors.Is(err, tc.wantErr) { t.Fatalf("err = %v, want %v", err, tc.wantErr) }
            if err == nil && got != tc.want { t.Fatalf("got %v, want %v", got, tc.want) }
        })
    }
}
```

```typescript
// TypeScript — vitest, same shape via it.each.
describe("applyCoupon", () => {
  const now = new Date("2026-06-01T00:00:00Z"); // injected clock
  it.each([
    [
      "valid 10% coupon",
      coupon("SAVE10", 10, addHours(now, 24)),
      100_00,
      90_00,
    ],
    ["zero total stays zero", coupon("SAVE10", 10, addHours(now, 24)), 0, 0],
  ])("%s", (_name, c, total, want) => {
    expect(applyCoupon(money(total), c, now)).toEqual(money(want));
  });

  it("rejects an expired coupon", () => {
    const expired = coupon("OLD", 10, addHours(now, -1));
    expect(() => applyCoupon(money(100_00), expired, now)).toThrow(
      CouponExpiredError,
    );
  });
});
```

## Integration — real DB via testcontainers, test owns its data

```go
// Go — testcontainers-go; gated with //go:build integration.
func TestOrderRepo_RoundTrip(t *testing.T) {
    db := startPostgres(t) // helper: container + migrations + t.Cleanup(terminate)
    repo := postgres.NewOrderRepo(db)
    o := testutil.NewOrder(testutil.WithID(uuid.New())) // unique per test

    require.NoError(t, repo.Save(context.Background(), o))
    got, err := repo.FindByID(context.Background(), o.ID())
    require.NoError(t, err)
    require.Equal(t, o.ID(), got.ID())
}
```

```typescript
// TypeScript — testcontainers + per-test unique IDs; no shared rows.
describe("OrderRepository (postgres)", () => {
  let container: StartedPostgreSqlContainer;
  let repo: PostgresOrderRepository;

  beforeAll(async () => {
    container = await new PostgreSqlContainer().start();
    repo = new PostgresOrderRepository(await poolFor(container));
  }, 60_000);
  afterAll(() => container.stop());

  it("round-trips an order", async () => {
    const order = buildOrder({ id: randomUUID() });
    await repo.save(order);
    expect((await repo.findById(order.id))?.id).toEqual(order.id);
  });
});
```

## Contract — one shared suite per boundary

```go
// Go — every implementation of the port passes the same suite.
func RepoContract(t *testing.T, make func(t *testing.T) OrderRepository) {
    t.Run("round-trip", func(t *testing.T) { /* save + find, as above */ })
    t.Run("missing id returns ErrNotFound", func(t *testing.T) {
        _, err := make(t).FindByID(context.Background(), uuid.New())
        require.ErrorIs(t, err, ErrNotFound)
    })
}
func TestInMemoryRepo(t *testing.T) { RepoContract(t, newMemRepo) }
func TestPostgresRepo(t *testing.T) { RepoContract(t, newPgRepo) } // integration tag
```

```typescript
// TypeScript — same idea as a parameterized describe block.
export function orderRepositoryContract(
  name: string,
  make: () => OrderRepository,
) {
  describe(`OrderRepository contract: ${name}`, () => {
    it("round-trips an order", async () => {
      /* ... */
    });
    it("returns null for an unknown id", async () => {
      expect(await make().findById(randomUUID())).toBeNull();
    });
  });
}
orderRepositoryContract("in-memory", () => new InMemoryOrderRepository());
```

## Regression test for a bug fix (always, both languages)

```go
// Bug #481: rounding truncated instead of half-up. This test FAILED before the fix.
func TestPrice_Rounding_HalfUp_Regression481(t *testing.T) {
    require.Equal(t, money(2_35), RoundPrice(2.345))
}
```

```typescript
// Bug #481 regression — failed before the fix, pins the behavior forever.
it("rounds half-up, not truncating (#481)", () => {
  expect(roundPrice(2.345)).toEqual(money(2_35));
});
```
