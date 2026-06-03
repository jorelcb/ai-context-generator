package desiredstate

import (
	"path/filepath"
	"testing"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

func TestLoad_MissingFileReturnsEmptyState(t *testing.T) {
	ds, err := Load(filepath.Join(t.TempDir(), "nope.yml"), catalog.ScopeProject)
	if err != nil {
		t.Fatalf("Load missing file should not error: %v", err)
	}
	if ds.Version != SchemaVersion || ds.Scope != "project" || len(ds.Packages) != 0 {
		t.Fatalf("empty state wrong: %+v", ds)
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".codify", "project.catalog.yml")
	ds := &DesiredState{Scope: "project"}
	ds.Add(
		Package{ID: "ddd-entity", Type: "skill"},
		Package{ID: "spec-driven-change", Type: "plugin", Source: "anthropics/claude-plugins-official"},
	)
	if err := Save(path, ds); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path, catalog.ScopeProject)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Packages) != 2 {
		t.Fatalf("round-trip lost packages: %+v", got.Packages)
	}
	// Plugin source must survive.
	var pluginSrc string
	for _, p := range got.Packages {
		if p.ID == "spec-driven-change" {
			pluginSrc = p.Source
		}
	}
	if pluginSrc != "anthropics/claude-plugins-official" {
		t.Errorf("plugin source lost: %q", pluginSrc)
	}
}

func TestAdd_DedupesByTypeAndID(t *testing.T) {
	ds := &DesiredState{Scope: "project"}
	added := ds.Add(Package{ID: "ddd-entity", Type: "skill"})
	if added != 1 {
		t.Fatalf("first add = %d, want 1", added)
	}
	// Same (type,id) again -> no new package.
	added = ds.Add(Package{ID: "ddd-entity", Type: "skill"})
	if added != 0 {
		t.Errorf("duplicate add = %d, want 0", added)
	}
	// Same id, different type -> distinct package.
	added = ds.Add(Package{ID: "ddd-entity", Type: "hook"})
	if added != 1 {
		t.Errorf("same id different type = %d, want 1", added)
	}
	if len(ds.Packages) != 2 {
		t.Errorf("packages = %d, want 2", len(ds.Packages))
	}
}

func TestAdd_UpdatesSourceOnExistingKey(t *testing.T) {
	ds := &DesiredState{Scope: "project"}
	ds.Add(Package{ID: "p", Type: "plugin", Source: "old/repo"})
	ds.Add(Package{ID: "p", Type: "plugin", Source: "new/repo"})
	if len(ds.Packages) != 1 || ds.Packages[0].Source != "new/repo" {
		t.Errorf("source not updated: %+v", ds.Packages)
	}
}

func TestAdd_EcosystemDistinguishesSameTypeAndID(t *testing.T) {
	ds := &DesiredState{Scope: "project"}
	// Same (type,id) but different ecosystem -> two distinct desired packages.
	ds.Add(Package{ID: "ddd-entity", Type: "skill"}) // defaults to claude
	added := ds.Add(Package{ID: "ddd-entity", Type: "skill", Ecosystem: "antigravity"})
	if added != 1 {
		t.Fatalf("antigravity variant should be distinct: added=%d", added)
	}
	// Re-adding the implicit-claude one is still a dup (default ecosystem).
	if got := ds.Add(Package{ID: "ddd-entity", Type: "skill", Ecosystem: "claude"}); got != 0 {
		t.Errorf("explicit claude should match implicit claude: added=%d", got)
	}
	if len(ds.Packages) != 2 {
		t.Errorf("packages = %d, want 2", len(ds.Packages))
	}
}

func TestRemove(t *testing.T) {
	ds := &DesiredState{Scope: "project"}
	ds.Add(Package{ID: "a", Type: "skill"}, Package{ID: "b", Type: "skill"})
	if !ds.Remove("skill", "a") {
		t.Error("Remove existing should return true")
	}
	if ds.Remove("skill", "a") {
		t.Error("Remove absent should return false")
	}
	if got := ds.ByType("skill"); len(got) != 1 || got[0] != "b" {
		t.Errorf("after remove, ByType = %v, want [b]", got)
	}
}

func TestByType_Sorted(t *testing.T) {
	ds := &DesiredState{Scope: "project"}
	ds.Add(
		Package{ID: "zeta", Type: "skill"},
		Package{ID: "alpha", Type: "skill"},
		Package{ID: "h1", Type: "hook"},
	)
	skills := ds.ByType("skill")
	if len(skills) != 2 || skills[0] != "alpha" || skills[1] != "zeta" {
		t.Errorf("ByType(skill) = %v, want [alpha zeta]", skills)
	}
	if hooks := ds.ByType("hook"); len(hooks) != 1 || hooks[0] != "h1" {
		t.Errorf("ByType(hook) = %v, want [h1]", hooks)
	}
}

func TestPathForScope(t *testing.T) {
	ws, err := PathForScope(catalog.ScopeWorkstation)
	if err != nil || filepath.Base(ws) != "workstation.catalog.yml" {
		t.Errorf("workstation path = %q (err %v)", ws, err)
	}
	pj, err := PathForScope(catalog.ScopeProject)
	if err != nil || filepath.Base(pj) != "project.catalog.yml" {
		t.Errorf("project path = %q (err %v)", pj, err)
	}
}
