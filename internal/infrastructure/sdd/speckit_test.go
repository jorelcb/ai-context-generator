package sdd

import (
	"strings"
	"testing"
)

func TestSpecKitAdapter_BasicContract(t *testing.T) {
	a := NewSpecKitAdapter()

	if a.ID() != "spec-kit" {
		t.Errorf("ID: got %q, want spec-kit", a.ID())
	}
	if a.DisplayName() == "" {
		t.Error("DisplayName must be non-empty")
	}
	if a.TemplateDir() != "spec-kit" {
		t.Errorf("TemplateDir: got %q, want spec-kit", a.TemplateDir())
	}
}

func TestSpecKitAdapter_BootstrapArtifacts(t *testing.T) {
	a := NewSpecKitAdapter()
	arts := a.BootstrapArtifacts()

	// Spec-Kit ships at minimum spec/plan/tasks (required) plus the optional
	// research/data-model/quickstart trio.
	if len(arts) < 3 {
		t.Fatalf("expected at least 3 artifacts, got %d", len(arts))
	}

	// Required ones: spec.md, plan.md, tasks.md.
	required := map[string]bool{}
	for _, art := range arts {
		if art.Required {
			required[art.FileName] = true
		}
	}
	for _, want := range []string{"spec.md", "plan.md", "tasks.md"} {
		if !required[want] {
			t.Errorf("expected required artifact %q, not found in required set %v", want, required)
		}
	}

	// Naming convention: lowercase with hyphens. ZERO uppercase. Validates
	// Spec-Kit's editorial constraint.
	for _, art := range arts {
		for _, r := range art.FileName {
			if r >= 'A' && r <= 'Z' {
				t.Errorf("Spec-Kit file names must be lowercase, got %q", art.FileName)
				break
			}
		}
	}
}

func TestSpecKitAdapter_ShipsProjectConstitution(t *testing.T) {
	a := NewSpecKitAdapter()
	var found bool
	for _, art := range a.BootstrapArtifacts() {
		if art.FileName != "constitution.md" {
			continue
		}
		found = true
		// The constitution is a project-level artifact: optional, lives under
		// .specify/memory/ (NOT specs/<feature>/), and must survive per-feature
		// re-runs (audit SDD-2 turned into a feature).
		if art.Required {
			t.Error("constitution.md should be optional, not required")
		}
		if !art.SkipIfExists {
			t.Error("constitution.md should be skip-if-exists (project-level, survives re-runs)")
		}
		if art.Dir != ".specify/memory" {
			t.Errorf("constitution.md Dir: got %q, want .specify/memory", art.Dir)
		}
	}
	if !found {
		t.Error("Spec-Kit must ship a constitution.md artifact (fixes the dangling Constitution Check)")
	}
}

func TestSpecKitAdapter_LifecycleWorkflowIDs(t *testing.T) {
	a := NewSpecKitAdapter()
	ids := a.LifecycleWorkflowIDs()

	if len(ids) == 0 {
		t.Fatal("Spec-Kit must declare lifecycle workflow IDs")
	}
	// Los workflows IDs son los slash commands (specify/plan/tasks).
	// Sus prefixes existen para no chocar con OpenSpec en el global mapping.
	for _, id := range ids {
		if !strings.HasPrefix(id, "speckit_") {
			t.Errorf("Spec-Kit workflow IDs should be namespaced (speckit_*), got %q", id)
		}
	}
}

func TestSpecKitAdapter_SystemPromptHints_MentionsKeyConventions(t *testing.T) {
	a := NewSpecKitAdapter()

	// The hints must remind the LLM of: lowercase names, the per-feature
	// layout, the constitution location (fixes SDD-2), and the upstream
	// clarification marker (SDD-3) — in both locales.
	for _, locale := range []string{"en", "es"} {
		hints := a.SystemPromptHints(locale)
		for _, want := range []string{"lowercase", "specs/<feature-id>/", ".specify/memory/constitution.md", "NEEDS CLARIFICATION"} {
			if !strings.Contains(hints, want) {
				t.Errorf("%s hints should mention %q, got: %s", locale, want, hints)
			}
		}
	}
}
