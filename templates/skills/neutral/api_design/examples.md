# api-design — Examples (Go + TypeScript)

> Illustrative — adapt to the project's stack. One resource (`orders`) end-to-end: spec fragment, error shape, pagination, and a handler in each language.

## OpenAPI fragment (the source of truth)

```yaml
paths:
  /v1/orders:
    get:
      summary: List orders
      parameters:
        - name: cursor
          in: query
          schema: { type: string }
        - name: limit
          in: query
          schema: { type: integer, maximum: 100, default: 50 }
      responses:
        "200":
          content:
            application/json:
              schema: { $ref: "#/components/schemas/OrderPage" }
    post:
      summary: Create an order
      parameters:
        - name: Idempotency-Key
          in: header
          required: true
          schema: { type: string }
      responses:
        "201":
          headers: { Location: { schema: { type: string } } }
        "422":
          content:
            application/problem+json:
              schema: { $ref: "#/components/schemas/Problem" }
components:
  schemas:
    Order: # representation DTO — NOT the DB row
      type: object
      required: [id, status, totalCents, createdAt]
      properties:
        id: { type: string, format: uuid }
        status: { type: string, enum: [open, shipped, cancelled] }
        totalCents: { type: integer }
        createdAt: { type: string, format: date-time }
    OrderPage:
      type: object
      required: [data]
      properties:
        data: { type: array, items: { $ref: "#/components/schemas/Order" } }
        nextCursor: { type: string, nullable: true }
```

Lint it (`spectral lint openapi.yaml`) and diff it (`oasdiff breaking`) in CI.

## Problem Details error (RFC 9457) — same shape on every endpoint

```json
{
  "type": "https://api.acme.dev/problems/validation",
  "title": "Request validation failed",
  "status": 422,
  "instance": "/v1/orders",
  "errors": [
    {
      "field": "items",
      "code": "min_items",
      "message": "at least one item is required"
    }
  ]
}
```

## Go handler — DTO mapping, pagination cap, Problem Details

```go
const maxLimit = 100

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
    limit := clampInt(queryInt(r, "limit", 50), 1, maxLimit) // cap server-side
    page, err := h.svc.ListOrders(r.Context(), ListQuery{
        Cursor: r.URL.Query().Get("cursor"),
        Status: r.URL.Query().Get("status"),
        Limit:  limit,
    })
    if errors.Is(err, ErrBadCursor) {
        writeProblem(w, 400, "invalid_cursor", "cursor is malformed or expired")
        return
    }
    if err != nil {
        writeProblem(w, 500, "internal", "unexpected error") // no internals leaked
        return
    }
    writeJSON(w, 200, OrderPage{Data: toOrderDTOs(page.Orders), NextCursor: page.Next})
}

// toOrderDTOs maps domain -> representation; DB-only fields never escape.
func toOrderDTOs(os []order.Order) []OrderDTO { /* explicit field mapping */ }
```

## TypeScript handler — zod schema validates AND documents

```typescript
const ListOrdersQuery = z.object({
  cursor: z.string().optional(),
  limit: z.coerce.number().int().min(1).max(100).default(50), // cap in the schema
  status: z.enum(["open", "shipped", "cancelled"]).optional(),
});

app.get("/v1/orders", async (req, res) => {
  const parsed = ListOrdersQuery.safeParse(req.query);
  if (!parsed.success) return problem(res, 400, "invalid_query", parsed.error);

  const page = await orderService.list(parsed.data);
  res.json({
    data: page.orders.map(toOrderDto),
    nextCursor: page.next ?? null,
  });
});

app.post("/v1/orders", async (req, res) => {
  const key = req.header("Idempotency-Key");
  if (!key) return problem(res, 400, "missing_idempotency_key");

  const result = await orderService.create(req.body, key); // replays return the original
  res.status(201).location(`/v1/orders/${result.id}`).json(toOrderDto(result));
});
```

With `zod-openapi`, `ListOrdersQuery` also generates the spec parameters — validation and contract cannot drift.

## Contract test pinning the contract (either stack)

```typescript
it("POST /v1/orders without items returns Problem Details 422", async () => {
  const res = await request(app)
    .post("/v1/orders")
    .set("Idempotency-Key", randomUUID())
    .send({ items: [] });
  expect(res.status).toBe(422);
  expect(res.headers["content-type"]).toContain("application/problem+json");
  expect(res.body.errors[0].code).toBe("min_items");
});
```

The status code, media type, and error `code` are asserted because all three are contract — renaming `min_items` would rightly fail this test ([[test-strategy]]).
