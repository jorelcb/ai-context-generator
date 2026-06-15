// Package catalog define el registro declarativo de categorías y opciones de skills.
package catalog

import (
	"fmt"
	"maps"
	"strings"
)

// SkillCategory representa una categoría de nivel 1 en el menú de skills.
type SkillCategory struct {
	Name      string        // identificador: "architecture", "workflow"
	Label     string        // display: "Architecture", "Workflow"
	Exclusive bool          // true = sub-opciones mutuamente excluyentes (sin "all")
	Options   []SkillOption // sub-opciones disponibles
}

// SkillOption representa una sub-opción dentro de una categoría.
type SkillOption struct {
	Name        string // identificador: "clean-ddd", "neutral", "conventional-commit"
	Label       string // display: "Clean (DDD, BDD, CQRS, Hexagonal)"
	TemplateDir string // directorio bajo templates/skills/
	// TemplateMapping mapea el SUBDIRECTORIO del skill (multi-archivo: SKILL.md
	// + reference.md + examples.md opcionales) bajo TemplateDir al guide name.
	TemplateMapping map[string]string
}

// ResolvedSelection es el resultado de resolver una selección del catálogo.
type ResolvedSelection struct {
	TemplateDir     string
	TemplateMapping map[string]string // nil = cargar todos los templates del directorio
}

// SkillMeta contiene metadata de un skill para generación de frontmatter estático.
type SkillMeta struct {
	Description string
	Triggers    []string // usado por ecosistemas que lo soportan (e.g. antigravity)
}

