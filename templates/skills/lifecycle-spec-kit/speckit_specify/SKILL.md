# Spec-Kit: Specify

Capture **what** a feature does and **why** as the single source of truth for the rest of the Spec-Kit lifecycle. Invoked as `/speckit-specify`.

**Lifecycle position** — this is step 1 of the chain:
`/speckit-specify` → [[speckit-clarify]] → [[speckit-plan]] → [[speckit-tasks]] → [[speckit-analyze]] → implement.

You produce `specs/<feature-id>/spec.md`. The next phase, [[speckit-clarify]], resolves any `[NEEDS CLARIFICATION]` markers you leave behind before planning starts.

## When to use

- Starting a new feature from a natural-language description.
- The user has an idea but no spec, plan, or tasks yet.
- Re-specifying an existing feature whose scope has shifted.

## What it produces

A populated `specs/<feature-id>/spec.md` containing:

- **User stories** prioritized **P1 / P2 / P3** (P1 = the MVP slice). Each is independently testable and written in user-value terms.
- **Functional Requirements** keyed `FR-001`, `FR-002`, … — each testable and unambiguous.
- **Success Criteria** keyed `SC-001`, … — measurable and technology-agnostic.
- **Key Entities** (only when the feature involves data).
- **`[NEEDS CLARIFICATION: <question>]`** markers where a real gap blocks correctness — **maximum 3**.

## Flow

1. **Derive a feature-id** — a 2-4 word kebab-case slug from the description (action-noun when possible, e.g. `user-auth`, `oauth2-api-integration`). Preserve acronyms (OAuth2, JWT, API).
2. **Create the feature directory** — `specs/<feature-id>/`, with `spec.md` as the artifact. One feature per invocation. Do not silently overwrite an existing spec.
3. **Load the constitution if present** — `.specify/memory/constitution.md` for governing principles and constraints.
4. **Extract concepts** from the description: actors, actions, data, constraints.
5. **Write user scenarios** — primary story plus acceptance scenarios in Given/When/Then form. Assign P1/P2/P3 by user value, with P1 as the smallest shippable increment.
6. **Generate Functional Requirements** (`FR-XXX`) — each testable. Use reasonable industry-standard defaults for unspecified details and record them in an **Assumptions** section rather than asking.
7. **Define Success Criteria** (`SC-XXX`) — measurable, user-focused, verifiable without implementation knowledge.
8. **Identify Key Entities** if data is involved.
9. **Mark gaps** with `[NEEDS CLARIFICATION: ...]` only when no reasonable default exists, the choice materially changes scope/UX, or multiple interpretations conflict. Prioritize: scope > security/privacy > UX > technical detail. **Hard cap of 3.**
10. **Validate** against the quality checklist below; iterate the spec until it passes.

## Quality criteria

- No implementation details (languages, frameworks, APIs, file paths).
- Focused on user value; readable by a non-technical stakeholder.
- Every requirement testable and unambiguous.
- Success criteria measurable and technology-agnostic.
- Acceptance scenarios and edge cases identified; scope clearly bounded.
- Dependencies and assumptions recorded.

**Measurable Success Criteria — good vs bad:**

- Good: "Users complete checkout in under 3 minutes"; "95% of searches return in under 1 second"; "System supports 10,000 concurrent users".
- Bad: "API response under 200ms"; "Redis cache hit rate above 80%"; "React components render efficiently" — these leak technology.

## Don'ts

- Don't specify HOW (tech stack, file paths, library names) — that is [[speckit-plan]].
- Don't write tasks — that is [[speckit-tasks]].
- Don't write code; `/speckit-specify` is design-time only.
- Don't exceed 3 `[NEEDS CLARIFICATION]` markers — make informed guesses and document them as assumptions instead.
- Don't embed checklists inside the spec; quality validation is a separate concern.

## Done when

- `specs/<feature-id>/spec.md` written, with P1/P2/P3 stories, `FR-XXX`, `SC-XXX`, and at most 3 `[NEEDS CLARIFICATION]` markers.
- The spec passes the quality criteria above (or remaining gaps are honest, scope-critical clarifications).
- Reported back: the feature directory, the spec path, and readiness for the next phase.

## Going deeper

Next step: [[speckit-clarify]] resolves the `[NEEDS CLARIFICATION]` markers before planning. Full chain: [[speckit-clarify]] · [[speckit-plan]] · [[speckit-tasks]] · [[speckit-analyze]].
