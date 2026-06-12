# ddd-entity — Examples (Go + TypeScript)

> Illustrative. Shows: a self-validating value object, an aggregate root whose invariant cannot be bypassed, a factory, and the invariant-failure test.

## Go

```go
package order

// Value Object: self-validating, immutable, equality by value.
type Money struct{ cents int64; currency string }

func NewMoney(cents int64, currency string) (Money, error) {
    if cents < 0 { return Money{}, errors.New("money cannot be negative") }
    if currency == "" { return Money{}, errors.New("currency required") }
    return Money{cents, currency}, nil
}
func (m Money) Add(o Money) (Money, error) {
    if m.currency != o.currency { return Money{}, errors.New("currency mismatch") }
    return Money{m.cents + o.cents, m.currency}, nil
}

// Aggregate root: invariant "total never negative", access only via methods.
type Order struct {
    id    ID
    total Money
    lines []Line
}

func New(customer CustomerID, lines []Line) (*Order, error) { // factory
    if len(lines) == 0 { return nil, errors.New("order needs at least one line") }
    total, err := sum(lines)
    if err != nil { return nil, err }
    return &Order{id: NewID(), total: total, lines: lines}, nil
}
func (o *Order) ID() ID { return o.id }
```

```go
// invariant-failure test
func TestNew_RejectsEmptyOrder(t *testing.T) {
    _, err := New("cust-1", nil)
    if err == nil { t.Fatal("expected empty order to be rejected") }
}
```

## TypeScript

```typescript
// Value Object: immutable, validated in factory, equality by value.
export class Money {
  private constructor(
    readonly cents: number,
    readonly currency: string,
  ) {}
  static of(cents: number, currency: string): Money {
    if (cents < 0) throw new DomainError("money cannot be negative");
    if (!currency) throw new DomainError("currency required");
    return new Money(cents, currency);
  }
  add(o: Money): Money {
    if (o.currency !== this.currency)
      throw new DomainError("currency mismatch");
    return new Money(this.cents + o.cents, this.currency);
  }
  equals(o: Money) {
    return this.cents === o.cents && this.currency === o.currency;
  }
}

// Aggregate root: invariant enforced in factory; members not mutable from outside.
export class Order {
  private constructor(
    readonly id: OrderId,
    private total: Money,
    private lines: Line[],
  ) {}
  static create(customer: CustomerId, lines: Line[]): Order {
    // factory
    if (lines.length === 0)
      throw new DomainError("order needs at least one line");
    return new Order(OrderId.next(), sum(lines), lines);
  }
}
```

```typescript
// invariant-failure test
it("rejects an empty order", () => {
  expect(() => Order.create(customerId, [])).toThrow(DomainError);
});
```

Both: invalid state is unrepresentable — there is no path to a negative `Money` or an empty `Order`.
