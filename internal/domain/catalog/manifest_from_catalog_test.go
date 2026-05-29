package catalog

import (
	"sort"
	"testing"
)

// TestManifestsFromSkillsCategory_ProducesOneManifestPerTemplate valida la
// regla central de la conversión skills: cada entry en TemplateMapping
// genera UN manifest. Si una category tiene 3 options con 2 templates
// cada una, el resultado son 6 manifests.
func TestManifestsFromSkillsCategory_ProducesOneManifestPerTemplate(t *testing.T) {
	// Usamos la category real "architecture" del catálogo embedded para
	// validar contra datos reales. El test se rompe si alguien remueve
	// templates sin cuidado — útil como regression detector.
	cat, err := FindCategory("architecture")
	if err != nil {
		t.Fatalf("FindCategory: %v", err)
	}

	manifests := ManifestsFromSkillsCategory(cat)
	if len(manifests) == 0 {
		t.Fatal("expected at least one manifest")
	}

	// Conteo esperado: suma de len(option.TemplateMapping) sobre todas
	// las options.
	expected := 0
	for _, opt := range cat.Options {
		expected += len(opt.TemplateMapping)
	}
	if len(manifests) != expected {
		t.Errorf("got %d manifests, want %d (sum of TemplateMapping sizes)",
			len(manifests), expected)
	}
}

// TestManifestsFromSkillsCategory_AllAreClaudeSkill valida el Target:
// todo skill convertido aterriza como TargetClaudeSkill (las skills hoy
// son Claude-native; multi-target shipping queda para futuro).
func TestManifestsFromSkillsCategory_AllAreClaudeSkill(t *testing.T) {
	cat, _ := FindCategory("architecture")
	manifests := ManifestsFromSkillsCategory(cat)
	for _, m := range manifests {
		if m.Target != TargetClaudeSkill {
			t.Errorf("manifest %q: got Target %q, want %q", m.ID, m.Target, TargetClaudeSkill)
		}
		if m.Claude == nil {
			t.Errorf("manifest %q: ClaudeMetadata is nil for claude-skill target", m.ID)
		}
	}
}

// TestManifestsFromSkillsCategory_LosslessMetadata valida que la
// description y los triggers de SkillMetadata se preservan en el manifest.
// Sin esto, la conversión sería destructiva.
func TestManifestsFromSkillsCategory_LosslessMetadata(t *testing.T) {
	cat, _ := FindCategory("architecture")
	manifests := ManifestsFromSkillsCategory(cat)

	// Buscamos un guide conocido (ddd_entity) — debe estar en la category.
	var found *PackageManifest
	for i, m := range manifests {
		if m.ID == "ddd-entity" {
			found = &manifests[i]
			break
		}
	}
	if found == nil {
		t.Fatal("ddd-entity manifest not found in conversion output")
	}

	expectedMeta := SkillMetadata["ddd_entity"]
	if found.Description != expectedMeta.Description {
		t.Errorf("description drift: got %q, want %q",
			found.Description, expectedMeta.Description)
	}
	if !equalStringSlices(found.Claude.Triggers, expectedMeta.Triggers) {
		t.Errorf("triggers drift: got %v, want %v",
			found.Claude.Triggers, expectedMeta.Triggers)
	}
}

// TestManifestsFromSkillsCategory_StableOrdering valida que dos llamadas
// produzcan el mismo orden, a pesar del recorrido de map (Go no garantiza
// orden de iteración). Necesario para tests deterministas y para que el
// lockfile no genere diff espurios entre regeneraciones.
func TestManifestsFromSkillsCategory_StableOrdering(t *testing.T) {
	cat, _ := FindCategory("architecture")
	a := ManifestsFromSkillsCategory(cat)
	b := ManifestsFromSkillsCategory(cat)
	if len(a) != len(b) {
		t.Fatalf("length differs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Errorf("position %d: got %q, want %q (ordering not stable)", i, b[i].ID, a[i].ID)
		}
	}
}