// SkillMetadata mapea guide names a su metadata para frontmatter.
var SkillMetadata = map[string]SkillMeta{
	// Architecture: clean-ddd
	"ddd_entity":       {Description: "Model domain entities, value objects and aggregates following DDD. Use when creating a new aggregate, extracting value objects from primitive obsession, encoding business invariants, building factory methods for valid construction, or defining domain events. Lives in the Domain layer — see clean-arch-layer for placement.", Triggers: []string{"entity", "aggregate", "value object", "domain model"}},
	"clean_arch_layer": {Description: "Place code in the correct architectural layer and wire dependencies following Clean Architecture + DDD. Use when adding features, creating application services, deciding where code goes, resolving layer/import violations, or designing the dependency direction. Pairs with ddd-entity, hexagonal-port and cqrs-command.", Triggers: []string{"layer", "architecture", "dependency rule"}},
	"bdd_scenario":     {Description: "Write a good Gherkin feature file fast — the mechanics of scenarios, steps, Backgrounds, Scenario Outlines and step definitions. Use when you already know the behavior (from discovery) and need to formulate or refactor a .feature file. For the BDD method (Three Amigos, Example Mapping, outside-in), see test-bdd.", Triggers: []string{"test", "scenario", "gherkin", "bdd"}},
	"cqrs_command":     {Description: "Implement use cases with CQRS — separate commands (write) from queries (read) as Application Services. Use when adding a use case, separating read/write paths, building a command handler or query handler, or designing input/output DTOs. Scope is simple model separation (no event sourcing). These handlers are the Application layer of clean-arch-layer.", Triggers: []string{"command", "query", "cqrs", "handler"}},
	"hexagonal_port":   {Description: "Define ports (interfaces) and implement adapters following Hexagonal Architecture (Ports & Adapters). Use when integrating an external service, defining a repository/gateway/notifier contract, building an API client or DB access, adding a messaging adapter, or deciding what is a port vs an adapter. Ports live in the Domain — see clean-arch-layer.", Triggers: []string{"port", "adapter", "hexagonal", "dependency inversion"}},
	// Architecture: hexagonal
	"port_definition":      {Description: "Define ports (interfaces) at the application-core boundary in Hexagonal Architecture — driven ports as Domain-owned interfaces named by capability with domain-typed signatures, and the Application Service as the driving entry point (driving interface optional). Use when contracting an external dependency (DB, API, message bus), exposing a new use case, or deciding whether a driving interface is justified.", Triggers: []string{"port", "interface", "contract", "boundary"}},
	"adapter_pattern":      {Description: "Implement adapters on both sides of the hexagon: thin driving adapters (HTTP/CLI/consumers) that translate triggers into Application Service calls, and driven adapters in Infrastructure that implement Domain ports with error translation at the boundary. Use when building a repository against a concrete database, integrating an external API behind a gateway port, writing an HTTP handler, or wiring the composition root.", Triggers: []string{"adapter", "implementation", "driver", "driven"}},
	"dependency_inversion": {Description: "Apply the Dependency Inversion Principle so the application core never depends on infrastructure: extract direct dependencies into Domain-owned driven ports, inject via constructor, wire concretes only at the composition root, and enforce the direction with dependency fitness functions. Use when refactoring a use case coupled to a DB/SDK, breaking core-to-infra imports, or fixing tests that require real infrastructure.", Triggers: []string{"dependency inversion", "dip", "decoupling"}},
	"hex_integration_test": {Description: "Write contract tests that prove adapter swappability in Hexagonal Architecture: one suite per port, parameterized by an adapter factory, run identically against real infrastructure (testcontainers) and the in-memory fake, asserting domain behavior and error translation — never adapter internals. Use when adding an adapter, validating one against its port, or creating the shared suite future implementations must pass.", Triggers: []string{"integration test", "adapter test", "contract test"}},
	// Architecture: event-driven
	"command_handler":   {Description: "Implement command handlers in event-driven architecture — imperative intent-revealing commands (PlaceOrder), one aggregate per transaction, events persisted via the transactional outbox, mandatory idempotency, and domain-vs-infra failure handling. Use when adding a use case that mutates state, refactoring a CRUD endpoint into a command, or splitting a service that mixes commands and queries. For simple CQRS without events, use cqrs-command.", Triggers: []string{"command", "handler", "cqrs"}},
	"domain_event":      {Description: "Model and publish domain events as immutable facts — past-tense naming (OrderPlaced, never OrderUpdated), envelope with correlation/causation IDs, payload sizing, schema evolution rules, and publishing through the transactional outbox. Use when designing the event a use case will emit, adding an event for a feature, refactoring status flags into events, or evolving the schema of an event already in production.", Triggers: []string{"event", "domain event", "publish", "subscribe"}},
	"event_projection":  {Description: "Build read-side projections that fold event streams into query-shaped read models — atomic checkpointing, idempotent applies, denormalized private tables, eventual-consistency UX, and truncate-and-replay rebuilds (including blue/green zero-downtime rebuilds). Use when creating a new query view, optimizing a query that hits the write model, or migrating from a shared read/write schema to separated read models.", Triggers: []string{"projection", "read model", "denormalization", "view"}},
	"saga_orchestrator": {Description: "Coordinate long-running flows across aggregates or services with orchestrated sagas — explicit step contracts (trigger, command, success/failure events, timeout, compensation), persisted state, reverse-order semantic compensation, and human escalation paths. A saga is NOT a distributed transaction. Use when a flow spans multiple aggregates, calls an external service mid-flow, or needs steps undone when a downstream operation fails.", Triggers: []string{"saga", "process manager", "compensation", "orchestration"}},
	"event_idempotency": {Description: "Guarantee exactly-once effects under at-least-once delivery — inbox/dedup tables claimed in the same transaction as effects, deterministic operation IDs, aggregate-version optimistic concurrency, naturally idempotent operations, and idempotency keys for external side effects. Use when adding an event consumer or command handler, debugging duplicate emails/charges in production, or auditing handlers for idempotency gaps.", Triggers: []string{"idempotency", "deduplication", "exactly-once", "at-least-once"}},
	// Architecture: neutral
	"code_review":     {Description: "Perform structured, severity-labeled code reviews: machine checks first, then correctness, edge cases, security, tests and design, ending in an explicit verdict. Use when reviewing a pull request or diff, doing pre-merge validation, auditing code quality, or asked whether a change is safe to ship. Pairs with test-strategy for coverage gaps and api-design for contract changes.", Triggers: []string{"review", "pull request", "code quality"}},
	"test_strategy":   {Description: "Design a test strategy around the pyramid: unit, integration (testcontainers), contract and E2E, with behavior-focused, deterministic, order-free tests and coverage as a gap detector. Use when setting up testing for a new project or module, deciding which test type code needs, fixing a slow or flaky suite, or improving coverage that proves nothing.", Triggers: []string{"test", "testing", "coverage", "test plan"}},
	"refactor_safely": {Description: "Refactor with behavior preserved: green baseline, characterization tests for untested code, one tool-driven transformation at a time, commit every green step, revert instead of debugging. Use when extracting functions, renaming, removing duplication, restructuring modules via parallel change, or preparing code before a feature. Never mixes refactoring with behavior changes.", Triggers: []string{"refactor", "cleanup", "tech debt"}},
	"api_design":      {Description: "Design REST/gRPC APIs spec-first: resource-oriented URLs, precise methods and status codes, RFC 9457 Problem Details errors, day-one pagination and versioning, breaking-change discipline enforced with spectral/oasdiff/buf. Use when designing new endpoints, evolving request/response schemas, choosing a versioning approach, or reviewing an API for consistency.", Triggers: []string{"api", "endpoint", "rest", "contract"}},
	// Workflow
	"conventional_commit": {Description: "Write commit messages following Conventional Commits 1.0.0 — types, scopes, breaking-change markers and footer rules. Use when committing changes, preparing pull requests, or generating changelogs. Commit types feed version bumps — pairs with semantic-versioning.", Triggers: []string{"commit", "git commit", "conventional"}},
	"semantic_versioning": {Description: "Determine version bumps following Semantic Versioning 2.0.0 — increment rules from conventional commits, precedence, pre-release identifiers, and the release workflow. Use when releasing a version, tagging, or deciding MAJOR/MINOR/PATCH after a set of changes. Pairs with conventional-commit.", Triggers: []string{"version", "release", "tag", "semver"}},
	// Testing
	"test_foundational": {Description: "Evaluate and improve tests using Kent Beck's Test Desiderata — twelve trade-off properties of good developer tests. Use when writing new tests, reviewing test quality or suite health, deciding what kind of test to write, or resolving test-design disagreements with objective criteria. Pairs with test-tdd.", Triggers: []string{"test", "testing", "test quality", "desiderata"}},
	"test_tdd":          {Description: "Practice Test-Driven Development with Red-Green-Refactor discipline — Kent Beck's cycle and strategies (Fake It, Triangulation, Obvious Implementation), Uncle Bob's Three Laws, and the Transformation Priority Premise. Use when implementing features or fixes test-first, designing APIs through usage, or enforcing baby-steps discipline. Builds on test-foundational.", Triggers: []string{"tdd", "test driven", "red green refactor", "test first"}},
	"test_bdd":          {Description: "Practice Behavior-Driven Development as a collaboration method, not just a test tool — Discovery (Three Amigos, Example Mapping), Formulation (Gherkin), Automation (step definitions). Use when defining acceptance criteria, running example mapping, doing outside-in development, or building living documentation. For the mechanics of writing a single feature file, see bdd-scenario.", Triggers: []string{"bdd", "gherkin", "cucumber", "given when then", "acceptance test"}},
	// Lifecycle (ex-workflows, ADR-0015)
	"bug_fix":         {Description: "Drive a bug from report to verified fix: reproduce, write a failing regression test, diagnose the root cause, fix narrowly, and confirm no regressions. Use when fixing a reported bug, when you need a regression guard before shipping, or when checking whether a quick fix addresses the symptom or the actual cause.", Triggers: []string{"bug", "fix", "regression", "reproduce", "root cause"}},
	"release_cycle":   {Description: "Take a codebase through a traceable, reversible release: confirm readiness, pick the SemVer bump, update version references, generate the changelog, commit, tag, push, deploy, and verify. Use when cutting a release, bumping the version, tagging and deploying, or auditing whether a release followed a safe sequence.", Triggers: []string{"release", "version", "changelog", "tag", "deploy"}},
	"speckit_specify": {Description: "Capture the WHAT and WHY of a feature as a spec.md with prioritized user stories (P1/P2/P3), testable FR-XXX requirements, and measurable SC-XXX success criteria. Step 1 of the Spec-Kit lifecycle. Use when starting a new feature from a description, writing or re-scoping a spec, before planning.", Triggers: []string{"specify", "spec", "requirements", "user stories", "speckit"}},
	"speckit_clarify": {Description: "Resolve ambiguity in a feature spec by asking up to 5 targeted questions and encoding the answers back into spec.md. Quality gate between specify and plan. Use when a spec has [NEEDS CLARIFICATION] markers or vague requirements, before planning.", Triggers: []string{"clarify", "open questions", "ambiguous spec", "clarifying questions"}},
	"speckit_plan":    {Description: "Translate a clarified spec.md into plan.md — Technical Context, Constitution Check, project structure, and Phase 0/1 artifacts (research.md, data-model.md, contracts/, quickstart.md). Step 3 of the lifecycle. Use when a spec is ready and you need the technical design before tasks.", Triggers: []string{"plan", "technical design", "implementation plan", "architecture"}},
	"speckit_tasks":   {Description: "Decompose plan.md into an executable, dependency-ordered tasks.md organized by user story (T001 IDs, [P] parallel markers, [US1] labels, MVP-first phases with checkpoints). Step 4 of the lifecycle. Use when a plan exists and you need an actionable breakdown.", Triggers: []string{"tasks", "task breakdown", "task list", "decompose"}},
	"speckit_analyze": {Description: "Read-only cross-artifact consistency and quality check across spec.md, plan.md, and tasks.md — coverage gaps, duplications, ambiguities, contradictions, constitution conflicts. Final gate before implementation. Use when all three artifacts exist and you want to verify them before coding.", Triggers: []string{"analyze", "consistency", "cross-check", "ready to implement"}},
	"spec_propose":    {Description: "Plan an OpenSpec change before any code: create openspec/changes/<id>/ with a proposal, design, atomic tasks, and spec deltas using literal ### Requirement: / #### Scenario: GIVEN/WHEN/THEN syntax. Use when starting a feature or fix, drafting spec deltas, or beginning the propose→apply→archive lifecycle.", Triggers: []string{"propose", "openspec change", "proposal", "spec delta"}},
	"spec_apply":      {Description: "Implement an approved OpenSpec change: execute tasks.md atomically in order, write tests, verify each spec-delta scenario, and open a PR. Use when building an approved proposal, working a task list, checking spec compliance, or applying a change.", Triggers: []string{"apply", "implement change", "execute tasks", "spec compliance"}},
	"spec_archive":    {Description: "Finalize an applied OpenSpec change: consolidate deltas into the canonical openspec/specs/<capability>/spec.md, merge the branch, and move the change to openspec/changes/archive/YYYY-MM-DD-<id>/. Use when a PR is approved and mergeable, consolidating specs, or archiving a change.", Triggers: []string{"archive", "consolidate specs", "finalize change", "merge spec"}},
}

