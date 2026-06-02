package catalog

import "testing"

func TestBuild_GroupsByDeclaredCategory(t *testing.T) {
	c := Build("project", "claude", []TabSpec{
		{Name: "Skills", Type: "skill", Pkgs: []Pkg{
			{ID: "ddd-entity", Category: "Architecture"},
			{ID: "test-bdd", Category: "Testing"},
			{ID: "hexagonal-port", Category: "Architecture"},
		}},
	})
	tab := c.Tabs[0]
	if !tab.HasCategories {
		t.Fatal("tab with declared categories should have HasCategories=true")
	}
	if len(tab.Categories) != 2 {
		t.Fatalf("got %d categories, want 2 (Architecture, Testing)", len(tab.Categories))
	}
	// Sorted: Architecture before Testing.
	if tab.Categories[0].Name != "Architecture" || tab.Categories[1].Name != "Testing" {
		t.Errorf("categories not sorted: %q, %q", tab.Categories[0].Name, tab.Categories[1].Name)
	}
	// Architecture holds 2 leaves, sorted by ID.
	arch := tab.Categories[0]
	if len(arch.Leaves) != 2 || arch.Leaves[0].ID != "ddd-entity" || arch.Leaves[1].ID != "hexagonal-port" {
		t.Errorf("architecture leaves wrong: %+v", arch.Leaves)
	}
}

func TestBuild_FlatWhenNoCategoryDeclared(t *testing.T) {
	c := Build("project", "claude", []TabSpec{
		{Name: "Plugins", Type: "plugin", Pkgs: []Pkg{
			{ID: "spec-driven-change"}, {ID: "release-cycle"}, {ID: "bug-fix"},
		}},
	})
	tab := c.Tabs[0]
	if tab.HasCategories {
		t.Fatal("tab with no declared categories should be flat (HasCategories=false)")
	}
	if len(tab.Categories) != 1 || tab.Categories[0].Name != "" {
		t.Fatalf("flat tab should have a single unnamed category, got %+v", tab.Categories)
	}
	// 3 leaves, sorted by ID: bug-fix, release-cycle, spec-driven-change.
	leaves := tab.Categories[0].Leaves
	if len(leaves) != 3 || leaves[0].ID != "bug-fix" {
		t.Errorf("flat leaves wrong/unsorted: %+v", leaves)
	}
}

func TestBuild_EmptyCategoryGoesToGeneral(t *testing.T) {
	c := Build("project", "claude", []TabSpec{
		{Name: "Skills", Type: "skill", Pkgs: []Pkg{
			{ID: "ddd-entity", Category: "Architecture"},
			{ID: "loose-one"}, // no category, but tab IS categorized
		}},
	})
	tab := c.Tabs[0]
	if !tab.HasCategories {
		t.Fatal("mixed tab should be categorized")
	}
	var foundGeneral bool
	for _, cat := range tab.Categories {
		if cat.Name == generalGroup {
			foundGeneral = true
			if len(cat.Leaves) != 1 || cat.Leaves[0].ID != "loose-one" {
				t.Errorf("General bucket wrong: %+v", cat.Leaves)
			}
		}
	}
	if !foundGeneral {
		t.Errorf("uncategorized package should land in %q; categories=%v", generalGroup, catNames(tab))
	}
}

func TestBuild_InstalledMarkerPropagates(t *testing.T) {
	c := Build("project", "claude", []TabSpec{
		{Name: "Skills", Type: "skill", Pkgs: []Pkg{
			{ID: "ddd-entity", Category: "Architecture", Installed: true},
			{ID: "hexagonal-port", Category: "Architecture"},
		}},
	})
	leaves := c.Tabs[0].Categories[0].Leaves
	if !leaves[0].Installed {
		t.Errorf("ddd-entity should be marked installed")
	}
	if leaves[1].Installed {
		t.Errorf("hexagonal-port should not be marked installed")
	}
}

// TestBuild_FeedsSelectorEndToEnd verifies a Built catalog supports the full
// selection mechanic (accumulation) without any further wiring.
func TestBuild_FeedsSelectorEndToEnd(t *testing.T) {
	c := Build("project", "claude", []TabSpec{
		{Name: "Skills", Type: "skill", Pkgs: []Pkg{{ID: "ddd-entity", Category: "Architecture"}}},
		{Name: "Plugins", Type: "plugin", Pkgs: []Pkg{{ID: "spec-driven-change"}}},
	})
	c.Active = 0
	c.markByID("ddd-entity")
	c.Active = 1
	c.markByID("spec-driven-change")
	got := c.Collect()
	if len(got) != 2 {
		t.Fatalf("Collect across built tabs = %d, want 2: %+v", len(got), got)
	}
}

func catNames(t Tab) []string {
	var n []string
	for _, c := range t.Categories {
		n = append(n, c.Name)
	}
	return n
}
