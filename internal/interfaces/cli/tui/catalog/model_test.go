package catalog

import "testing"

// newTestCatalog builds a representative catalog: a Skills tab WITH categories
// (architecture/testing/conventions) and Hooks/Plugins tabs as flat lists
// (HasCategories=false), mirroring codify's real shape.
func newTestCatalog() *Catalog {
	return &Catalog{
		Scope:     "project",
		Ecosystem: "claude",
		Tabs: []Tab{
			{
				Name: "Skills", Type: "skill", HasCategories: true,
				Categories: []Category{
					{Name: "architecture", Leaves: []Leaf{
						{ID: "ddd-entity"}, {ID: "hexagonal-port"}, {ID: "clean-arch-layer"}, {ID: "cqrs-command"},
					}},
					{Name: "testing", Leaves: []Leaf{{ID: "test-bdd"}, {ID: "bdd-scenario"}}},
					{Name: "conventions", Leaves: []Leaf{{ID: "conventional-commit"}, {ID: "semantic-versioning"}}},
				},
			},
			{
				Name: "Hooks", Type: "hook", HasCategories: false,
				Categories: []Category{{Name: "", Leaves: []Leaf{
					{ID: "format-on-save"}, {ID: "block-secrets"}, {ID: "lint-staged"},
				}}},
			},
			{
				Name: "Plugins", Type: "plugin", HasCategories: false,
				Categories: []Category{{Name: "", Leaves: []Leaf{
					{ID: "spec-driven-change"}, {ID: "release-cycle"}, {ID: "bug-fix"},
				}}},
			},
		},
	}
}

// mark checks a leaf by ID in the active tab (helper for tests).
func (c *Catalog) markByID(id string) {
	t := c.ActiveTab()
	for ci := range t.Categories {
		for li := range t.Categories[ci].Leaves {
			if t.Categories[ci].Leaves[li].ID == id {
				t.Categories[ci].Leaves[li].Checked = true
				return
			}
		}
	}
}

func TestCollect_AccumulatesAcrossTabs(t *testing.T) {
	c := newTestCatalog()
	// Mark 2 in Skills (tab 0).
	c.Active = 0
	c.markByID("ddd-entity")
	c.markByID("test-bdd")
	// Switch to Plugins (tab 2) and mark 1.
	c.Active = 2
	c.markByID("spec-driven-change")

	got := c.Collect()
	if len(got) != 3 {
		t.Fatalf("Collect()=%d, want 3: %+v", len(got), got)
	}
	wantType := map[string]string{
		"ddd-entity":         "skill",
		"test-bdd":           "skill",
		"spec-driven-change": "plugin",
	}
	for _, s := range got {
		if wantType[s.ID] != s.TabType {
			t.Errorf("%q has TabType %q, want %q", s.ID, s.TabType, wantType[s.ID])
		}
		delete(wantType, s.ID)
	}
	if len(wantType) != 0 {
		t.Errorf("missing from Collect: %v", wantType)
	}
	if c.TotalChecked() != 3 {
		t.Errorf("TotalChecked()=%d, want 3", c.TotalChecked())
	}
}

func TestCategory_TriState(t *testing.T) {
	c := newTestCatalog()
	arch := &c.Tabs[0].Categories[0] // architecture, 4 leaves
	if arch.State() != None {
		t.Fatalf("fresh category = %v, want None", arch.State())
	}
	arch.Leaves[0].Checked = true
	arch.Leaves[1].Checked = true
	if arch.State() != Some {
		t.Errorf("2/4 checked = %v, want Some ([~])", arch.State())
	}
	if arch.State().Glyph() != "[~]" {
		t.Errorf("Some.Glyph()=%q, want [~]", arch.State().Glyph())
	}
	arch.SetAll(true)
	if arch.State() != All {
		t.Errorf("4/4 checked = %v, want All ([x])", arch.State())
	}
	arch.SetAll(false)
	if arch.State() != None {
		t.Errorf("0/4 checked = %v, want None ([ ])", arch.State())
	}
}

func TestToggleCurrent_CategoryMarksAllChildren(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0
	c.Cursor = 0 // architecture header (collapsed -> header is first visible row)
	c.ToggleCurrent()
	if c.Tabs[0].Categories[0].State() != All {
		t.Errorf("after toggle on header: %v, want All", c.Tabs[0].Categories[0].State())
	}
	c.ToggleCurrent() // toggle again -> unmark all
	if c.Tabs[0].Categories[0].State() != None {
		t.Errorf("after second toggle: %v, want None", c.Tabs[0].Categories[0].State())
	}
}

func TestVisibleRows_RespectsCollapse(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0 // Skills, all categories collapsed by default
	rows := c.VisibleRows()
	if len(rows) != 3 { // 3 category headers, no leaves visible
		t.Fatalf("collapsed Skills: %d rows, want 3 headers", len(rows))
	}
	c.Tabs[0].Categories[0].Expanded = true
	rows = c.VisibleRows()
	// 3 headers + 4 architecture leaves = 7
	if len(rows) != 7 {
		t.Fatalf("architecture expanded: %d rows, want 7", len(rows))
	}
}

func TestVisibleRows_FlatTabShowsLeaves(t *testing.T) {
	c := newTestCatalog()
	c.Active = 1 // Hooks, HasCategories=false
	rows := c.VisibleRows()
	// 1 (synthetic category header, hidden by render) + 3 leaves
	leafCount := 0
	for _, r := range rows {
		if r.Kind == RowLeaf {
			leafCount++
		}
	}
	if leafCount != 3 {
		t.Fatalf("flat Hooks tab: %d leaf rows, want 3", leafCount)
	}
}

func TestNextPrevTab_ResetsCursor(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0
	c.Cursor = 2
	c.NextTab()
	if c.Active != 1 || c.Cursor != 0 {
		t.Errorf("NextTab: active=%d cursor=%d, want 1/0", c.Active, c.Cursor)
	}
	c.PrevTab()
	if c.Active != 0 || c.Cursor != 0 {
		t.Errorf("PrevTab: active=%d cursor=%d, want 0/0", c.Active, c.Cursor)
	}
}

func TestToggleAllInActiveTab_IsolatedToTab(t *testing.T) {
	c := newTestCatalog()
	c.Active = 1 // Hooks
	c.ToggleAllInActiveTab()
	if c.Tabs[1].CheckedCount() != 3 {
		t.Errorf("Hooks after toggle-all: %d, want 3", c.Tabs[1].CheckedCount())
	}
	if c.Tabs[0].CheckedCount() != 0 || c.Tabs[2].CheckedCount() != 0 {
		t.Errorf("toggle-all leaked to other tabs: skills=%d plugins=%d",
			c.Tabs[0].CheckedCount(), c.Tabs[2].CheckedCount())
	}
}

func TestCollapse_FromLeafMovesCursorToParent(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0
	c.Tabs[0].Categories[0].Expanded = true
	c.Cursor = 2 // a leaf under architecture (row 0=header,1=leaf,2=leaf)
	c.Collapse()
	// Cursor should land on the architecture header (row 0) and it should be collapsed.
	if c.Tabs[0].Categories[0].Expanded {
		t.Error("Collapse did not collapse the category")
	}
	r, ok := c.CurrentRow()
	if !ok || r.Kind != RowCategory || r.Cat != 0 {
		t.Errorf("after collapse, cursor at %+v (ok=%v), want architecture header", r, ok)
	}
}
