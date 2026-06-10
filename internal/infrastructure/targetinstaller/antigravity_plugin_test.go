package targetinstaller

import (
	"context"
	"strings"
	"testing"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

func agyPluginManifest() catalog.PackageManifest {
	return catalog.PackageManifest{
		ID:       "deployer",
		Target:   catalog.TargetAntigravityPlugin,
		Source:   catalog.SourceRef{Kind: "antigravity-marketplace", URI: "acme/plugins"},
		Metadata: map[string]string{catalog.MetaKeyMarketplace: "acme-tools"},
	}
}

func TestAntigravityPluginInstaller_Handles(t *testing.T) {
	a := NewAntigravityPluginInstaller()
	if !a.Handles(catalog.TargetAntigravityPlugin) {
		t.Error("should handle antigravity-plugin")
	}
	if a.Handles(catalog.TargetClaudePlugin) || a.Handles(catalog.TargetAntigravitySkill) {
		t.Error("should not handle other targets")
	}
}

// TestAntigravityPluginInstaller_InstallBlocked guards the R-8 contract: the
// stub fails loudly with the agy blocker explained (ADR-0012 §5), never
// pretending to install.
func TestAntigravityPluginInstaller_InstallBlocked(t *testing.T) {
	a := NewAntigravityPluginInstaller()
	err := a.Install(context.Background(), agyPluginManifest(), catalog.PackageContent{}, catalog.ScopeWorkstation)
	if err == nil {
		t.Fatal("install must be blocked until agy supports marketplace registration")
	}
	for _, want := range []string{"BLOCKED", "agy", "marketplace add", "deployer"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("blocked error should mention %q, got: %v", want, err)
		}
	}
}

func TestAntigravityPluginInstaller_UninstallBlocked(t *testing.T) {
	a := NewAntigravityPluginInstaller()
	if err := a.Uninstall(context.Background(), agyPluginManifest(), catalog.ScopeWorkstation); err == nil {
		t.Fatal("uninstall must be blocked too")
	}
}

func TestAntigravityPluginInstaller_InstalledList_Empty(t *testing.T) {
	a := NewAntigravityPluginInstaller()
	got, err := a.InstalledList(context.Background(), catalog.ScopeWorkstation)
	if err != nil {
		t.Fatalf("InstalledList should not error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("nothing can be installed, expected empty list, got %v", got)
	}
}