// GenerateFrontmatter genera YAML frontmatter para un skill según el ecosistema target.
func GenerateFrontmatter(guideName, target string) string {
	name := strings.ReplaceAll(guideName, "_", "-")
	meta, ok := SkillMetadata[guideName]
	if !ok {
		meta = SkillMeta{Description: fmt.Sprintf("Agent skill for %s", name)}
	}

	switch target {
	case "codex":
		return fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n", name, meta.Description)
	case "antigravity":
		var triggers strings.Builder
		for _, t := range meta.Triggers {
			triggers.WriteString(fmt.Sprintf("  - %s\n", t))
		}
		return fmt.Sprintf("---\nname: %s\ndescription: %s\ntriggers:\n%s---\n", name, meta.Description, triggers.String())
	default: // claude
		return fmt.Sprintf("---\nname: %s\ndescription: %s\nuser-invocable: true\n---\n", name, meta.Description)
	}
}

// Categories es el registro global de categorías de skills.
var Categories = []SkillCategory{
	{
		Name:      "architecture",
		Label:     "Architecture",
		Exclusive: true,
		Options: []SkillOption{
			{
				Name:        "neutral",
				Label:       "Neutral (Code review, testing, API design, refactoring) — recommended default",
				TemplateDir: "neutral",
				TemplateMapping: map[string]string{
					"code_review":     "code_review",
					"test_strategy":   "test_strategy",
					"refactor_safely": "refactor_safely",
					"api_design":      "api_design",
				},
			},
			{
				Name:        "clean-ddd",
				Label:       "Clean + DDD (DDD, BDD, CQRS, Hexagonal port skill)",
				TemplateDir: "clean-ddd",
				TemplateMapping: map[string]string{
					"ddd_entity":       "ddd_entity",
					"clean_arch_layer": "clean_arch_layer",
					"bdd_scenario":     "bdd_scenario",
					"cqrs_command":     "cqrs_command",
					"hexagonal_port":   "hexagonal_port",
				},
			},
			{
				Name:        "hexagonal",
				Label:       "Hexagonal (Ports & Adapters — lighter than clean-ddd)",
				TemplateDir: "hexagonal",
				TemplateMapping: map[string]string{
					"port_definition":      "port_definition",
					"adapter_pattern":      "adapter_pattern",
					"dependency_inversion": "dependency_inversion",
					"hex_integration_test": "hex_integration_test",
				},
			},
			{
				Name:        "event-driven",
				Label:       "Event-Driven (CQRS + Event Sourcing + Sagas)",
				TemplateDir: "event-driven",
				TemplateMapping: map[string]string{
					"command_handler":   "command_handler",
					"domain_event":      "domain_event",
					"event_projection":  "event_projection",
					"saga_orchestrator": "saga_orchestrator",
					"event_idempotency": "event_idempotency",
				},
			},
		},
	},
	{
		Name:      "testing",
		Label:     "Testing",
		Exclusive: true,
		Options: []SkillOption{
			{
				Name:        "foundational",
				Label:       "Foundational (Test Desiderata — properties of good tests)",
				TemplateDir: "testing",
				TemplateMapping: map[string]string{
					"test_foundational": "test_foundational",
				},
			},
			{
				Name:        "tdd",
				Label:       "TDD (Test-Driven Development — includes foundational)",
				TemplateDir: "testing",
				TemplateMapping: map[string]string{
					"test_tdd": "test_tdd",
				},
			},
			{
				Name:        "bdd",
				Label:       "BDD (Behavior-Driven Development — includes foundational)",
				TemplateDir: "testing",
				TemplateMapping: map[string]string{
					"test_bdd": "test_bdd",
				},
			},
		},
	},
	{
		Name:      "conventions",
		Label:     "Conventions",
		Exclusive: false,
		Options: []SkillOption{
			{
				Name:        "conventional-commit",
				Label:       "Conventional Commits",
				TemplateDir: "conventions",
				TemplateMapping: map[string]string{
					"conventional_commit": "conventional_commit",
				},
			},
			{
				Name:        "semantic-versioning",
				Label:       "Semantic Versioning",
				TemplateDir: "conventions",
				TemplateMapping: map[string]string{
					"semantic_versioning": "semantic_versioning",
				},
			},
		},
	},
	{
		// Lifecycle skills — the recipes that used to be `workflows` (ADR-0015).
		// bug-fix and release-cycle are plain skills; spec-kit and openspec are
		// the SDD lifecycle bundles (one per standard — no SDDAware magic; the
		// user installs the bundle matching their --sdd-standard).
		Name:      "lifecycle",
		Label:     "Lifecycle",
		Exclusive: false,
		Options: []SkillOption{
			{
				Name:        "bug-fix",
				Label:       "Bug Fix (reproduce → diagnose → fix → test → PR)",
				TemplateDir: "lifecycle",
				TemplateMapping: map[string]string{
					"bug_fix": "bug_fix",
				},
			},
			{
				Name:        "release-cycle",
				Label:       "Release Cycle (version → changelog → tag → deploy)",
				TemplateDir: "lifecycle",
				TemplateMapping: map[string]string{
					"release_cycle": "release_cycle",
				},
			},
			{
				Name:        "spec-kit",
				Label:       "Spec-Kit SDD lifecycle (specify → clarify → plan → tasks → analyze)",
				TemplateDir: "lifecycle-spec-kit",
				TemplateMapping: map[string]string{
					"speckit_specify": "speckit_specify",
					"speckit_clarify": "speckit_clarify",
					"speckit_plan":    "speckit_plan",
					"speckit_tasks":   "speckit_tasks",
					"speckit_analyze": "speckit_analyze",
				},
			},
			{
				Name:        "openspec",
				Label:       "OpenSpec SDD lifecycle (propose → apply → archive)",
				TemplateDir: "lifecycle-openspec",
				TemplateMapping: map[string]string{
					"spec_propose": "spec_propose",
					"spec_apply":   "spec_apply",
					"spec_archive": "spec_archive",
				},
			},
		},
	},
}

