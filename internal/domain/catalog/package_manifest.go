package catalog

import (
	"context"
	"encoding/json"
)

// PackageManifest define el formato unificado interno para cualquier
// installable del catálogo de codify (Claude skill, Claude hook, Gemini
// extension, Antigravity workflow, SDD standard declarativo, etc.).
//
// Ver ADR-0010 §7 para razones, alternatives y consequences. Refinado
// durante el spike D.0 (2026-05-08) a partir del sketch original — este
// archivo encarna esos refinements.
//
// Decisiones encarnadas acá:
//
//   - **Atómico**: un manifest = un primitivo del ecosystem destino. No
//     compound packages (ADR-0010 decisión 2).
//   - **No localización**: el manifest no tiene Locale field. Aliniado con
//     la convención de registries (npm/PyPI/brew/Anthropic plugins). Si una
//     skill viene "talegada" con un idioma, eso es contenido del package,
//     no una dimensión del manifest.
//   - **No InstallPath**: el path se deriva de (Target, Scope) por el
//     TargetInstaller. El manifest no carga la receta de instalación.
//   - **Metadata target-namespaced**: campos típicos por ecosystem (Claude
//     triggers/allowedTools/userInvocable, Gemini, SDD declarativo) viven
//     en sub-structs opcionales. `Metadata map[string]string` queda como
//     escape hatch para extensiones scalar puntuales.
//
// El package no define implementaciones — solo tipos + interfaces. Los
// adapters concretos (EmbeddedSource, ClaudeInstaller, etc.) viven en
// internal/infrastructure/packagesource/ y internal/infrastructure/packageinstall/.

// Target identifies the kind of primitive a package represents within an
// ecosystem. Selects the TargetInstaller responsible for installing the
// package. Open-ended via string + named constants — third-party adapters
// can register new Target values (validated at install time by the
// installer registry).
type Target string

const (
	// Content packages — files installed under the ecosystem's conventions.

	// TargetClaudeSkill: a single SKILL.md installed under .claude/skills/<id>/.
	TargetClaudeSkill Target = "claude-skill"
	// TargetClaudeHook: a script bundle plus a settings.json fragment merged
	// into .claude/settings.json.
	TargetClaudeHook Target = "claude-hook"
	// TargetClaudePlugin: the broader Claude Code plugin format that may
	// bundle skills + hooks + slash commands.
	TargetClaudePlugin Target = "claude-plugin"
	// TargetClaudeSlash: a slash command installed under .claude/commands/.
	TargetClaudeSlash Target = "claude-slash-command"
	// TargetAntigravitySkill: an Agent-Skills markdown file installed flat
	// under ~/.gemini/antigravity-cli/skills/<id>.md (global) or
	// <project>/.agents/skills/<id>.md (workspace). Written directly by the
	// AntigravityInstaller — no CLI delegation. See ADR-0012 §5.
	TargetAntigravitySkill Target = "antigravity-skill"
	// TargetAntigravityWF: an Antigravity workflow (flat .md with execution
	// annotations). Workflows are exempt from the catalog UX (ADR-0010 §5)
	// but remain installable via flags.
	//
	// Note: there is no Gemini CLI target. Gemini CLI is being deprecated in
	// favor of Antigravity CLI (ADR-0012 §4); codify aligns early and does
	// not model gemini-extension packages. (The Gemini *model* API, used for
	// generation, is unaffected and lives in infrastructure/llm.)
	TargetAntigravityWF Target = "antigravity-workflow"

	// Behavior packages — declarative metadata that a generic adapter
	// consumes at runtime to register new behaviors. The first instance
	// is SDD standards (OpenSpec, Spec-Kit, future third-party standards).

	// TargetCodifySDDStandard: a declarative SpecStandard adapter spec.
	// The runtime SDD registry consumes the SDDMetadata sub-struct (and
	// optionally additional template files in PackageContent) to construct
	// a service.SpecStandard implementation without Go code execution.
	TargetCodifySDDStandard Target = "codify-sdd-standard"
)

// Scope determines whether a package installs at workstation level
// (~/.claude/skills/, ~/.gemini/extensions/, ~/.codify/) or project level
// (.claude/skills/, .gemini/extensions/, .codify/). Resolved by the
// TargetInstaller at install time using the manifest's Target.
type Scope string

