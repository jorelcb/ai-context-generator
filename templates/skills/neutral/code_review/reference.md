# code-review — Reference

## Machine-first tooling (run before human review)

| Ecosystem   | Format                                                                       | Lint                          | Types          | Security             | Tests                      |
| ----------- | ---------------------------------------------------------------------------- | ----------------------------- | -------------- | -------------------- | -------------------------- |
| Go          | `gofmt -l .`                                                                 | `golangci-lint run`, `go vet` | compiler       | `gosec ./...`        | `go test -race ./...`      |
| TypeScript  | `prettier -c .`                                                              | `eslint .`                    | `tsc --noEmit` | `npm audit`, semgrep | `vitest run` / `jest`      |
| Python      | `ruff format --check`                                                        | `ruff check`                  | `mypy .`       | `bandit -r .`        | `pytest`                   |
| Java/Kotlin | spotless                                                                     | checkstyle / detekt           | compiler       | OWASP dep-check      | `mvn test` / `gradle test` |
| Any diff    | `git diff --stat`, `git log --oneline main..HEAD` — scope and commit hygiene |                               |                |                      |                            |

Rule: a comment a linter could have made is a configuration gap, not a review comment. Fix the config (separate PR) instead.

## Pre-approval checklist

- [ ] I can state the change's intent in one sentence.
- [ ] CI is green; I ran the tests locally for non-trivial diffs.
- [ ] Every new behavior has at least one test; every bug fix has a regression test.
- [ ] Error paths reviewed: no swallowed errors, no `catch {}` / `_ = err`.
- [ ] No new endpoint/handler without an authz check.
- [ ] No secrets, tokens, or credentials anywhere in the diff (including test fixtures).
- [ ] List endpoints paginate; queries on new access patterns have index support.
- [ ] Public API/schema changes are backward compatible or versioned ([[api-design]]).
- [ ] Every blocker/major comment includes a concrete fix.
- [ ] Verdict given explicitly (approve / approve-with-comments / request changes).

## Security review checklist (every diff that touches I/O)

- [ ] External input validated at the boundary (type, length, range, allowlist).
- [ ] SQL/commands built with parameters/placeholders — never string concatenation.
- [ ] Output encoding for HTML/templates (XSS); no `dangerouslySetInnerHTML` / `template.HTML` with user data.
- [ ] AuthN verified AND authZ verified (right user _and_ right permission) on new routes.
- [ ] Sensitive fields excluded from logs and error messages.
- [ ] File paths from input canonicalized (no `../` traversal); uploads constrained by type and size.
- [ ] Crypto: standard library / vetted libs only; no homemade hashing for passwords (use bcrypt/argon2).

## Writing the comment — criteria

A good review comment has four parts:

1. **Severity label** — `blocker:` / `major:` / `suggestion:` / `nit:`.
2. **Observation** — what the code does, quoting the line.
3. **Consequence** — what breaks, when, and for whom.
4. **Concrete alternative** — code sketch or precise instruction.

Tone rules: comment on the code, never the author ("this function ignores the error" not "you ignored the error"); ask when unsure ("is the empty-slice case intentional?") instead of asserting; one issue per comment thread.

## Sizing and scope

| Diff size      | Strategy                                                          |
| -------------- | ----------------------------------------------------------------- |
| < 200 lines    | Full line-by-line review                                          |
| 200–800 lines  | Review by commit; ask for a commit-level narrative if missing     |
| > 800 lines    | Request a split unless it is mechanical (rename, generated code)  |
| Generated code | Verify the generator input + spot-check; don't line-review output |

Review latency matters: a same-day shallow-but-honest review beats a perfect review three days later. Say explicitly which parts you reviewed deeply.

## When the review finds pre-existing problems

Issues in code the diff merely touches (not introduces) are **out of scope for blocking**: file an issue or a follow-up task, mention it as a non-blocking note. Exception: the diff makes a pre-existing vulnerability reachable — that's a blocker.
