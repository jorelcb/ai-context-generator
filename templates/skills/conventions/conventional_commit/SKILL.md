# Conventional Commits

Write commit messages that strictly follow the Conventional Commits 1.0.0 specification: analyze the staged changes and produce a structured, meaningful message.

## When to use

- Committing code changes or preparing a pull request
- Generating or maintaining a changelog
- Any git commit operation where the message must follow Conventional Commits
- Determining the semantic impact of a change — commit types feed version bumps, see [[semantic-versioning]]

## Commit format

The strict format from the specification:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Rules:

- `type` is required: a noun describing the category of change
- `scope` is optional: a noun in parentheses describing the section of the codebase
- `!` before the colon signals a BREAKING CHANGE
- `description` is required: short summary in imperative mood, lowercase, no period
- `body` is optional: free-form, one blank line after description
- `footer(s)` are optional: one blank line after body, format `token: value` or `token #value`

## Commit types

Ordered by semantic impact (the type drives the version bump — see [[semantic-versioning]]):

- **feat**: new feature (triggers MINOR version bump in SemVer)
- **fix**: bug fix (triggers PATCH version bump in SemVer)
- **docs**: documentation only
- **style**: formatting, semicolons, whitespace (no logic change)
- **refactor**: code restructuring (no behavior change)
- **perf**: performance improvement
- **test**: adding or correcting tests
- **build**: build system or external dependencies
- **ci**: CI/CD configuration
- **chore**: maintenance tasks (deps update, config)

## Breaking Changes

Two valid ways to signal breaking changes (triggers MAJOR version bump):

1. Append `!` after type/scope: `feat!: remove deprecated endpoint`
2. Footer: `BREAKING CHANGE: removed /v1/users endpoint`
   Both can be combined. Breaking changes can apply to any type.

## Scope Guidelines

- Use the module, package, or component name: `feat(auth):`, `fix(parser):`
- Keep scopes consistent within a project
- Omit scope when the change is cross-cutting

## Analysis process

1. Read the staged diff (`git diff --staged`)
2. Identify the primary intent: is this a feature, fix, refactor, etc.?
3. Determine scope from the files/modules changed
4. Check if this introduces breaking changes to a public API
5. Write the description in imperative mood ("add", not "added" or "adds")
6. Add body if the diff is non-trivial or the "why" is not obvious
7. Add footer for breaking changes or issue references

## Forbidden footers (STRICT)

NEVER add author-attribution trailers to a commit message or to tag release notes:

- No `Co-Authored-By:` trailers of any kind.
- No LLM/AI credit whatsoever (e.g. `Co-Authored-By: Claude ...`, "Generated with Claude Code", "🤖 Generated with ...").
- Commits and tag release notes MUST contain zero AI/assistant attribution.

## Anti-patterns

- Using `update` or `change` as type (not in the spec)
- Past tense in description: "added feature" instead of "add feature"
- Overly vague descriptions: "fix bug", "update code", "misc changes"
- Mixing unrelated changes in one commit (split them)
- Omitting BREAKING CHANGE when a public API changes
- Using commit body to repeat what the diff already shows
- Adding `Co-Authored-By:` or any LLM/AI credit trailer (strictly forbidden)

## Examples

```
feat(auth): add JWT token refresh endpoint

fix: prevent null pointer on empty user profile

refactor(db)!: change connection pool from sync to async

BREAKING CHANGE: NewPool() now returns (Pool, error) instead of Pool

docs: update API reference for v2 endpoints

chore(deps): bump go.mod dependencies to latest

feat: add CSV export for reports

Closes #142
```

## Verification

Before committing:

- [ ] Type is one of the standard types
- [ ] Description is imperative mood, lowercase, no trailing period
- [ ] Scope (if used) matches an existing module/package
- [ ] Breaking changes are marked with `!` AND/OR `BREAKING CHANGE:` footer
- [ ] Body explains "why" (not "what" — the diff shows "what")
- [ ] Each commit is atomic (one logical change)
- [ ] No `Co-Authored-By:` / LLM / AI attribution trailer is present
