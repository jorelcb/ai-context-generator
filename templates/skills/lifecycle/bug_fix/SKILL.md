# Bug Fix Lifecycle

Drive a bug from report to verified fix through a disciplined sequence: reproduce, capture a failing test, diagnose the root cause, fix narrowly, and confirm no regressions. This prevents symptom-only patches and bugs that quietly return.

## When to use

- A bug report or issue needs a structured, end-to-end fix
- A defect was reproduced and you are about to change code
- You want a regression guard in place before shipping the fix
- Reviewing whether a "quick fix" actually addresses the root cause

As a Claude Code skill, this may declare `allowed-tools` so trusted, repeatable steps (running the test suite, linting) execute without re-prompting. Keep destructive or remote steps (merge, deploy) interactive.

## Process

### 1. Create the fix branch

Branch from the latest `main`/`develop`. Follow the project's naming convention (e.g. `fix/<issue-number>-short-description`) and reference the bug report so the branch is traceable back to the issue.

### 2. Reproduce the bug

Confirm the defect before touching code:

- Read the report — steps to reproduce, expected vs. actual behavior
- Recreate the environment: the specific data, configuration, and state
- Execute the reproduction steps
- Capture the broken behavior: error messages, logs, screenshots

If you cannot reproduce it, stop and gather more information — do not guess at a fix.

### 3. Write a failing test

When the bug is testable, encode it as a test first:

- The test must FAIL against the current code, proving the bug exists
- It should assert the expected correct behavior
- This test becomes the permanent regression guard

### 4. Diagnose the root cause

Investigate systematically rather than patching the first suspicious line:

- Trace the execution path from the reproduction steps
- Find WHERE behavior diverges from expectation
- Find WHY — the underlying cause, not the surface symptom
- Check whether the same flawed pattern exists elsewhere
- Record the findings for the commit message or PR description

### 5. Implement the fix

Fix the cause with the smallest change that works:

- Make the minimum change needed to correct the behavior
- Confirm the failing test from step 3 now passes
- Do not refactor unrelated code in the same commit
- If a broader refactor is warranted, file it as a follow-up

### 6. Verify no regressions

Run the full test suite, not just the new test:

- All existing tests still pass
- The new regression test passes
- Linting and static analysis are clean

### 7. Cover the edge cases

When the bug touches boundary conditions, probe the neighbors:

- Empty or null inputs
- Maximum and minimum values
- Concurrent-access scenarios
- Add tests for any uncovered edge case you surface

### 8. Open a focused pull request

- Title: `fix: <what was broken>`
- Description: root-cause analysis, what the fix does, how to verify
- Link the original bug report and include before/after behavior

### 9. Merge and confirm

After approval, merge with the project's preferred strategy, verify the fix in the target environment, close the issue with a reference to the fix, and delete the branch.

## Anti-patterns

- Patching symptoms without understanding the root cause
- Bundling the fix with unrelated changes in one PR
- Shipping without a regression test — the bug will return
- Fixing one bug and introducing another by skipping the full suite
- Leaving the root cause undocumented, making future debugging harder

## Verification

Before merging:

- [ ] Bug was reproduced and the broken behavior captured
- [ ] A regression test failed before the fix and passes after
- [ ] Root cause (not just the symptom) is identified and documented
- [ ] Full test suite, linting, and static analysis are green
- [ ] Relevant edge cases have test coverage
- [ ] PR is scoped to the fix only and linked to the issue
