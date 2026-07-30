package command

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jorelcb/codify-og/internal/application/dto"
	"github.com/jorelcb/codify-og/internal/domain/service"
	"github.com/jorelcb/codify-og/internal/infrastructure/filesystem"
	"github.com/jorelcb/codify-og/internal/infrastructure/llm"
)

// padded keeps mock output above the validator length threshold.
const padded = "body padded long enough so output validators do not flag length warnings during the test run."

// TestGenerateSpec_PlacesArtifactsByDir pins the per-artifact path model:
// each file lands at OutputPath/<Dir with {feature} expanded>/<FileName>.
func TestGenerateSpec_PlacesArtifactsByDir(t *testing.T) {
	tmp := t.TempDir()
	mock := llm.NewMockProvider()
	mock.Responses = map[string]string{
		"speckit_constitution": "# Constitution " + padded,
		"speckit_spec":         "# Spec " + padded,
	}

	// Two artifacts: a project-level constitution under .specify/memory/ and a
	// feature-scoped spec under specs/<feature>/.
	artifacts := []service.SpecArtifact{
		{GuideName: "speckit_constitution", FileName: "constitution.md", Dir: ".specify/memory", Required: false, SkipIfExists: true},
		{GuideName: "speckit_spec", FileName: "spec.md", Dir: "specs/" + service.FeatureToken, Required: true},
	}
	cfg := &dto.SpecConfig{
		ProjectName:     "test",
		FromContextPath: tmp,
		OutputPath:      tmp,
		Locale:          "en",
		FeatureID:       "checkout",
		Artifacts:       artifacts,
	}
	guides := []service.TemplateGuide{
		{Name: "speckit_constitution", Content: "g", OutputFileName: "constitution.md"},
		{Name: "speckit_spec", Content: "g", OutputFileName: "spec.md"},
	}

	cmd := NewGenerateSpecCommand(mock, filesystem.NewFileWriter(), filesystem.NewDirectoryManager())
	result, err := cmd.Execute(context.Background(), cfg, "EXISTING CONTEXT", guides)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result.GeneratedFiles) != 2 {
		t.Fatalf("GeneratedFiles: got %d, want 2", len(result.GeneratedFiles))
	}
	if _, err := os.Stat(filepath.Join(tmp, ".specify", "memory", "constitution.md")); err != nil {
		t.Errorf("constitution not at .specify/memory/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "specs", "checkout", "spec.md")); err != nil {
		t.Errorf("spec not at specs/checkout/: %v", err)
	}

	last := mock.LastCall()
	if last.Mode != "spec" {
		t.Fatalf("Mode: got %q, want spec", last.Mode)
	}
	if last.ExistingContext != "EXISTING CONTEXT" {
		t.Fatalf("ExistingContext: got %q", last.ExistingContext)
	}
}

// TestGenerateSpec_SkipIfExists pins that a project-level artifact already on
// disk is left untouched on a per-feature re-run.
func TestGenerateSpec_SkipIfExists(t *testing.T) {
	tmp := t.TempDir()
	existing := "# hand-edited constitution, do not clobber\n"
	memDir := filepath.Join(tmp, ".specify", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "constitution.md"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	mock := llm.NewMockProvider()
	mock.Responses = map[string]string{"speckit_constitution": "# Regenerated " + padded}
	cfg := &dto.SpecConfig{
		ProjectName: "test", FromContextPath: tmp, OutputPath: tmp, Locale: "en", FeatureID: "x",
		Artifacts: []service.SpecArtifact{
			{GuideName: "speckit_constitution", FileName: "constitution.md", Dir: ".specify/memory", SkipIfExists: true},
		},
	}
	guides := []service.TemplateGuide{{Name: "speckit_constitution", Content: "g", OutputFileName: "constitution.md"}}

	cmd := NewGenerateSpecCommand(mock, filesystem.NewFileWriter(), filesystem.NewDirectoryManager())
	result, err := cmd.Execute(context.Background(), cfg, "ctx", guides)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(memDir, "constitution.md"))
	if string(got) != existing {
		t.Errorf("skip-if-exists violated; constitution was overwritten:\n%s", got)
	}
	if len(result.GeneratedFiles) != 0 {
		t.Errorf("skipped artifact must not be reported as generated, got %v", result.GeneratedFiles)
	}
}
