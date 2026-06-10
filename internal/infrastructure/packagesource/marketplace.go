package packagesource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// Compile-time guard: PluginMarketplaceSource satisfies the frozen
// catalog.PackageSource contract.
var _ catalog.PackageSource = (*PluginMarketplaceSource)(nil)

// marketplaceEcosystem captura lo que varía entre ecosystems en el formato
// marketplace.json compartido (ADR-0012 §4): el Kind del source, el Target que
// emiten los manifests, y el directorio del manifest dentro del repo.
type marketplaceEcosystem struct {
	kind        string         // Source.Kind (badges UI + routing de Fetch)
	target      catalog.Target // Target emitido por cada manifest
	manifestDir string         // directorio del marketplace.json en el repo
}

var marketplaceEcosystems = map[string]marketplaceEcosystem{
	"claude": {
		kind:        "claude-marketplace",
		target:      catalog.TargetClaudePlugin,
		manifestDir: ".claude-plugin",
	},
	// Antigravity (R-8 scaffolding): el formato converge con Claude (ADR-0012
	// §4 — mismo marketplace.json, distinto namespace dir). El directorio
	// `.agents/plugins` está confirmado para Codex y es la mejor hipótesis
	// para Antigravity (verificación pendiente, item D.2.b). La instalación
	// funcional sigue BLOQUEADA en `agy` (ver AntigravityPluginInstaller).
	"antigravity": {
		kind:        "antigravity-marketplace",
		target:      catalog.TargetAntigravityPlugin,
		manifestDir: ".agents/plugins",
	},
}

// PluginMarketplaceSource is a catalog.PackageSource over the shared
// marketplace.json plugin-catalog format (ADR-0012). It reads the
// marketplace.json and emits one plugin manifest per entry, with the Target
// and Source.Kind of the ecosystem it was built for.
//
// The format is shared/parallel across ecosystems (Claude/Codex/Antigravity),
// so this reader is parameterized by ecosystem rather than hardcoded to one
// (R-8); the git/URL transport is an injectable fetcher. Materialization is
// NOT this source's job — Fetch returns empty content, and the per-ecosystem
// TargetInstaller delegates the actual install to the native plugin CLI
// (e.g. `claude plugin install <id>@<marketplace>`; see ADR-0012 §3).
type PluginMarketplaceSource struct {
	// ref locates the marketplace: a `owner/repo` GitHub shorthand or a full
	// URL to the marketplace.json. Passed verbatim to the installer as the
	// `claude plugin marketplace add <ref>` argument.
	ref string
	// eco fixes the ecosystem-specific knobs (Kind, Target, manifest dir).
	eco marketplaceEcosystem
	// fetch reads the raw marketplace.json bytes for ref. Injectable so tests
	// run without network; defaults to httpFetchMarketplace.
	fetch func(ctx context.Context, ref string) ([]byte, error)
}

// NewPluginMarketplaceSource builds a Claude-ecosystem source for the
// marketplace at ref (`owner/repo` or a marketplace.json URL).
func NewPluginMarketplaceSource(ref string) *PluginMarketplaceSource {
	s, _ := NewPluginMarketplaceSourceFor("claude", ref)
	return s
}

// NewPluginMarketplaceSourceFor builds a source for the given ecosystem
// ("claude" or "antigravity"). Unknown ecosystems error so callers fail loudly
// instead of silently reading the wrong manifest path.
func NewPluginMarketplaceSourceFor(ecosystem, ref string) (*PluginMarketplaceSource, error) {
	eco, ok := marketplaceEcosystems[ecosystem]
	if !ok {
		return nil, fmt.Errorf("marketplace source: unknown ecosystem %q (claude or antigravity)", ecosystem)
	}
	s := &PluginMarketplaceSource{ref: ref, eco: eco}
	s.fetch = s.httpFetchMarketplace
	return s, nil
}

// withFetcher overrides the fetcher (tests).
func (s *PluginMarketplaceSource) withFetcher(f func(ctx context.Context, ref string) ([]byte, error)) *PluginMarketplaceSource {
	s.fetch = f
	return s
}

// Kind identifies this source for UI badges and Fetch routing.
func (s *PluginMarketplaceSource) Kind() string { return s.eco.kind }

