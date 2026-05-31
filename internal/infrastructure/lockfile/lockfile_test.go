package lockfile

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

func TestLoad_MissingFile_Empty(t *testing.T) {
	lf, err := Load(filepath.Join(t.TempDir(), "nope.lock"))
	if err != nil {
		t.Fatalf("Load missing should not error: %v", err)
	}
	if len(lf.Packages) != 0 || lf.Version != SchemaVersion {
		t.Errorf("expected empty v%d lockfile, got %+v", SchemaVersion, lf)
	}
}

func TestUpsertRemoveAndSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "project.lock")
	lf, _ := Load(path)
	lf.Scope = "project"
	lf.Upsert(Entry{ID: "ddd-entity", Target: "claude-skill", Version: "2.3.0", InstalledAt: "t1"})
	lf.Upsert(Entry{ID: "linting", Target: "claude-hook", InstalledAt: "t1"})
	// Replace ddd-entity (same id+target) — should not duplicate.
	lf.Upsert(Entry{ID: "ddd-entity", Target: "claude-skill", Version: "2.4.0", InstalledAt: "t2"})
	if len(lf.Packages) != 2 {
		t.Fatalf("expected 2 packages after upsert-replace, got %d", len(lf.Packages))
	}
	if err := lf.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Round-trip.
	got, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(got.Packages) != 2 {
		t.Fatalf("reloaded %d packages, want 2", len(got.Packages))
	}
	// Sorted by (target, id): claude-hook < claude-skill.
	if got.Packages[0].ID != "linting" || got.Packages[1].ID != "ddd-entity" {
		t.Errorf("packages not sorted: %+v", got.Packages)
	}
	if got.Packages[1].Version != "2.4.0" {
		t.Errorf("ddd-entity not replaced, version %q", got.Packages[1].Version)
	}

	if !got.Remove("linting", "claude-hook") {
		t.Error("Remove should report true")
	}
	if got.Remove("ghost", "claude-skill") {
		t.Error("Remove of absent entry should be false")
	}
	if len(got.Packages) != 1 {
		t.Errorf("after remove expected 1, got %d", len(got.Packages))
	}
}

func TestLoad_Malformed_Errors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.lock")
	_ = os.WriteFile(path, []byte("{not json"), 0o644)
	if _, err := Load(path); err == nil {
		t.Fatal("expected error on malformed lockfile")
	}
}

func TestPathForScope(t *testing.T) {
	wp, err := PathForScope(catalog.ScopeWorkstation)
	if err != nil {
		t.Fatalf("workstation: %v", err)
	}
	if !strings.HasSuffix(wp, filepath.Join(".codify", "workstation.lock")) {
		t.Errorf("workstation path: %q", wp)
	}
	pp, err := PathForScope(catalog.ScopeProject)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	if !strings.HasSuffix(pp, filepath.Join(".codify", "project.lock")) {
		t.Errorf("project path: %q", pp)
	}
	if _, err := PathForScope(catalog.Scope("nope")); err == nil {
		t.Error("expected error for unknown scope")
	}
}

func TestRecorder_RecordAndForget(t *testing.T) {
	dir := t.TempDir()
	fixedPath := func(catalog.Scope) (string, error) { return filepath.Join(dir, "project.lock"), nil }
	fixedNow := func() time.Time { return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC) }
	rec := NewRecorderWith(fixedPath, fixedNow)
	ctx := context.Background()

	manifests := []catalog.PackageManifest{
		{ID: "ddd-entity", Target: catalog.TargetClaudeSkill, Version: "2.3.0", Source: catalog.SourceRef{Kind: "embedded"}, SourceChecksum: "abc"},
		{ID: "gopls-lsp", Target: catalog.TargetClaudePlugin, Version: "1.0.0", Source: catalog.SourceRef{Kind: "claude-marketplace"}},
	}
	if err := rec.Record(ctx, catalog.ScopeProject, manifests); err != nil {
		t.Fatalf("Record: %v", err)
	}

	lf, _ := Load(filepath.Join(dir, "project.lock"))
	if len(lf.Packages) != 2 {
		t.Fatalf("expected 2 recorded, got %d", len(lf.Packages))
	}
	var ddd *Entry
	for i := range lf.Packages {
		if lf.Packages[i].ID == "ddd-entity" {
			ddd = &lf.Packages[i]
		}
	}
	if ddd == nil || ddd.Version != "2.3.0" || ddd.SourceKind != "embedded" || ddd.Checksum != "abc" {
		t.Errorf("ddd entry wrong: %+v", ddd)
	}
	if ddd.InstalledAt != "2026-05-31T12:00:00Z" {
		t.Errorf("installedAt not stamped from clock: %q", ddd.InstalledAt)
	}

	// Forget one.
	if err := rec.Forget(ctx, catalog.ScopeProject, []string{"ddd-entity"}, catalog.TargetClaudeSkill); err != nil {
		t.Fatalf("Forget: %v", err)
	}
	lf, _ = Load(filepath.Join(dir, "project.lock"))
	if len(lf.Packages) != 1 || lf.Packages[0].ID != "gopls-lsp" {
		t.Errorf("after forget expected only gopls-lsp, got %+v", lf.Packages)
	}
}

func TestRecorder_RecordEmpty_NoOp(t *testing.T) {
	dir := t.TempDir()
	rec := NewRecorderWith(func(catalog.Scope) (string, error) { return filepath.Join(dir, "x.lock"), nil }, time.Now)
	if err := rec.Record(context.Background(), catalog.ScopeProject, nil); err != nil {
		t.Fatalf("Record empty: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "x.lock")); !os.IsNotExist(err) {
		t.Error("empty Record should not create a lockfile")
	}
}
