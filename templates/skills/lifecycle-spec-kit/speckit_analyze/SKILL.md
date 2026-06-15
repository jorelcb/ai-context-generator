# Spec-Kit: Analyze

Run a non-destructive, **read-only** cross-artifact consistency and quality check across `spec.md`, `plan.md`, and `tasks.md` before implementation. This is the final quality gate. Invoked as `/speckit-analyze`.

**Lifecycle position** — step 5 of the chain:
[[speckit-specify]] → [[speckit-clarify]] → [[speckit-plan]] → [[speckit-tasks]] → `/speckit-analyze` → implement.

Runs only after [[speckit-tasks]] has produced a complete `tasks.md`. It validates the trio of artifacts and clears the path to implementation — it does **not** modify them.

## When to use

- All three artifacts exist (`spec.md`, `plan.md`, `tasks.md`).
- Immediately before starting implementation, to catch gaps, contradictions, and uncovered requirements.

## Operating constraints

- **Strictly read-only.** Never modify any file. Output a structured report and, optionally, a remediation plan the user must explicitly approve.
- **Constitution is authority.** Any conflict with a MUST principle in `.specify/memory/constitution.md` is automatically CRITICAL and resolved by fixing the spec/plan/tasks — never by diluting the principle.

## Flow

1. **Load artifacts** (minimal, progressive disclosure):
   - From `spec.md`: overview, Functional Requirements, Success Criteria, user stories, edge cases.
   - From `plan.md`: architecture/stack, data-model references, phases, technical constraints.
   - From `tasks.md`: task IDs, descriptions, phase grouping, `[P]` markers, referenced file paths.
   - From `.specify/memory/constitution.md`: principle names and MUST/SHOULD statements.
   - Abort if any required file is missing — tell the user which prerequisite command to run.
2. **Build semantic models** (internal, not echoed): requirements inventory keyed by `FR-###` / `SC-###`; user-story/action inventory; task→requirement coverage mapping; constitution rule set. Include only Success Criteria that require buildable work — exclude post-launch KPIs.
3. **Run detection passes** (high-signal; cap at 50 findings, summarize overflow):
   - **Duplication** — near-duplicate requirements; flag the weaker phrasing for consolidation.
   - **Ambiguity** — vague adjectives (fast, scalable, secure, robust) without measurable criteria; unresolved placeholders (TODO, ???, `<placeholder>`).
   - **Underspecification** — requirements missing object/measurable outcome; stories missing acceptance criteria; tasks referencing undefined components.
   - **Constitution alignment** — any element conflicting with a MUST principle, or a missing mandated gate.
   - **Coverage gaps** — requirements with zero tasks; tasks with no mapped requirement; buildable Success Criteria not reflected in tasks.
   - **Inconsistency** — terminology drift; entities in plan but absent from spec (or vice versa); task-ordering contradictions; directly conflicting requirements.
4. **Assign severity:** CRITICAL (constitution MUST violation, missing core artifact, zero-coverage requirement blocking baseline) · HIGH (conflicting/duplicate requirement, untestable security/performance attribute) · MEDIUM (terminology drift, missing non-functional coverage, underspecified edge case) · LOW (wording/redundancy not affecting execution).
5. **Emit a compact report** (see below).
6. **Offer remediation** — ask whether to suggest concrete edits for the top issues. Never apply them automatically.

## Report structure

- A findings table: `| ID | Category | Severity | Location(s) | Summary | Recommendation |` (one row per finding; stable IDs prefixed by category initial, e.g. `A1`, `D2`).
- A **Coverage Summary** table: `| Requirement Key | Has Task? | Task IDs | Notes |`.
- **Constitution Alignment Issues** and **Unmapped Tasks** sections (if any).
- **Metrics**: total requirements, total tasks, coverage % (requirements with ≥1 task), ambiguity count, duplication count, critical-issue count.
- **Next Actions**: if CRITICAL issues exist, recommend resolving before implement; otherwise the user may proceed. Suggest explicit follow-ups (re-run [[speckit-specify]] to refine, [[speckit-plan]] to adjust architecture, or hand-edit `tasks.md` to close a coverage gap).

## Quality criteria

- Deterministic — rerunning without changes yields consistent IDs and counts.
- High-signal — cite specific instances, not generic patterns; never hallucinate missing sections.
- Constitution violations always surface as CRITICAL.
- Zero-issue runs still emit a success report with coverage statistics.

## Don'ts

- Don't modify any file — this is read-only analysis.
- Don't apply remediation edits without explicit user approval.
- Don't dilute, reinterpret, or silently ignore a constitution principle.
- Don't hallucinate sections that are absent — report them as missing.
- Don't exceed 50 findings in the table; aggregate the rest in an overflow summary.

## Done when

- All three artifacts (plus constitution, if present) loaded and cross-analyzed.
- A structured report emitted with findings, coverage summary, metrics, and next actions.
- Remediation offered but not auto-applied; the user knows whether it's safe to implement.

## Going deeper

Previous: [[speckit-tasks]]. Next: implementation (execute `tasks.md` once this gate is clear). Full chain: [[speckit-specify]] · [[speckit-clarify]] · [[speckit-plan]] · [[speckit-tasks]].
