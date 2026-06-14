package dto

import (
	"github.com/jorelcb/codify/internal/domain/service"
	"github.com/jorelcb/codify/internal/domain/shared"
)

// SpecConfig holds configuration for generating SDD specifications.
//
// Each artifact carries its own directory (SpecArtifact.Dir, possibly with the
// {feature} token); FeatureID supplies the slug that fills that token, so a
// single field expresses every standard's layout (Spec-Kit's specs/<feature>/,
// OpenSpec's openspec/specs/<capability>/, etc.). StandardID is persisted so
// logs and downstream consumers know which adapter produced the files.
type SpecConfig struct {
	ProjectName     string
	FromContextPath string // path to existing output directory (contains AGENTS.md and context/)
	OutputPath      string
	Model           string
	Locale          string

	// FeatureID is the slug substituted for the {feature} token in each
	// artifact's directory. The caller typically derives it from the
	// projectName (a feature/capability slug).
	FeatureID string

	// StandardID identifies the active SpecStandard (e.g., "spec-kit",
	// "openspec"). Useful for logs and downstream validations.
	StandardID string

	// StandardHints is the block the active SpecStandard appends to the spec
	// system prompt to reinforce its conventions (literal requirement syntax,
	// file-naming rules, etc.).
	StandardHints string

	// Artifacts are the active standard's bootstrap artifacts (with their Dir
	// and SkipIfExists). The command places each generated file at
	// OutputPath/<Dir with {feature} expanded>/<FileName>.
	Artifacts []service.SpecArtifact
}

// Validate validates the spec configuration
func (sc *SpecConfig) Validate() error {
	if sc.ProjectName == "" {
		return shared.ErrInvalidInput("project name is required")
	}
	if sc.FromContextPath == "" {
		return shared.ErrInvalidInput("from-context path is required")
	}
	if sc.OutputPath == "" {
		return shared.ErrInvalidInput("output path is required")
	}
	if sc.FeatureID == "" {
		return shared.ErrInvalidInput("feature id is required")
	}
	return nil
}
