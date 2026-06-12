# adapter-pattern — Examples (Go + TypeScript)

> Illustrative — adapt to the stack. One driven adapter (Postgres behind `order.Repository`), one driving adapter (HTTP calling `PlaceOrder`), and the composition root. Ports come from [[port-definition]].

## Go

```go
// infrastructure/postgres/order_repository.go — DRIVEN ADAPTER
package postgres

type OrderRepository struct{ db *sql.DB }

func NewOrderRepository(db *sql.DB) *OrderRepository { return &OrderRepository{db: db} }

func (r *OrderRepository) Save(ctx context.Context, o *order.Order) error {
    _, err := r.db.ExecContext(ctx,
        `INSERT INTO orders (id, customer_id, status) VALUES ($1, $2, $3)`,
        string(o.ID()), o.CustomerID(), o.Status())
    if isUniqueViolation(err) {
        return order.ErrDuplicate // error TRANSLATED at the boundary
    }
    return err
}

func (r *OrderRepository) FindByID(ctx context.Context, id order.ID) (*order.Order, error) {
    row := r.db.QueryRowContext(ctx, `SELECT ... FROM orders WHERE id = $1`, string(id))
    var rec orderRecord // persistence model, PRIVATE to the adapter
    if err := row.Scan(&rec.id, &rec.customerID, &rec.status); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, order.ErrNotFound // never leak sql.ErrNoRows
        }
        return nil, fmt.Errorf("order repo: %w", err)
    }
    return rec.toDomain()
}
```

```go
// interfaces/http/place_order_handler.go — DRIVING ADAPTER (thin)
func PlaceOrderHandler(svc *placeorder.PlaceOrder) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req placeOrderRequest // wire DTO, private to the adapter
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "bad request", http.StatusBadRequest) // format only
            return
        }
        id, err := svc.Execute(r.Context(), req.toCommand())
        switch {
        case errors.Is(err, order.ErrDuplicate):
            http.Error(w, "duplicate order", http.StatusConflict) // domain → protocol
        case err != nil:
            http.Error(w, "internal error", http.StatusInternalServerError)
        default:
            w.WriteHeader(http.StatusCreated)
            json.NewEncoder(w).Encode(map[string]string{"id": string(id)})
        }
    }
}

// cmd/api/main.go — COMPOSITION ROOT: the only place adapters are constructed
func main() {
    db := mustOpenDB()
    repo := postgres.NewOrderRepository(db)          // driven adapter
    notifier := smtp.NewPlacedNotifier(mustSMTP())   // driven adapter
    svc := placeorder.New(repo, notifier)            // app service gets ports
    http.Handle("POST /orders", PlaceOrderHandler(svc)) // driving adapter
}
```

## TypeScript

```typescript
// infrastructure/postgres/order-repository.ts — DRIVEN ADAPTER
export class PostgresOrderRepository implements OrderRepository {
  constructor(private readonly pool: Pool) {}

  async save(order: Order): Promise<void> {
    try {
      await this.pool.query(
        "INSERT INTO orders (id, customer_id, status) VALUES ($1, $2, $3)",
        [order.id.value, order.customerId.value, order.status],
      );
    } catch (e) {
      if (isUniqueViolation(e)) throw new DuplicateOrderError(); // translated
      throw e;
    }
  }

  async findById(id: OrderId): Promise<Order | null> {
    const res = await this.pool.query("SELECT ... WHERE id = $1", [id.value]);
    return res.rows[0] ? toDomain(res.rows[0]) : null; // row model stays inside
  }
}
```

```typescript
// interfaces/http/place-order-handler.ts — DRIVING ADAPTER (thin)
export function placeOrderHandler(svc: PlaceOrder) {
  return async (req: Request, res: Response) => {
    const parsed = placeOrderSchema.safeParse(req.body); // format validation only
    if (!parsed.success) return res.status(400).json(parsed.error.flatten());
    try {
      const id = await svc.execute(toCommand(parsed.data));
      res.status(201).json({ id: id.value });
    } catch (e) {
      if (e instanceof DuplicateOrderError) return res.status(409).end();
      res.status(500).end(); // domain → protocol mapping, nothing else
    }
  };
}

// main.ts — COMPOSITION ROOT
const repo = new PostgresOrderRepository(pool);
const notifier = new SmtpPlacedNotifier(transport);
const placeOrder = new PlaceOrder(repo, notifier);
app.post("/orders", placeOrderHandler(placeOrder));
```

Both adapters are swappable (in-memory repo for tests, gRPC instead of HTTP) because all knowledge of the technology — including its errors — stays inside the adapter. The shared contract test that keeps them honest is in [[hex-integration-test]].
