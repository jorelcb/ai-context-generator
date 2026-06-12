# Test Strategy Skill

> A test suite is a product: its users are future changes. Optimize for **fast, trustworthy feedback** — tests that fail when behavior breaks and stay green when only implementation moves. Language-agnostic; Go + TypeScript used as reference stacks.

## When to use

Setting up testing for a new project or module, deciding which test type a piece of code needs, improving coverage that exists but proves nothing, or diagnosing a slow/flaky suite.

## The test pyramid (normative shape)

| Level           | Verifies                                       | Doubles?                                     | Speed target        | Share of suite |
| --------------- | ---------------------------------------------- | -------------------------------------------- | ------------------- | -------------- |
| **Unit**        | One unit of logic, isolated                    | Fake/mock all I/O                            | ms; whole run < 30s | ~70%           |
| **Integration** | Real component pairs: DB queries, HTTP clients | Real infra (testcontainers) only at the edge | seconds             | ~20%           |
| **Contract**    | A boundary both sides agree on (API, port)     | One shared suite per contract                | seconds             | per boundary   |
| **E2E**         | Critical user journeys, fully wired            | Nothing                                      | minutes; run in CI  | ~5–10, not %   |

Inverted pyramids (many E2E, few unit) produce slow, flaky, undebuggable suites. If a behavior is testable at a lower level, test it there.

## What to test where — decision table

| Code under test                          | Test type                                                  |
| ---------------------------------------- | ---------------------------------------------------------- |
| Business rules, calculations, validation | Unit, exhaustively (table-driven), including edge cases    |
| SQL / repository implementations         | Integration against a real DB (testcontainers)             |
| External API clients                     | Integration against a recorded/mock server + contract test |
| HTTP handlers / CLI commands             | Unit with faked service + a few wired integration tests    |
| Critical user journeys (checkout, login) | One E2E each                                               |
| Getters, framework glue, generated code  | Don't test directly — covered transitively                 |

Bug fixes always get a **regression test that fails before the fix** — no exceptions.

## Test quality rules (normative)

- **Assert behavior, not implementation**: test through the public API; never assert "method X was called" when you can assert the observable outcome.
- **One behavior per test**; the name states scenario + expectation (`TestDiscount_ExpiredCoupon_Rejected`, `it("rejects an expired coupon")`).
- **Independent and order-free**: each test creates its own fixtures; no shared mutable state; parallel-safe (`t.Parallel()`, default in vitest).
- **Deterministic**: inject clocks, seed randomness, never `sleep` to wait — poll or use synchronization.
- **Arrange–Act–Assert** visible in every test body; extract builders for noisy setup.
- Tests mirror source structure and live next to (Go) or beside (`__tests__`/`*.test.ts`) the code they test.

## Coverage policy

Coverage is a **gap detector, not a goal**. Measure it (`go test -cover`, `vitest run --coverage`), look at _what_ is uncovered, and decide if it matters. A hard threshold (e.g. 80%) invites assertion-free tests written to pass the gate; instead enforce: critical-path packages reviewed for uncovered branches in [[code-review]].

## Anti-patterns

- Testing implementation details — suite breaks on every refactor ([[refactor-safely]] depends on this not happening).
- Mocking everything, including the thing under test's collaborators' collaborators — green tests, untested system.
- Shared mutable fixtures (global test DB rows, package-level vars) — order-dependent flakiness.
- Sleep-based waits and real-clock assertions — flaky by design.
- Assertion-free or snapshot-everything tests that pass no matter what.
- One E2E per acceptance criterion — push them down the pyramid.

## Verification (executable)

The strategy is enforced by commands, not intentions: `go test -race -cover ./...` (Go), `vitest run --coverage` or `jest --coverage` (TS) in CI on every PR; integration tests gated behind a tag/profile so the unit loop stays under 30s; flakiness surfaced with `go test -count=10` / `vitest --retry=0`. Full tooling per ecosystem and the flaky-test triage checklist are in **[reference.md](reference.md)**.

## Going deeper

- **[reference.md](reference.md)** — per-ecosystem tooling table, suite-layout conventions, flaky-test triage, mutation testing.
- **[examples.md](examples.md)** — table-driven unit test, integration test with testcontainers, contract test, in Go and TypeScript.
