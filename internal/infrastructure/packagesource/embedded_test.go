package packagesource

import (
	"context"
	"strings"
	"testing"

	root "github.com/jorelcb/codify"
	"github.com/jorelcb/codify/internal/domain/catalog"
)

// TestEmbeddedSource_Kind congela el identificador.
func TestEmbeddedSource_Kind(t *testing.T) {
	src := NewEmbeddedSource(root.TemplatesFS, "test-version")
	if got := src.Kind(); got != "embedded" {
		t.Errorf("Kind() = %q, want %q", got, "embedded")
	}
}

// TestEmbeddedSource_List_HasSkillsAndHooks valida que List() compone
// ambas categorías. Útil regression detector si alguna se cae del flujo
// silenciosamente.
func TestEmbeddedSource_List_HasSkillsAndHooks(t *testing.T) {
	src := NewEmbeddedSource(root.TemplatesFS, "2.3.0")
	manifests, err := src.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	skillCount := 0
	hookCount := 0
	for _, m := range manifests {
		switch m.Target {
		case catalog.TargetClaudeSkill:
			skillCount++
		case catalog.TargetClaudeHook:
			hookCount++
		}
	}
	if skillCount == 0 {
		t.Error("expected at least one TargetClaudeSkill manifest, got 0")
	}
	if hookCount == 0 {
		t.Error("expected at least one TargetClaudeHook manifest, got 0")
	}
}

// TestEmbeddedSource_List_DecoratesEveryManifest confirma que cada
// manifest emitido tiene Source, Version y SourceChecksum populados.
// Sin esto la diferencia con los converters puros (D.1.b) sería
// imperceptible y el source no aporta valor.
func TestEmbeddedSource_List_DecoratesEveryManifest(t *testing.T) {
	src := NewEmbeddedSource(root.TemplatesFS, "2.3.0")
	manifests, err := src.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(manifests) == 0 {
		t.Fatal("List returned no manifests")
	}

	for _, m := range manifests {
		if m.Source.Kind != "embedded" {
			t.Errorf("manifest %q: Source.Kind = %q, want embedded", m.ID, m.Source.Kind)
		}
		if m.Version != "2.3.0" {
			t.Errorf("manifest %q: Version = %q, want 2.3.0", m.ID, m.Version)
		}
		if m.SourceChecksum == "" {
			t.Errorf("manifest %q: SourceChecksum is empty", m.ID)
		}
		if len(m.SourceChecksum) != 64 {
			t.Errorf("manifest %q: SourceChecksum length = %d, want 64 (sha256 hex)",
				m.ID, len(m.SourceChecksum))
		}
	}
}

// TestEmbeddedSource_List_ChecksumStable corre List() dos veces y
// confirma que los checksums son idénticos. Sin determinismo el lockfile
// (D.9) generaría diff espurios entre runs.
func TestEmbeddedSource_List_ChecksumStable(t *testing.T) {
	src := NewEmbeddedSource(root.TemplatesFS, "2.3.0")
	a, _ := src.List(context.Background())
	b, _ := src.List(context.Background())

	if len(a) != len(b) {
		t.Fatalf("count mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatalf("position %d: ID drift %q vs %q", i, a[i].ID, b[i].ID)
		}
		if a[i].SourceChecksum != b[i].SourceChecksum {
			t.Errorf("manifest %q: checksum drift %q vs %q",
				a[i].ID, a[i].SourceChecksum, b[i].SourceChecksum)
		}
	}
}

