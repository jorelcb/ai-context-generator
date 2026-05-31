package targetinstaller

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	root "github.com/jorelcb/codify"
	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/infrastructure/packagesource"
)

func newAntigravity(t *testing.T) (*AntigravityInstaller, string) {
	t.Helper()
	dir := t.TempDir()
	return NewAntigravityInstallerWithRoots(dir, dir), dir
}

func TestAntigravityInstaller_Handles(t *testing.T) {
	a := NewAntigravityInstallerWithRoots("", "")
	if !a.Handles(catalog.TargetAntigravitySkill) {
		t.Error("should handle antigravity-skill")
	}
	if a.Handles(catalog.TargetClaudeSkill) {
		t.Error("should not handle claude-skill")
	}
}

func TestAntigravityInstaller_Install_Workstation(t *testing.T) {
	a, dir := newAntigravity(t)
	m := catalog.PackageManifest{ID: "ddd-entity", Target: catalog.TargetAntigravitySkill}
	content := catalog.PackageContent{Files: map[string][]byte{"SKILL.md": []byte("---\nname: ddd-entity\n---\nbody")}}

	if err := a.Install(context.Background(), m, content, catalog.ScopeWorkstation); err != nil {
		t.Fatalf("Install: %v", err)
	}
	// Flat file under ~/.gemini/antigravity-cli/skills/<id>.md
	got, err := os.ReadFile(filepath.Join(dir, ".gemini", "antigravity-cli", "skills", "ddd-entity.md"))
	if err != nil {
		t.Fatalf("read installed skill: %v", err)
	}
	if !strings.Contains(string(got), "name: ddd-entity") {
		t.Errorf("content wrong: %q", got)
	}
}

func TestAntigravityInstaller_Install_Project(t *testing.T) {
	a, dir := newAntigravity(t)
	m := catalog.PackageManifest{ID: "bdd-scenario", Target: catalog.TargetAntigravitySkill}
	content := catalog.PackageContent{Files: map[string][]byte{"SKILL.md": []byte("x")}}
	if err := a.Install(context.Background(), m, content, catalog.ScopeProject); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".agents", "skills", "bdd-scenario.md")); err != nil {
		t.Errorf("project skill not written: %v", err)
	}
}

func TestAntigravityInstaller_Install_MissingContent(t *testing.T) {
	a, _ := newAntigravity(t)
	err := a.Install(context.Background(), catalog.PackageManifest{ID: "x", Target: catalog.TargetAntigravitySkill}, catalog.PackageContent{}, catalog.ScopeProject)
	if err == nil {
		t.Fatal("expected error when content has no SKILL.md")
	}
}

func TestAntigravityInstaller_UninstallAndList(t *testing.T) {
	a, _ := newAntigravity(t)
	ctx := context.Background()
	content := catalog.PackageContent{Files: map[string][]byte{"SKILL.md": []byte("x")}}
	_ = a.Install(ctx, catalog.PackageManifest{ID: "s1", Target: catalog.TargetAntigravitySkill}, content, catalog.ScopeWorkstation)
	_ = a.Install(ctx, catalog.PackageManifest{ID: "s2", Target: catalog.TargetAntigravitySkill}, content, catalog.ScopeWorkstation)

	list, err := a.InstalledList(ctx, catalog.ScopeWorkstation)
	if err != nil {
		t.Fatalf("InstalledList: %v", err)
	}
	if len(list) != 2 || list[0].ID != "s1" || list[1].ID != "s2" {
		t.Fatalf("unexpected list: %+v", list)
	}

	if err := a.Uninstall(ctx, catalog.PackageManifest{ID: "s1", Target: catalog.TargetAntigravitySkill}, catalog.ScopeWorkstation); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	list, _ = a.InstalledList(ctx, catalog.ScopeWorkstation)
	if len(list) != 1 || list[0].ID != "s2" {
		t.Errorf("after uninstall expected only s2, got %+v", list)
	}
	// Idempotent.
	if err := a.Uninstall(ctx, catalog.PackageManifest{ID: "s1", Target: catalog.TargetAntigravitySkill}, catalog.ScopeWorkstation); err != nil {
		t.Errorf("second uninstall should be no-op: %v", err)
	}
}

// End-to-end: AntigravitySkillSource.Fetch → AntigravityInstaller.Install,
// proving the cross-ecosystem read/write pair works for Antigravity skills.
func TestAntigravity_EndToEnd(t *testing.T) {
	ctx := context.Background()
	src := packagesource.NewAntigravitySkillSource(packagesource.NewEmbeddedSource(root.TemplatesFS, "test"))
	manifests, err := src.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var m *catalog.PackageManifest
	for i := range manifests {
		if manifests[i].ID == "ddd-entity" {
			m = &manifests[i]
			break
		}
	}
	if m == nil {
		t.Fatal("ddd-entity not found")
	}

	content, err := src.Fetch(ctx, *m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	a, dir := newAntigravity(t)
	if err := a.Install(ctx, *m, content, catalog.ScopeProject); err != nil {
		t.Fatalf("Install: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, ".agents", "skills", "ddd-entity.md"))
	if err != nil {
		t.Fatalf("read e2e skill: %v", err)
	}
	if !strings.Contains(string(got), "name: ddd-entity") || strings.Contains(string(got), "user-invocable") {
		t.Errorf("e2e content should be antigravity-framed: %q", got[:min(len(got), 100)])
	}
}
