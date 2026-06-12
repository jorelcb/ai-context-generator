# clean-arch-layer — Reference

## Reference directory tree

Same logical structure in both languages; adapt names to your project conventions.

```
Go                                  TypeScript
internal/                           src/
  domain/                             domain/
    order/                              order/
      order.go        (entity)            order.ts          (entity)
      money.go        (value object)      money.ts          (value object)
      repository.go   (DRIVEN PORT)       order-repository.ts (DRIVEN PORT)
      events.go       (domain events)     events.ts         (domain events)
  application/                          application/
    place_order.go    (app service)       place-order.ts    (app service / command handler)
    get_order.go      (query handler)     get-order.ts      (query handler)
    dto.go            (DTOs)              dto.ts            (DTOs)
  infrastructure/                       infrastructure/
    postgres/                             postgres/
      order_repo.go   (adapter)            order-repository.ts (adapter)
  interfaces/                           interfaces/
    http/                                 http/
      order_handler.go (primary adapter)   order-controller.ts (primary adapter)
  cmd/main.go         (composition root)  main.ts           (composition root)
```

Rule of thumb for "where does this go?": ask _what does it depend on?_ If it needs the DB/HTTP/framework → outer ring. If it's a pure business rule → Domain. If it coordinates domain objects and ports → Application.

## Enforcing the dependency rule (fitness functions)

| Ecosystem   | Tool                                             | Enforces                          |
| ----------- | ------------------------------------------------ | --------------------------------- |
| Go          | `go-arch-lint`, `depguard` (golangci-lint)       | Allowed import graph per layer    |
| TypeScript  | `eslint-plugin-boundaries`, `dependency-cruiser` | Layer boundaries, forbidden edges |
| Java/Kotlin | ArchUnit                                         | Layer access rules as unit tests  |
| Python      | `import-linter`                                  | Contract-based import layering    |
| PHP         | `deptrac`                                        | Layer dependency rules            |

Wire the chosen tool into CI so a violation fails the build. A passing architecture test is the real "verification" — the manual grep is only a fallback.

### Minimal manual checks

- Domain package has zero imports of infrastructure/framework/transport packages.
- No `Interfaces → Infrastructure` import (outer ring must not couple to itself).
- Every driven port interface has at least one adapter implementation registered at the composition root.

## Layer placement decision table

| You are writing…                                        | Layer                             |
| ------------------------------------------------------- | --------------------------------- |
| A business invariant / rule                             | Domain (entity or domain service) |
| An interface for "we need to persist/fetch/notify X"    | Domain (driven port)              |
| Code that coordinates entities + ports for one use case | Application (application service) |
| SQL, an HTTP client to a 3rd party, a queue publisher   | Infrastructure (adapter)          |
| Parsing an HTTP request / rendering a response          | Interfaces (primary adapter)      |
| `container.Provide(...)` / `new Service(repo)` wiring   | Composition root                  |

## Notes on terminology

- "Interfaces Layer" ≡ hexagonal **primary / driving adapters**.
- "Infrastructure Layer" ≡ hexagonal **secondary / driven adapters**.
- "Application Service" ≡ CQRS **command/query handler** ≡ a thin orchestrator (no business logic).