const (
	// ScopeWorkstation installs into the user's home tree, shared across
	// all projects. Used by codify config + codify catalog standalone.
	ScopeWorkstation Scope = "workstation"
	// ScopeProject installs into the current project's tree, versioned with
	// the repo. Used by codify init + codify catalog (project mode).
	ScopeProject Scope = "project"
)

// MetaKeyMarketplace is the PackageManifest.Metadata key carrying the
// marketplace name a plugin belongs to. Written by a plugin marketplace
// source and read by the plugin installer to address `<id>@<marketplace>`.
// Shared contract between the two infrastructure adapters.
const MetaKeyMarketplace = "marketplace"

// PackageRef points to another package by ID and optional version.
// Used in Dependencies and Conflicts to express relationships between
// packages without requiring the referenced package to be resolved at
// declaration time.
type PackageRef struct {
	ID      string
	Version string // empty = any version
}

// SourceRef identifies where the package was sourced from. The Kind value
// is also surfaced as a badge in the catalog UI per ADR-0010 §8 + D.0
// spike Gap #8 ("[embedded]", "[git]", "[konfio-private]").
type SourceRef struct {
	// Kind is a stable identifier for the source type. Values match the
	// PackageSource implementations: "embedded", "local-fs", "git",
	// "http-registry". Third-party sources can declare custom kinds.
	Kind string
	// URI carries the source-specific locator. Empty for "embedded";
	// directory path for "local-fs"; Git URL for "git"; base HTTP URL for
	// "http-registry".
	URI string
}

// PackageManifest is the unified internal format for any installable in
// codify's catalog. Atomic (one manifest = one primitive). Localization
// is intentionally NOT a manifest dimension. See package-level docs for
// the full rationale.
type PackageManifest struct {
	// Identity

	// ID is the machine identifier (kebab-case, e.g., "ddd-entity"). Used
	// as the install directory name for content packages and as the lookup
	// key throughout codify.
	ID string
	// Label is the human display label (e.g., "DDD Entity"). Shown in the
	// catalog UI tree rows.
	Label string
	// Version is a semver-ish string. Concrete semantics depend on Source:
	//   - embedded: derived from codify binary version (.version file)
	//   - local-fs / git: declared in the source's manifest file
	//   - http-registry: returned by the registry API
	// D.2 specifies the per-source contract.
	Version string

	// Description is a one- or two-sentence summary shown in the catalog
	// UI detail view. Should be ≤250 chars (Antigravity constraint mirrored
	// here as a soft cap).
	Description string

	// Target identifies which TargetInstaller handles this package.
	Target Target

	// Source identifies the origin of this manifest. Source.Kind is
	// surfaced as a UI badge.
	Source SourceRef

	// Relationships

	// Dependencies are packages required for this one to function. The
	// catalog UI auto-marks dependencies when the user selects a package.
	// Resolution semantics (transitive closure, version pinning) live in
	// D.4+.
	Dependencies []PackageRef
	// Conflicts are packages incompatible with this one. The catalog UI
	// shows a warning and refuses to mark both for install in the same
	// session.
	Conflicts []PackageRef

	// Verification

	// SourceChecksum is a hash of the source declaration (the manifest
	// content + any template/declarative bytes the source ships with it).
	// Used to detect changes between catalog refreshes. The hash of the
	// rendered file as installed on disk lives in the lockfile (D.9), not
	// here.
	SourceChecksum string

	// Target-namespaced typed metadata. Exactly one of these is populated,
	// determined by Target. The catalog UI uses these for search, filter,
	// detail view, and conflict detection without fetching content.

	// Claude is populated for any claude-* Target.
	Claude *ClaudeMetadata
	// SDD is populated for TargetCodifySDDStandard. Carries the declarative
	// spec the runtime registry consumes to register a new SpecStandard.
	SDD *SDDMetadata

	// Metadata is a forward-compat scalar escape hatch for Target-specific
	// extensions that are not common enough to warrant a typed sub-struct.
	// Prefer typed sub-structs whenever the field is shared across multiple
	// packages of the same Target.
	Metadata map[string]string
}

