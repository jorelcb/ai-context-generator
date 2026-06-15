package dto

import "github.com/jorelcb/codify/internal/domain/shared"

// ValidTargets maps valid target ecosystem names.
var ValidTargets = map[string]bool{
	"claude":      true,
	"codex":       true,
	"antigravity": true,
}

// Skills install scopes
const (
	InstallScopeGlobal  = "global"
	InstallScopeProject = "project"
)

// SkillsConfig holds configuration for delivering static Agent Skills.
type SkillsConfig struct {
	Category   string // "architecture", "testing", "conventions"
	Preset     string // "clean-ddd", "neutral", "conventional-commit", "all", etc.
	Locale     string // "en" or "es"
	Target     string // target ecosystem: "claude", "codex", "antigravity"
	OutputPath string
	Install    string // install scope: "global", "project", or "" (custom output)
}

// Validate validates the skills configuration
func (sc *SkillsConfig) Validate() error {
	if sc.OutputPath == "" {
		return shared.ErrInvalidInput("output path is required")
	}
	if !ValidTargets[sc.Target] {
		return shared.ErrInvalidInput("invalid target: must be claude, codex, or antigravity")
	}
	return nil
}
