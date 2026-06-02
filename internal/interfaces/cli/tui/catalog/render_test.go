package catalog

import (
	"strings"
	"testing"
)

// withDescriptions returns the test catalog enriched with descriptions, so the
// rendered frames look like the real thing.
func withDescriptions() *Catalog {
	c := newTestCatalog()
	desc := map[string]string{
		"ddd-entity":          "Model domain entities & value objects",
		"hexagonal-port":      "Ports & adapters",
		"clean-arch-layer":    "Layer placement & wiring",
		"cqrs-command":        "Command/query separation",
		"test-bdd":            "BDD method (Three Amigos, Example Mapping)",
		"bdd-scenario":        "Write Gherkin feature files",
		"conventional-commit": "Conventional Commits 1.0.0",
		"semantic-versioning": "SemVer 2.0.0 bumps",
		"spec-driven-change":  "OpenSpec: propose / apply / archive",
		"release-cycle":       "version bump + changelog + tag",
		"bug-fix":             "structured bug-fix workflow",
	}
	for ti := range c.Tabs {
		for ci := range c.Tabs[ti].Categories {
			for li := range c.Tabs[ti].Categories[ci].Leaves {
				id := c.Tabs[ti].Categories[ci].Leaves[li].ID
				c.Tabs[ti].Categories[ci].Leaves[li].Desc = desc[id]
			}
		}
	}
	return c
}

func TestView_RendersTabsTreeAndState(t *testing.T) {
	c := withDescriptions()
	c.Active = 0
	c.Tabs[0].Categories[0].Expanded = true
	c.markByID("ddd-entity")
	c.markByID("clean-arch-layer")
	m := NewModel(c)
	out := m.View()

	for _, want := range []string{
		"Skills", "Hooks", "Plugins", // tabs
		"architecture",       // category
		"[~]",                // partial parent (2/4 marked)
		"[x] ddd-entity",     // checked leaf
		"[ ] hexagonal-port", // unchecked leaf
		"(2/4)",              // counter
		"2 elegidos",         // header total
		"instalar",           // footer
	} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q\n---\n%s", want, out)
		}
	}
}

func TestView_FlatTabHidesCategoryHeader(t *testing.T) {
	c := withDescriptions()
	c.Active = 2 // Plugins (flat)
	out := NewModel(c).View()
	if strings.Contains(out, "▸") || strings.Contains(out, "▾") {
		t.Errorf("flat tab should not render expand arrows:\n%s", out)
	}
	if !strings.Contains(out, "spec-driven-change") {
		t.Errorf("flat tab should list leaves directly:\n%s", out)
	}
}

// TestRenderFrames logs real frames (run with NO_COLOR=1 -v to read them as
// plain text). Not an assertion test — a living preview of the UX.
func TestRenderFrames(t *testing.T) {
	// Frame A: initial, Skills collapsed.
	a := withDescriptions()
	t.Logf("\n--- Frame A · Skills, colapsado ---\n%s\n", NewModel(a).View())

	// Frame B: architecture expanded, 2/4 checked, cursor on a leaf.
	b := withDescriptions()
	b.Active = 0
	b.Tabs[0].Categories[0].Expanded = true
	b.markByID("ddd-entity")
	b.markByID("clean-arch-layer")
	b.Cursor = 2
	t.Logf("\n--- Frame B · architecture expandido, 2/4, padre [~] ---\n%s\n", NewModel(b).View())

	// Frame C: switched to Plugins, one checked there, Skills still shows (2).
	cc := withDescriptions()
	cc.Active = 0
	cc.markByID("ddd-entity")
	cc.markByID("clean-arch-layer")
	cc.Active = 2
	cc.markByID("spec-driven-change")
	cc.Cursor = 0
	t.Logf("\n--- Frame C · tab Plugins, acumulado cross-tab (Skills sigue en (2)) ---\n%s\n", NewModel(cc).View())
}