// ClaudeMetadata carries Claude-specific structured metadata. Populated
// when Target is one of the claude-* values. Different Claude targets use
// different subsets of these fields; a TargetClaudeSkill package may set
// Triggers/AllowedTools/UserInvocable but not HookEvents, while a
// TargetClaudeHook package sets HookEvents but not the others.
type ClaudeMetadata struct {
	// Triggers are keyword routing hints used by some Claude agents to
	// pick a skill from natural-language input. Optional.
	Triggers []string
	// AllowedTools maps to the Claude skill frontmatter "allowed-tools"
	// field, restricting what tools the skill can invoke. Optional.
	AllowedTools []string
	// UserInvocable maps to the Claude skill frontmatter "user-invocable"
	// field, exposing the skill as a slash command.
	UserInvocable bool

	// HookEvents (only for TargetClaudeHook) lists the lifecycle events
	// the hook bundle reacts to: PreToolUse, PostToolUse, etc.
	HookEvents []string
}

// SDDMetadata is the declarative spec for behavior packages of type
// TargetCodifySDDStandard. The runtime SDD registry consumes this to
// construct a service.SpecStandard adapter without Go code execution.
type SDDMetadata struct {
	// BootstrapArtifacts mirrors service.SpecStandard.BootstrapArtifacts()
	// in declarative form.
	BootstrapArtifacts []SpecArtifactDescriptor
	// LifecycleWorkflowIDs mirrors service.SpecStandard.LifecycleWorkflowIDs()
	// in declarative form.
	LifecycleWorkflowIDs []string
	// OutputLayout is "flat" or "feature-grouped".
	OutputLayout string
	// TemplateDir is the directory name under templates/sdd/ where this
	// standard's templates live.
	TemplateDir string
	// SystemPromptHints carries the per-locale prompt addendum the standard
	// adds to the spec system prompt. Map keys are ISO locale codes ("en",
	// "es"). NOT package localization — this is prompt content for the LLM.
	SystemPromptHints map[string]string
}

// SpecArtifactDescriptor mirrors service.SpecArtifact in declarative form.
// Kept separate from service.SpecArtifact so the manifest can evolve
// independently (e.g., add JSON tags) without churning the domain port.
type SpecArtifactDescriptor struct {
	GuideName string
	FileName  string
	Required  bool
}

// PackageContent is the resolved payload for a package — what the
// TargetInstaller writes to disk (or, for behavior packages, registers in
// memory). Different targets carry different shapes:
//
//   - claude-skill: Files has one entry, the rendered SKILL.md.
//   - claude-hook: Files has multiple entries (scripts) and SettingsFragment
//     carries the JSON fragment to merge.
//   - codify-sdd-standard: Files may carry templates; the bulk of the spec
//     lives in the manifest's SDDMetadata sub-struct.
//
// The TargetInstaller for the package's Target knows how to interpret
// Content.
type PackageContent struct {
	// Files maps relative paths to file bytes. For single-file installs
	// (skills), Files has one entry; for bundles (hooks), multiple.
	Files map[string][]byte

	// SettingsFragment is an optional JSON object to merge into the
	// ecosystem's settings file (~/.claude/settings.json for Claude hooks).
	// Nil for packages that don't touch settings. Use json.RawMessage to
	// preserve the source's JSON shape verbatim and avoid re-marshaling.
	SettingsFragment json.RawMessage
}

