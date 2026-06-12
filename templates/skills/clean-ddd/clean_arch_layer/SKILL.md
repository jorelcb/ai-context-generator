# Clean Architecture + DDD — Layer Skill

> **Style:** Clean Architecture reconciled with DDD. Ports (interfaces) live in the **Domain**; orchestration is done by **Application Services** (not anemic "use case classes"). This is the **single source of truth** the other three skills reference — see [The Architecture Map](#the-architecture-map).

## When to use

Adding a new feature, creating an application service, implementing an adapter, deciding which layer code belongs to, or fixing a dependency-direction / import violation.

## The four layers

| Layer              | Contains                                                                                    | Knows about                  |
| ------------------ | ------------------------------------------------------------------------------------------- | ---------------------------- |
| **Domain**         | Entities, value objects, aggregates, domain services, domain events, **ports (interfaces)** | Nothing (no imports outward) |
| **Application**    | **Application Services** that orchestrate use cases (= CQRS command/query handlers), DTOs   | Domain only                  |
| **Infrastructure** | Implementations of driven ports: repositories, API clients, message publishers              | Application + Domain         |
| **Interfaces**     | Primary adapters: HTTP/CLI/gRPC handlers, message consumers                                 | Application + Domain         |

`Interfaces` and `Infrastructure` are both the _outer ring_; neither depends on the other.

## The Dependency Rule (non-negotiable)

- Dependencies point **inward only**: Interfaces/Infrastructure → Application → Domain.
- The **Domain imports nothing** outside itself (no DB, no HTTP, no framework).
- Outer layers depend on Domain **abstractions** (ports), never on concretions.
- Wiring happens once at the **composition root** (main / bootstrap) via dependency injection.

## Ports: where they live (the reconciled rule)

- **Driven ports** (outbound: repositories, gateways, notifiers) → interface in **Domain**, **always**. Implemented in Infrastructure. See [[hexagonal-port]].
- **Driving side** (inbound) → the **Application Service is the entry point** (a concrete type). A driving _interface_ is **optional** — add it only when more than one primary adapter needs the abstraction or to test a handler against a contract. **Do not** wrap every service in an interface "just because" (ceremony anti-pattern).

## Adding a feature — workflow

1. Model the domain concept → use [[ddd-entity]] (entity / VO / aggregate + invariants).
2. Declare the driven ports it needs as **interfaces in Domain** → use [[hexagonal-port]].
3. Write the **Application Service** that orchestrates the use case → use [[cqrs-command]] (command vs query).
4. Implement driven adapters in Infrastructure; implement the primary adapter in Interfaces.
5. Wire everything at the composition root.

## Common violations

- Domain importing infrastructure/framework packages.
- Application service performing HTTP/DB calls directly instead of through a port.
- Business logic in handlers/controllers (push it into the domain).
- Circular dependencies between layers; outer→outer coupling (Interfaces importing Infrastructure).

## Verification (executable)

Architecture rules must be enforced by a fitness function, not by hope. See the per-ecosystem tooling table and the directory-tree reference in **[reference.md](reference.md)**. Minimum manual check: grep the Domain package for outward imports — there must be none.

## Examples

Reference implementations (Go + TypeScript) of a full vertical slice across the four layers: **[examples.md](examples.md)**.

## The Architecture Map

How the four architecture skills divide the work — the shared mental model:

- **[[ddd-entity]]** → models the **Domain** (entities, VOs, aggregates, invariants).
- **[[hexagonal-port]]** → defines the **boundaries** (ports in Domain, adapters in outer ring).
- **[[cqrs-command]]** → structures the **Application Services** (commands change state, queries read).
- **[[clean-arch-layer]]** (this skill) → organizes the **layers** and enforces the dependency rule.

One architecture, four lenses. When they seem to disagree, **this skill's rules win**.
