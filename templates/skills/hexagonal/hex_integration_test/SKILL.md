# Hexagonal Integration Testing — Contract Tests

> **Terminology (normative):** **inbound = driving = primary**, **outbound = driven = secondary**. Driven ports are interfaces in the Domain ([[port-definition]]); driven adapters in Infrastructure implement them ([[adapter-pattern]]). This skill is the executable proof that the hexagon delivers what it promises: **swappability**.

## When to use

Adding a new adapter, validating that an existing adapter satisfies its port, creating the shared suite any future implementation must pass, or deciding what an adapter test should (and should not) assert.

## The contract test concept

A **contract test** is a behavioral specification of the port, written once and run against **every** implementation:

- Postgres adapter — passes the suite.
- MongoDB adapter — passes the suite.
- In-memory fake used by use-case unit tests — passes the **same** suite.

If all pass, the adapters are interchangeable and the fake your unit tests rely on ([[dependency-inversion]]) is honest. Without contract tests, "swappable" is wishful thinking. Write the suite **parameterized by a factory** that produces the adapter under test.

## Test structure

1. **Setup** — instantiate the adapter via the factory (real or containerized infrastructure for real adapters; nothing for in-memory).
2. **Exercise** — call port methods through the **port type**, covering happy paths and the failure modes the port documents.
3. **Verify** — assert observable port behavior (returned values, domain errors), never adapter internals.

## What to test

- **Happy path:** the operation succeeds with the expected domain output (e.g. save → find round-trip).
- **Domain error paths:** `ErrNotFound`, `ErrDuplicate` — produced correctly from each technology's native failure.
- **Edge cases:** empty results, large payloads, concurrent access where the port promises safety.
- **Idempotency**, where the port specifies it.

## What NOT to test

- Adapter internals — which SQL it generates, which HTTP verb it uses, how it retries. Implementation details; asserting them welds the suite to one technology.
- The port itself — it is an interface; it has no logic.
- Business rules — those live in Domain/Application unit tests, not here.

## Test infrastructure

- **Driven adapters (DB, message bus, external API client):** real or containerized infrastructure — testcontainers, a local broker, a vendor sandbox/mock-server. **Never mock the database or HTTP server**: the adapter's whole job is speaking that protocol correctly; mocking it tests nothing.
- **In-memory fakes:** run in the same suite with zero setup — that is the point.
- **Driving adapters (HTTP handlers, CLI):** integration-test against the wired-up Application Service with **stubbed driven ports**; assert protocol mapping (status codes, payloads, error translation).

## Anti-patterns

- A test that passes on Postgres but fails on MongoDB — it asserts storage details, not the contract.
- Mocking the entire stack — an "integration" test that integrates nothing.
- Reaching into adapter privates (reflection, exported test hooks) to assert state.
- Skipping the in-memory fake in the suite — the fake silently drifts from production behavior.
- Contract tests that only cover happy paths — error translation is where adapters diverge most.

## Verification

The suite is correct when the **same test code** passes against every valid implementation of the port, and a failing run pinpoints which adapter broke the contract. Pair it with a dependency fitness function so the boundary holds at compile time too — tooling in [reference.md](reference.md).

If you use full Clean Architecture + DDD, the clean-ddd pack's integrated `hexagonal-port` skill includes a condensed version of this strategy; architecture packs are mutually exclusive at install time.

## Going deeper

- [reference.md](reference.md) — suite-design checklist, infrastructure matrix, per-ecosystem fitness functions.
- [examples.md](examples.md) — a parameterized contract suite for `OrderRepository` run against Postgres and in-memory, in Go and TypeScript.
