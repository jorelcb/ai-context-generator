package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/infrastructure/desiredstate"
)

// TestCatalogApply_InstallsDesiredSkills exercises the full declarative path
// end-to-end with the embedded source (no network, no external CLI): write a
// desired-state file, run `catalog --apply`, and assert the skills land on disk.
func TestCatalogApply_InstallsDesiredSkills(t *testing.T) {
	ctx := context.Background()

	// Discover two real embedded skill IDs so the test isn't pinned to specific
	// catalog contents.
	avail, err := embeddedService().Available(ctx, catalog.TargetClaudeSkill)
	if err != nil {
		t.Fatalf("available skills: %v", err)
	}
	if len(avail) < 2 {
		t.Skipf("need >=2 embedded skills, got %d", len(avail))
	}
	id1, id2 := avail[0].ID, avail[1].ID

	dir := t.TempDir()
	t.Chdir(dir) // project scope resolves .codify / .claude under cwd

	// Author the desired state (what a user would commit).
	ds := &desiredstate.DesiredState{Scope: "project"}
	ds.Add(
		desiredstate.Package{Ecosystem: "claude", ID: id1, Type: "skill"},
		desiredstate.Package{Ecosystem: "claude", ID: id2, Type: "skill"},
	)
	path, err := desiredstate.PathForScope(catalog.ScopeProject)
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if err := desiredstate.Save(path, ds); err != nil {
		t.Fatalf("save desired-state: %v", err)
	}

	// Reproduce the environment from the file.
	if err := catalogApply(ctx, catalogParams{scope: "project", marketplace: defaultMarketplace}); err != nil {
		t.Fatalf("catalogApply: %v", err)
	}

	for _, id := range []string{id1, id2} {
		skillDir := filepath.Join(dir, ".claude", "skills", id)
		if info, err := os.Stat(skillDir); err != nil || !info.IsDir() {
			t.Errorf("skill %q not installed (expected dir %s): %v", id, skillDir, err)
		}
	}
}

// TestCatalogApply_EmptyStateIsNoOp verifies apply on a scope with no
// desired-state file does nothing and does not error.
func TestCatalogApply_EmptyStateIsNoOp(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := catalogApply(context.Background(), catalogParams{scope: "project"}); err != nil {
		t.Fatalf("apply on empty state should be a no-op, got: %v", err)
	}
}
