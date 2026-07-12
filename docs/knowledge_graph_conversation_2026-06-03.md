# Knowledge Graph — Conversation Scope — 2026-06-03

> **Type:** Session handoff knowledge graph · **Scope:** `conversation` (this thread)
> **Depth:** Compact (references sources, does not duplicate) · **Format:** MD tables + YAML
> **Audience:** Another AI agent taking over · **Project:** `codify` (AI Context Generator CLI)
> **Project root:** `/Users/jorge.corredor/workspace-go/codify`
> **Validated:** git tag + filesystem + merged PRs at generation time (see §6-validation in chat)

---

## 0. How to use this document (READ FIRST)

You inherit a session whose entire arc was: **audit the v3.0.0 release → correct the documented "truth" → plan the fix → execute it (Track 0/1/2) → update docs → publish v3.1.0**. The release is DONE and live.

- **This is an index, not authority.** For full detail of any entity, ALWAYS open the session sources in §9 (the research docs, ADRs, and `MEMORY.md`). Do not treat this file as the source of truth for code — the repo is.
- **It is a snapshot at 2026-06-03.** If the thread continued after, re-confirm before acting.
- **Hard operational restrictions (do NOT violate):**
  1. **NEVER push a tag / publish a release without explicit operator permission** — confirm the exact tag-push first (irreversible, public).
  2. **NEVER include AI credits in commits** (no `Co-Authored-By`, no model mentions). Absolute.
  3. **Conventional Commits**; feature work on `feature/*` → PR → merge (a protected-branch hook blocks direct pushes to `main`; tag pushes are exempt).
  4. **Broken tests are top priority**, always.
  5. **Greenfield** for codify's own platform code (no backward-compat/migrations; replace, don't wrap).
  6. **Spanish** in conversation + domain comments; **"Track"** not "Pista"; do NOT calque "cut a release" → "cortar el release" (use **"publicar/lanzar la versión"**).
  7. **ADRs (`docs/adr/`) and `research/` are git-ignored** — local-only; don't assume teammates see them, don't commit them.
- **§11 YAML** is the machine-parseable edge list (IDs match §1). Consume it for automated reasoning; prose elsewhere is the elaboration.
- **Re-confirm any "deferred" item (§7) with the operator before acting** — the thread may have moved on.

---

## 1. System — macro entities

| ID            | Entity                                       | Type              | Status                                                                      |
| ------------- | -------------------------------------------- | ----------------- | --------------------------------------------------------------------------- |
| `REL-3.1.0`   | v3.1.0                                       | Release / git tag | **Published 2026-06-03** (tag `f6f367b`, non-draft, 4 tarballs + checksums) |
| `REL-3.0.0`   | v3.0.0                                       | Release / git tag | Shipped 2026-05-31 — its gaps are what this whole session fixed             |
| `AUDIT`       | v3.0.0 gap audit                             | Finding set       | Done — `research/V3.0.0_GAP_AUDIT.md`                                       |
| `PLAN`        | v3.1.0 implementation plan                   | Plan (3 tracks)   | All tracks shipped — `research/V3.1.0_IMPLEMENTATION_PLAN.md`               |
| `T0`          | Track 0 — stack/UX decision                  | Work track        | Done → ADR-0013                                                             |
| `T1`          | Track 1 — MCP `generate_specs` fix           | Work track        | Done → PR #22                                                               |
| `T2`          | Track 2 — rich catalog TUI + catalog-as-code | Work track        | Done → PRs #23/#24/#25                                                      |
| `SUB-TUI`     | Rich catalog selector                        | Subsystem (new)   | Shipped — `internal/interfaces/cli/tui/catalog/`                            |
| `SUB-DESIRED` | Desired-state (catalog-as-code)              | Subsystem (new)   | Shipped — `internal/infrastructure/desiredstate/`                           |
| `SUB-SDDHELP` | Shared SDD template helper                   | Subsystem (new)   | Shipped — `internal/infrastructure/sdd/specsupport.go`                      |
| `ADR-0013`    | CLI interaction stack                        | ADR (local-only)  | Accepted — bubbletea+lipgloss, rich TUI, declarative backbone               |

