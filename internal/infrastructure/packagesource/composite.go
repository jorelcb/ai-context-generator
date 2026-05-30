package packagesource

import (
	"context"
	"fmt"
	"sort"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// Compile-time guard: CompositeSource satisfies the frozen
// catalog.PackageSource contract.
var _ catalog.PackageSource = (*CompositeSource)(nil)

// CompositeSource merges several PackageSources into one, so the catalog can
// offer built-in packages alongside personal/team ones from local or remote
// sources (ADR-0010 §8: "la composición de varias sources se hace en una capa
// superior").
//
// Precedence: sources are given in increasing-priority order — when two
// sources expose a package with the same (Target, ID), the LATER source wins.
// So `NewCompositeSource(embedded, local)` lets a local package override a
// built-in one of the same ID (the "personal override" use case).
//
// Fetch routes back to the source that produced a manifest, matched on
// Source.Kind. (v0 assumes at most one source per Kind; multi-source-per-kind
// would additionally match on Source.URI.)
type CompositeSource struct {
	sources []catalog.PackageSource
}

// NewCompositeSource composes sources in increasing-priority order (later
// overrides earlier on (Target, ID) collisions).
func NewCompositeSource(sources ...catalog.PackageSource) *CompositeSource {
	return &CompositeSource{sources: sources}
}

// Kind identifies the composite for diagnostics. Individual manifests keep
// their own Source.Kind badge from their originating source.
func (c *CompositeSource) Kind() string { return "composite" }

// List gathers manifests from every source and de-duplicates by (Target, ID),
// later sources winning. The result is sorted deterministically.
func (c *CompositeSource) List(ctx context.Context) ([]catalog.PackageManifest, error) {
	type key struct {
		target catalog.Target
		id     string
	}
	merged := map[key]catalog.PackageManifest{}
	for _, src := range c.sources {
		manifests, err := src.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("composite: list %s: %w", src.Kind(), err)
		}
		for _, m := range manifests {
			merged[key{m.Target, m.ID}] = m // later source overrides
		}
	}

	out := make([]catalog.PackageManifest, 0, len(merged))
	for _, m := range merged {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Target != out[j].Target {
			return out[i].Target < out[j].Target
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// Fetch routes to the source whose Kind matches the manifest's Source.Kind.
func (c *CompositeSource) Fetch(ctx context.Context, m catalog.PackageManifest) (catalog.PackageContent, error) {
	for _, src := range c.sources {
		if src.Kind() == m.Source.Kind {
			return src.Fetch(ctx, m)
		}
	}
	return catalog.PackageContent{}, fmt.Errorf("composite: no source of kind %q to fetch %q", m.Source.Kind, m.ID)
}
