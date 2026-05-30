package packagesource

import (
	"context"
	"strings"
	"testing"

	root "github.com/jorelcb/codify"
	"github.com/jorelcb/codify/internal/domain/catalog"
)

func TestCompositeSource_MergesAndDedups(t *testing.T) {
	ctx := context.Background()
	embedded := NewEmbeddedSource(root.TemplatesFS, "test")

	// Local source defines a NEW skill plus an OVERRIDE of an embedded one.
	localRoot := t.TempDir()
	writePackage(t, localRoot, "my-local-skill", "id: my-local-skill\ntarget: claude-skill\n",
		map[string]string{"SKILL.md": "---\nname: my-local-skill\n---\nlocal"})
	writePackage(t, localRoot, "ddd-entity", "id: ddd-entity\ntarget: claude-skill\n",
		map[string]string{"SKILL.md": "---\nname: ddd-entity\n---\nLOCAL OVERRIDE"})
	local := NewLocalDirectorySource(localRoot)

	emCount := mustList(t, embedded, ctx)
	composite := NewCompositeSource(embedded, local) // local overrides embedded

	all, err := composite.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// One brand-new skill added; ddd-entity dedup'd (not double-counted).
	if len(all) != emCount+1 {
		t.Errorf("expected %d packages, got %d", emCount+1, len(all))
	}

	// ddd-entity now resolves to the LOCAL source (override won).
	var ddd *catalog.PackageManifest
	for i := range all {
		if all[i].ID == "ddd-entity" && all[i].Target == catalog.TargetClaudeSkill {
			ddd = &all[i]
		}
	}
	if ddd == nil {
		t.Fatal("ddd-entity missing")
	}
	if ddd.Source.Kind != "local-fs" {
		t.Errorf("ddd-entity should be overridden by local-fs, got source %q", ddd.Source.Kind)
	}

	// Fetch routes to the local source for the override.
	content, err := composite.Fetch(ctx, *ddd)
	if err != nil {
		t.Fatalf("Fetch override: %v", err)
	}
	if !strings.Contains(string(content.Files["SKILL.md"]), "LOCAL OVERRIDE") {
		t.Errorf("override Fetch returned embedded content: %q", content.Files["SKILL.md"])
	}
}

func TestCompositeSource_Fetch_RoutesByKind(t *testing.T) {
	ctx := context.Background()
	embedded := NewEmbeddedSource(root.TemplatesFS, "test")
	composite := NewCompositeSource(embedded)

	// An embedded skill fetches via the embedded source.
	manifests, _ := composite.List(ctx)
	var skill *catalog.PackageManifest
	for i := range manifests {
		if manifests[i].Target == catalog.TargetClaudeSkill {
			skill = &manifests[i]
			break
		}
	}
	if skill == nil {
		t.Fatal("no embedded skill found")
	}
	if _, err := composite.Fetch(ctx, *skill); err != nil {
		t.Fatalf("Fetch embedded via composite: %v", err)
	}

	// A manifest from an unregistered kind errors.
	_, err := composite.Fetch(ctx, catalog.PackageManifest{
		ID: "ghost", Target: catalog.TargetClaudeSkill, Source: catalog.SourceRef{Kind: "git"},
	})
	if err == nil {
		t.Error("expected error fetching a manifest whose source kind is not composed")
	}
}

func mustList(t *testing.T, s catalog.PackageSource, ctx context.Context) int {
	t.Helper()
	m, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return len(m)
}