// CategoryNames devuelve los nombres de todas las categorías registradas.
func CategoryNames() []string {
	names := make([]string, len(Categories))
	for i, c := range Categories {
		names[i] = c.Name
	}
	return names
}

// AllSkillPresetNames devuelve los nombres de todos los presets registrados en
// todas las categorías de skills, mas el alias "all" para las categorías que
// permiten selección compuesta. Util para validación con enums (MCP).
func AllSkillPresetNames() []string {
	seen := map[string]bool{"all": true}
	names := []string{"all"}
	for _, c := range Categories {
		for _, o := range c.Options {
			if seen[o.Name] {
				continue
			}
			seen[o.Name] = true
			names = append(names, o.Name)
		}
	}
	return names
}

// FindCategory busca una categoría por nombre.
func FindCategory(name string) (*SkillCategory, error) {
	for i := range Categories {
		if Categories[i].Name == name {
			return &Categories[i], nil
		}
	}
	return nil, fmt.Errorf("unknown category: %s", name)
}

// Resolve resuelve la selección de una sub-opción (o "all") dentro de la
// categoría usando los campos estáticos TemplateDir/TemplateMapping.
func (c *SkillCategory) Resolve(preset string) (*ResolvedSelection, error) {
	if preset == "all" {
		if c.Exclusive {
			return nil, fmt.Errorf("category %q does not support 'all' (options are mutually exclusive)", c.Name)
		}
		return c.resolveAll(), nil
	}

	for _, opt := range c.Options {
		if opt.Name == preset {
			return &ResolvedSelection{TemplateDir: opt.TemplateDir, TemplateMapping: opt.TemplateMapping}, nil
		}
	}
	return nil, fmt.Errorf("unknown preset %q in category %q", preset, c.Name)
}

