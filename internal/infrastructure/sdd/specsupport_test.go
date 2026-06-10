package sdd

import (
	"strings"
	"testing"
)

// TestSpecsReferenceSection cubre el helper compartido CLI↔MCP. Antes de
// extraerlo existían dos copias y la del MCP ya había divergido (omitía el
// prefijo de featureID en el layout agrupado) — este test fija el contrato
// para ambos consumidores.
func TestSpecsReferenceSection_FlatLayout(t *testing.T) {
	std := NewOpenSpecAdapter() // LayoutFlat

	section := SpecsReferenceSection("en", std, "")

	if !strings.Contains(section, "## Specifications") {
		t.Error("en locale should render the Specifications header")
	}
	for _, a := range std.BootstrapArtifacts() {
		want := "- `specs/" + a.FileName + "`"
		if !strings.Contains(section, want) {
			t.Errorf("flat layout should list %q, got:\n%s", want, section)
		}
	}
}

func TestSpecsReferenceSection_FeatureGroupedLayout(t *testing.T) {
	std := NewSpecKitAdapter() // LayoutFeatureGrouped

	section := SpecsReferenceSection("en", std, "my-feature")

	for _, a := range std.BootstrapArtifacts() {
		want := "- `specs/my-feature/" + a.FileName + "`"
		if !strings.Contains(section, want) {
			t.Errorf("grouped layout should prefix the feature id: want %q, got:\n%s", want, section)
		}
	}
}

func TestSpecsReferenceSection_SpanishHeader(t *testing.T) {
	std := NewOpenSpecAdapter()

	section := SpecsReferenceSection("es", std, "")

	if !strings.Contains(section, "## Especificaciones") {
		t.Errorf("es locale should render the Spanish header, got:\n%s", section)
	}
}