// PackageSource enumerates and fetches packages from a single origin
// (embedded catalog, local directory, Git repo, HTTP registry). See
// ADR-0010 §8.
//
// Implementations live in internal/infrastructure/packagesource/. The
// catalog command composes multiple sources at runtime via a registry,
// applying ordering and deduplication.
//
// FROZEN CONTRACT (ratified post-D.1, before D.2/D.3). This signature is
// the stable contract every adapter implements; reconciles two deliberate
// deviations from the ADR-0010 §8 sketch — do not "restore" the sketch:
//
//   - Fetch takes the full PackageManifest, NOT (id, version). The caller
//     always holds the manifest from a prior List, so the source needs no
//     re-lookup, and per-source versioning falls out naturally (the
//     manifest already carries Version + Source.URI). Richer input than
//     coordinates, and strictly more capable for remote/versioned sources.
//   - Kind() string instead of Capabilities() SourceCapabilities. The only
//     near-term consumer (the catalog command, D.4) badges provenance and
//     distinguishes offline vs network sources — both derivable from Kind
//     via a pure helper. A SourceCapabilities struct is deferred until a
//     real consumer needs it (marketplace/auth, D.10) rather than shipped
//     speculatively.
//
// Each adapter must add a compile-time `var _ catalog.PackageSource`
// assertion so signature drift fails the build instead of surfacing later.
type PackageSource interface {
	// Kind returns a stable identifier for the source type ("embedded",
	// "local-fs", "git", "http-registry"). Used by the catalog UI to badge
	// each manifest with its provenance and by the registry to identify
	// the source in user-facing diagnostics.
	Kind() string

	// List returns all manifests this source knows about. Filtering by
	// Target (for the active ecosystem tab) happens at the catalog level,
	// not here — sources return everything they have.
	List(ctx context.Context) ([]PackageManifest, error)

	// Fetch resolves the content payload for a specific package. Caller
	// already has the manifest from a prior List call; Fetch produces the
	// bytes (or declarative spec) needed to install.
	Fetch(ctx context.Context, m PackageManifest) (PackageContent, error)
}

// TargetInstaller knows how to install/uninstall packages onto a Scope for
// one ecosystem. See ADR-0010 §9.
//
// The install recipe lives here, not in the manifest — hook bundles copy
// multiple files and merge settings.json; skills write a single SKILL.md;
// SDD standards register a runtime adapter. Each TargetInstaller
// implementation encodes the conventions for its ecosystem.
//
// Ratified for D.3 as PER-ECOSYSTEM, not per-Target: one installer
// (ClaudeInstaller, GeminiInstaller, AntigravityInstaller) owns all the
// Targets of its ecosystem, since they share path roots (~/.claude/, etc.)
// and settings mechanics. Hence Handles(Target) bool rather than the
// per-Target Target() of the ADR sketch — the installer registry routes a
// manifest to the first installer that Handles its Target.
type TargetInstaller interface {
	// Handles reports whether this installer can install the given Target.
	// The catalog command's installer registry routes packages by asking
	// each registered installer.
	Handles(t Target) bool

	// Install writes the package to the appropriate filesystem location
	// under the given Scope. For behavior packages (SDD standards), this
	// also registers the runtime adapter.
	Install(ctx context.Context, m PackageManifest, c PackageContent, s Scope) error

	// Uninstall removes the package from the given Scope. Idempotent —
	// must succeed if the package is not present (no-op).
	Uninstall(ctx context.Context, m PackageManifest, s Scope) error

	// InstalledList enumerates packages of this Target currently present
	// in the given Scope. Used by the catalog UI to render already-installed
	// state and by the lockfile (D.9) to compute install/uninstall plans.
	InstalledList(ctx context.Context, s Scope) ([]InstalledPackage, error)
}

// InstalledPackage tracks a package that's currently on disk in a given
// scope. Distinct from PackageManifest — InstalledPackage describes what
// is, not what could be.
type InstalledPackage struct {
	// ID matches the manifest ID at install time.
	ID string
	// Version captures the package version as installed. May lag the
	// manifest's current Version if the catalog has been refreshed but
	// the user hasn't reinstalled.
	Version string
	// Target identifies the kind of primitive (informational; the disk
	// location is determined by Target + Scope at install time).
	Target Target
	// Scope identifies where the package was installed.
	Scope Scope
	// SourceURI is the source-specific locator the package came from (the
	// marketplace ref for plugins, the directory for local-fs; empty for
	// embedded). Provenance carried by the lockfile reader so re-apply (sync)
	// can rebuild the originating source. Live installers leave it empty —
	// they enumerate what's on disk, not where it came from.
	SourceURI string
	// InstalledAt is an ISO 8601 timestamp captured at install time.
	InstalledAt string
	// InstalledChecksum is a hash of the file(s) as installed. Differs
	// from PackageManifest.SourceChecksum: this captures post-render state
	// (frontmatter generated, settings merged, etc.) for drift detection
	// against hand-edits.
	InstalledChecksum string
}
