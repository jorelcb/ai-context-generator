// Package sdd contiene los adapters concretos del port SpecStandard
// (ver internal/domain/service/spec_standard.go y ADR-0011).
//
// Cada adapter encapsula el conocimiento específico de un estándar de
// Spec-Driven Development: file names, layout en disco, hints para el LLM,
// y workflows del lifecycle. La selección del adapter activo se resuelve
// en el wiring de la CLI según la precedencia documentada en ADR-0011.
package sdd

import (
	"github.com/jorelcb/codify/internal/domain/service"
)

// OpenSpecAdapter implementa service.SpecStandard para el formato OpenSpec
// (https://github.com/Fission-AI/OpenSpec). Genera la estructura REAL de
// OpenSpec sobre la que sus comandos y skills operan:
//
//   - openspec/project.md — contexto del proyecto (stack, convenciones).
//   - openspec/specs/<capability>/spec.md — capacidad con requirements en
//     formato `### Requirement:` + `#### Scenario:` (GIVEN/WHEN/THEN).
//
// alignedWith documenta la versión upstream contra la que se calibró el
// formato — revisar en cada release de codify (ver Track 4.3).
//
// La implementación es stateless. El template path completo lo arma el
// consumidor vía sdd.SpecTemplatePath.
type OpenSpecAdapter struct{}

// NewOpenSpecAdapter construye el adapter. No tiene dependencias externas.
func NewOpenSpecAdapter() *OpenSpecAdapter {
	return &OpenSpecAdapter{}
}

// alignedWith records the upstream version this adapter's format tracks.
// Bump when re-aligning to a newer OpenSpec release (Track 4.3 process).
const openSpecAlignedWith = "OpenSpec v1.x (Fission-AI/OpenSpec, 2026-06)"

// AlignedWith reports the upstream version this adapter's format tracks
// (Track 4.3 process). Consumed via an optional interface — not part of the
// SpecStandard contract — so callers print it when available.
func (OpenSpecAdapter) AlignedWith() string { return openSpecAlignedWith }

// ID returns the stable identifier "openspec".
func (OpenSpecAdapter) ID() string { return "openspec" }

// DisplayName returns "OpenSpec".
func (OpenSpecAdapter) DisplayName() string { return "OpenSpec" }

// BootstrapArtifacts returns the real OpenSpec structure: a project context
// file at the openspec/ root plus one capability spec under
// openspec/specs/<capability>/. The capability slug fills {feature}.
func (OpenSpecAdapter) BootstrapArtifacts() []service.SpecArtifact {
	return []service.SpecArtifact{
		{GuideName: "openspec_project", FileName: "project.md", Dir: "openspec", Required: true},
		{GuideName: "openspec_spec", FileName: "spec.md", Dir: "openspec/specs/" + service.FeatureToken, Required: true},
	}
}

// TemplateDir returns "openspec". Templates live at
// templates/{locale}/sdd/openspec/spec/.
func (OpenSpecAdapter) TemplateDir() string { return "openspec" }

// SystemPromptHints returns OpenSpec's literal requirement/scenario syntax so
// the LLM emits specs that `openspec validate` accepts (audit SDD-7).
func (OpenSpecAdapter) SystemPromptHints(locale string) string {
	if locale == "es" {
		return `<sdd_standard_hints>
Estándar activo: OpenSpec.
Convenciones obligatorias de formato:
- La estructura raíz es openspec/: project.md (contexto) y specs/<capability>/spec.md (capacidades).
- Cada requirement usa exactamente este encabezado: "### Requirement: <Nombre>" seguido de una frase con lenguaje SHALL/MUST (RFC 2119).
- Cada requirement incluye al menos un escenario: "#### Scenario: <Nombre>" con viñetas GIVEN / WHEN / THEN.
- Para cambios futuros, los deltas usan "## ADDED Requirements", "## MODIFIED Requirements", "## REMOVED Requirements", "## RENAMED Requirements"; en MODIFIED indica "(Previously: <texto anterior>)".
- NO uses nombres de archivo en mayúsculas ni un layout plano specs/<FILE>.md — eso es el formato legacy, eliminado.
</sdd_standard_hints>
`
	}
	return `<sdd_standard_hints>
Active standard: OpenSpec.
Required format conventions:
- The root structure is openspec/: project.md (context) and specs/<capability>/spec.md (capabilities).
- Every requirement uses exactly this header: "### Requirement: <Name>" followed by a SHALL/MUST sentence (RFC 2119 language).
- Every requirement includes at least one scenario: "#### Scenario: <Name>" with GIVEN / WHEN / THEN bullets.
- For future changes, deltas use "## ADDED Requirements", "## MODIFIED Requirements", "## REMOVED Requirements", "## RENAMED Requirements"; in MODIFIED note "(Previously: <prior text>)".
- Do NOT use uppercase file names or a flat specs/<FILE>.md layout — that is the removed legacy format.
</sdd_standard_hints>
`
}
