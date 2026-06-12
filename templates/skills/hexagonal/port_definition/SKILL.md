# Port Definition — Hexagonal Architecture

> **Terminology (normative):** **inbound = driving = primary** — HTTP handlers, CLI commands, schedulers, message consumers; they _call_ the application. **Outbound = driven = secondary** — repositories, gateways, notifiers, publishers; the application _needs_ them. **Driven ports are always interfaces owned by the Domain.** On the driving side the **Application Service is the entry point** (a concrete class/function); a driving interface is **optional**.

## When to use

Defining the contract for an external dependency (database, external API, message bus), introducing a new use case as the application's inbound API, or refactoring a direct concrete dependency into an explicit port (see [[dependency-inversion]] for the refactor itself).

## Inbound vs outbound ports

|                | Driven (outbound / secondary)                              | Driving (inbound / primary)                       |
| -------------- | ---------------------------------------------------------- | ------------------------------------------------- |
| Direction      | The application **calls** it                               | It **calls** the application                      |
| Examples       | `OrderRepository`, `PaymentGateway`, `OrderPlacedNotifier` | HTTP handler, CLI command, gRPC service, consumer |
| Contract       | **Interface in the Domain, always**                        | App Service **is** the entry; interface optional  |
| Implemented by | Driven adapters in Infrastructure ([[adapter-pattern]])    | Primary adapters call it directly                 |

## Driven ports (mandatory rules)

- Declare the interface **in the Domain**, next to the concept it serves (e.g. `OrderRepository` lives with `Order`).
- Name by **domain intent, never technology**: `OrderRepository`, not `PostgresOrderDAO`; `PaymentGateway`, not `StripeClient`.
- Signatures use **domain types only** — entities, value objects, IDs, domain errors. Never `*sql.Rows`, `*http.Request`, ORM rows, or vendor SDK types.
- Return **domain errors** (`ErrOrderNotFound`, `ErrPaymentDeclined`); the adapter translates technical errors before they cross the boundary.
- Keep each port a **single cohesive capability**. One repository per aggregate root; one gateway per external capability. Split when an interface accumulates unrelated reasons to change.

## Driving side (the optional interface)

The default entry point is the **Application Service itself** — `PlaceOrder` as a concrete class/function that primary adapters invoke directly. Add a driving interface **only** when:

- more than one primary adapter must depend on the same abstraction (e.g. HTTP + CLI + scheduler), **or**
- you want to test the primary adapter against a contract/double of the service.

An interface per service "just because" is ceremony, not architecture. When you do define one, name it after the use case (`PlaceOrderUseCase`), not after a generic service (`OrderManager`).

## Signature design

- Accept and return the language's idiomatic context/cancellation mechanism where applicable (`context.Context` in Go, `AbortSignal` in TS when meaningful).
- Document **concurrency expectations** (safe for concurrent use? per-request instance?) and **lifetime** (singleton, scoped) on the port, because every adapter must honor them.
- Prefer specific queries (`FindByID`, `FindPendingOlderThan`) over generic escape hatches (`Query(string)`) that leak technology through the back door.

## Anti-patterns

- Adapter/vendor types in port signatures — the most common and most corrosive leak.
- Technical errors crossing the boundary (`sql.ErrNoRows`, Axios errors) instead of domain errors.
- One god-port covering persistence + notification + payment.
- Generic `IRepository<T>` for everything — an abstraction must be _about something_.
- A driving interface for every Application Service by reflex.

## Verification

A well-defined port compiles in the core with **zero adapters present**, and can be implemented by at least two technologies without changing the port. Prove it with a dependency fitness function and a contract test suite ([[hex-integration-test]]) — tooling per ecosystem in [reference.md](reference.md).

If you use full Clean Architecture + DDD, the clean-ddd pack ships an integrated view of this topic (its `hexagonal-port` skill); architecture packs are mutually exclusive at install time, so use one or the other.

## Going deeper

- [reference.md](reference.md) — port checklist, naming table, per-ecosystem fitness functions, manual checks.
- [examples.md](examples.md) — driven ports and the driving entry point for `Order`/`PlaceOrder`, in Go and TypeScript.
