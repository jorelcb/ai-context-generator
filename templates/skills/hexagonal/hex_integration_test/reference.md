# hex-integration-test — Reference

## Contract suite design checklist

- [ ] The suite is written against the **port type**, parameterized by a factory producing the adapter under test.
- [ ] Every production adapter AND the in-memory fake run the same suite.
- [ ] Each documented domain error of the port has at least one test forcing it (e.g. find a missing ID → `ErrNotFound`; double save → `ErrDuplicate`).
- [ ] Assertions use only what crosses the port: return values, domain errors, observable state via other port methods.
- [ ] No assertion mentions SQL, collections, HTTP verbs, or any vendor concept.
- [ ] Each run starts from a clean state (truncate/rollback/fresh container) so order does not matter.
- [ ] Concurrency tests exist if (and only if) the port documents concurrent safety ([[port-definition]]).

## Infrastructure matrix

| Adapter under test         | Backing infrastructure for the test                          | Notes                                          |
| -------------------------- | ------------------------------------------------------------ | ---------------------------------------------- |
| SQL repository             | testcontainers (Postgres/MySQL) or embedded DB               | Never a mocked driver                          |
| Document-store repository  | testcontainers (Mongo, etc.)                                 | Same suite as the SQL one — that's the point   |
| External API gateway       | Vendor sandbox or recorded mock-server (WireMock, nock, VCR) | Mock the _remote server_, not your HTTP client |
| Message publisher/consumer | Containerized broker (Kafka/Rabbit) or in-process broker     | Assert delivery via the consuming port         |
| In-memory fake             | None                                                         | Must run in the same contract suite            |
| Driving adapter (HTTP/CLI) | Wired-up Application Service + stubbed driven ports          | Assert protocol mapping, not business rules    |

## What a failing contract run means

- **One adapter fails, others pass** → that adapter violates the contract (usually error translation). Fix the adapter, not the test.
- **All adapters fail** → the contract changed; this is a **port change** — update port docs and every implementation deliberately ([[port-definition]]).
- **Only the in-memory fake fails** → the fake drifted; your use-case unit tests have been lying. Highest-priority fix.

## Fitness functions per ecosystem (compile-time complement)

Contract tests prove behavioral swappability at runtime; a dependency rule proves the boundary at build time. Use both.

| Ecosystem   | Tool                         | What to configure                                                     |
| ----------- | ---------------------------- | --------------------------------------------------------------------- |
| Go          | `go-arch-lint` or `depguard` | Forbid `domain`/`application` → `infrastructure` and driver packages  |
| TypeScript  | `eslint-plugin-boundaries`   | Inward-only edges between `domain`/`application`/`infrastructure`     |
| Python      | `import-linter`              | `layers` contract with `domain` at the bottom                         |
| Java/Kotlin | ArchUnit                     | `..domain..` must not depend on `..infrastructure..` (runs as a test) |
| PHP         | `deptrac`                    | `Domain` layer with empty allowlist                                   |

Config sketches live in [[dependency-inversion]]'s reference.

## Minimal manual check (no tooling yet)

- Run the contract suite with the factory pointed at the in-memory fake **and** at the real adapter; both green.
- Swap drill: change only the composition root to the alternative adapter; the application boots and end-to-end smoke passes.
- `grep` the contract test file for technology words (`sql`, `pool`, `collection`, `axios`) — should appear only in the per-adapter factory, never in test bodies.

## CI layout suggestion

- **Fast lane (every push):** use-case unit tests with fakes + contract suite against in-memory adapters.
- **Integration lane (PR gate):** contract suite against containerized infrastructure (testcontainers).
- Tag/skip mechanism so the integration lane is skippable locally but mandatory before merge.
