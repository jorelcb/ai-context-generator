# adapter-pattern — Reference

## Adapter checklist

A **driven adapter** is correct when:

- [ ] It lives in Infrastructure and implements exactly one driven port from the Domain.
- [ ] Nothing in Domain or Application imports it (only the composition root does).
- [ ] Every technical error is translated to a domain error before returning.
- [ ] All technology concerns (SQL, HTTP, retries, serialization, pooling) stay inside it.
- [ ] Persistence/wire models are private to the adapter; only domain types cross the port.
- [ ] It passes the port's shared contract test suite ([[hex-integration-test]]).

A **driving adapter** is correct when:

- [ ] It only parses/validates format, builds a command, calls the Application Service, maps the result.
- [ ] Domain errors are mapped to protocol responses (table below); none leak raw.
- [ ] It contains zero business rules — deleting it loses no business behavior.
- [ ] It depends on the Application Service (or its optional driving interface), never on driven adapters.

## Error translation

| Technical signal (inside adapter) | Domain error (crosses the port) | Driving-adapter mapping (HTTP) |
| --------------------------------- | ------------------------------- | ------------------------------ |
| `sql.ErrNoRows` / empty result    | `ErrNotFound`                   | 404                            |
| Unique-constraint violation       | `ErrDuplicate`                  | 409                            |
| PSP declines (4xx from vendor)    | `ErrPaymentDeclined`            | 402 / 422                      |
| Timeout, connection refused       | wrapped opaque infra error      | 503 + retry semantics          |
| Malformed request payload         | (never reaches domain)          | 400 at the driving adapter     |

Rule: the use case may branch on **domain** errors; it must never inspect vendor errors.

## Fitness functions per ecosystem

The rule to enforce: **adapters depend inward on ports; the core never depends on adapters.**

| Ecosystem   | Tool                         | What to configure                                                                   |
| ----------- | ---------------------------- | ----------------------------------------------------------------------------------- |
| Go          | `go-arch-lint` or `depguard` | `domain`/`application` components may not depend on `infrastructure` or driver pkgs |
| TypeScript  | `eslint-plugin-boundaries`   | `infrastructure` may import `domain`; `domain` imports nothing outside itself       |
| Python      | `import-linter`              | `forbidden` contract: `myapp.domain` must not import `myapp.infrastructure`         |
| Java/Kotlin | ArchUnit                     | Adapters in `..infrastructure..` may depend on `..domain..`; never the reverse      |
| PHP         | `deptrac`                    | `Infrastructure -> Domain` allowed; `Domain -> Infrastructure` violation            |

Run in CI as a build-failing check.

## Minimal manual check (no tooling yet)

- `grep -rE "infrastructure|adapters" src/domain/ src/application/` (imports only) — must return nothing.
- Go: `go list -deps ./internal/application/... | grep -i infrastructure` — empty.
- Swap drill: point the composition root at the in-memory adapter; the application must boot and unit tests must pass unchanged.

## Thinness criteria for driving adapters

Allowed in a handler/CLI command:

1. Deserialize + validate **shape** (not business rules).
2. Authn/authz delegation to middleware or an application policy.
3. Build the command/query DTO; call the service.
4. Map result/domain errors to the protocol.

Anything else (calculations, conditionals on domain state, orchestration of multiple ports) belongs in the Application Service.

## One adapter = one port

- An adapter implementing several unrelated ports couples their lifecycles and makes the swap drill impossible piecemeal.
- Exception: one technology _module_ (e.g. a `postgres` package) may host several single-port adapters that share a connection pool — sharing infra, not interfaces.
