# clean-arch-layer — Examples (Go + TypeScript)

> Illustrative vertical slice of "place an order". Adapt to your stack — these show _placement and dependency direction_, not production detail.

## Go

```go
// domain/order/repository.go  — DRIVEN PORT (interface in Domain)
package order
type Repository interface {
    Save(ctx context.Context, o *Order) error
    FindByID(ctx context.Context, id ID) (*Order, error)
}

// application/place_order.go — APPLICATION SERVICE (concrete, orchestrates; no business logic)
package application
type PlaceOrder struct{ repo order.Repository } // depends on Domain abstraction
func NewPlaceOrder(r order.Repository) *PlaceOrder { return &PlaceOrder{r} }
func (s *PlaceOrder) Handle(ctx context.Context, cmd PlaceOrderCmd) (string, error) {
    o, err := order.New(order.CustomerID(cmd.CustomerID), toLines(cmd.Items)) // domain enforces invariants
    if err != nil { return "", err }
    if err := s.repo.Save(ctx, o); err != nil { return "", err }
    return string(o.ID()), nil
}

// infrastructure/postgres/order_repo.go — DRIVEN ADAPTER (implements the port)
package postgres
type OrderRepo struct{ db *sql.DB }
func (r *OrderRepo) Save(ctx context.Context, o *order.Order) error { /* SQL */ return nil }

// interfaces/http/order_handler.go — PRIMARY ADAPTER (calls the app service)
func (h *OrderHandler) Post(w http.ResponseWriter, req *http.Request) {
    cmd := decode(req)            // transport → DTO
    id, err := h.placeOrder.Handle(req.Context(), cmd)
    write(w, id, err)
}

// cmd/main.go — COMPOSITION ROOT (wiring; the only place that knows concretions)
repo := postgres.NewOrderRepo(db)
svc  := application.NewPlaceOrder(repo)
handler := http.NewOrderHandler(svc)
```

## TypeScript

```typescript
// domain/order/order-repository.ts — DRIVEN PORT
export interface OrderRepository {
  save(o: Order): Promise<void>;
  findById(id: OrderId): Promise<Order | null>;
}

// application/place-order.ts — APPLICATION SERVICE (concrete orchestrator)
export class PlaceOrder {
  constructor(private readonly repo: OrderRepository) {} // depends on Domain abstraction
  async handle(cmd: PlaceOrderCmd): Promise<string> {
    const order = Order.create(
      new CustomerId(cmd.customerId),
      toLines(cmd.items),
    ); // domain enforces invariants
    await this.repo.save(order);
    return order.id.value;
  }
}

// infrastructure/postgres/order-repository.ts — DRIVEN ADAPTER
export class PostgresOrderRepository implements OrderRepository {
  constructor(private readonly db: Pool) {}
  async save(o: Order): Promise<void> {
    /* SQL */
  }
  async findById(id: OrderId) {
    /* SQL */ return null;
  }
}

// interfaces/http/order-controller.ts — PRIMARY ADAPTER
export class OrderController {
  constructor(private readonly placeOrder: PlaceOrder) {}
  async post(req: Request, res: Response) {
    const id = await this.placeOrder.handle(toCmd(req.body)); // DTO in
    res.status(201).json({ id });
  }
}

// main.ts — COMPOSITION ROOT
const repo = new PostgresOrderRepository(pool);
const svc = new PlaceOrder(repo);
const controller = new OrderController(svc);
```

Notice in both: the **driving side has no interface** (PlaceOrder is concrete) because there is a single primary adapter — exactly the "interface optional" rule.
