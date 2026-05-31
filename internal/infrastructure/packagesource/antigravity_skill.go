package packagesource

import (
	"context"
	"fmt"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// Compile-time guard: AntigravitySkillSource satisfies catalog.PackageSource.
var _ catalog.PackageSource = (*AntigravitySkillSource)(nil)

// AntigravitySkillSource re-frames the built-in skill catalog for the
// Antigravity ecosystem (ADR-0012 §5). It decorates an EmbeddedSource: the
// skill bodies are ecosystem-agnostic, so only the Target and the frontmatter
// differ. List exposes the skills as `antigravity-skill` packages; Fetch
// re-renders each with Antigravity (Agent Skills) frontmatter.
//
// Antigravity hooks/plugins are out of scope (ADR-0012 §5: agy's plugin
// marketplace model is immature), so this source emits skills only — the
// EmbeddedSource (Claude) is left untouched.
type AntigravitySkillSource struct {
	inner *EmbeddedSource
}

// NewAntigravitySkillSource wraps an embedded source to serve Antigravity
// skills.
func NewAntigravitySkillSource(inner *EmbeddedSource) *AntigravitySkillSource {
	return &AntigravitySkillSource{inner: inner}
}

// Kind badges these packages distinctly from the Claude embedded source.
func (s *AntigravitySkillSource) Kind() string { return "embedded-antigravity" }

// List returns the built-in skills re-stamped as antigravity-skill packages.
// Hooks (and any non-skill manifest) from the inner source are dropped.
func (s *AntigravitySkillSource) List(ctx context.Context) ([]catalog.PackageManifest, error) {
	inner, err := s.inner.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]catalog.PackageManifest, 0, len(inner))
	for _, m := range inner {
		if m.Target != catalog.TargetClaudeSkill {
			continue
		}
		m.Target = catalog.TargetAntigravitySkill
		m.Source = catalog.SourceRef{Kind: s.Kind(), URI: ""}
		m.SourceChecksum = computeManifestChecksum(m)
		out = append(out, m)
	}
	return out, nil
}

// Fetch re-renders the skill with Antigravity frontmatter. The body comes from
// the same embedded template the Claude path uses; only the frontmatter
// differs (per-ecosystem), so the content is produced here rather than baked
// into the source.
func (s *AntigravitySkillSource) Fetch(_ context.Context, m catalog.PackageManifest) (catalog.PackageContent, error) {
	if m.Target != catalog.TargetAntigravitySkill {
		return catalog.PackageContent{}, fmt.Errorf("antigravity source: target %q not supported (antigravity-skill only)", m.Target)
	}
	guideName, body, err := s.inner.skillTemplate(m)
	if err != nil {
		return catalog.PackageContent{}, err
	}
	frontmatter := catalog.GenerateFrontmatter(guideName, "antigravity")
	content := frontmatter + "\n" + string(body)
	return catalog.PackageContent{
		Files: map[string][]byte{"SKILL.md": []byte(content)},
	}, nil
}
