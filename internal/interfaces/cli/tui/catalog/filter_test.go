package catalog

import "testing"

// leafIDsVisible returns the IDs of leaf rows currently visible in the active tab.
func leafIDsVisible(c *Catalog) []string {
	var ids []string
	t := c.ActiveTab()
	for _, r := range c.VisibleRows() {
		if r.Kind == RowLeaf {
			ids = append(ids, t.Categories[r.Cat].Leaves[r.Leaf].ID)
		}
	}
	return ids
}

func TestFilter_ShowsOnlyMatchingLeavesAndTheirCategories(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0 // Skills (categorized)
	c.Query = "bdd"

	ids := leafIDsVisible(c)
	if len(ids) != 2 {
		t.Fatalf("filter 'bdd' visible leaves=%v, want test-bdd & bdd-scenario", ids)
	}
	// Only the 'testing' category header should appear (architecture/conventions
	// have no matches and are dropped entirely).
	cats := 0
	for _, r := range c.VisibleRows() {
		if r.Kind == RowCategory {
			cats++
			if c.ActiveTab().Categories[r.Cat].Name != "testing" {
				t.Errorf("unexpected category header in filtered view: %q", c.ActiveTab().Categories[r.Cat].Name)
			}
		}
	}
	if cats != 1 {
		t.Errorf("filtered view has %d category headers, want 1 (testing)", cats)
	}
}

func TestFilter_NoMatchesYieldsEmptyView(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0
	c.Query = "zzzznope"
	if rows := c.VisibleRows(); len(rows) != 0 {
		t.Errorf("no-match filter should yield 0 rows, got %d", len(rows))
	}
}

func TestFilter_FlatTabFiltersLeaves(t *testing.T) {
	c := newTestCatalog()
	c.Active = 2 // Plugins (flat: spec-driven-change, release-cycle, bug-fix)
	c.Query = "release"
	ids := leafIDsVisible(c)
	if len(ids) != 1 || ids[0] != "release-cycle" {
		t.Errorf("flat-tab filter 'release' => %v, want [release-cycle]", ids)
	}
}

func TestFilterInput_BuildsQueryAndPositionsCursor(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0
	c.StartFilter()
	for _, r := range "cqrs" {
		c.FilterInput(r)
	}
	if c.Query != "cqrs" {
		t.Fatalf("Query=%q, want cqrs", c.Query)
	}
	// Cursor must sit on the first visible LEAF (not a category header).
	r, ok := c.CurrentRow()
	if !ok || r.Kind != RowLeaf {
		t.Fatalf("cursor not on a leaf after typing: %+v ok=%v", r, ok)
	}
	leaf, _ := c.CurrentLeaf()
	if leaf.ID != "cqrs-command" {
		t.Errorf("cursor leaf=%q, want cqrs-command", leaf.ID)
	}
}

func TestFilterBackspace_EditsQuery(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0
	c.Query = "bddx"
	c.Filtering = true
	c.FilterBackspace()
	if c.Query != "bdd" {
		t.Errorf("after backspace Query=%q, want bdd", c.Query)
	}
	// Now it should match again.
	if len(leafIDsVisible(c)) != 2 {
		t.Errorf("after backspace to 'bdd', expected 2 matches, got %v", leafIDsVisible(c))
	}
}

func TestCommitFilter_KeepsQueryExitsInput(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0
	c.Query = "bdd"
	c.Filtering = true
	c.CommitFilter()
	if c.Filtering {
		t.Error("CommitFilter should exit input mode")
	}
	if c.Query != "bdd" {
		t.Error("CommitFilter should keep the query applied")
	}
}

func TestClearFilter_ResetsEverything(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0
	c.Query = "bdd"
	c.Filtering = true
	c.ClearFilter()
	if c.Filtering || c.Query != "" {
		t.Errorf("ClearFilter left state Filtering=%v Query=%q", c.Filtering, c.Query)
	}
	// Back to the unfiltered tree (3 collapsed category headers).
	if rows := c.VisibleRows(); len(rows) != 3 {
		t.Errorf("after clear, expected 3 collapsed headers, got %d rows", len(rows))
	}
}

func TestFilter_SelectionPersistsAcrossFilterAndClear(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0
	c.markByID("test-bdd")
	c.Query = "cqrs" // filters test-bdd out of view
	if c.TotalChecked() != 1 {
		t.Errorf("selection lost under filter: TotalChecked=%d, want 1", c.TotalChecked())
	}
	c.ClearFilter()
	if c.TotalChecked() != 1 {
		t.Errorf("selection lost after clearing filter: %d", c.TotalChecked())
	}
}

func TestFilter_ToggleAllOnlyAffectsVisibleLeaves(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0
	c.Query = "bdd" // visible: test-bdd, bdd-scenario (2 of 8 in the tab)
	c.ToggleAllInActiveTab()
	if c.Tabs[0].CheckedCount() != 2 {
		t.Fatalf("toggle-all under filter marked %d, want only the 2 visible", c.Tabs[0].CheckedCount())
	}
	// The two checked must be exactly the filtered ones.
	for _, want := range []string{"test-bdd", "bdd-scenario"} {
		found := false
		for _, s := range c.Collect() {
			if s.ID == want {
				found = true
			}
		}
		if !found {
			t.Errorf("expected %q checked by filtered toggle-all", want)
		}
	}
	// Toggling again clears just those.
	c.ToggleAllInActiveTab()
	if c.Tabs[0].CheckedCount() != 0 {
		t.Errorf("second toggle-all should clear the visible leaves, left %d", c.Tabs[0].CheckedCount())
	}
}

func TestFilter_ToggleMarksFilteredLeaf(t *testing.T) {
	c := newTestCatalog()
	c.Active = 0
	c.StartFilter()
	for _, r := range "cqrs" {
		c.FilterInput(r)
	}
	c.ToggleCurrent() // cursor is on cqrs-command
	c.CommitFilter()
	c.ClearFilter()
	sel := c.Collect()
	if len(sel) != 1 || sel[0].ID != "cqrs-command" {
		t.Errorf("toggling a filtered leaf => %+v, want [cqrs-command]", sel)
	}
}
