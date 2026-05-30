package catalog

import (
	"encoding/json"
	"testing"
)

// TestTargetConstants_StableValues fija los string values de los Target
// constants. Cualquier cambio acá implica una migración para usuarios con
// lockfiles persistidos — el unit test obliga a tomar la decisión de
// forma explícita en lugar de cambiar un literal silenciosamente.
func TestTargetConstants_StableValues(t *testing.T) {
	tests := []struct {
		got, want string
	}{
		{string(TargetClaudeSkill), "claude-skill"},
		{string(TargetClaudeHook), "claude-hook"},
		{string(TargetClaudePlugin), "claude-plugin"},
		{string(TargetClaudeSlash), "claude-slash-command"},
		{string(TargetAntigravityWF), "antigravity-workflow"},
		{string(TargetCodifySDDStandard), "codify-sdd-standard"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("Target value drift: got %q, want %q", tt.got, tt.want)
		}
	}
}

// TestScopeConstants_StableValues fija scope identifiers, mismo razonamiento
// que TestTargetConstants_StableValues.
func TestScopeConstants_StableValues(t *testing.T) {
	if string(ScopeWorkstation) != "workstation" {
		t.Errorf("ScopeWorkstation drift: got %q", string(ScopeWorkstation))
	}
	if string(ScopeProject) != "project" {
		t.Errorf("ScopeProject drift: got %q", string(ScopeProject))
	}
}

// TestPackageManifest_NoLocaleField confirma que el manifest no expone una
// dimensión Locale. Es una decisión deliberada (D.0 spike Gap #2): los
// registries no manejan i18n. Si alguien intenta agregar el field por
// reflejo, este test debería evolucionar a una conversación explícita.
func TestPackageManifest_NoLocaleField(t *testing.T) {
	m := PackageManifest{ID: "x"}
	// Round-trip JSON y verificar que el shape no contiene "Locale" /
	// "locale". Esto se rompe si el field se agrega de vuelta.
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	parsed := map[string]any{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for key := range parsed {
		if key == "Locale" || key == "locale" {
			t.Errorf("PackageManifest exposes %q field — locale must not be a manifest dimension (see ADR-0010 + D.0 spike)", key)
		}
	}
}

// TestPackageManifest_TargetMetadataIsExclusive valida la convención: para
// cualquier manifest válido, exactamente uno de los typed sub-structs
// (Claude / Gemini / SDD) está populado, según el Target. El struct no
// fuerza esto en runtime (todos son pointers nilable), pero el test deja
// la convención explícita para que cualquiera que escriba un constructor
// sepa el invariante.
func TestPackageManifest_TargetMetadataIsExclusive(t *testing.T) {
	cases := []struct {
		name     string
		manifest PackageManifest
		valid    bool
	}{
		{
			name: "claude-skill with Claude metadata",
			manifest: PackageManifest{
				ID: "x", Target: TargetClaudeSkill,
				Claude: &ClaudeMetadata{Triggers: []string{"a"}},
			},
			valid: true,
		},
		{
			name: "claude-skill with SDD metadata is invalid",
			manifest: PackageManifest{
				ID: "x", Target: TargetClaudeSkill,
				SDD: &SDDMetadata{},
			},
			valid: false,
		},
		{
			name: "sdd-standard with SDD metadata",
			manifest: PackageManifest{
				ID: "openspec", Target: TargetCodifySDDStandard,
				SDD: &SDDMetadata{TemplateDir: "openspec"},
			},
			valid: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ok := manifestMetadataMatchesTarget(c.manifest)
			if ok != c.valid {
				t.Errorf("manifestMetadataMatchesTarget = %v, want %v", ok, c.valid)
			}
		})
	}
}

// manifestMetadataMatchesTarget es un helper de validación usado por el
// test. Vive acá (en el _test.go) en lugar del struct file porque el
// struct mismo no fuerza el invariante en runtime — solo lo documenta.
// Si en el futuro queremos un Validate() público, este es el cuerpo.
func manifestMetadataMatchesTarget(m PackageManifest) bool {
	switch m.Target {
	case TargetClaudeSkill, TargetClaudeHook, TargetClaudePlugin, TargetClaudeSlash:
		return m.Claude != nil && m.SDD == nil
	case TargetCodifySDDStandard:
		return m.SDD != nil && m.Claude == nil
	case TargetAntigravityWF:
		// Antigravity no tiene typed metadata todavía — ningún sub-struct
		// debe estar populado.
		return m.Claude == nil && m.SDD == nil
	default:
		return true // Target desconocido: el validator a nivel install registry decidirá
	}
}

// TestPackageContent_SettingsFragmentIsRawMessage valida que el campo es
// json.RawMessage y no un []byte u otro tipo. La diferencia importa
// porque RawMessage preserva el shape JSON original cuando el manifest
// se serializa, mientras que []byte se base64-codea.
func TestPackageContent_SettingsFragmentIsRawMessage(t *testing.T) {
	c := PackageContent{
		SettingsFragment: json.RawMessage(`{"hooks":{"PreToolUse":[]}}`),
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Si SettingsFragment fuera []byte, la salida tendría base64. Como es
	// RawMessage, el JSON inline se preserva.
	out := string(data)
	if !contains(out, `"hooks"`) {
		t.Errorf("expected SettingsFragment to serialize inline, got: %s", out)
	}
}

// contains es un substring helper local (evita import de strings en este
// test pequeño).
func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