// TestEmbeddedSource_Fetch_Skill_ProducesRenderedSkillMd valida el
// path end-to-end de skills: List → pick uno → Fetch → SKILL.md
// renderizado con frontmatter Claude + body del template.
func TestEmbeddedSource_Fetch_Skill_ProducesRenderedSkillMd(t *testing.T) {
	src := NewEmbeddedSource(root.TemplatesFS, "2.3.0")
	manifests, _ := src.List(context.Background())

	var ddd *catalog.PackageManifest
	for i, m := range manifests {
		if m.ID == "ddd-entity" && m.Target == catalog.TargetClaudeSkill {
			ddd = &manifests[i]
			break
		}
	}
	if ddd == nil {
		t.Fatal("ddd-entity skill manifest not found")
	}

	content, err := src.Fetch(context.Background(), *ddd)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	skillMd, ok := content.Files["SKILL.md"]
	if !ok {
		t.Fatal("Files does not contain SKILL.md")
	}
	skill := string(skillMd)

	// Frontmatter check — formato Claude (---\n...---\n).
	if !strings.HasPrefix(skill, "---\n") {
		t.Errorf("SKILL.md should start with --- (frontmatter), got first 80 chars: %q", skill[:min(len(skill), 80)])
	}
	if !strings.Contains(skill, "name: ddd-entity") {
		t.Error("SKILL.md frontmatter missing 'name: ddd-entity'")
	}
	if !strings.Contains(skill, "user-invocable: true") {
		t.Error("SKILL.md frontmatter missing 'user-invocable: true'")
	}

	// Body check — debería estar el contenido del template (no solo
	// frontmatter).
	if len(skill) < 200 {
		t.Errorf("SKILL.md is suspiciously short: %d bytes", len(skill))
	}

	// SettingsFragment debe estar vacío para skills.
	if len(content.SettingsFragment) != 0 {
		t.Errorf("skills should have empty SettingsFragment, got %d bytes", len(content.SettingsFragment))
	}
}

// TestEmbeddedSource_Fetch_Hook_BundlesScriptsAndSettings valida que
// para un hook bundle Fetch separa scripts → Files y hooks.json →
// SettingsFragment.
func TestEmbeddedSource_Fetch_Hook_BundlesScriptsAndSettings(t *testing.T) {
	src := NewEmbeddedSource(root.TemplatesFS, "2.3.0")
	manifests, _ := src.List(context.Background())

	var linting *catalog.PackageManifest
	for i, m := range manifests {
		if m.ID == "linting" && m.Target == catalog.TargetClaudeHook {
			linting = &manifests[i]
			break
		}
	}
	if linting == nil {
		t.Fatal("linting hook manifest not found")
	}

	content, err := src.Fetch(context.Background(), *linting)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Esperamos al menos 1 script (lint.sh) en Files.
	if _, ok := content.Files["lint.sh"]; !ok {
		t.Errorf("expected lint.sh in Files, got keys: %v", keysOf(content.Files))
	}

	// hooks.json NO debe estar en Files — debe estar en SettingsFragment.
	if _, ok := content.Files["hooks.json"]; ok {
		t.Error("hooks.json should be in SettingsFragment, not Files")
	}

	// SettingsFragment debe contener un JSON con la key "hooks".
	if len(content.SettingsFragment) == 0 {
		t.Fatal("SettingsFragment is empty for hook bundle")
	}
	if !strings.Contains(string(content.SettingsFragment), "PostToolUse") {
		t.Errorf("SettingsFragment should contain 'PostToolUse', got: %s",
			string(content.SettingsFragment)[:min(len(content.SettingsFragment), 100)])
	}
}

// TestEmbeddedSource_Fetch_UnsupportedTarget devuelve error explícito
// para targets fuera de v0 scope (workflows, gemini, etc.).
func TestEmbeddedSource_Fetch_UnsupportedTarget(t *testing.T) {
	src := NewEmbeddedSource(root.TemplatesFS, "2.3.0")
	m := catalog.PackageManifest{
		ID:     "anything",
		Target: catalog.TargetCodifySDDStandard, // no soportado por EmbeddedSource (solo skill+hook)
	}
	_, err := src.Fetch(context.Background(), m)
	if err == nil {
		t.Fatal("expected error for unsupported target")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error should mention 'not supported', got: %v", err)
	}
}

// TestEmbeddedSource_Fetch_UnknownSkillID devuelve error claro cuando
// el manifest apunta a un skill que no existe en el catálogo embedded.
// Defensivo contra manifests stale en el lockfile (D.9).
func TestEmbeddedSource_Fetch_UnknownSkillID(t *testing.T) {
	src := NewEmbeddedSource(root.TemplatesFS, "2.3.0")
	m := catalog.PackageManifest{
		ID:     "phantom-skill",
		Target: catalog.TargetClaudeSkill,
	}
	_, err := src.Fetch(context.Background(), m)
	if err == nil {
		t.Fatal("expected error for unknown skill ID")
	}
}

// keysOf es un helper local para mensajes de error (evita ordenamiento
// inestable de map iteration).
func keysOf(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
