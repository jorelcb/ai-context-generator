# hexagonal-port — Reference

## Port vs Adapter — checklist

A **port** is correct when:

- [ ] It is an interface in the Domain (or, rarely, a driving interface in Application).
- [ ] Its name reflects domain intent, not a vendor/technology.
- [ ] Its signatures use only domain types (entities, VOs, IDs, domain errors).
- [ ] It represents one cohesive capability (not a grab-bag).

An **adapter** is correct when:

- [ ] It lives in Infrastructure (driven) or Interfaces (driving).
- [ ] It implements/consumes a port; nothing in the Domain imports it.
- [ ] All technology concerns (SQL, HTTP, retries, serialization) stay inside it.
- [ ] It is registered at the composition root and is swappable.

## Contract tests (the key to swappability)

Write **one** test suite per port, parameterized by the adapter under test. Every adapter (Postgres, in-memory, mock-server) must pass the same suite. This is what guarantees adapters are truly interchangeable — without it, "swappable" is wishful thinking.

## Swappability verification (fitness functions)

| Ecosystem   | Tool                                             | Use                                                  |
| ----------- | ------------------------------------------------ | ---------------------------------------------------- |
| Go          | `depguard` / `go-arch-lint`                      | Forbid Domain importing infra packages               |
| TypeScript  | `dependency-cruiser`, `eslint-plugin-boundaries` | Forbid domain→infra edges                            |
| Java/Kotlin | ArchUnit                                         | "domain classes should not depend on infrastructure" |
| Python      | `import-linter`                                  | Layer contract: domain independent                   |
| PHP         | `deptrac`                                        | Domain layer has no outbound infra deps              |

## Driving side — when to add an interface

Default: the Application Service **is** the driving entry point, called directly by the primary adapter. Add a driving interface only when:

- More than one primary adapter must depend on the same abstraction, **or**
- You want to test the primary adapter against a contract/double of the service.

Otherwise an interface per service is pure ceremony. (Aligned with [[clean-arch-layer]].)

## One port per what?

- One **repository** per aggregate root ([[ddd-entity]]).
- One **gateway** per external system capability (`PaymentGateway`, `ShippingGateway`).
- Split if a single interface starts mixing unrelated reasons to change.
