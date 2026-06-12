# refactor-safely — Reference

## Tool-driven transformations per ecosystem

| Ecosystem   | Rename / move                               | Mechanical rewrite                       | Complexity check               | Dead code                                  |
| ----------- | ------------------------------------------- | ---------------------------------------- | ------------------------------ | ------------------------------------------ |
| Go          | `gopls rename` (LSP rename in any editor)   | `gofmt -r 'old -> new'`, `gofix`         | `gocyclo -over 15 .`           | `deadcode ./...`, `golangci-lint` (unused) |
| TypeScript  | LSP rename (VS Code F2), `ts-morph` scripts | `eslint --fix`, codemods (`jscodeshift`) | eslint `complexity`, `sonarjs` | `knip`, `ts-prune`                         |
| Python      | `rope`, LSP rename (pyright/pylsp)          | `ruff check --fix`, `libcst` codemods    | `radon cc`                     | `vulture`                                  |
| Java/Kotlin | IDE refactorings (IntelliJ)                 | OpenRewrite recipes                      | PMD/detekt                     | IDE inspections                            |

Rule: if the language server can do the transformation atomically (rename, move, change signature), **never** do it with find-and-replace — string matching renames comments, SQL, and unrelated symbols.

## Git safety mechanics

```bash
git stash                      # park unrelated WIP before starting
git commit -m "refactor: ..."  # after EVERY green step (small commits = cheap undo)
git checkout -- . && git clean -fd   # red tests? back to last green commit
git revert <sha>               # undo one landed step without losing the rest
git rebase main                # keep the branch short-lived; merge within days
```

- One refactoring branch per goal; if it lives longer than a few days, it's a big bang in disguise — land what's green and restart.
- Commit messages: `refactor:` prefix (Conventional Commits) so reviewers know "no behavior change" is the claim to verify.

## Characterization tests — recipe for untested code

Purpose: pin **current** behavior (including bugs) so structure can change safely. Fixing the bugs comes after, as separate `fix:` commits with proper tests.

1. Identify the seam: the narrowest public entry point of the code to refactor.
2. Write a test calling it with representative inputs; assert **whatever it currently returns** (run it, copy the actual output into the assertion).
3. Cover the branches you're about to touch — use coverage (`go test -cover`, `vitest --coverage`) to confirm the target lines run.
4. Golden/snapshot files are acceptable here (`testdata/`, `toMatchSnapshot()`) — these tests are scaffolding, deletable after the refactor when real behavior tests exist ([[test-strategy]]).
5. Only then start the safety loop.

## Parallel change (expand–migrate–contract) — for high-risk moves

For signature changes, module restructures, or data-flow changes with many callers:

1. **Expand**: introduce the new function/module/field alongside the old one. Old path untouched; everything green.
2. **Migrate**: move callers one by one (or one package at a time), each migration a green commit. The compiler is the worklist — change the new path, follow the errors.
3. **Contract**: when no callers remain (`grep`/`knip`/`deadcode` confirms), delete the old path in its own commit.

Never skip to contract while both paths have callers — that's where behavior diverges silently.

## Step-size calibration

| Symptom while refactoring                      | Diagnosis                   | Action                                          |
| ---------------------------------------------- | --------------------------- | ----------------------------------------------- |
| Tests red and the cause isn't obvious in 5 min | Step too big                | Revert to green, halve the step                 |
| Compile errors across > ~10 files              | Missing expand phase        | Switch to parallel change                       |
| Diff includes changed test assertions          | Behavior change smuggled in | Split into `refactor:` + `fix:`/`feat:` commits |
| Branch older than a few days                   | Big bang forming            | Land green steps now, re-scope the rest         |

## Pre-merge checklist for a refactoring PR

- [ ] Suite green before and after; no test assertions changed (or changes are purely mechanical renames).
- [ ] Diff is `refactor:`-only — no behavior, config, or dependency changes mixed in.
- [ ] All renames/moves were tool-driven (reviewer can trust them as atomic).
- [ ] Complexity/dead-code metrics improved or held (`gocyclo`, eslint `complexity`, `knip`).
- [ ] Public API unchanged, or the change is flagged and versioned deliberately ([[api-design]] if it's an HTTP/RPC surface).
- [ ] PR description states the one-sentence goal so [[code-review]] can verify scope.