// resolveAll combina todas las opciones de la categoría en una sola selección.
func (c *SkillCategory) resolveAll() *ResolvedSelection {
	merged := make(map[string]string)
	var dir string
	for _, opt := range c.Options {
		dir = opt.TemplateDir
		maps.Copy(merged, opt.TemplateMapping)
	}
	return &ResolvedSelection{
		TemplateDir:     dir,
		TemplateMapping: merged,
	}
}

// OptionNames devuelve los nombres de las sub-opciones de la categoría.
func (c *SkillCategory) OptionNames() []string {
	names := make([]string, len(c.Options))
	for i, o := range c.Options {
		names[i] = o.Name
	}
	return names
}

// OptionLabels devuelve los labels de las sub-opciones (para el menú interactivo).
func (c *SkillCategory) OptionLabels() []string {
	labels := make([]string, len(c.Options))
	for i, o := range c.Options {
		labels[i] = o.Label
	}
	return labels
}

// LegacyPresetMapping mapea presets legados (--preset flag del comando skills)
// al nuevo modelo de category+option. La entrada "default" fue removida en
// v2.0 per ADR-001; "clean" se mantiene como alias del rename hecho en v1.21
// (clean → clean-ddd) ya que es un cambio terminológico, no un default.
var LegacyPresetMapping = map[string][2]string{
	"clean":        {"architecture", "clean-ddd"},
	"clean-ddd":    {"architecture", "clean-ddd"},
	"hexagonal":    {"architecture", "hexagonal"},
	"event-driven": {"architecture", "event-driven"},
	"neutral":      {"architecture", "neutral"},
	"workflow":     {"conventions", "all"},
	"conventions":  {"conventions", "all"},
}
