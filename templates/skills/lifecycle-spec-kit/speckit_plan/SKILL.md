# Spec-Kit: Plan

Translate `spec.md` into a concrete technical plan that captures **how** the feature will be built — before any code is written. Invoked as `/speckit-plan`.

**Lifecycle position** — step 3 of the chain:
[[speckit-specify]] → [[speckit-clarify]] → `/speckit-plan` → [[speckit-tasks]] → [[speckit-analyze]] → implement.

Consumes the clarified spec from [[speckit-clarify]] and produces `specs/<feature-id>/plan.md` plus supporting design artifacts that [[speckit-tasks]] decomposes next.

## When to use

- The spec is complete and clarified (`[NEEDS CLARIFICATION]` markers resolved).
- You need to decide stack, structure, and design before generating tasks.

## What it produces

- `specs/<feature-id>/plan.md` — the implementation plan.
- Phase 0: `research.md` — decisions resolving any remaining unknowns.
- Phase 1: `data-model.md`, `contracts/`, `quickstart.md` (as applicable).
- An updated agent context reference pointing at the new plan.

## Flow

1. **Load context.** Read `spec.md` and, if present, `.specify/memory/constitution.md`. Load the plan template as the structure to fill.
2. **Fill Technical Context.** Language, frameworks, datastore, deployment target, test strategy — match the existing repo's conventions. Mark genuine unknowns as `NEEDS CLARIFICATION` (resolved in Phase 0, not deferred to coding).
3. **Constitution Check.** Verify the plan honors the project's principles — architectural pattern, naming, dependency policy, test layering. **ERROR on any gate violation that isn't explicitly justified.** Misalignment caught here is cheap; caught after coding it is expensive.
4. **Project structure.** Exact paths for new and modified files.
5. **Phase 0 — Research.** For each unknown / dependency / integration, run a research task and consolidate findings in `research.md` using: **Decision** (what was chosen), **Rationale** (why), **Alternatives considered**. Exit criterion: zero unresolved `NEEDS CLARIFICATION`.
6. **Phase 1 — Design & Contracts.**
   - Extract entities from the spec → `data-model.md` (fields, relationships, validation rules, state transitions).
   - Define interface contracts → `contracts/` (public APIs for libraries, command schemas for CLIs, endpoints for services, grammars for parsers). Skip if the project is purely internal.
   - Write `quickstart.md` — runnable validation scenarios proving the feature works end-to-end (prerequisites, setup, run/test commands, expected outcomes). Reference contracts and the data model instead of duplicating them; no full implementation code.
   - Update the agent context reference to point at this plan.
7. **Re-run the Constitution Check** after design to catch violations introduced by Phase 1.

## Quality criteria

- Technical Context is concrete and matches repo conventions — no lingering `NEEDS CLARIFICATION` after Phase 0.
- Constitution Check is green both pre- and post-design, or every deviation is explicitly justified.
- Project structure lists exact file paths.
- `research.md` records Decision / Rationale / Alternatives for each non-obvious choice.
- `quickstart.md` is a validation guide, not an implementation dump.
- Absolute paths for filesystem operations; project-relative paths for references in docs.

## Don'ts

- Don't enumerate tasks here — that is [[speckit-tasks]] territory.
- Don't skip or hand-wave the Constitution Check.
- Don't write implementation code or full test suites — those belong to the implement phase; quickstart only references them.
- Don't leave unknowns unresolved; Phase 0 exists to close them.
- Don't restate requirements from the spec — link back to it.

## Done when

- `plan.md` written with Technical Context, a green Constitution Check, and project structure.
- Phase 0 (`research.md`) and Phase 1 (`data-model.md`, `contracts/`, `quickstart.md`) artifacts generated as applicable.
- Reported back: the plan path and the list of generated artifacts.

## Going deeper

Previous: [[speckit-clarify]]. Next: [[speckit-tasks]] decomposes this plan into an ordered, executable task list. Full chain: [[speckit-specify]] · [[speckit-clarify]] · [[speckit-tasks]] · [[speckit-analyze]].
