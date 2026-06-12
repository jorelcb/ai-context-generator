# Dependency Inversion — Hexagonal Architecture

> **Terminology (normative):** **inbound = driving = primary**, **outbound = driven = secondary**. DIP is what makes the hexagon work: the application core declares **driven ports as interfaces in the Domain** ([[port-definition]]) and Infrastructure implements them ([[adapter-pattern]]) — so source dependencies always point inward, even when runtime calls go outward.

## When to use

Refactoring a direct infrastructure dependency out of the core, introducing a new external dependency the right way from the start, breaking a cyclic import between core and infrastructure, or fixing a test setup that can't run without a real database/network.

## The principle

- High-level modules (Domain, Application Services) **must not** depend on low-level modules (databases, frameworks, vendor SDKs).
- Both depend on **abstractions**: the driven ports the core owns.
- Abstractions do not depend on details; details (adapters) depend on abstractions.
- Direction test: at the source level, **all imports point toward the Domain**. If `domain` or `application` imports `infrastructure`, DIP is broken — no exceptions.

## Practical application

1. The core defines the abstraction: a driven port **in the Domain**, named by capability, signed with domain types.
2. Infrastructure implements it as an adapter.
3. The Application Service receives the port via **constructor injection** — it never constructs an adapter, never calls a factory that resolves to a concrete one.
4. The **composition root** (`cmd/main.go`, app bootstrap, DI container config) is the single place where concrete adapters meet abstract ports.

On the driving side DIP needs no interface by default: the Application Service is a concrete entry point that primary adapters call directly. Add a driving interface only with >1 primary adapter or for contract-testing the adapter — see [[port-definition]].

## Refactor workflow (direct dependency → port)

1. **Identify** the offending import in the core (e.g. `application` imports `database/sql` or a vendor SDK).
2. **Extract** the operations the core actually uses into a driven port interface owned by the Domain — capability-shaped, not a mirror of the vendor API.
3. **Move** the concrete code into an adapter package in Infrastructure; translate technical errors to domain errors there.
4. **Inject** the port into the use case via constructor; delete the direct import.
5. **Wire** the adapter at the composition root; add a contract test so the in-memory fake and the real adapter stay interchangeable ([[hex-integration-test]]).

## The testability payoff

With DIP applied, use cases unit-test against fakes implementing the ports: no database, no HTTP, no clock, no flakiness. If a use-case test needs Docker or network, DIP is broken somewhere — treat the test pain as the signal.

## Anti-patterns

- **Service locator** — dependencies resolved from a global registry hide the coupling DIP is supposed to expose.
- **Static factories / singletons in the core** that return concrete infrastructure (`db.GetConnection()` called from a use case).
- **Generic `IRepository<T>`** over capability-shaped ports — an abstraction must be _about something_; "abstract everything" inverts nothing.
- **Port mirroring the vendor API** (`ExecuteQuery(sql string)`) — the import is inverted but the coupling survives intact.
- Interfaces declared **next to the adapter** in Infrastructure — ownership inverted; the core must own its abstractions.

## Verification

DIP holds when the Domain and Application packages have **zero imports** of database drivers, HTTP frameworks, or vendor SDKs, and all use-case tests run with no external infrastructure. Enforce it with a dependency fitness function in CI — per-ecosystem tooling and config sketches in [reference.md](reference.md).

If you use full Clean Architecture + DDD, the clean-ddd pack covers this rule inside its integrated `hexagonal-port` skill; architecture packs are mutually exclusive at install time.

## Going deeper

- [reference.md](reference.md) — fitness functions per ecosystem with config sketches, manual checks, violation catalog.
- [examples.md](examples.md) — before/after refactor of a use case coupled to SQL, in Go and TypeScript.