**Project identity (do not misread):** codify is a CLI that uses an **LLM** to generate context/spec files from a project description, and installs agent "equipment" (skills/hooks/plugins) into Claude/Antigravity at workstation/project scope, reproducibly. NOT a deterministic template engine.

---

## 2. Session timeline

| Seq | Phase                  | What happened                                                                                                                                                                  | Outcome                                      |
| --- | ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------- |
| T0  | Open                   | Prior thread had just shipped v3.0.0; this session opened on its closeout, then the operator **installed v3.0.0 and found the catalog UX differed hugely from the agreements** | Triggered everything below                   |
| T1  | **(4) Audit**          | 6 parallel auditors compared v3.0.0 shipped-vs-agreed (ADRs/specs). Found 2 critical issues                                                                                    | `research/V3.0.0_GAP_AUDIT.md`               |
| T2  | **(2) Correct truth**  | Amended ADR-0010 (as-built addendum), `MEMORY.md`, `CHANGELOG` (Known issues), KG banner                                                                                       | docs corrected                               |
| T3  | **(1) Plan + Track 0** | Wrote v3.1.0 plan; deep stack/UX research; **operator chose rich TUI**; spike bubbletea vs tview → **bubbletea**                                                               | ADR-0013, `TRACK_0_STACK_AND_UX_RESEARCH.md` |
| T4  | Track 1                | Fixed MCP `generate_specs` (shared SDD helper + regression test)                                                                                                               | PR #22 merged                                |
| T5  | Track 2                | Pure-logic engine → render → builder → integration → init/config reuse → catalog-as-code (`--apply`) → ASCII/overflow + deps/conflicts reserved                                | PRs #23/#24/#25 merged                       |
| T6  | Docs + release-prep    | Updated README/README_ES/command-reference/lifecycle-matrix; CHANGELOG `[3.1.0]`; `.version`→3.1.0; MCP `serverVersion`→3.1.0                                                  | PR #26 merged                                |
| T7  | **Publish**            | Pushed tag `v3.1.0` (with explicit operator OK) → goreleaser built + published the GH release                                                                                  | **v3.1.0 live**                              |

---

## 3. Files touched in session (by area; new = created)

| Area                       | Path (under project root)                                                                                                             | Change                                                                            |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| TUI (new pkg)              | `internal/interfaces/cli/tui/catalog/{model,render,build,glyphs}.go` (+ tests)                                                        | created — selector engine/render/builder/ASCII                                    |
| Desired-state (new pkg)    | `internal/infrastructure/desiredstate/desiredstate.go` (+ test)                                                                       | created — catalog-as-code                                                         |
| SDD helper (new)           | `internal/infrastructure/sdd/specsupport.go`                                                                                          | created — shared spec template path/mapping                                       |
| MCP                        | `internal/interfaces/mcp/server.go`                                                                                                   | edited — fixed `executeSpecs`, `sdd_standard` param, `serverVersion`→3.1.0        |
| Catalog cmd                | `internal/interfaces/cli/commands/catalog.go`                                                                                         | edited — rich selector wiring, `runCatalogSelector`, `--apply`, write-through     |
| Bootstrap reuse            | `internal/interfaces/cli/commands/{config,init,config_install}.go`                                                                    | edited — `PromptInstallPackages` reuse (Decision 6); removed old `promptInstall*` |
| Domain                     | `internal/domain/catalog/{package_manifest,manifest_from_catalog}.go`                                                                 | edited — `MetaKeyCategory/Tags`, stamp category, deps/conflicts marked RESERVED   |
| Marketplace                | `internal/infrastructure/packagesource/marketplace.go`                                                                                | edited — use category/tags constants                                              |
| CLI spec                   | `internal/interfaces/cli/commands/spec.go`                                                                                            | edited — DRY onto shared SDD helper                                               |
| Docs                       | `README.md`, `README_ES.md`, `docs/{command-reference,lifecycle-matrix}.md`, `CHANGELOG.md`, `.version`                               | edited — reflect new reality + release prep                                       |
| ADR/research (git-ignored) | `docs/adr/0010-*.md`, `docs/adr/0013-*.md`, `research/{V3.0.0_GAP_AUDIT,V3.1.0_IMPLEMENTATION_PLAN,TRACK_0_STACK_AND_UX_RESEARCH}.md` | created/edited                                                                    |
| Auto-memory (outside repo) | `…/memory/MEMORY.md`                                                                                                                  | edited — audit, plan, v3.1.0 released, terminology pref                           |

