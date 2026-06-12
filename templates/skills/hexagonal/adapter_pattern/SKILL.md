# Adapter Pattern — Hexagonal Architecture

> **Terminology (normative):** **inbound = driving = primary** adapters translate external triggers (HTTP, CLI, schedulers, messages) into calls to the application; **outbound = driven = secondary** adapters implement the **driven ports** the application needs (repositories, gateways, notifiers). Driven ports are always interfaces in the Domain ([[port-definition]]); driven adapters in Infrastructure implement them. On the driving side, adapters call the **Application Service directly** — a driving interface is optional.

## When to use

Implementing a repository against a concrete database, integrating an external HTTP API behind a gateway port, building an HTTP/gRPC handler or CLI command that invokes an Application Service, or wiring a message consumer to a use case.

## Driving adapters (inbound / primary)

- Translate the external trigger into a call on the Application Service: parse + validate **format** (types, required fields), build the command/query, invoke, map the result.
- Map domain errors back to the protocol: `ErrOrderNotFound` → HTTP 404, `ErrDuplicate` → 409, validation → 400.
- Keep them **thin**. Business rules creeping into a handler belong in the Application Service or the Domain. The test: deleting the adapter must lose zero business behavior.
- In Clean Architecture terms, the "Interfaces layer" _is_ the driving-adapter side — same concept, two vocabularies.

## Driven adapters (outbound / secondary)

- Implement **one driven port** against one technology: `PostgresOrderRepository implements order.Repository`.
- **Translate errors at the boundary**: `sql.ErrNoRows` → `order.ErrNotFound`, HTTP 402 from the PSP → `ErrPaymentDeclined`. No vendor error type ever crosses the port.
- Own all technical concerns — SQL, serialization, retries, circuit breakers, timeouts, connection pooling. The use case never knows they exist.
- Map between domain objects and persistence/wire models **inside** the adapter; ORM entities and DTOs never leak out.

## Composition root wiring

- Adapters are instantiated and injected **only at the composition root** (`cmd/main.go`, the DI container, the app bootstrap).
- The Application Service receives driven ports via constructor injection ([[dependency-inversion]]) and never constructs an adapter itself.
- Swapping Postgres → MongoDB, or real PSP → sandbox, is a one-line change at the root.

## Testing adapters

- **Driven adapters:** contract tests against the port — the same suite every implementation must pass — plus integration tests on real/containerized infrastructure. See [[hex-integration-test]].
- **Driving adapters:** integration-test against the wired-up core with stubbed driven ports; assert protocol mapping (status codes, payload shape, error translation).

## Anti-patterns

- Business logic in adapters (the #1 erosion vector).
- The core importing a concrete adapter — skipping the port entirely.
- Vendor exceptions/error types leaking past the port boundary.
- One adapter implementing several unrelated ports (one adapter = one port, generally).
- ORM entities doubling as domain entities.

## Verification

A correct adapter implements its port fully, passes the **same contract test suite** as any alternative implementation, and can be replaced (Postgres → MongoDB, REST → gRPC client) without touching the use case. Enforce the direction with a dependency fitness function — per-ecosystem tooling in [reference.md](reference.md).

If you use full Clean Architecture + DDD, the clean-ddd pack offers an integrated port-and-adapter view (its `hexagonal-port` skill); architecture packs are mutually exclusive at install time.

## Going deeper

- [reference.md](reference.md) — adapter checklist, error-translation table, per-ecosystem fitness functions.
- [examples.md](examples.md) — Postgres driven adapter + HTTP driving adapter for `Order`/`PlaceOrder`, in Go and TypeScript.
