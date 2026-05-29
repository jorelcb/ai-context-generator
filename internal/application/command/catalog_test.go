package command_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	root "github.com/jorelcb/codify"
	"github.com/jorelcb/codify/internal/application/command"
	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/infrastructure/packagesource"
	"github.com/jorelcb/codify/internal/infrastructure/targetinstaller"
)

func newService(t *testing.T) (*command.CatalogService, string) {
	t.Helper()
	dir := t.TempDir()
	src := packagesource.NewEmbeddedSource(root.TemplatesFS, "test")
	reg := targetinstaller.NewRegistry(targetinstaller.NewClaudeInstallerWithRoots(dir, dir))
	return command.NewCatalogService(src, reg), dir
}

func TestCatalogService_Available_FiltersByTarget(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	all, err := svc.Available(ctx, "")
	if err != nil {
		t.Fatalf("Available(all): %v", err)
	}
	skills, err := svc.Available(ctx, catalog.TargetClaudeSkill)
	if err != nil {
		t.Fatalf("Available(skill): %v", err)
	}
	hooks, err := svc.Available(ctx, catalog.TargetClaudeHook)
	if err != nil {
		t.Fatalf("Available(hook): %v", err)
	}

	if len(skills) == 0 || len(hooks) == 0 {
		t.Fatalf("expected skills and hooks, got %d/%d", len(skills), len(hooks))
	}
	if len(skills)+len(hooks) != len(all) {
		t.Errorf("skill+hook (%d) should equal all (%d)", len(skills)+len(hooks), len(all))
	}
	for _, m := range skills {
		if m.Target != catalog.TargetClaudeSkill {
			t.Errorf("skill filter leaked %q", m.Target)
		}
	}
}

func TestCatalogService_Install_And_Installed(t *testing.T) {
	svc, dir := newService(t)
	ctx := context.Background()

	out, err := svc.Install(ctx, command.InstallRequest{
		Target: catalog.TargetClaudeSkill,
		IDs:    []string{"ddd-entity"},
		Scope:  catalog.ScopeProject,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(out.Installed) != 1 || out.Installed[0] != "ddd-entity" {
		t.Fatalf("expected ddd-entity installed, got %+v", out)
	}
	if len(out.NotFound) != 0 {
		t.Errorf("unexpected NotFound: %v", out.NotFound)
	}

	// File landed on disk.
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "ddd-entity", "SKILL.md")); err != nil {
		t.Errorf("SKILL.md not written: %v", err)
	}

	// Installed reflects it.
	installed, err := svc.Installed(ctx, catalog.TargetClaudeSkill, catalog.ScopeProject)
	if err != nil {
		t.Fatalf("Installed: %v", err)
	}
	found := false
	for _, p := range installed {
		if p.ID == "ddd-entity" {
			found = true
		}
	}
	if !found {
		t.Errorf("ddd-entity not reported installed, got %+v", installed)
	}
}

func TestCatalogService_Install_UnknownID_GoesToNotFound(t *testing.T) {
	svc, _ := newService(t)
	out, err := svc.Install(context.Background(), command.InstallRequest{
		Target: catalog.TargetClaudeSkill,
		IDs:    []string{"ddd-entity", "does-not-exist"},
		Scope:  catalog.ScopeProject,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(out.Installed) != 1 || out.Installed[0] != "ddd-entity" {
		t.Errorf("expected ddd-entity installed, got %+v", out.Installed)
	}
	if len(out.NotFound) != 1 || out.NotFound[0] != "does-not-exist" {
		t.Errorf("expected does-not-exist in NotFound, got %+v", out.NotFound)
	}
}

func TestCatalogService_Install_HookBundle(t *testing.T) {
	svc, dir := newService(t)
	out, err := svc.Install(context.Background(), command.InstallRequest{
		Target: catalog.TargetClaudeHook,
		IDs:    []string{"linting"},
		Scope:  catalog.ScopeProject,
	})
	if err != nil {
		t.Fatalf("Install hook: %v", err)
	}
	if len(out.Installed) != 1 {
		t.Fatalf("expected linting installed, got %+v", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "hooks", "lint.sh")); err != nil {
		t.Errorf("lint.sh not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "settings.json")); err != nil {
		t.Errorf("settings.json not written: %v", err)
	}
}