---

## 4. Findings / issues — index (from the audit)

| ID       | Finding                                                                   | Severity | Status                             |
| -------- | ------------------------------------------------------------------------- | -------- | ---------------------------------- |
| `F-MCP`  | MCP `generate_specs` broken at runtime (loaded removed template path)     | HIGH     | ✅ Fixed (T1 / PR #22)             |
| `F-UX`   | Rich catalog UX (ADR-0010 §3/4/6) never built — shipped a `huh` wizard    | MED      | ✅ Built (T2 / PR #23)             |
| `F-DEC6` | `init`/`config` never reused the selector (Decision 6)                    | MED      | ✅ Done (T2 / PR #23)              |
| `F-DEAD` | `Dependencies`/`Conflicts`/`PackageRef` dead fields, documented-as-active | LOW-MED  | ✅ Downgraded to RESERVED (PR #25) |
| `F-DOCS` | User docs described the old wizard, missed selector/`--apply`/reuse       | MED      | ✅ Updated (PR #26)                |

---

## 5. Emergent findings (insights surfaced this session)

| ID   | Finding                                                                                                                                                                                                                                                                                              |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `E1` | **Stack choice is ~invisible to UX** — bubbletea vs tview look ~90% identical; decide on dep hygiene/testability, not aesthetics. Chose bubbletea (already transitive vs net-new tcell; `teatest` deterministic; single charm ecosystem).                                                            |
| `E2` | **Category must be source-declared, not codify-invented** (operator's framing) — as an aggregator of external catalogs, codify can't impose architecture/testing taxonomies; the `marketplace.json` standard already carries `category`/`tags`. Tree groups by `Metadata[category]`; flat when none. |
| `E3` | **The lockfile was already half the Nix/Terraform model** — codify ships a lockfile + sync that Homebrew removed for lack of consumers; making it user-authorable (catalog-as-code) is the differentiator.                                                                                           |
| `E4` | **TUIs are a trap on accessibility/CI/maintenance** — mitigated with non-TTY fallback (flags/`--apply`), ASCII fallback, and pure-logic-separated-from-render for testability.                                                                                                                       |
| `E5` | **The v3.0.0 bug existed because no test covered the MCP spec path** — every new subsystem this session put pure logic in a terminal-free, unit-tested core.                                                                                                                                         |

---

## 6. Decisions taken this session

| ID   | Decision                                                                                                                     | Origin                                                          |
| ---- | ---------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `D1` | Sequence = **(4) audit → (2) correct truth → (1) plan**, then build                                                          | Operator: _"hagamos (4), luego (2) y por ultimo (1)"_           |
| `D2` | Register the MCP bug and **follow the plan** (don't hotfix out-of-band)                                                      | Operator: _"Registrarlo y seguir el plan"_                      |
| `D3` | Experience = **rich TUI** (tabs+tree+accumulated), full spec, recovering ADR-0010 §3/4/6                                     | Operator (chose TUI over research's tentative "go declarative") |
| `D4` | Stack = **bubbletea + lipgloss**, decided by symmetric spike vs tview → ADR-0013                                             | Operator (accepted spike recommendation)                        |
| `D5` | Category grouping = **source-declared only** (Metadata[category]); flat when absent                                          | Operator's own architectural framing                            |
| `D6` | Declarative file = **YAML, per-scope** (`.codify/*.catalog.yml`), NOT the ADR's tentative `.toml` — to add no new dependency | Agent, dep-hygiene; documented in package + ADR                 |
| `D7` | **Publish v3.1.0** (push tag)                                                                                                | Operator explicit OK                                            |
| `D8` | (Terminology) don't calque "cut a release" in Spanish                                                                        | Operator correction                                             |

---

## 7. Deferred items (NOT debt — out of v3.1.0 scope by plan §6)

| ID         | Item                                                                       | Trigger to resume                    |
| ---------- | -------------------------------------------------------------------------- | ------------------------------------ |
| `R-6`      | Fuzzy filter (`/`) inside the tree + package detail-view side panel        | A "polish the selector" task         |
| `R-7`      | Plugin install fallback when `claude` not on PATH (B2/B1)                  | Demand from agentless environments   |
| `R-8`      | Parameterize `PluginMarketplaceSource` per ecosystem (Antigravity plugins) | When `agy` (Antigravity CLI) matures |
| `R-4r`     | Full CLI↔MCP SDD parity (beyond the `sdd_standard` param shipped)          | If demand appears                    |
| `huh-drop` | Possibly drop `huh` entirely to consolidate on bubbletea                   | Greenfield cleanup pass              |

---

## 8. Operational conventions (binding for the next agent)

- **Release = goreleaser-driven by tag push** (`.github/workflows/release.yml`); tag → builds darwin/linux × amd64/arm64 + checksums → GH release. `.version` is the source of truth. **Never tag without operator OK.**
- **Sub-agent chain:** `git-workflow-manager` (on push) → `version-manager` (bump+tag) → `doc-updater` (version refs only — does NOT rewrite feature docs; those are manual).
- **The protected-branch hook** can mis-parse compound commands containing `origin main` — run pushes as isolated commands.
- **Architecture:** Go DDD/Clean Arch; domain → application (CQRS) → infrastructure → interfaces (cli, mcp). New TUI lives in `interfaces/cli/tui/`, pure logic with zero terminal imports.
- **Two distinct files, do not conflate:** lockfile (`.codify/*.lock`, JSON, codify's install RECORD, `--status`/`--sync`) vs desired-state (`.codify/*.catalog.yml`, YAML, user INTENT, `--apply`).
- **One selector, three entry points:** `codify catalog` / `init` / `config` all funnel to `runCatalogSelector` (ADR-0010 Decision 6).

---

## 9. Session sources (open these for detail — index, not authority)

**Auto-memory (primary cross-session context):**

- `/Users/jorge.corredor/.claude/projects/-Users-jorge-corredor-workspace-go-codify/memory/MEMORY.md`

**Research / ADRs (git-ignored, local-only):**

- `/Users/jorge.corredor/workspace-go/codify/research/V3.0.0_GAP_AUDIT.md`
- `/Users/jorge.corredor/workspace-go/codify/research/V3.1.0_IMPLEMENTATION_PLAN.md`
- `/Users/jorge.corredor/workspace-go/codify/research/TRACK_0_STACK_AND_UX_RESEARCH.md`
- `/Users/jorge.corredor/workspace-go/codify/research/cli-checkbox-tree-tabs-spec.md` (the original UX spec)
- `/Users/jorge.corredor/workspace-go/codify/docs/adr/0013-cli-interaction-stack.md`
- `/Users/jorge.corredor/workspace-go/codify/docs/adr/0010-catalog-command-architecture.md` (has the "as-built" addendum)

**Code (the new subsystems):**

- `/Users/jorge.corredor/workspace-go/codify/internal/interfaces/cli/tui/catalog/` (model/render/build/glyphs)
- `/Users/jorge.corredor/workspace-go/codify/internal/infrastructure/desiredstate/desiredstate.go`
- `/Users/jorge.corredor/workspace-go/codify/internal/infrastructure/sdd/specsupport.go`
- `/Users/jorge.corredor/workspace-go/codify/internal/interfaces/cli/commands/catalog.go`

**Release / public:**

- `https://github.com/jorelcb/codify/releases/tag/v3.1.0`
- PRs `#22` (MCP fix), `#23` (rich selector + reuse), `#24` (catalog-as-code), `#25` (polish), `#26` (release prep)

---

## 10. Open pendings at session close

**Nothing blocking.** v3.1.0 is published; the tree is clean on `main` at `f6f367b` (= tag); suite green. The only items are the opt-in deferrals in §7. Most likely next asks:

1. Pick a §7 follow-up (R-6 fuzzy filter is the natural next selector enhancement).
2. Or open an unrelated new front — re-confirm with the operator.

> One untracked artifact remains on disk: `docs/Knowledge_Graph_conversation_2026-05-31.md` (a prior snapshot, carries a correction banner). Not committed; safe to ignore or remove.

---

## 11. Critical relationships (YAML)

```yaml
edges:
  # cause → fix
  - { from: REL-3.0.0, rel: had-gaps-found-by, to: AUDIT }
  - { from: AUDIT, rel: motivated, to: PLAN }
  - { from: PLAN, rel: delivered-by, to: REL-3.1.0 }
  - { from: F-MCP, rel: fixed-by, to: T1 }
  - { from: F-UX, rel: built-by, to: T2 }
  - { from: F-DEC6, rel: done-by, to: T2 }

  # tracks → PRs (merged)
  - { from: T1, rel: shipped-as, to: "PR#22" }
  - { from: T2, rel: shipped-as, to: ["PR#23", "PR#24", "PR#25"] }
  - { from: T0, rel: produced, to: ADR-0013 }

  # subsystems
  - { from: T2, rel: introduces, to: SUB-TUI }
  - { from: T2, rel: introduces, to: SUB-DESIRED }
  - { from: T1, rel: introduces, to: SUB-SDDHELP }
  - { from: SUB-SDDHELP, rel: used-by, to: ["cli/spec.go", "mcp/server.go"] }

  # the one selector (Decision 6)
  - {
      from: "runCatalogSelector",
      rel: invoked-by,
      to: ["codify catalog", "codify init", "codify config"],
    }

  # decisions
  - {
      from: D6,
      rel: diverges-from,
      to: "ADR-0013 codify.catalog.toml mock",
      note: "YAML per-scope, dep hygiene",
    }
  - {
      from: D3,
      rel: overrides,
      to: "research tentative go-declarative recommendation",
    }

release_facts:
  tag: v3.1.0
  commit: f6f367b
  version_file: "3.1.0"
  date: 2026-06-03
  published_at: "2026-06-03T16:49:28Z"
  draft: false
  prerelease: false
  bump: MINOR
  prs: ["#22", "#23", "#24", "#25", "#26"]
  url: https://github.com/jorelcb/codify/releases/tag/v3.1.0

deferred: [R-6, R-7, R-8, R-4r, huh-drop] # see §7 — out of scope by plan §6
```

---

## 12. Closing — 30-second summary

1. **`codify v3.1.0` is PUBLISHED and live** (tag `f6f367b` on `main`, GH release non-draft, 4 tarballs + checksums, `2026-06-03`).
2. It **closes the v3.0.0 gap audit**: built the rich catalog selector (tabs+tree+cross-tab checkboxes) that v3.0.0 faked with a wizard, and **fixed the MCP `generate_specs` runtime bug**.
3. **Decision 6 recovered:** `codify catalog`, `init`, and `config` now share ONE selector (`runCatalogSelector`).
4. **New: catalog-as-code** — the selector writes a committable `.codify/*.catalog.yml`; `codify catalog --apply` reproduces it. YAML, per-scope. Distinct from the lockfile (intent vs record).
5. **Category grouping is source-declared, never codify-imposed** (operator's framing); flat when the source declares none.
6. **Nothing is blocked.** Deferred items (R-6/R-7/R-8) are opt-in, by plan design — not debt.
7. **Key restrictions:** never tag/release without operator OK; no AI credits in commits; broken tests are #1; greenfield; ADRs/research are git-ignored; in Spanish say "publicar/lanzar la versión", not "cortar el release".
8. **Immediate next move:** ask the operator which §7 follow-up (or new front) to start. Re-confirm before acting — this snapshot may be stale.

---

_Index with pointers — for authoritative detail open §9. Snapshot 2026-06-03; re-invoke knowledge-graph-extractor if the project/conversation moved on._
