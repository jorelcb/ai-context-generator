# refactor-safely — Examples (Go + TypeScript)

> Illustrative — adapt to the project's stack. Both examples preserve behavior: the tests shown do not change between "before" and "after".

## Extract function (Go) — the workhorse

The test that stays untouched through the whole refactor:

```go
func TestQuote_TotalWithVolumeDiscount(t *testing.T) {
    q := NewQuote(items(3, 50_00)) // 3 items, $50 each
    require.Equal(t, money(135_00), q.Total()) // 10% off over $100
}
```

**Before** — one function, mixed concerns:

```go
func (q *Quote) Total() Money {
    var sum Money
    for _, it := range q.items {
        sum += it.Price * Money(it.Qty)
    }
    if sum > 100_00 {            // discount rule buried in the loop's tail
        sum = sum - sum/10
    }
    return sum
}
```

**Steps** (each one: edit → `go test ./...` → commit):

1. Extract `subtotal()` — commit `refactor: extract Quote.subtotal`.
2. Extract `applyVolumeDiscount()` — commit `refactor: extract volume discount rule`.

**After** — same behavior, named rules:

```go
func (q *Quote) Total() Money { return applyVolumeDiscount(q.subtotal()) }

func (q *Quote) subtotal() Money {
    var sum Money
    for _, it := range q.items { sum += it.Price * Money(it.Qty) }
    return sum
}

func applyVolumeDiscount(sum Money) Money {
    if sum > 100_00 { return sum - sum/10 }
    return sum
}
```

## Replace conditional with polymorphism (TypeScript)

The pinned behavior:

```typescript
it.each([
  ["card", 100_00, 102_90], // 2.9% fee
  ["bank", 100_00, 100_50], // flat 50¢
])("charges the %s fee", (method, amount, want) => {
  expect(totalWithFees(payment(method, amount))).toEqual(want);
});
```

**Before** — type-tag switch duplicated wherever payments are handled:

```typescript
function totalWithFees(p: Payment): number {
  switch (p.method) {
    case "card":
      return p.amount + Math.round(p.amount * 0.029);
    case "bank":
      return p.amount + 50;
    default:
      throw new Error(`unknown method ${p.method}`);
  }
}
```

**Steps** (each one: edit → `vitest run` → commit):

1. **Expand**: add a `FeePolicy` interface + one class per method; `totalWithFees` still uses the switch. Green, commit.
2. **Migrate**: route `totalWithFees` through a `policies` map. Green, commit.
3. **Contract**: delete the switch; `knip`/`tsc` confirm nothing references it. Green, commit.

**After**:

```typescript
interface FeePolicy {
  total(amount: number): number;
}

const policies: Record<PaymentMethod, FeePolicy> = {
  card: { total: (a) => a + Math.round(a * 0.029) },
  bank: { total: (a) => a + 50 },
};

function totalWithFees(p: Payment): number {
  const policy = policies[p.method];
  if (!policy) throw new Error(`unknown method ${p.method}`);
  return policy.total(p.amount);
}
```

Adding a new payment method is now one entry in `policies` — no scattered switches to hunt down.

## Characterization test before refactoring legacy code (Go)

No tests exist for `legacyParse`; pin what it does **today**, bugs included:

```go
func TestLegacyParse_Characterization(t *testing.T) {
    // Assertions copied from ACTUAL current output — not from a spec.
    got := legacyParse("a;b;;c")
    require.Equal(t, []string{"a", "b", "c"}, got) // NB: drops empties — current behavior, kept for now
}
```

Only after this is green does the safety loop start. The empty-segment drop, if it's a bug, becomes a separate `fix:` with its own regression test afterwards.

## The revert move (either language)

```bash
go test ./pricing/...        # red after a step — cause not obvious
git checkout -- . && git clean -fd   # back to last green commit, zero debugging
# retry as two smaller steps: extract first, THEN rename
```

Reverting a 5-minute step costs 5 minutes. Debugging a broken refactor costs the afternoon.
