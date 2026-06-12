# Code Review Skill

> Reviews exist to catch defects and share knowledge **before merge** — not to enforce taste. Machines check style; humans (and agents) check correctness, design and risk. Language-agnostic; Go + TypeScript used as reference stacks.

## When to use

Reviewing a pull request or diff, auditing a module's quality, pre-merge validation, or being asked "is this change safe to ship?".

## Review process (in this order)

1. **Understand the intent** — read the PR description, linked issue, and commit messages. If you cannot state in one sentence what the change is for, ask before reviewing.
2. **Run the machines first** — formatter, linter, type checker, tests, security scanner (see tooling in [reference.md](reference.md)). Never spend human comments on what a tool flags.
3. **Check correctness** — does the code do what the description claims? Trace the main path by hand.
4. **Check edge cases** — empty/nil/null inputs, zero and negative values, boundaries (off-by-one), error paths, concurrency (shared state, races), partial failure.
5. **Check security** — injection (SQL/command/template), authn/authz on every new endpoint, secrets in code or logs, unsafe deserialization, unvalidated external input.
6. **Check tests** — new behavior has tests; tests assert behavior, not implementation; failure cases are covered. Gaps here are findings, not nitpicks (see [[test-strategy]]).
7. **Check design & readability** — naming, duplication, coupling, function size, dead code. Suggest, don't mandate, unless it blocks maintainability.
8. **Deliver the verdict** — approve, approve-with-comments, or request changes. Every "request changes" must list the concrete blockers.

## Severity taxonomy (normative)

| Label          | Meaning                                                  | Merge action                |
| -------------- | -------------------------------------------------------- | --------------------------- |
| **blocker**    | Bug, security flaw, data loss, broken contract           | Must fix before merge       |
| **major**      | Wrong design, missing tests for new behavior, perf cliff | Fix or justify before merge |
| **suggestion** | Better approach exists; current one works                | Author's call               |
| **nitpick**    | Style/preference not covered by a linter                 | Author's call; never block  |

Prefix every comment with its label. A review with ten unlabeled comments is noise.

## What to look for, by category

- **Bugs**: logic errors, off-by-one, nil/undefined handling, ignored errors (`err` discarded in Go, unhandled promise rejections in TS), race conditions on shared state.
- **Security**: string-built SQL/commands, missing authz checks, secrets/tokens in diffs, overly broad CORS, sensitive data in logs.
- **Performance**: N+1 queries, unbounded loops/queries (no LIMIT, no pagination), allocations in hot paths, missing indexes for new query patterns, sync I/O in request paths.
- **Maintainability**: misleading names, copy-paste duplication, god functions, new public surface without need, leaking internals across module boundaries.
- **Contracts**: breaking changes to APIs/schemas without versioning — review against [[api-design]] when the diff touches endpoints.

## Feedback rules (normative)

- Point to the **exact file and line**; quote the code in question.
- Every blocker/major comes with a **concrete fix or alternative** — "this is wrong" alone is not a finding.
- Explain the **why** (what breaks, when) — not just the what.
- Praise non-obvious good decisions; it calibrates the author on what to keep doing.
- Large refactor ideas go to an issue, not 30 inline comments — see [[refactor-safely]] for how to land them separately.

## Anti-patterns

- Bikeshedding style while a logic bug sits unreviewed.
- Rewriting the PR in comments instead of stating the problem and letting the author solve it.
- Approving without reading ("LGTM" on a 600-line diff in 2 minutes).
- Blocking on subjective preference dressed up as a blocker.
- Reviewing the diff only — ignoring how the change interacts with surrounding code it doesn't touch.

## Verification (executable)

A review is grounded, not vibes: run `git diff --stat` to scope it, run the project's lint/type/test commands on the branch, and confirm CI is green before approving. Per-ecosystem commands (Go: `golangci-lint`, `go vet`, `gosec`; TS: `eslint`, `tsc --noEmit`; etc.) and the full pre-approval checklist are in **[reference.md](reference.md)**. Minimum: every blocker has a concrete fix, and you actually executed the tests.

## Going deeper

- **[reference.md](reference.md)** — per-ecosystem tooling table, pre-approval checklist, security review checklist, comment-writing criteria.
- **[examples.md](examples.md)** — canonical review findings (bad vs good comments) on Go and TypeScript diffs.
