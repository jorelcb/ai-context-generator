# saga-orchestrator — Examples (Go + TypeScript)

> Illustrative — adapt to the stack. Order fulfillment: `OrderPlaced` → `ReserveStock` → `TakePayment`. On `PaymentDeclined`, compensate in reverse: `ReleaseStock`. The decision function is pure — `(state, event) → (newState, commands)` — so it unit-tests without infra; the runner persists state + commands in one transaction.

## Go

```go
// saga/order_fulfillment.go — PURE decision function (no I/O)
type State struct {
    SagaID string; Step string // "reserve-stock" | "take-payment" | "done" | "compensating" | "compensated"
    OrderID string; PaymentID string
}

func Decide(s State, e Envelope) (State, []Command) {
    switch ev := e.Payload.(type) {
    case OrderPlaced:
        if s.Step != "" { return s, nil } // dedup: saga already started
        s.Step, s.OrderID = "reserve-stock", ev.AggregateID
        return s, []Command{ReserveStock{
            CommandID: cmdID(s.SagaID, "reserve-stock"), // deterministic → handler dedups
            OrderID:   ev.AggregateID, Lines: ev.Lines,
        }}
    case StockReserved:
        if s.Step != "reserve-stock" { return s, nil } // redelivery: step already passed
        s.Step = "take-payment"
        return s, []Command{TakePayment{
            CommandID: cmdID(s.SagaID, "take-payment"),
            OrderID:   s.OrderID, AmountCents: ev.TotalCents,
        }}
    case PaymentApproved:
        if s.Step != "take-payment" { return s, nil }
        s.Step = "done" // hardest-to-undo step was last; nothing left to compensate
        return s, nil
    case PaymentDeclined: // business rejection — designed outcome, not an error
        if s.Step != "take-payment" { return s, nil }
        s.Step = "compensating"
        return s, []Command{ReleaseStock{ // reverse order of completed steps
            CommandID: cmdID(s.SagaID, "reserve-stock", "compensate"),
            OrderID:   s.OrderID,
        }}
    case StockReleased:
        if s.Step != "compensating" { return s, nil }
        s.Step = "compensated" // honest terminal state
        return s, nil
    }
    return s, nil
}

// saga/runner.go — persistence + dispatch (one transaction, OCC on the saga row)
func (r *Runner) Handle(ctx context.Context, e Envelope) error {
    return r.uow.Transact(ctx, func(tx Tx) error {
        s, version, err := r.store.Load(tx, sagaIDFor(e)) // "order-fulfillment:{orderID}"
        if err != nil { return err }
        next, cmds := Decide(s, e)
        if err := r.store.Save(tx, next, version); err != nil { return err } // OCC
        return r.outbox.AppendCommands(tx, cmds) // dispatched via outbox, never directly
    })
}
```

## TypeScript

```typescript
// saga/order-fulfillment.ts — PURE decision function (no I/O)
type Step =
  | ""
  | "reserve-stock"
  | "take-payment"
  | "done"
  | "compensating"
  | "compensated";
interface State {
  sagaId: string;
  step: Step;
  orderId: string;
}

export function decide(
  s: State,
  e: EventEnvelope,
): { state: State; commands: Command[] } {
  switch (e.eventType) {
    case "OrderPlaced": {
      if (s.step !== "") return { state: s, commands: [] }; // dedup: already started
      const ev = e.payload as OrderPlaced;
      return {
        state: { ...s, step: "reserve-stock", orderId: ev.aggregateId },
        commands: [
          {
            type: "ReserveStock",
            commandId: cmdId(s.sagaId, "reserve-stock"), // deterministic → handler dedups
            orderId: ev.aggregateId,
            lines: ev.lines,
          },
        ],
      };
    }
    case "StockReserved": {
      if (s.step !== "reserve-stock") return { state: s, commands: [] }; // redelivery
      const ev = e.payload as StockReserved;
      return {
        state: { ...s, step: "take-payment" },
        commands: [
          {
            type: "TakePayment",
            commandId: cmdId(s.sagaId, "take-payment"),
            orderId: s.orderId,
            amountCents: ev.totalCents,
          },
        ],
      };
    }
    case "PaymentApproved":
      if (s.step !== "take-payment") return { state: s, commands: [] };
      return { state: { ...s, step: "done" }, commands: [] };
    case "PaymentDeclined": // designed outcome → compensate in reverse order
      if (s.step !== "take-payment") return { state: s, commands: [] };
      return {
        state: { ...s, step: "compensating" },
        commands: [
          {
            type: "ReleaseStock",
            commandId: cmdId(s.sagaId, "reserve-stock", "compensate"),
            orderId: s.orderId,
          },
        ],
      };
    case "StockReleased":
      if (s.step !== "compensating") return { state: s, commands: [] };
      return { state: { ...s, step: "compensated" }, commands: [] }; // honest terminal
    default:
      return { state: s, commands: [] };
  }
}

// saga/runner.ts — persistence + dispatch (one transaction, OCC on the saga row)
export class SagaRunner {
  async handle(e: EventEnvelope): Promise<void> {
    await this.uow.transact(async (tx) => {
      const { state, version } = await this.store.load(tx, sagaIdFor(e));
      const { state: next, commands } = decide(state, e);
      await this.store.save(tx, next, version); // OCC: WHERE version = expected
      await this.outbox.appendCommands(tx, commands); // via outbox, never direct dispatch
    });
  }
}
```

Note in both: step guards make every transition redelivery-safe; command IDs derive from `(sagaId, step)`; payment (hardest to undo) runs last so the only compensation needed is `ReleaseStock`; and the saga never touches `Order` or stock directly — only commands cross the boundary.
