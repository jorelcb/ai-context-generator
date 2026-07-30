package packagesource

import (
	"context"
	"strings"
	"testing"

	root "github.com/jorelcb/codify-og"
	"github.com/jorelcb/codify-og/internal/domain/catalog"
)

func TestAntigravitySkillSource_List_RestampsSkillsOnly(t *testing.T) {
	inner := NewEmbeddedSource(root.TemplatesFS, "test")
	src := NewAntigravitySkillSource(inner)

	got, err := src.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected skills")
	}
	for _, m := range got {
		if m.Target != catalog.TargetAntigravitySkill {
			t.Errorf("manifest %q has target %q, want antigravity-skill", m.ID, m.Target)
		}
		if m.Source.Kind != "embedded-antigravity" {
			t.Errorf("manifest %q source kind %q, want embedded-antigravity", m.ID, m.Source.Kind)
		}
	}
	// Hooks must be dropped — count antigravity skills equals embedded skills.
	innerList, _ := inner.List(context.Background())
	skillCount := 0
	for _, m := range innerList {
		if m.Target == catalog.TargetClaudeSkill {
			skillCount++
		}
	}
	if len(got) != skillCount {
		t.Errorf("expected %d skills (hooks dropped), got %d", skillCount, len(got))
	}
}

func TestAntigravitySkillSource_Fetch_AntigravityFrontmatter(t *testing.T) {
	inner := NewEmbeddedSource(root.TemplatesFS, "test")
	src := NewAntigravitySkillSource(inner)
	m := catalog.PackageManifest{ID: "ddd-entity", Target: catalog.TargetAntigravitySkill}

	content, err := src.Fetch(context.Background(), m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	skill := string(content.Files["SKILL.md"])
	if !strings.Contains(skill, "name: ddd-entity") {
		t.Errorf("missing name frontmatter: %q", skill[:min(len(skill), 80)])
	}
	// Antigravity frontmatter has no `user-invocable` (that's Claude's).
	if strings.Contains(skill, "user-invocable") {
		t.Errorf("antigravity skill should not have user-invocable frontmatter")
	}
	if len(skill) < 100 {
		t.Errorf("skill body too short: %d", len(skill))
	}
}

func TestAntigravitySkillSource_Fetch_RejectsNonSkill(t *testing.T) {
	src := NewAntigravitySkillSource(NewEmbeddedSource(root.TemplatesFS, "test"))
	_, err := src.Fetch(context.Background(), catalog.PackageManifest{ID: "x", Target: catalog.TargetClaudeHook})
	if err == nil {
		t.Fatal("expected error fetching a non-antigravity-skill target")
	}
}

func TestAntigravitySkillSource_Kind(t *testing.T) {
	if NewAntigravitySkillSource(NewEmbeddedSource(root.TemplatesFS, "t")).Kind() != "embedded-antigravity" {
		t.Error("Kind() should be embedded-antigravity")
	}
}
