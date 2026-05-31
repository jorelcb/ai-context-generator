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

// PluginMarketplaceSource is a catalog.PackageSource over a Claude-style
// plugin marketplace — the `.claude-plugin/marketplace.json` catalog format
// (ADR-0012). It reads the marketplace.json and emits one TargetClaudePlugin
// manifest per plugin entry.
//
// The format is shared/parallel across ecosystems (Claude/Codex/Antigravity),
// so this reader is intentionally about the *format*, not a specific
// ecosystem; the git/URL transport is an injectable fetcher. Materialization
// is NOT this source's job — Fetch returns empty content, and the
// per-ecosystem TargetInstaller delegates the actual install to the native
// plugin CLI (e.g. `claude plugin install <id>@<marketplace>`; see ADR-0012
// §3, validated by the 2026-05-30 spike).
type PluginMarketplaceSource struct {
	// ref locates the marketplace: a `owner/repo` GitHub shorthand or a full
	// URL to the marketplace.json. Passed verbatim to the installer as the
	// `claude plugin marketplace add <ref>` argument.
	ref string
	// fetch reads the raw marketplace.json bytes for ref. Injectable so tests
	// run without network; defaults to httpFetchMarketplace.
	fetch func(ctx context.Context, ref string) ([]byte, error)
}

// NewPluginMarketplaceSource builds a source for the marketplace at ref
// (`owner/repo` or a marketplace.json URL).
func NewPluginMarketplaceSource(ref string) *PluginMarketplaceSource {
	return &PluginMarketplaceSource{ref: ref, fetch: httpFetchMarketplace}
}

// withFetcher overrides the fetcher (tests).
func (s *PluginMarketplaceSource) withFetcher(f func(ctx context.Context, ref string) ([]byte, error)) *PluginMarketplaceSource {
	s.fetch = f
	return s
}

// Kind identifies this source for UI badges and Fetch routing.
func (s *PluginMarketplaceSource) Kind() string { return "claude-marketplace" }

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
			Target:      catalog.TargetClaudePlugin,
			Source:      catalog.SourceRef{Kind: s.Kind(), URI: s.ref},
			Metadata:    map[string]string{catalog.MetaKeyMarketplace: doc.Name},
		}
		if p.Category != "" {
			m.Metadata["category"] = p.Category
		}
		if len(p.Tags) > 0 {
			m.Metadata["tags"] = strings.Join(p.Tags, ",")
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
	if m.Target != catalog.TargetClaudePlugin {
		return catalog.PackageContent{}, fmt.Errorf("marketplace source: target %q not supported (claude-plugin only)", m.Target)
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
// (resolved to the default branch's `.claude-plugin/marketplace.json` via the
// GitHub contents API, which avoids guessing the branch name).
func httpFetchMarketplace(ctx context.Context, ref string) ([]byte, error) {
	url, accept := marketplaceURL(ref)

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
// media type so the default branch is followed without hardcoding it.
func marketplaceURL(ref string) (url, accept string) {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref, ""
	}
	return fmt.Sprintf("https://api.github.com/repos/%s/contents/.claude-plugin/marketplace.json", ref),
		"application/vnd.github.raw+json"
}
