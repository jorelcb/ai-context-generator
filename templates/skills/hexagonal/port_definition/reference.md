# port-definition — Reference

## Port checklist

A **driven port** is correct when:

- [ ] It is an interface declared in the Domain (never in Infrastructure).
- [ ] Its name states domain intent, with no vendor or technology hint.
- [ ] Every parameter and return type is a domain type (entity, value object, ID, domain error).
- [ ] It represents one cohesive capability (one repository per aggregate; one gateway per external capability).
- [ ] Concurrency and lifetime expectations are documented on the interface.
- [ ] At least one contract test suite exercises it ([[hex-integration-test]]).

The **driving side** is correct when:

- [ ] The Application Service is the concrete entry point primary adapters call.
- [ ] A driving interface exists **only** if >1 primary adapter shares it, or the adapter is tested against a contract of the service.
- [ ] No business logic lives in the primary adapter itself ([[adapter-pattern]]).

## Naming table

| Bad (technology)         | Good (domain intent)  | Why                                   |
| ------------------------ | --------------------- | ------------------------------------- |
| `PostgresOrderDAO`       | `OrderRepository`     | Storage choice is the adapter's job   |
| `StripeClient`           | `PaymentGateway`      | Vendor is swappable, capability isn't |
| `SmtpMailer`             | `OrderPlacedNotifier` | Names the _event_, not the protocol   |
| `KafkaProducer`          | `OrderEventPublisher` | Transport is infrastructure           |
| `OrderManager` (driving) | `PlaceOrderUseCase`   | Use-case-shaped, not a grab-bag       |

## Fitness functions per ecosystem

The rule to enforce: **the Domain (and Application) packages import zero infrastructure/vendor packages.** Ports compile alone.

| Ecosystem   | Tool                         | What to configure                                                                                                         |
| ----------- | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Go          | `go-arch-lint` or `depguard` | Component `domain` may not depend on `infrastructure`, `database/sql`, drivers, SDKs                                      |
| TypeScript  | `eslint-plugin-boundaries`   | Element type `domain` has `allow: [domain]` only; `infrastructure` may import `domain`                                    |
| Python      | `import-linter`              | `layers` contract: `domain` below `application` below `infrastructure`                                                    |
| Java/Kotlin | ArchUnit                     | `noClasses().that().resideInAPackage("..domain..").should().dependOnClassesThat().resideInAPackage("..infrastructure..")` |
| PHP         | `deptrac`                    | Layer `Domain` has empty allowlist; `Infrastructure` may depend on `Domain`                                               |

Run the fitness function in CI; a violation is a build failure, not a review comment.

## Minimal manual check (no tooling yet)

- Go: `go list -deps ./internal/domain/... | grep -E 'database/sql|net/http|github.com/(lib|jackc|aws|stripe)'` — must return nothing.
- TS/Python/PHP: `grep -rE "from ['\"]\.\./(infrastructure|adapters)" src/domain/` — must return nothing.
- Compile/typecheck the domain package in isolation: it must build with every adapter deleted.

## Granularity criteria — when to split a port

Split when any of these holds:

1. Two methods change for unrelated reasons (persistence vs notification).
2. An adapter implements half the interface and stubs the rest (`NotImplemented` is a smell).
3. A consumer depends on the port but uses one method out of many (interface segregation).

Keep together when methods share a transactional or consistency boundary (e.g. `Save` + `FindByID` of one aggregate).

## Error design across the boundary

- Define domain errors next to the port: `ErrOrderNotFound`, `ErrDuplicateOrder`.
- The port's documentation states **which** errors each method may return; the contract test asserts them.
- "Unknown technical failure" is allowed as a wrapped/opaque error — but never as a vendor type the core must inspect.
