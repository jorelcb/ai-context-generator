# test-bdd — Reference

## Test Desiderata (Kent Beck) — 12 properties

Good developer tests balance these. They are **trade-offs**, not a checklist to maximize all at once.

1. **Isolated** — no test depends on another; any order, subset, parallelism.
2. **Composable** — any combination of tests yields valid results.
3. **Fast** — slow tests run less, losing value.
4. **Inspiring** — green gives confidence to deploy; lingering unease ⇒ the suite fails this.
5. **Writable** — low friction to create; heavy setup discourages testing.
6. **Readable** — tests are documentation; behavior understood without tracing internals.
7. **Behavioral** — sensitive to behavior changes.
8. **Structure-insensitive** — refactoring internals doesn't break them.
9. **Automated** — no human step to run/verify.
10. **Specific** — a failure points precisely to the problem.
11. **Deterministic** — same code, same result; no flakiness.
12. **Predictive** — green means it works in production.

### The trade-off thesis

The properties exist in tension: Fast ↔ Predictive, Isolated ↔ Predictive, Writable ↔ Readable, Specific ↔ Structure-insensitive. The right balance is contextual. **BDD scenarios deliberately optimize for Readable, Behavioral and Inspiring** (and accept being slower than unit tests).

## Declarative vs imperative (the decisive style rule)

**Imperative (anti-pattern)** — couples the scenario to the UI:

```gherkin
Given I am on the login page
And I enter "user@test.com" in the email field
And I click the "Login" button
Then I should see "Welcome"
```

**Declarative (correct)** — survives a UI redesign:

```gherkin
Given I am a registered customer
When I sign in with valid credentials
Then I should be welcomed back
```

If moving a button breaks your scenario, you're testing UI, not behavior.

## Gherkin keyword semantics

| Keyword                              | Meaning                                                |
| ------------------------------------ | ------------------------------------------------------ |
| **Feature**                          | Groups related scenarios; one feature per file.        |
| **Rule** (Gherkin 6+)                | Groups scenarios under one business rule.              |
| **Scenario** / **Example**           | One concrete, independent, self-contained example.     |
| **Background**                       | Steps run before each scenario (shared preconditions). |
| **Scenario Outline + Examples**      | Data-driven scenario with `<placeholders>`.            |
| **And / But**                        | Continue the previous Given/When/Then for readability. |
| **Tags** (`@wip`, `@smoke`, `@slow`) | Metadata for filtering and hooks.                      |

## Sources

Dan North — _Introducing BDD_ (2006). Matt Wynne & Aslak Hellesoy — _The Cucumber Book_; Example Mapping. Seb Rose & Gáspár Nagy — _The BDD Books (Discovery / Formulation)_. Chris Matts & Dan North — Given/When/Then. Kent Beck — Test Desiderata. Eric Evans — Ubiquitous Language ([[ddd-entity]]).
