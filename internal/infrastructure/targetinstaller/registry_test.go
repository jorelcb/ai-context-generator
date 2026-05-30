package targetinstaller

import (
	"testing"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

func TestRegistry_For(t *testing.T) {
	claude := NewClaudeInstallerWithRoots("", "")
	reg := NewRegistry(claude)

	got, err := reg.For(catalog.TargetClaudeSkill)
	if err != nil {
		t.Fatalf("For(claude-skill): %v", err)
	}
	if got != claude {
		t.Errorf("For(claude-skill) returned a different installer")
	}

	if _, err := reg.For(catalog.TargetCodifySDDStandard); err == nil {
		t.Error("expected error routing an unregistered target")
	}
}
