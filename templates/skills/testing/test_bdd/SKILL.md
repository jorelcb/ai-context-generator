# Behavior-Driven Development Skill

> **Scope:** the **method** of BDD as defined by its founders (Dan North 2006; Wynne, Rose, Hellesoy; Kent Beck on test quality). For the focused mechanics of writing one good `.feature` file, use **[[bdd-scenario]]**. BDD's ubiquitous language is the same DDD concept — see **[[ddd-entity]]**.

## When to use

Defining acceptance criteria, running discovery on a user story, specifying business rules with concrete examples, bridging business/dev/QA, doing outside-in development, or producing living documentation.

## The core insight (Dan North, 2006)

The word **"test"** makes people think _verification_; BDD reframes it as **specifying behavior** in natural language. This dissolves North's five TDD pains: where to start, what to test, how much, what to name it, why it failed.

## The three practices (Wynne & Rose) — in order of value

1. **Discovery — the most important.** Structured conversation _before_ code.
   - **Three Amigos**: Business (what problem/value?), Dev (how/edge cases?), QA (what breaks/boundaries?).
   - **Example Mapping** (Matt Wynne, ~25 min): pick a story → rules on cards → a concrete example per rule → questions for the unknowns. Too many questions ⇒ story not understood; too many rules ⇒ split the story. Output: examples all three amigos agree mean "done".
2. **Formulation.** Translate the agreed examples into Gherkin using ubiquitous language. Writing precise Gherkin surfaces ambiguities the conversation missed. _(Mechanics → [[bdd-scenario]].)_
3. **Automation — the least important.** Wire scenarios to step definitions so the spec becomes executable living documentation.

> **The #1 reason BDD fails:** teams skip Discovery + Formulation and jump to Automation, treating Cucumber as a mere test runner. **The value is in the conversations, not the tooling.**

## Given / When / Then (Matts & North)

`Given` precondition (state, no actions/assertions) · `When` the single trigger (one per scenario) · `Then` observable outcome. This is Arrange-Act-Assert raised to business language.

## Outside-in development

1. Write a failing acceptance scenario first.
2. Drop to unit level only as needed to implement a step.
3. Let the outer scenario pull the design inward, each layer driving the next.
   Start from the behavior the stakeholder cares about — not from the database.

## Ubiquitous language & living documentation

Scenarios must use the **business's words** (if they say "policy holder", the code's "customer" is a bug waiting to happen — see [[ddd-entity]]). Because scenarios _are_ the tests, the documentation can never silently go stale: divergence makes them fail.

## Anti-patterns (process)

- Skipping Discovery — Gherkin written with no collaborative conversation.
- BDD-as-testing-tool — Cucumber for automation without the collaboration.
- Scenarios written _after_ implementation (documentation, not specification).
- No Three Amigos — one person writes everything, losing cross-functional perspective.

_(Writing-level anti-patterns live with the mechanics in [[bdd-scenario]].)_

## Verification

- [ ] The team did Discovery (Three Amigos / Example Mapping) before formulating.
- [ ] Scenarios are readable by non-developers and use ubiquitous language.
- [ ] Each scenario specifies one business rule with one trigger.
- [ ] Scenarios drive development (written first), not document it after.
- [ ] Feature files act as living documentation (they fail when behavior diverges).

## Reference

Kent Beck's Test Desiderata (the 12 properties + the trade-off thesis), declarative-vs-imperative, and Gherkin keyword semantics: **[reference.md](reference.md)**.

## Examples

An Example Mapping output, a full feature file, a Scenario Outline, and step definitions in Go (godog) and TypeScript (cucumber-js): **[examples.md](examples.md)**.
