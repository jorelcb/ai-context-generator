# Spec Apply (OpenSpec)

Implement an approved OpenSpec change. Execute the change's `tasks.md` in order, write tests, commit atomically, verify the implementation matches the spec deltas, and open a reviewable pull request. This is phase 2 of 3 in the OpenSpec lifecycle: `/spec-propose` -> `/spec-apply` -> `/spec-archive`. It requires an approved change workspace produced by [[spec-propose]].

## When to use

- A change under `openspec/changes/<change-id>/` has approved intent and is ready to build
- You are executing tasks from a `tasks.md` and want them done atomically and in order
- You need to confirm the code actually satisfies the spec deltas before review
- Continuing the OpenSpec lifecycle from [[spec-propose]] toward [[spec-archive]]

Note: this phase touches the deltas under `openspec/changes/<change-id>/specs/`, never the source-of-truth `openspec/specs/` — consolidation belongs to [[spec-archive]].

## Process

### 1. Verify the proposal is approved

Do not start without an approved change. Confirm `openspec/changes/<change-id>/proposal.md` exists, `tasks.md` is present with tasks not all checked, and the deltas describe the intended changes. If anything is missing, return to [[spec-propose]] before continuing.

### 2. Switch to the feature branch

Ensure work happens on the right branch: check the current branch matches the change's `feature/<change-id>`, pull the latest base branch and rebase if needed, and confirm the working tree is clean before starting.

### 3. Load the change context

Read everything needed to implement correctly: `proposal.md` for motivation and scope, `design.md` for technical decisions, every delta under `openspec/changes/<change-id>/specs/`, and `tasks.md` to find the first uncompleted task.

### 4. Implement tasks sequentially

Work through `tasks.md` one unchecked task at a time, in order:

- Read the task description carefully
- Make the minimal code change that satisfies it
- Write or update tests covering the new behavior — for each delta scenario, the GIVEN/WHEN/THEN should map to a concrete test
- Run the tests for the affected module
- Mark the task complete by changing `- [ ]` to `- [x]` in `tasks.md`
- Commit atomically as `feat: <task-description>` or `fix: <task-description>`, referencing the change-id in the commit body

Implementing out of order is allowed only with explicit justification (e.g. an unblocking dependency) — note it in the commit.

### 5. Run the full test suite

Once every task is checked, run the unit tests across the whole codebase, the integration tests, and the linters and static analysis. All must pass before proceeding.

### 6. Verify spec compliance

Check the implementation against the deltas, scenario by scenario:

- **ADDED** — at least one test covers each scenario of each new requirement
- **MODIFIED** — the old behavior is replaced cleanly and the revised scenarios pass
- **REMOVED** — the code path is gone with no dead references left behind

If reality diverges from the deltas, update the deltas to reflect what was actually built (not the source-of-truth specs), or return to [[spec-propose]] if the intent itself changed.

### 7. Self-review the diff

Read the complete diff as if it were someone else's PR: no leftover debug code, TODOs, or commented-out blocks; no hardcoded credentials or secrets; commit messages follow Conventional Commits ([[conventional-commit]]); tests are meaningful rather than coverage padding; and the code matches the `design.md` decisions.

### 8. Reconcile tasks.md with reality

Keep `tasks.md` as the honest record of what was built. Add discovered tasks as completed (`- [x] 4.5 Discovered: handle empty input`) and mark anything skipped with a reason (`- [-] 3.2 Skipped: covered by 3.1`).

### 9. Open the pull request

Open the PR as ready (not draft) once tests pass: title `<change-id>: <one-line summary>`, description linking `proposal.md`, summarizing the spec deltas, and listing the verification steps. Request review from the appropriate reviewers.

### 10. Address review feedback

Iterate in NEW commits — do not force-push or amend during active review. Update the deltas if scope shifted, update `tasks.md` if new tasks emerged, and re-request review once every comment is addressed.

## Anti-patterns

- Implementing tasks out of order without justification
- Skipping tests to "save time" — the scenarios are part of the spec
- Force-pushing or amending during active review
- Editing `openspec/specs/` directly — deltas only until [[spec-archive]]
- Bundling unrelated fixes into the same PR
- Marking tasks done without real implementation, or silencing failures with skip annotations

## Verification

Before requesting review:

- [ ] An approved proposal and deltas exist under `openspec/changes/<change-id>/`
- [ ] Every `tasks.md` item is checked or explicitly skipped with a reason
- [ ] Each delta scenario maps to a passing test (ADDED/MODIFIED/REMOVED all reconciled)
- [ ] Full test suite, linting, and static analysis are green
- [ ] Diff is clean and commits follow [[conventional-commit]]
- [ ] PR is scoped to this change and links the proposal

Previous phase: [[spec-propose]]. Next, once the PR is approved: [[spec-archive]].
