# test-strategy — Reference

## Tooling per ecosystem

| Ecosystem   | Runner                  | Assertions/doubles                    | Integration infra                    | Coverage                                        | E2E                   |
| ----------- | ----------------------- | ------------------------------------- | ------------------------------------ | ----------------------------------------------- | --------------------- |
| Go          | `go test`               | stdlib + `testify`, hand-rolled fakes | `testcontainers-go`, `httptest`      | `go test -coverprofile` + `go tool cover -html` | `chromedp`, API-level |
| TypeScript  | `vitest` / `jest`       | built-in + `testdouble`/`vi.fn`       | `testcontainers`, `supertest`, `msw` | `--coverage` (v8/istanbul)                      | Playwright            |
| Python      | `pytest`                | built-in + `unittest.mock`            | `testcontainers-python`              | `coverage.py` / `pytest-cov`                    | Playwright            |
| Java/Kotlin | JUnit 5                 | AssertJ + Mockito                     | Testcontainers                       | JaCoCo                                          | Selenium/Playwright   |
| BDD layer   | godog (Go), cucumber-js | Gherkin features over the API         |                                      |                                                 |                       |

## CI pipeline shape (normative)

1. **PR loop (must stay fast):** lint + type check + unit tests with race/coverage. Budget: < 5 min total, unit tests < 30s.
2. **Merge loop:** integration + contract tests (testcontainers). Budget: < 15 min.
3. **Scheduled / pre-release:** E2E suite, mutation testing run, dependency audit.

Commands worth pinning in CI:

```bash
# Go
go test -race -shuffle=on -coverprofile=cover.out ./...
go test -tags=integration ./...           # gated integration tests
# TypeScript
vitest run --coverage
vitest run --sequence.shuffle             # order-independence check
```

## Suite layout conventions

| Concern               | Go                                         | TypeScript                                |
| --------------------- | ------------------------------------------ | ----------------------------------------- |
| Unit test location    | `foo_test.go` next to `foo.go`             | `foo.test.ts` next to / `__tests__/`      |
| Black-box enforcement | `package foo_test` (external test package) | import only from the public entry point   |
| Integration gating    | build tag `//go:build integration`         | separate vitest project / `*.int.test.ts` |
| Fixtures/builders     | `internal/testutil/` builders              | `test/builders/` factory functions        |
| E2E                   | `tests/e2e/`                               | `e2e/` with Playwright config             |

## Flaky-test triage checklist

A test that fails intermittently is **quarantined the same day** (skipped with a tracking issue) — a suite people retry is a suite people ignore.

- [ ] Reproduce: `go test -count=20 -race ./pkg/...` / `vitest run --retry=0 --sequence.shuffle`.
- [ ] Time: real clock, timezones, `time.Now()` not injected, midnight/DST boundaries?
- [ ] Order: passes alone, fails in suite → shared state; run shuffled to confirm.
- [ ] Concurrency: race detector clean? Channels/promises awaited?
- [ ] Sleep-based waits → replace with polling (`require.Eventually`, `vi.waitFor`) or events.
- [ ] External dependency (network, real service) in a unit test → fake it or move to integration.
- [ ] Resource leaks: ports, temp dirs, containers not cleaned between tests.

## Mutation testing (trust, beyond coverage)

Coverage says the line ran; mutation testing says a bug there would be caught.

| Ecosystem  | Tool                        | Invocation               |
| ---------- | --------------------------- | ------------------------ |
| Go         | `go-mutesting` / `gremlins` | `gremlins unleash ./...` |
| TypeScript | StrykerJS                   | `npx stryker run`        |
| Python     | `mutmut`                    | `mutmut run`             |

Use sparingly: run on the highest-value packages (money, auth, core domain), not the whole repo; treat surviving mutants as missing test cases.

## Test data rules

- Builders/factories with sensible defaults and per-test overrides; never share fixture instances across tests.
- Each integration test owns its rows: unique IDs per test, truncate-or-transaction-rollback between tests.
- Golden files (`testdata/` in Go, `__snapshots__` in TS) only for genuinely stable, human-reviewed output — regenerating them must be a deliberate, reviewed act ([[code-review]] flags wholesale snapshot updates).

## What "done" looks like for a strategy

- [ ] Every level of the pyramid has at least one example test acting as a template.
- [ ] PR loop runs unit tests automatically and fails on regression.
- [ ] Integration tests run against real infra (containers), not mocks of the DB.
- [ ] Each public boundary has a contract test (shared suite per port/API).
- [ ] Flaky-test quarantine policy is written down and applied.
- [ ] Coverage is reported per package and reviewed, not gate-only.
