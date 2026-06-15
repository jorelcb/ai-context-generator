# Spec-Kit: Clarify

Detect and reduce ambiguity in the feature spec by asking up to **5 targeted questions** and encoding the answers back into `spec.md`. This is the quality gate **before** planning. Invoked as `/speckit-clarify`.

**Lifecycle position** — step 2 of the chain:
[[speckit-specify]] → `/speckit-clarify` → [[speckit-plan]] → [[speckit-tasks]] → [[speckit-analyze]] → implement.

Run this after [[speckit-specify]] and **before** [[speckit-plan]]. If the user explicitly skips clarification (e.g. an exploratory spike), you may proceed — but warn that downstream rework risk rises.

## When to use

- A spec exists (from [[speckit-specify]]) and may contain `[NEEDS CLARIFICATION]` markers or vague language.
- Before planning, to lock down decisions that affect architecture, data modeling, or acceptance tests.

## What it produces

The same `spec.md`, updated in place with:

- A `## Clarifications` section containing a `### Session YYYY-MM-DD` subheading.
- One bullet per accepted answer: `- Q: <question> → A: <answer>`.
- The corresponding spec sections (requirements, user stories, data model, success criteria, edge cases) updated to reflect each answer.

## Flow

1. **Locate the spec.** If no spec file exists, stop and tell the user to run [[speckit-specify]] first — do not create one here.
2. **Load the constitution if present** (`.specify/memory/constitution.md`).
3. **Scan for ambiguity** across this taxonomy, marking each category Clear / Partial / Missing:
   - Functional scope & behavior (goals, out-of-scope, roles)
   - Domain & data model (entities, identity, lifecycle, scale)
   - Interaction & UX flow (journeys, error/empty/loading states, a11y)
   - Non-functional attributes (performance, scalability, reliability, observability, security/privacy, compliance)
   - Integration & external dependencies (APIs, failure modes, formats)
   - Edge cases & failure handling (negative paths, throttling, conflicts)
   - Constraints & tradeoffs; terminology consistency; completion signals (testable acceptance criteria); leftover placeholders / vague adjectives ("robust", "intuitive").
4. **Build a prioritized question queue (max 5).** Include a question only if its answer materially impacts architecture, data modeling, task decomposition, test design, UX, operational readiness, or compliance. Rank by `Impact × Uncertainty`. Each question must be answerable by a **2-5 option multiple choice** OR a **≤5-word short answer**.
5. **Ask one question at a time** (sequential loop):
   - For multiple choice: state your **Recommended** option with 1-2 sentences of reasoning, then render all options as a Markdown table, then let the user reply with a letter, "yes"/"recommended", or their own short answer.
   - For short answer: state a **Suggested** answer with brief reasoning, constrain to ≤5 words.
   - Never reveal future queued questions in advance.
6. **Integrate each answer immediately** (incremental, atomic save after each):
   - Append the `- Q: … → A: …` bullet under `## Clarifications`.
   - Apply the answer to the most appropriate section: functional ambiguity → Functional Requirements; actor/role → User Stories; data shape → Key Entities/Data Model; non-functional → Success Criteria (convert vague adjective to a metric); edge case → Edge Cases; terminology → normalize the term spec-wide.
   - If an answer invalidates earlier text, **replace** it — leave no contradictory statement behind.
7. **Stop early** when all critical ambiguities are resolved, the user signals completion ("done", "stop", "proceed"), or you reach 5 asked questions.
8. **Re-validate the quality checklist** if `checklists/requirements.md` exists — toggle only the `[ ]`/`[x]` markers whose pass state actually changed.

## Quality criteria

- Exactly one `## Clarifications` bullet per accepted answer; no duplicates.
- Total asked questions ≤ 5 (re-asking the same question for disambiguation doesn't count as new).
- Updated sections contain no leftover vague placeholder the answer was meant to resolve.
- One canonical term used consistently across all touched sections.
- Only new headings introduced are `## Clarifications` and `### Session YYYY-MM-DD`.

## Don'ts

- Don't exceed 5 questions, and don't ask low-impact or purely stylistic ones.
- Don't ask speculative tech-stack questions unless their absence blocks functional clarity — those belong in [[speckit-plan]].
- Don't create a spec here; if it's missing, defer to [[speckit-specify]].
- Don't reorder unrelated sections or disturb the heading hierarchy.
- Don't leave contradictory text after an answer supersedes it.

## Done when

- Ambiguities identified and each accepted answer integrated into `spec.md`.
- Quality checklist re-validated (if present).
- Reported back: questions asked/answered, sections touched, a coverage summary (Resolved / Deferred / Clear / Outstanding), and whether to proceed to [[speckit-plan]] or re-run clarify later.

## Going deeper

Previous: [[speckit-specify]]. Next: [[speckit-plan]] translates the now-clarified spec into a technical plan. Full chain: [[speckit-specify]] · [[speckit-plan]] · [[speckit-tasks]] · [[speckit-analyze]].
