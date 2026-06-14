package sdd

import (
	"strings"
	"testing"
)

// TestSpecsReferenceSection cubre el helper compartido CLI↔MCP. Deriva los
// paths del mismo modelo per-artifact (Dir + {feature}) que usa el writer,
// así la lista siempre coincide con lo que se generó.
func TestSpecsReferenceSection_OpenSpec(t *testing.T) {
	std := NewOpenSpecAdapter()

	section := SpecsReferenceSection("en", std, "payments")

	if !strings.Contains(section, "## Specifications") {
		t.Error("en locale should render the Specifications header")
	}
	for _, want := range []string{"- `openspec/project.md`", "- `openspec/specs/payments/spec.md`"} {
		if !strings.Contains(section, want) {
			t.Errorf("OpenSpec section should list %q, got:\n%s", want, section)
		}
	}
}

func TestSpecsReferenceSection_SpecKit(t *testing.T) {
	std := NewSpecKitAdapter()

	section := SpecsReferenceSection("en", std, "my-feature")

	for _, want := range []string{
		"- `.specify/memory/constitution.md`",
		"- `specs/my-feature/spec.md`",
		"- `specs/my-feature/tasks.md`",
	} {
		if !strings.Contains(section, want) {
			t.Errorf("Spec-Kit section should list %q, got:\n%s", want, section)
		}
	}
}

func TestSpecsReferenceSection_SpanishHeader(t *testing.T) {
	std := NewOpenSpecAdapter()

	section := SpecsReferenceSection("es", std, "payments")

	if !strings.Contains(section, "## Especificaciones") {
		t.Errorf("es locale should render the Spanish header, got:\n%s", section)
	}
}

// TestArtifactPath pins the {feature}-token resolution used everywhere paths
// are computed (writer + AGENTS.md reference).
func TestArtifactPath(t *testing.T) {
	for _, std := range []interface{ ID() string }{NewSpecKitAdapter(), NewOpenSpecAdapter()} {
		_ = std
	}
	specKit := NewSpecKitAdapter()
	for _, a := range specKit.BootstrapArtifacts() {
		got := ArtifactPath(a, "checkout")
		if strings.Contains(got, "{feature}") {
			t.Errorf("unresolved {feature} token in %q", got)
		}
		if a.FileName == "constitution.md" && got != ".specify/memory/constitution.md" {
			t.Errorf("constitution path: got %q", got)
		}
		if a.FileName == "spec.md" && got != "specs/checkout/spec.md" {
			t.Errorf("spec path: got %q", got)
		}
	}
}
