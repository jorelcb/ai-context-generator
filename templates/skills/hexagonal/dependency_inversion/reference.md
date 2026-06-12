# dependency-inversion — Reference

## Fitness functions per ecosystem

The rule to enforce in CI: **no source dependency from Domain/Application toward Infrastructure, drivers, frameworks, or vendor SDKs.** A violation fails the build.

| Ecosystem   | Tool                         | What to configure                                                            |
| ----------- | ---------------------------- | ---------------------------------------------------------------------------- |
| Go          | `go-arch-lint` or `depguard` | Component rules: `domain` depends on nothing; `application` only on `domain` |
| TypeScript  | `eslint-plugin-boundaries`   | Element types `domain`/`application`/`infrastructure` with inward-only rules |
| Python      | `import-linter`              | `layers` contract (domain < application < infrastructure)                    |
| Java/Kotlin | ArchUnit                     | Layered-architecture rule or package-dependency assertions in a unit test    |
| PHP         | `deptrac`                    | Layers by namespace; ruleset allows only inward edges                        |

### Config sketches

```yaml
# Go — .go-arch-lint.yml
components:
  domain: { in: internal/domain/** }
  application: { in: internal/application/** }
  infrastructure: { in: internal/infrastructure/** }
deps:
  domain: { mayDependOn: [] }
  application: { mayDependOn: [domain] }
  infrastructure: { mayDependOn: [domain, application] }
```

```jsonc
// TypeScript — eslint-plugin-boundaries (excerpt)
"boundaries/element-types": ["error", {
  "default": "disallow",
  "rules": [
    { "from": "domain", "allow": ["domain"] },
    { "from": "application", "allow": ["domain", "application"] },
    { "from": "infrastructure", "allow": ["domain", "application", "infrastructure"] }
  ]
}]
```

```ini
# Python — .importlinter
[importlinter:contract:layers]
name = Hexagonal layers
type = layers
layers =
    myapp.infrastructure
    myapp.application
    myapp.domain
```

```java
// Java — ArchUnit (runs as a unit test)
noClasses().that().resideInAPackage("..domain..")
    .should().dependOnClassesThat().resideInAPackage("..infrastructure..");
```

```yaml
# PHP — deptrac.yaml (ruleset)
ruleset:
  Domain: []
  Application: [Domain]
  Infrastructure: [Domain, Application]
```

## Minimal manual check (no tooling yet)

- Go: `go list -deps ./internal/domain/... ./internal/application/... | grep -E 'database/sql|net/http|github.com/'` — only stdlib non-IO and your own domain packages should appear.
- TS: `grep -rE "from ['\"].*(infrastructure|adapters|node_modules-vendor)" src/domain/ src/application/` — empty.
- Python: `grep -rE "^(import|from) (sqlalchemy|requests|boto3|django)" src/myapp/domain/` — empty.
- Test drill: run the use-case unit tests with the network disabled and no containers running. They must pass.

## Violation catalog → fix

| Symptom                                                          | Violation                            | Fix                                                               |
| ---------------------------------------------------------------- | ------------------------------------ | ----------------------------------------------------------------- |
| Use case constructs `sql.DB` / `new PrismaClient()`              | Core depends on detail               | Extract driven port; inject via constructor ([[port-definition]]) |
| Use case calls `Container.resolve("repo")`                       | Service locator                      | Constructor injection from the composition root                   |
| `domain` imports `infrastructure` for an interface defined there | Inverted ownership                   | Move the interface into the Domain; adapter implements it         |
| Port method `Query(sql string)`                                  | Vendor API mirrored as "abstraction" | Reshape the port around the domain capability                     |
| Unit test needs Docker for a use case                            | Hidden infra dependency              | Find the concrete import; apply the refactor workflow             |
| `IRepository<T>` used by every aggregate                         | Abstraction about nothing            | One capability-shaped port per aggregate ([[port-definition]])    |

## Composition root rules

- Exactly **one** per executable (per `main`, per lambda entry, per worker).
- It may import everything — it is the only module allowed to know all layers.
- Keep it boring: construct, inject, start. Logic in the root escapes all tests.
- Swapping an adapter (real ↔ fake, Postgres ↔ Mongo) must be a change **only** here; verify with the contract suite from [[hex-integration-test]].
