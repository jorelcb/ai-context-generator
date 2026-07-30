package packagesource

import (
	"context"
	"errors"
	"testing"

	"github.com/jorelcb/codify-og/internal/domain/catalog"
)

// fakeFetcher returns fixed bytes (or an error) for any ref.
func fakeFetcher(data string, err error) func(context.Context, string) ([]byte, error) {
	return func(context.Context, string) ([]byte, error) {
		if err != nil {
			return nil, err
		}
		return []byte(data), nil
	}
}

const sampleMarketplace = `{
  "name": "acme-tools",
  "owner": {"name": "Acme"},
  "plugins": [
    {"name": "code-formatter", "description": "Format on save", "version": "2.1.0", "category": "quality", "tags": ["fmt","lint"]},
    {"name": "deployer", "description": "Deploy automation"}
  ]
}`

func newMarketplace(t *testing.T, data string, err error) *PluginMarketplaceSource {
	t.Helper()
	return NewPluginMarketplaceSource("acme/plugins").withFetcher(fakeFetcher(data, err))
}

func TestMarketplaceSource_List_MapsPluginsToManifests(t *testing.T) {
	src := newMarketplace(t, sampleMarketplace, nil)
	got, err := src.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(got))
	}
	// Sorted by ID: code-formatter before deployer.
	cf := got[0]
	if cf.ID != "code-formatter" || cf.Target != catalog.TargetClaudePlugin {
		t.Errorf("unexpected first manifest: %+v", cf)
	}
	if cf.Version != "2.1.0" {
		t.Errorf("version: got %q, want 2.1.0", cf.Version)
	}
	if cf.Source.Kind != "claude-marketplace" || cf.Source.URI != "acme/plugins" {
		t.Errorf("source not decorated: %+v", cf.Source)
	}
	if MarketplaceName(cf) != "acme-tools" {
		t.Errorf("marketplace name: got %q, want acme-tools", MarketplaceName(cf))
	}
	if cf.Metadata["category"] != "quality" || cf.Metadata["tags"] != "fmt,lint" {
		t.Errorf("metadata not mapped: %+v", cf.Metadata)
	}
	if cf.SourceChecksum == "" {
		t.Error("checksum not computed")
	}

	// A plugin without a version label defaults to "latest".
	if got[1].ID != "deployer" || got[1].Version != "latest" {
		t.Errorf("expected deployer@latest, got %+v", got[1])
	}
}

func TestMarketplaceSource_Fetch_EmptyForPlugins(t *testing.T) {
	src := newMarketplace(t, sampleMarketplace, nil)
	m := catalog.PackageManifest{ID: "code-formatter", Target: catalog.TargetClaudePlugin}
	content, err := src.Fetch(context.Background(), m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(content.Files) != 0 || len(content.SettingsFragment) != 0 {
		t.Errorf("plugin Fetch should return empty content, got %+v", content)
	}
}

func TestMarketplaceSource_Fetch_RejectsNonPlugin(t *testing.T) {
	src := newMarketplace(t, sampleMarketplace, nil)
	_, err := src.Fetch(context.Background(), catalog.PackageManifest{ID: "x", Target: catalog.TargetClaudeSkill})
	if err == nil {
		t.Fatal("expected error fetching a non-plugin target")
	}
}

func TestMarketplaceSource_List_FetchError(t *testing.T) {
	src := newMarketplace(t, "", errors.New("network down"))
	if _, err := src.List(context.Background()); err == nil {
		t.Fatal("expected List to propagate fetch error")
	}
}

func TestMarketplaceSource_List_MalformedJSON(t *testing.T) {
	src := newMarketplace(t, "{not json", nil)
	if _, err := src.List(context.Background()); err == nil {
		t.Fatal("expected error on malformed marketplace.json")
	}
}

func TestMarketplaceSource_List_MissingName(t *testing.T) {
	src := newMarketplace(t, `{"plugins":[{"name":"p"}]}`, nil)
	if _, err := src.List(context.Background()); err == nil {
		t.Fatal("expected error when marketplace has no name")
	}
}

func TestMarketplaceSource_List_PluginWithoutName(t *testing.T) {
	src := newMarketplace(t, `{"name":"m","plugins":[{"description":"x"}]}`, nil)
	if _, err := src.List(context.Background()); err == nil {
		t.Fatal("expected error when a plugin entry has no name")
	}
}

func TestMarketplaceURL(t *testing.T) {
	cases := []struct{ ref, dir, wantURL, wantAccept string }{
		{"owner/repo", ".claude-plugin", "https://api.github.com/repos/owner/repo/contents/.claude-plugin/marketplace.json", "application/vnd.github.raw+json"},
		{"owner/repo", ".agents/plugins", "https://api.github.com/repos/owner/repo/contents/.agents/plugins/marketplace.json", "application/vnd.github.raw+json"},
		{"https://example.com/marketplace.json", ".claude-plugin", "https://example.com/marketplace.json", ""},
	}
	for _, c := range cases {
		gotURL, gotAccept := marketplaceURL(c.ref, c.dir)
		if gotURL != c.wantURL || gotAccept != c.wantAccept {
			t.Errorf("marketplaceURL(%q,%q) = (%q,%q), want (%q,%q)", c.ref, c.dir, gotURL, gotAccept, c.wantURL, c.wantAccept)
		}
	}
}

func TestMarketplaceSource_Kind(t *testing.T) {
	if NewPluginMarketplaceSource("x").Kind() != "claude-marketplace" {
		t.Error("Kind() should be claude-marketplace")
	}
}

// TestMarketplaceSource_EcosystemParameterization guards R-8: the SAME
// marketplace.json read through each ecosystem's source emits that ecosystem's
// Kind and Target.
func TestMarketplaceSource_EcosystemParameterization(t *testing.T) {
	doc := `{"name":"mkt","plugins":[{"name":"p1","description":"d"}]}`

	cases := []struct {
		ecosystem  string
		wantKind   string
		wantTarget catalog.Target
	}{
		{"claude", "claude-marketplace", catalog.TargetClaudePlugin},
		{"antigravity", "antigravity-marketplace", catalog.TargetAntigravityPlugin},
	}
	for _, c := range cases {
		src, err := NewPluginMarketplaceSourceFor(c.ecosystem, "owner/repo")
		if err != nil {
			t.Fatalf("NewPluginMarketplaceSourceFor(%s): %v", c.ecosystem, err)
		}
		src.withFetcher(func(context.Context, string) ([]byte, error) { return []byte(doc), nil })

		if src.Kind() != c.wantKind {
			t.Errorf("%s: Kind()=%q, want %q", c.ecosystem, src.Kind(), c.wantKind)
		}
		got, err := src.List(context.Background())
		if err != nil {
			t.Fatalf("%s: List: %v", c.ecosystem, err)
		}
		if len(got) != 1 || got[0].Target != c.wantTarget {
			t.Errorf("%s: manifest target=%q, want %q", c.ecosystem, got[0].Target, c.wantTarget)
		}
		if got[0].Source.Kind != c.wantKind {
			t.Errorf("%s: Source.Kind=%q, want %q", c.ecosystem, got[0].Source.Kind, c.wantKind)
		}
	}
}

func TestNewPluginMarketplaceSourceFor_UnknownEcosystem(t *testing.T) {
	if _, err := NewPluginMarketplaceSourceFor("gemini", "x"); err == nil {
		t.Error("unknown ecosystem should error (gemini CLI is deprecated, not modeled)")
	}
}
