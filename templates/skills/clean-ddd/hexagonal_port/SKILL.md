# Hexagonal Architecture — Port & Adapter Skill

> **Reconciled model (Clean Arch + DDD):** **driven ports** are interfaces in the **Domain**, implemented by adapters in **Infrastructure**. On the **driving** side the entry point is an **Application Service** ([[cqrs-command]]); a driving interface is **optional**. The layer rules and the "interface optional" decision come from [[clean-arch-layer]].

## When to use

Integrating an external service, defining a repository/gateway/notifier contract, writing an API client or DB access, adding a queue producer/consumer, or asking "is this a port or an adapter?".

## Port design (normative)

- A **port** is an interface expressed in **domain terms**, owned by the inner layers.
- Name by **domain intent, not technology**: `OrderRepository`, not `PostgresOrderStore`; `PaymentGateway`, not `StripeClient`.
- Port methods speak **domain types** (entities, value objects, IDs) — never `sql.DB`, `http.Request`, ORM rows.
- Split by **cohesive capability**; many small ports beat one god-interface.

## Driven vs driving

|                | Driven (secondary / outbound)                           | Driving (primary / inbound)                               |
| -------------- | ------------------------------------------------------- | --------------------------------------------------------- |
| Direction      | The app **calls** it                                    | It **calls** the app                                      |
| Examples       | Repository, PaymentGateway, EmailSender, EventPublisher | HTTP handler, CLI command, gRPC service, message consumer |
| Interface      | **In Domain, always**                                   | App Service is the entry; interface **optional**          |
| Implemented in | Infrastructure                                          | Interfaces layer                                          |

## Adapter implementation — workflow

1. Identify the domain need (what capability is required?).
2. Define the **driven port** as an interface in the Domain, in domain types.
3. Implement the **adapter** in Infrastructure (it depends inward on the port).
4. Register the adapter at the **composition root** via dependency injection.
5. Write **adapter-specific tests** against real infrastructure (testcontainers / in-memory).

## Testing strategy

- **Domain / application logic**: unit tests with fake/mock adapters.
- **Adapters**: integration tests against real infra (testcontainers, in-memory DB).
- **Ports**: **contract tests** — one shared test suite every adapter of a port must pass (keeps swappability honest).

## Anti-patterns to avoid

- Technology-specific types in port signatures (`sql.DB`, `http.Request`, ORM entities).
- Adapter logic (retries, SQL, serialization) leaking into the domain.
- Skipping the port and coupling the app directly to a concrete client.
- One giant port; an interface per service "just because" on the driving side (see the optional-interface rule in [[clean-arch-layer]]).

## Verification (executable)

The promise of hexagonal is _swappability_; prove it with a fitness function, not by inspection. Per-ecosystem tooling, a port/adapter checklist, and a contract-test pattern are in **[reference.md](reference.md)**. Minimum: the Domain package imports zero infrastructure, and the app's unit tests run with no external dependencies.

## Examples

A driven port (Domain) + two swappable adapters (Postgres + in-memory) + a contract test, in Go and TypeScript: **[examples.md](examples.md)**.
