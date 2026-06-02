package mcp

import (
	"testing"

	"github.com/jorelcb/codify/internal/infrastructure/sdd"
)

// TestLoadSpecGuides_AllStandardsAndLocales is the regression guard for the
// v3.0.0 MCP break: executeSpecs loaded spec templates from the legacy path
// templates/{locale}/spec (removed in v2.2.0), so LoadAll() errored before any
// LLM call and generate_specs / with_specs were dead at runtime — uncaught
// because no test exercised the MCP spec template path. This test drives the
// exact loader the MCP uses (loadSpecGuides) for every standard × locale and
// fails if the templates don't resolve.
func TestLoadSpecGuides_AllStandardsAndLocales(t *testing.T) {
	registry := sdd.NewDefaultRegistry()

	for _, stdID := range []string{"openspec", "spec-kit"} {
		standard, err := registry.Resolve(stdID, "", "")
		if err != nil {
			t.Fatalf("resolve standard %q: %v", stdID, err)
		}
		for _, locale := range []string{"en", "es"} {
			guides, err := loadSpecGuides(locale, standard)
			if err != nil {
				t.Fatalf("loadSpecGuides(locale=%s, standard=%s) error: %v", locale, stdID, err)
			}
			if len(guides) == 0 {
				t.Fatalf("loadSpecGuides(locale=%s, standard=%s) returned 0 guides", locale, stdID)
			}
			// Every loaded guide must carry the standard's per-standard output
			// file name (the bug also left these blank for the MCP path).
			for _, g := range guides {
				if g.OutputFileName == "" {
					t.Errorf("guide %q (locale=%s, standard=%s) has empty OutputFileName", g.Name, locale, stdID)
				}
			}
		}
	}
}

// TestLoadSpecGuides_DefaultStandard verifies the empty-standard path (what the
// MCP passes when sdd_standard is omitted) resolves to OpenSpec and loads.
func TestLoadSpecGuides_DefaultStandard(t *testing.T) {
	standard, err := sdd.NewDefaultRegistry().Resolve("", "", "")
	if err != nil {
		t.Fatalf("resolve default standard: %v", err)
	}
	if standard.ID() != "openspec" {
		t.Fatalf("default standard = %q, want openspec", standard.ID())
	}
	guides, err := loadSpecGuides("en", standard)
	if err != nil {
		t.Fatalf("loadSpecGuides(en, default) error: %v", err)
	}
	if len(guides) == 0 {
		t.Fatal("loadSpecGuides(en, default) returned 0 guides")
	}
}
