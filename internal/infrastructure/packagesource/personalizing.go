package packagesource

import (
	"context"
	"fmt"

	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/domain/service"
)

// Compile-time guard: PersonalizingSource satisfies the frozen
// catalog.PackageSource contract.
var _ catalog.PackageSource = (*PersonalizingSource)(nil)

// PersonalizingSource decorates an EmbeddedSource so that skill packages are
// LLM-adapted to a project before install, instead of shipped as the static
// template. It is the personalization path of `codify catalog` expressed as
// just another source: List/Kind delegate to the inner source, and only
// Fetch differs — for skills it runs the LLM over the raw template; for hooks
// (and any non-skill target) it delegates verbatim, since those have no
// personalized form.
//
// Keeping personalization behind the PackageSource port means CatalogService
// and the TargetInstaller stay untouched — the caller swaps the source based
// on the requested mode and everything downstream is identical.
type PersonalizingSource struct {
	inner          *EmbeddedSource
	provider       service.LLMProvider
	projectContext string
	locale         string
	target         string // ecosystem target for the skills prompt ("claude")
}

// NewPersonalizingSource wraps inner with an LLM provider and the project
// context the adaptation is conditioned on. projectContext must be non-empty;
// Fetch errors otherwise rather than producing a generic (pointless) result.
func NewPersonalizingSource(inner *EmbeddedSource, provider service.LLMProvider, projectContext, locale, target string) *PersonalizingSource {
	if locale == "" {
		locale = "en"
	}
	if target == "" {
		target = "claude"
	}
	return &PersonalizingSource{
		inner:          inner,
		provider:       provider,
		projectContext: projectContext,
		locale:         locale,
		target:         target,
	}
}

// Kind reports the inner source kind suffixed to signal personalization, so
// the catalog UI can badge adapted packages distinctly from static ones.
func (p *PersonalizingSource) Kind() string { return p.inner.Kind() + "+llm" }

// List delegates to the inner source — the set of available packages is the
// same; only how a skill's content is produced on Fetch changes.
func (p *PersonalizingSource) List(ctx context.Context) ([]catalog.PackageManifest, error) {
	return p.inner.List(ctx)
}

// Fetch returns LLM-adapted content for skills and delegates everything else
// to the inner source.
func (p *PersonalizingSource) Fetch(ctx context.Context, m catalog.PackageManifest) (catalog.PackageContent, error) {
	if m.Target != catalog.TargetClaudeSkill {
		// Hooks and other targets have no personalized form — ship the
		// static content unchanged.
		return p.inner.Fetch(ctx, m)
	}
	if p.projectContext == "" {
		return catalog.PackageContent{}, fmt.Errorf("personalizing source: project context is required to adapt skill %q", m.ID)
	}

	guideName, body, err := p.inner.skillTemplate(m)
	if err != nil {
		return catalog.PackageContent{}, err
	}

	req := service.GenerationRequest{
		TemplateGuides: []service.TemplateGuide{{Name: guideName, Content: string(body)}},
		Mode:           "skills",
		Target:         p.target,
		Locale:         p.locale,
		ProjectContext: p.projectContext,
	}
	resp, err := p.provider.GenerateContext(ctx, req)
	if err != nil {
		return catalog.PackageContent{}, fmt.Errorf("personalizing source: adapt skill %q: %w", m.ID, err)
	}
	if len(resp.Files) == 0 {
		return catalog.PackageContent{}, fmt.Errorf("personalizing source: LLM returned no content for skill %q", m.ID)
	}

	return catalog.PackageContent{
		Files: map[string][]byte{"SKILL.md": []byte(resp.Files[0].Content)},
	}, nil
}
