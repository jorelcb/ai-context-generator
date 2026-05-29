package packagesource

import (
	"context"
	"strings"
	"testing"

	root "github.com/jorelcb/codify"
	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/infrastructure/llm"
)

func findSkill(t *testing.T, src catalog.PackageSource, id string) catalog.PackageManifest {
	t.Helper()
	manifests, err := src.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, m := range manifests {
		if m.ID == id && m.Target == catalog.TargetClaudeSkill {
			return m
		}
	}
	t.Fatalf("skill %q not found", id)
	return catalog.PackageManifest{}
}

func TestPersonalizingSource_Fetch_AdaptsSkillViaLLM(t *testing.T) {
	inner := NewEmbeddedSource(root.TemplatesFS, "test")
	mock := llm.NewMockProvider()
	mock.Responses["ddd_entity"] = "---\nname: ddd-entity\n---\nADAPTED to my Go microservice with rich domain entities."

	src := NewPersonalizingSource(inner, mock, "Go microservice, DDD, rich domain", "en", "claude")
	m := findSkill(t, src, "ddd-entity")

	content, err := src.Fetch(context.Background(), m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	got := string(content.Files["SKILL.md"])
	if !strings.Contains(got, "ADAPTED") {
		t.Errorf("expected LLM-adapted content, got: %q", got)
	}

	// The provider was driven in skills mode with the project context.
	last := mock.LastCall()
	if last.Mode != "skills" {
		t.Errorf("expected Mode=skills, got %q", last.Mode)
	}
	if last.ProjectContext == "" {
		t.Error("expected project context to be passed to the LLM")
	}
	if len(last.TemplateGuides) != 1 || last.TemplateGuides[0].Name != "ddd_entity" {
		t.Errorf("expected the ddd_entity guide passed, got %+v", last.TemplateGuides)
	}
}

func TestPersonalizingSource_Fetch_DelegatesHooks(t *testing.T) {
	inner := NewEmbeddedSource(root.TemplatesFS, "test")
	mock := llm.NewMockProvider()
	src := NewPersonalizingSource(inner, mock, "ctx", "en", "claude")

	manifests, _ := src.List(context.Background())
	var hook catalog.PackageManifest
	for _, m := range manifests {
		if m.ID == "linting" && m.Target == catalog.TargetClaudeHook {
			hook = m
		}
	}
	content, err := src.Fetch(context.Background(), hook)
	if err != nil {
		t.Fatalf("Fetch hook: %v", err)
	}
	// Hooks are not personalized: scripts ship verbatim, LLM untouched.
	if _, ok := content.Files["lint.sh"]; !ok {
		t.Errorf("expected lint.sh delegated verbatim, got keys %v", keysOf(content.Files))
	}
	if len(mock.Calls) != 0 {
		t.Errorf("LLM should not be called for hooks, got %d calls", len(mock.Calls))
	}
}

func TestPersonalizingSource_Fetch_RequiresContext(t *testing.T) {
	inner := NewEmbeddedSource(root.TemplatesFS, "test")
	src := NewPersonalizingSource(inner, llm.NewMockProvider(), "", "en", "claude")
	m := findSkill(t, src, "ddd-entity")
	if _, err := src.Fetch(context.Background(), m); err == nil {
		t.Fatal("expected error when project context is empty")
	}
}

func TestPersonalizingSource_KindAndList(t *testing.T) {
	inner := NewEmbeddedSource(root.TemplatesFS, "test")
	src := NewPersonalizingSource(inner, llm.NewMockProvider(), "ctx", "en", "claude")
	if src.Kind() != "embedded+llm" {
		t.Errorf("Kind() = %q, want embedded+llm", src.Kind())
	}
	got, err := src.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) == 0 {
		t.Error("List should delegate to inner and return packages")
	}
}
