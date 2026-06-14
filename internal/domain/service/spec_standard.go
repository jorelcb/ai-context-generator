package service

// SpecStandard abstrae el formato de SDD (Spec-Driven Development) que codify
// produce. Diferentes estándares (OpenSpec, GitHub Spec-Kit, custom internos)
// difieren en file names, layout en disco, y workflows del lifecycle.
//
// Ver ADR-0011 para razones, alternatives, y consequences.
//
// Adapters concretos viven en internal/infrastructure/sdd/. La selección del
// estándar activo se resuelve por precedencia: flag CLI > project config >
// user config > built-in default (OpenSpec).
type SpecStandard interface {
	// ID returns a stable identifier used in flags and config (e.g.,
	// "openspec", "spec-kit"). Lowercase, no spaces.
	ID() string

	// DisplayName returns a human-readable name for prompts and CLI output.
	DisplayName() string

	// BootstrapArtifacts returns the artifacts that `codify spec` produces.
	// The slice order also drives the order of file generation and listing.
	BootstrapArtifacts() []SpecArtifact

	// TemplateDir returns the directory name under templates/{locale}/sdd/
	// where this standard's templates live (e.g., "openspec", "spec-kit").
	// The full path is templates/{locale}/sdd/{TemplateDir()}/{kind}/...
	// where kind is "spec" or "workflows".
	TemplateDir() string

	// SystemPromptHints returns standard-specific instructions appended to
	// the base spec system prompt. May vary by locale for consistent tone.
	// OpenSpec returns delta-format reminders; Spec-Kit returns per-feature
	// directory conventions; etc.
	SystemPromptHints(locale string) string

	// LifecycleWorkflowIDs returns the workflow guide IDs that implement
	// this standard's lifecycle. Consumed by the workflows command when the
	// user installs the spec-driven-change preset to know which workflow
	// templates to ship for the active standard.
	//
	// OpenSpec → ["spec_propose", "spec_apply", "spec_archive"].
	// Spec-Kit → ["speckit_specify", "speckit_plan", "speckit_tasks"].
	LifecycleWorkflowIDs() []string
}

// SpecArtifact describes one file that `codify spec` generates. Dir + FileName
// give each artifact its full on-disk location, so a single per-standard slice
// expresses any layout the upstream tool uses without a fixed enum:
//   - Spec-Kit:  Dir "specs/{feature}", FileName "spec.md"
//   - Spec-Kit constitution: Dir ".specify/memory", FileName "constitution.md"
//   - OpenSpec:  Dir "openspec/specs/{feature}", FileName "spec.md"; project
//     doc Dir "openspec", FileName "project.md"
type SpecArtifact struct {
	// GuideName matches the template guide identifier used by the prompt
	// builder and the template file name (e.g., "speckit_spec", "openspec_spec").
	GuideName string

	// FileName is the on-disk file name (e.g., "spec.md", "constitution.md").
	FileName string

	// Dir is the artifact's directory relative to the spec output root. It may
	// contain the token "{feature}", expanded to the feature/capability slug at
	// generation time. Empty Dir means the output root itself.
	Dir string

	// Required indicates whether the spec command must generate this file.
	// Optional artifacts (Spec-Kit's research.md, etc.) may be skipped.
	Required bool

	// SkipIfExists, when true, leaves an already-present file untouched instead
	// of regenerating it. Used for project-level artifacts that must survive
	// per-feature re-runs (e.g. Spec-Kit's .specify/memory/constitution.md).
	SkipIfExists bool
}

// FeatureToken is the placeholder inside SpecArtifact.Dir replaced with the
// feature/capability slug when resolving the on-disk path.
const FeatureToken = "{feature}"
