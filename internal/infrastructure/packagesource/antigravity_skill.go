package packagesource

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jorelcb/codify-og/internal/domain/catalog"
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

// Fetch re-renders the skill with Antigravity frontmatter. The body comes
// from the same embedded skill directory the Claude path uses; only the
// frontmatter differs (per-ecosystem).
//
// Antigravity installs skills as a FLAT <id>.md (no directory), so the
// progressive-disclosure sidecars (reference.md, examples.md) cannot ship as
// separate files. They are INLINED after the body, in deterministic order,
// with a provenance marker — full content preserved, no dangling
// "see reference.md" pointers, and no silently dropped files (SK-4).
func (s *AntigravitySkillSource) Fetch(_ context.Context, m catalog.PackageManifest) (catalog.PackageContent, error) {
	if m.Target != catalog.TargetAntigravitySkill {
		return catalog.PackageContent{}, fmt.Errorf("antigravity source: target %q not supported (antigravity-skill only)", m.Target)
	}
	guideName, files, err := s.inner.skillFiles(m)
	if err != nil {
		return catalog.PackageContent{}, err
	}

	var sb strings.Builder
	sb.WriteString(catalog.GenerateFrontmatter(guideName, "antigravity"))
	sb.WriteString("\n")
	sb.Write(files["SKILL.md"])

	sidecars := make([]string, 0, len(files)-1)
	for name := range files {
		if name != "SKILL.md" {
			sidecars = append(sidecars, name)
		}
	}
	sort.Strings(sidecars)
	for _, name := range sidecars {
		fmt.Fprintf(&sb, "\n\n<!-- inlined from %s (flat Antigravity layout) -->\n\n", name)
		sb.Write(files[name])
	}

	return catalog.PackageContent{
		Files: map[string][]byte{"SKILL.md": []byte(sb.String())},
	}, nil
}
