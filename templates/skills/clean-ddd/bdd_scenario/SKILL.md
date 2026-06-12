# BDD Scenario Writing Skill

> **Focus:** the **mechanics** of formulating one `.feature` file well. The collaboration _method_ that should precede this (Discovery, Three Amigos, Example Mapping) lives in **[[test-bdd]]** — do that first. Use the business's ubiquitous language ([[ddd-entity]]).

## When to use

You have agreed examples (from discovery) and now need to write or refactor a feature file, name scenarios, structure a Background, parameterize with a Scenario Outline, or wire step definitions.

## The recipe

1. **Feature** — one business capability per file; add a one-line "why" (the value).
2. **Background** — only steps shared by _every_ scenario; keep it tiny.
3. **Rule** — group scenarios under the business rule they demonstrate.
4. **Scenario** — one behavior, named by behavior not implementation.
5. **Given / When / Then** — precondition (state) / single trigger / observable outcome.
6. **Scenario Outline + Examples** — when the same behavior varies only by data.

## Hard rules

- **One `When` per scenario.** Two triggers ⇒ two scenarios.
- **Declarative, not imperative.** Express business intent, not clicks/fields — a scenario must survive a UI redesign. (Side-by-side example in [[test-bdd]]'s reference.)
- **Independent & self-contained.** No scenario depends on another's state or order.
- **No incidental detail.** Every value in the scenario must matter to the behavior under test.
- **Behavior, not implementation.** Never mention tables, class names or endpoints.
- **≤ ~7 steps.** More is a smell — the scenario or the story is too big.

## Scenario naming

Name the _behavior and its outcome_, readable by a non-developer:

- ✅ `Withdrawal over balance is declined`
- ❌ `Test withdraw() returns false when amount > balance`

## Step definitions (Automation)

- Reusable steps via Cucumber expressions (`{int}`, `{word}`) or regex.
- Share state between steps through a context object/struct — never globals leaking across scenarios.
- Steps are **thin**: translate Gherkin to domain calls and assert outcomes; **business rules belong in the domain** ([[ddd-entity]]), not in steps.
- Use Background/hooks for setup-teardown; table steps for structured data.

## Anti-patterns (writing-level)

- Incidental details unrelated to the behavior.
- Imperative, UI-coupled steps.
- Too many scenarios per feature (feature too large — split it).
- Interdependent scenarios.
- Multiple `When` clauses.
- Steps coupled to a data format (brittle).

## Verification

- [ ] A non-developer can read each scenario and recognize the rule.
- [ ] Exactly one `When` per scenario; ≤ ~7 steps.
- [ ] Declarative — no UI mechanics; survives a redesign.
- [ ] Scenarios are independent and deterministic.
- [ ] Step definitions hold no business logic.

## Examples

Feature file, Background, Scenario Outline, and a thin step-definition skeleton (Go + TypeScript): **[examples.md](examples.md)**.
