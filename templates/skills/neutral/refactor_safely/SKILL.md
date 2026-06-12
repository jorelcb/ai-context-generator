# Safe Refactoring Skill

> Refactoring changes **structure without changing behavior** — and the only acceptable proof is a passing test suite, before and after. No green baseline, no refactoring. Language-agnostic; Go + TypeScript used as reference stacks.

## When to use

Reducing complexity, extracting functions/abstractions, renaming for clarity, removing duplication, moving code to where it belongs, or preparing the ground before a feature ("make the change easy, then make the easy change").

## The safety loop (normative)

1. **Green baseline** — run the full relevant suite. If it's red, fix that first; if coverage is missing on the target, write **characterization tests** that pin current behavior (even ugly behavior) before touching anything.
2. **State the goal** in one sentence ("extract pricing rules out of the handler") — it bounds the scope.
3. **One mechanical transformation at a time** — prefer tool-driven moves (rename via `gopls`/IDE) over hand edits.
4. **Run the tests after every transformation.** Seconds-fast unit loop is the enabler — see [[test-strategy]].
5. **Commit every green step** (`refactor: extract PriceCalculator`). Small commits make `git revert`/`git reset` your undo button.
6. **Red? Revert, don't debug.** `git checkout -- .` (or `git stash`) back to the last green commit and take a smaller step. Debugging mid-refactor means the step was too big.
7. **Stop at the goal** — resist adjacent "while I'm here" edits; queue them instead.

## Risk ladder — choose step size by risk

| Risk       | Transformations                                                      | Required safety                                                        |
| ---------- | -------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| **Low**    | Rename, extract function/variable, inline, reorder private code      | Existing tests + tool-driven edit                                      |
| **Medium** | Extract class/interface, change a signature, move across packages    | Tests on all call sites; compiler-guided ripple; one commit per move   |
| **High**   | Restructure modules, change data flow, touch shared/concurrent state | Characterization tests first; feature-flag or parallel-change strategy |

For high-risk changes use **parallel change** (expand–migrate–contract): add the new path alongside the old, migrate callers incrementally (each step green), then delete the old path.

## Core catalog (know these by name)

- **Extract function/method** — the workhorse; name reveals intent, shrinks complexity.
- **Rename** — cheapest readability win; always tool-driven, never find-and-replace on strings.
- **Inline** — removes indirection that no longer earns its keep.
- **Move function/field** — puts behavior next to the data it uses.
- **Extract interface** — decouples a dependency so it can be substituted (do it when a second implementation or a test double is needed — not "just because").
- **Replace conditional with polymorphism** — when the same `switch`/`if` chain on a type tag appears in multiple places.
- **Introduce parameter object** — when the same group of parameters travels together.

## When NOT to refactor

- **No tests and no time to add them** — write characterization tests first or don't touch it.
- **Mid-incident / urgent fix** — fix minimally, refactor in a follow-up.
- **Together with behavior changes** — never mix `refactor:` and `feat:`/`fix:` in one commit; reviewers can't verify "no behavior change" in a mixed diff ([[code-review]]).
- **Code scheduled for deletion** — polishing a corpse.
- **Speculative generality** — refactoring toward flexibility nobody asked for is gold plating.

## Anti-patterns

- **Big bang**: a 2,000-line "cleanup" branch that drifts for two weeks and merges with conflicts and surprises.
- Refactoring without a green baseline ("the tests were already failing").
- Hand-editing renames across files when the language server does it atomically.
- "While I'm here" scope creep — each extra edit multiplies revert cost.
- Treating a red test as "the test is wrong" and editing the assertion to match the new behavior — that's a behavior change, not a refactor.

## Verification (executable)

Behavior preservation is proven, not claimed: the same test suite passes before and after (`go test ./...` / `vitest run`); the diff contains **zero test-assertion changes** (`git diff -- '*_test.go' '*.test.ts'` is empty or mechanical); complexity actually dropped (`gocyclo`, eslint `complexity` rule). Tool-driven transformation commands per ecosystem and the characterization-test recipe are in **[reference.md](reference.md)**.

## Going deeper

- **[reference.md](reference.md)** — per-ecosystem refactoring tooling, characterization-test recipe, parallel-change walkthrough, git safety mechanics.
- **[examples.md](examples.md)** — extract-function and replace-conditional-with-polymorphism, before/after with untouched tests, in Go and TypeScript.
