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

// fakeRecorder captures Record calls.
type fakeRecorder struct {
	scope    catalog.Scope
	recorded []catalog.PackageManifest
}

func (f *fakeRecorder) Record(_ context.Context, scope catalog.Scope, installed []catalog.PackageManifest) error {
	f.scope = scope
	f.recorded = append(f.recorded, installed...)
	return nil
}

func TestCatalogService_Install_RecordsToLockfile(t *testing.T) {
	dir := t.TempDir()
	src := packagesource.NewEmbeddedSource(root.TemplatesFS, "test")
	reg := targetinstaller.NewRegistry(targetinstaller.NewClaudeInstallerWithRoots(dir, dir))
	rec := &fakeRecorder{}
	svc := command.NewCatalogService(src, reg).WithRecorder(rec)

	_, err := svc.Install(context.Background(), command.InstallRequest{
		Target: catalog.TargetClaudeSkill,
		IDs:    []string{"ddd-entity", "does-not-exist"},
		Scope:  catalog.ScopeProject,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	// Only the successfully-installed manifest is recorded (not the NotFound).
	if len(rec.recorded) != 1 || rec.recorded[0].ID != "ddd-entity" {
		t.Fatalf("expected only ddd-entity recorded, got %+v", rec.recorded)
	}
	if rec.scope != catalog.ScopeProject {
		t.Errorf("recorder scope: got %q, want project", rec.scope)
	}
	if rec.recorded[0].Target != catalog.TargetClaudeSkill {
		t.Errorf("recorded manifest carries the target: %+v", rec.recorded[0])
	}
}

// fakeReader serves a fixed recorded set (the lockfile).
type fakeReader struct{ recorded []catalog.InstalledPackage }

func (f *fakeReader) Recorded(_ context.Context, _ catalog.Scope) ([]catalog.InstalledPackage, error) {
	return f.recorded, nil
}

// fakeInstaller reports a fixed live set; routes every target to itself.
type fakeInstaller struct{ live []catalog.InstalledPackage }

func (f *fakeInstaller) Handles(catalog.Target) bool { return true }
func (f *fakeInstaller) Install(context.Context, catalog.PackageManifest, catalog.PackageContent, catalog.Scope) error {
	return nil
}

func (f *fakeInstaller) Uninstall(context.Context, catalog.PackageManifest, catalog.Scope) error {
	return nil
}

func (f *fakeInstaller) InstalledList(context.Context, catalog.Scope) ([]catalog.InstalledPackage, error) {
	return f.live, nil
}

type fakeRegistry struct{ inst catalog.TargetInstaller }

func (r fakeRegistry) For(catalog.Target) (catalog.TargetInstaller, error) { return r.inst, nil }

func TestCatalogService_Status_ReportsDrift(t *testing.T) {
	reader := &fakeReader{recorded: []catalog.InstalledPackage{
		{ID: "ddd-entity", Target: catalog.TargetClaudeSkill, Version: "2.3.0"},
		{ID: "linting", Target: catalog.TargetClaudeSkill, Version: "1.0.0"}, // removed by hand
	}}
	// Live state: only ddd-entity remains.
	reg := fakeRegistry{inst: &fakeInstaller{live: []catalog.InstalledPackage{
		{ID: "ddd-entity", Target: catalog.TargetClaudeSkill},
	}}}
	svc := command.NewCatalogService(nil, reg).WithReader(reader)

	report, err := svc.Status(context.Background(), catalog.ScopeProject)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if report.Scope != catalog.ScopeProject {
		t.Errorf("report scope: %q", report.Scope)
	}
	if len(report.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(report.Entries))
	}
	got := map[string]command.DriftStatus{}
	for _, e := range report.Entries {
		got[e.Package.ID] = e.Status
	}
	if got["ddd-entity"] != command.DriftInSync {
		t.Errorf("ddd-entity should be in-sync, got %q", got["ddd-entity"])
	}
	if got["linting"] != command.DriftMissing {
		t.Errorf("linting should be missing, got %q", got["linting"])
	}
	if report.Missing() != 1 {
		t.Errorf("Missing() = %d, want 1", report.Missing())
	}
}

func TestCatalogService_Status_NoReader_Errors(t *testing.T) {
	svc, _ := newService(t)
	if _, err := svc.Status(context.Background(), catalog.ScopeProject); err == nil {
		t.Fatal("expected error when no reader configured")
	}
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