// TestManifestsFromSkillsCategory_NilCategory valida el guard.
func TestManifestsFromSkillsCategory_NilCategory(t *testing.T) {
	if got := ManifestsFromSkillsCategory(nil); got != nil {
		t.Errorf("nil category should return nil, got %v", got)
	}
}

// TestManifestsFromHooksCategory_OneManifestPerBundle valida que cada
// SkillOption (bundle) produzca UN manifest, no uno por archivo dentro
// del bundle. Es la diferencia clave frente a skills.
func TestManifestsFromHooksCategory_OneManifestPerBundle(t *testing.T) {
	cat, err := FindHookCategory("hooks")
	if err != nil {
		t.Fatalf("FindHookCategory: %v", err)
	}

	manifests := ManifestsFromHooksCategory(cat)
	if len(manifests) != len(cat.Options) {
		t.Errorf("got %d manifests, want %d (one per bundle option)",
			len(manifests), len(cat.Options))
	}

	for _, m := range manifests {
		if m.Target != TargetClaudeHook {
			t.Errorf("manifest %q: got Target %q, want %q",
				m.ID, m.Target, TargetClaudeHook)
		}
		if m.Claude == nil {
			t.Errorf("manifest %q: ClaudeMetadata nil for claude-hook target", m.ID)
		}
	}
}

// TestManifestsFromHooksCategory_PreservesLabels valida que el Label
// del bundle (más legible que el ID) se preserva. Critical UX en el
// catalog UI.
func TestManifestsFromHooksCategory_PreservesLabels(t *testing.T) {
	cat, _ := FindHookCategory("hooks")
	manifests := ManifestsFromHooksCategory(cat)

	for _, m := range manifests {
		if m.Label == "" {
			t.Errorf("manifest %q: Label is empty", m.ID)
		}
	}
}

// TestHumanLabel_PreservesAcronyms valida que humanLabel mantenga
// acrónimos comunes en mayúscula. "ddd-entity" → "DDD Entity",
// no "Ddd Entity".
func TestHumanLabel_PreservesAcronyms(t *testing.T) {
	tests := []struct {
		id, want string
	}{
		{"ddd-entity", "DDD Entity"},
		{"clean-arch-layer", "Clean Arch Layer"},
		{"bdd-scenario", "BDD Scenario"},
		{"cqrs-command", "CQRS Command"},
		{"api-design", "API Design"},
		{"test-tdd", "Test TDD"},
		{"single", "Single"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := humanLabel(tt.id); got != tt.want {
			t.Errorf("humanLabel(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

// TestNormalizeID_UnderscoresToDashes lock simple convertion contract.
func TestNormalizeID_UnderscoresToDashes(t *testing.T) {
	tests := []struct{ in, want string }{
		{"ddd_entity", "ddd-entity"},
		{"single", "single"},
		{"a_b_c", "a-b-c"},
		{"already-dashed", "already-dashed"},
	}
	for _, tt := range tests {
		if got := normalizeID(tt.in); got != tt.want {
			t.Errorf("normalizeID(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestManifestMatchesTargetMetadata_ConvertersProduceValidManifests cierra
// el círculo: cada manifest emitido por los converters cumple el invariante
// de exclusividad target ↔ sub-struct (definido en package_manifest_test.go).
// Sin esto, los converters podrían emitir manifests internamente válidos
// pero inconsistentes con la convención del struct.
func TestConverters_ProduceManifestsConsistentWithTargetMetadataInvariant(t *testing.T) {
	skillsCat, _ := FindCategory("architecture")
	hooksCat, _ := FindHookCategory("hooks")

	all := append([]PackageManifest{}, ManifestsFromSkillsCategory(skillsCat)...)
	all = append(all, ManifestsFromHooksCategory(hooksCat)...)

	for _, m := range all {
		if !manifestMetadataMatchesTarget(m) {
			t.Errorf("manifest %q (target %q) violates target↔metadata exclusivity invariant",
				m.ID, m.Target)
		}
	}
}

// equalStringSlices compara contenido sin importar orden — útil porque
// SkillMetadata.Triggers es declarado en orden arbitrario y Triggers
// del manifest no se reordena.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string{}, a...)
	bc := append([]string{}, b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}
