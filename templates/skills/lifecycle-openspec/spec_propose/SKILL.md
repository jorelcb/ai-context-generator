# Spec Propose (OpenSpec)

Plan a change before any code is written. Create an isolated OpenSpec change workspace under `openspec/changes/<id>/` holding a proposal, an optional design note, an atomic task list, and spec deltas — then get the intent reviewed and approved. This is phase 1 of 3 in the OpenSpec lifecycle: `/spec-propose` -> `/spec-apply` -> `/spec-archive`.

## When to use

- A new feature, fix, or capability change needs a formal plan before implementation
- You are about to touch behavior described in `openspec/specs/` and want a delta-based audit trail
- You need reviewers to approve intent (the "what" and "why") before code exists
- Starting the OpenSpec lifecycle — the downstream phases are [[spec-apply]] then [[spec-archive]]

The change workspace is the contract for the whole lifecycle: [[spec-apply]] executes its `tasks.md`, and [[spec-archive]] consolidates its deltas into the source-of-truth specs.

## Process

### 1. Understand the request

Before writing any artifact, get the change clear in your head:

- What feature or fix is being proposed, and what user-visible behavior changes?
- Which capabilities does it affect?
- What is the motivation — compliance, performance, UX, a bug?

### 2. Read the existing specs

The bootstrap generates real `openspec/specs/<capability>/spec.md` files, so this step finds actual content. List the directories under `openspec/specs/`, read the `spec.md` of each candidate capability, and identify which requirements you will ADD, MODIFY, or REMOVE. If `openspec/project.md` exists, read it for project-wide conventions.

### 3. Choose a change identifier

Pick a kebab-case `<verb>-<short-description>` id (e.g. `add-2fa`, `fix-session-expiry`, `remove-legacy-auth`). No version numbers or dates — the id becomes the directory name under `openspec/changes/`.

### 4. Create the change workspace

Initialize the directories:

- `openspec/changes/<change-id>/` for the proposal artifacts
- `openspec/changes/<change-id>/specs/` for the deltas, with one subdirectory per affected capability

### 5. Write proposal.md

Capture the rationale concisely (under ~100 lines):

- **Title** — one-line summary
- **Motivation** — why this matters; link issues or tickets
- **Scope** — what is and is NOT included
- **Impact** — affected capabilities, breaking changes, migrations

### 6. Write design.md (when trade-offs exist)

Skip it for trivial changes; write it whenever there is a real decision to record: architecture decisions and the patterns/libraries chosen, trade-offs (what was considered and rejected, and why), integration points (APIs, schemas, contracts touched), and the testing strategy.

### 7. Write tasks.md

Decompose the work into atomic, sequential tasks grouped by phase (data, business logic, interfaces, UI). Each task is one indivisible unit — one file, one function, one schema change — in checkbox form:

```markdown
## 1. Data layer

- [ ] 1.1 Add `two_factor_secret` column to users table
- [ ] 1.2 Backfill nulls in migration

## 2. Business logic

- [ ] 2.1 Implement TOTP generation in AuthService
```

Aim for 5-20 tasks. If you need many more, the change is too large — split it into separate proposals.

### 8. Write the spec deltas

For each affected capability, write `openspec/changes/<change-id>/specs/<capability>/spec.md` using delta sections. This is the syntax `openspec validate` requires — follow it literally.

Group requirements under exactly these delta headers:

```markdown
## ADDED Requirements

### Requirement: Two-factor authentication

The system SHALL require a valid TOTP code when a user with 2FA enabled signs in.

#### Scenario: Correct code accepted

- **GIVEN** a user with 2FA enabled
- **WHEN** they submit a valid TOTP code at sign-in
- **THEN** the session is established

#### Scenario: Wrong code rejected

- **GIVEN** a user with 2FA enabled
- **WHEN** they submit an invalid TOTP code
- **THEN** sign-in is denied and an attempt is logged

## MODIFIED Requirements

### Requirement: Session expiry

The system SHALL expire idle sessions after 15 minutes. (Previously: 60 minutes)

#### Scenario: Idle session expires

- **GIVEN** an authenticated session idle for 15 minutes
- **WHEN** the next request arrives
- **THEN** the session is invalid and re-authentication is required

## REMOVED Requirements

### Requirement: Legacy password-only login

Removed: superseded by mandatory 2FA for privileged accounts.
```

Rules to honor:

- Every requirement uses a `### Requirement:` heading and SHALL-language prose.
- Every requirement carries at least one `#### Scenario:` written in GIVEN/WHEN/THEN.
- MODIFIED requirements include the **complete** revised requirement text and annotate the change with `(Previously: …)`.
- REMOVED requirements state the reason for deprecation.
- RENAMED requirements (`## RENAMED Requirements`) note the old and new names.

### 9. Create the feature branch

Pull the latest base branch, then branch as `feature/<change-id>` (or per project convention) and push to establish tracking.

### 10. Commit and request intent review

Stage `openspec/changes/<change-id>/`, commit as `propose: <change-id> - one-line summary`, and push. Open a draft PR or share the change directory and have reviewers focus on `proposal.md` and the spec deltas — the intent, not code. Iterate until intent is approved; implementation starts only after approval, in [[spec-apply]].

## Anti-patterns

- Writing code before the proposal is approved
- Vague tasks ("implement feature") instead of atomic, single-unit steps
- Spec deltas without GIVEN/WHEN/THEN scenarios, or missing the literal `### Requirement:` / `#### Scenario:` headings
- MODIFIED requirements that show only the diff instead of the full revised text and a `(Previously: …)` note
- Editing `openspec/specs/` directly instead of writing deltas under the change workspace
- Bundling multiple unrelated changes into one proposal
- Skipping `design.md` when real trade-offs exist

## Verification

Before requesting review:

- [ ] `openspec/changes/<change-id>/` holds proposal.md, tasks.md, and per-capability deltas
- [ ] Every requirement has a `### Requirement:` heading and at least one GIVEN/WHEN/THEN `#### Scenario:`
- [ ] MODIFIED requirements carry full text plus `(Previously: …)`; REMOVED give a reason
- [ ] `openspec validate` passes on the change
- [ ] Tasks are atomic, phased, and number 5-20
- [ ] Intent is approved before moving to [[spec-apply]]

Next in the lifecycle: [[spec-apply]]. Final phase: [[spec-archive]].