// marketplaceDoc is the subset of marketplace.json we read.
type marketplaceDoc struct {
	Name    string             `json:"name"`
	Plugins []marketplaceEntry `json:"plugins"`
}

type marketplaceEntry struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

// List fetches and parses the marketplace.json, emitting one manifest per
// plugin. Manifests carry the marketplace name in Metadata so the installer
// can address the plugin as `<id>@<marketplace>`.
func (s *PluginMarketplaceSource) List(ctx context.Context) ([]catalog.PackageManifest, error) {
	data, err := s.fetch(ctx, s.ref)
	if err != nil {
		return nil, fmt.Errorf("marketplace source: fetch %q: %w", s.ref, err)
	}
	var doc marketplaceDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("marketplace source: parse %q: %w", s.ref, err)
	}
	if doc.Name == "" {
		return nil, fmt.Errorf("marketplace source: %q has no top-level \"name\"", s.ref)
	}

	out := make([]catalog.PackageManifest, 0, len(doc.Plugins))
	for _, p := range doc.Plugins {
		if p.Name == "" {
			return nil, fmt.Errorf("marketplace source: %q has a plugin entry with no name", s.ref)
		}
		version := p.Version
		if version == "" {
			// No pin in the marketplace → the agent resolves to the git
			// commit at install time. We label it "latest" for display.
			version = "latest"
		}
		m := catalog.PackageManifest{
			ID:          p.Name,
			Version:     version,
			Description: p.Description,
			Target:      s.eco.target,
			Source:      catalog.SourceRef{Kind: s.Kind(), URI: s.ref},
			Metadata:    map[string]string{catalog.MetaKeyMarketplace: doc.Name},
		}
		if p.Category != "" {
			m.Metadata[catalog.MetaKeyCategory] = p.Category
		}
		if len(p.Tags) > 0 {
			m.Metadata[catalog.MetaKeyTags] = strings.Join(p.Tags, ",")
		}
		m.SourceChecksum = computeManifestChecksum(m)
		out = append(out, m)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Fetch returns empty content for plugins: a plugin is materialized by the
// agent's own plugin CLI (delegated by the TargetInstaller), not fetched and
// written by codify. The manifest alone (id + marketplace + ref) is enough to
// install.
func (s *PluginMarketplaceSource) Fetch(ctx context.Context, m catalog.PackageManifest) (catalog.PackageContent, error) {
	if m.Target != s.eco.target {
		return catalog.PackageContent{}, fmt.Errorf("marketplace source: target %q not supported (%s only)", m.Target, s.eco.target)
	}
	return catalog.PackageContent{}, nil
}

// MarketplaceName returns the marketplace name a manifest belongs to, read
// from Metadata. The installer needs it to address `<id>@<marketplace>`.
func MarketplaceName(m catalog.PackageManifest) string {
	if m.Metadata == nil {
		return ""
	}
	return m.Metadata[catalog.MetaKeyMarketplace]
}

// httpFetchMarketplace is the default fetcher. It accepts either a full
// http(s) URL to a marketplace.json, or a GitHub `owner/repo` shorthand
// (resolved to the default branch's `<manifestDir>/marketplace.json` via the
// GitHub contents API, which avoids guessing the branch name). Bound as a
// method so the ecosystem's manifest dir parameterizes the shorthand path.
func (s *PluginMarketplaceSource) httpFetchMarketplace(ctx context.Context, ref string) ([]byte, error) {
	url, accept := marketplaceURL(ref, s.eco.manifestDir)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20)) // 5 MiB cap
}

// marketplaceURL maps a ref to a fetch URL. A full URL is used as-is; an
// `owner/repo` shorthand resolves via the GitHub contents API with the raw
// media type so the default branch is followed without hardcoding it. The
// manifestDir is the ecosystem's marketplace.json directory in the repo
// (`.claude-plugin` for Claude, `.agents/plugins` for Antigravity/Codex).
func marketplaceURL(ref, manifestDir string) (url, accept string) {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref, ""
	}
	return fmt.Sprintf("https://api.github.com/repos/%s/contents/%s/marketplace.json", ref, manifestDir),
		"application/vnd.github.raw+json"
}
