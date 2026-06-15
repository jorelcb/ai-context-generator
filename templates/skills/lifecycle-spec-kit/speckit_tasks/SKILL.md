# Spec-Kit: Tasks

Decompose `plan.md` into an actionable, dependency-ordered `tasks.md` that an engineer (human or AI) can execute end-to-end. Invoked as `/speckit-tasks`.

**Lifecycle position** — step 4 of the chain:
[[speckit-specify]] → [[speckit-clarify]] → [[speckit-plan]] → `/speckit-tasks` → [[speckit-analyze]] → implement.

Consumes the plan and design artifacts from [[speckit-plan]] and produces the task list that [[speckit-analyze]] cross-checks before implementation begins.

## When to use

- A complete `plan.md` exists with a green Constitution Check.
- You need an executable breakdown organized for incremental, independently testable delivery.

## What it produces

`specs/<feature-id>/tasks.md`, organized **by user story** and **MVP-first**, with task IDs, file paths, parallel markers, a dependency graph, and an implementation strategy.

## Flow

1. **Load design documents** from the feature directory:
   - Required: `plan.md` (tech stack, structure), `spec.md` (user stories with P1/P2/P3 priorities).
   - Optional: `data-model.md` (entities), `contracts/` (interfaces), `research.md` (decisions), `quickstart.md` (scenarios). Generate from whatever is available.
   - Load `.specify/memory/constitution.md` if present.
2. **Map components to user stories.** Each entity, service, and interface contract maps to the story it serves. Shared infrastructure goes to Setup or Foundational.
3. **Build phases** (see structure below) and a **dependency graph** showing story completion order.
4. **Validate completeness** — each user story has all the tasks it needs and is independently testable.

## Task format (REQUIRED)

Every task strictly follows:

```text
- [ ] [TaskID] [P?] [Story?] Description with file path
```

- **Checkbox** — always `- [ ]`.
- **Task ID** — sequential `T001`, `T002`, … in execution order.
- **`[P]`** — include **only** if parallelizable: different files, no dependency on an incomplete task.
- **`[Story]`** — `[US1]`, `[US2]`, … required on user-story-phase tasks only (not Setup, Foundational, or Polish).
- **Description** — a clear action with an exact file path. Generic tasks like "implement the feature" are forbidden.

Correct: `- [ ] T012 [P] [US1] Create User model in src/models/user.py`
Wrong: `- [ ] Create User model` (no ID, story, or path) · `- [ ] T001 [US1] Create model` (no file path).

## Phase structure

- **Phase 1 — Setup**: project initialization. No story label.
- **Phase 2 — Foundational**: blocking prerequisites that must complete before any user story. No story label.
- **Phase 3+ — User Stories** in priority order (P1, then P2, then P3…). Each story is its own phase: within it, order as Models → Services → Endpoints → Integration (and Tests first **only if** explicitly requested / TDD). Each phase is a complete, independently testable increment with a **checkpoint**.
- **Final Phase — Polish & Cross-Cutting Concerns**. No story label.

Tests are **optional** — generate test tasks only when the spec or user requests TDD.

## Quality criteria

- Tasks organized by user story; P1 (the MVP) deliverable on its own.
- Every task has a checkbox, sequential ID, correct labels, and an exact file path.
- `[P]` only on truly independent tasks; dependencies otherwise explicit in the dependency graph.
- Each story phase ends at a testable checkpoint.
- The list is immediately executable — each task is specific enough to complete without extra context.

## Don'ts

- Don't invent tasks not derivable from `plan.md`.
- Don't omit file paths or task IDs, or skip the `[Story]` label on story-phase tasks.
- Don't mark dependent tasks `[P]` — that breaks parallel execution.
- Don't add test tasks unless tests were requested.
- Don't bury MVP scope — User Story 1 should stand alone as the first shippable slice.

## Done when

- `tasks.md` generated with all phases, sequential IDs, correct labels, and file paths.
- Dependency graph and MVP scope (typically just User Story 1) identified.
- Reported back: total task count, per-story breakdown, parallel opportunities, and confirmation that every task follows the checklist format.

## Going deeper

Previous: [[speckit-plan]]. Next: [[speckit-analyze]] cross-checks spec ↔ plan ↔ tasks for consistency before implementation. Full chain: [[speckit-specify]] · [[speckit-clarify]] · [[speckit-plan]] · [[speckit-analyze]].
