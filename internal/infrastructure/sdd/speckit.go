package sdd

import (
	"github.com/jorelcb/codify-og/internal/domain/service"
)

// SpecKitAdapter implementa service.SpecStandard para el formato GitHub
// Spec-Kit (https://github.com/github/spec-kit). Es el DEFAULT desde v4.0.0.
//
// Diferencias clave frente a OpenSpec:
//
//   - Layout por feature: specs/<feature-id>/<file>.md.
//   - File names lowercase con guiones (spec.md, plan.md, data-model.md).
//   - Constitution SÍ existe: vive en .specify/memory/constitution.md (NO bajo
//     specs/<feature>/) y es el primer paso del flujo upstream — su Constitution
//     Check se gatea en plan.md (corrige audit SDD-2).
//   - Files opcionales adicionales: research.md, data-model.md, quickstart.md.
//
// alignedWith documenta la versión upstream contra la que se calibró el
// formato de los artefactos (spec/plan/tasks: user stories P1/P2/P3, FR-XXX,
// SC-XXX). Revisar en cada release de codify (ver Track 4.3).
//
// El adapter es stateless. El feature-id (subdir bajo specs/) lo provee el
// caller — para `codify spec`, el projectName slugificado.
type SpecKitAdapter struct{}

// NewSpecKitAdapter construye el adapter.
func NewSpecKitAdapter() *SpecKitAdapter {
	return &SpecKitAdapter{}
}

// alignedWith records the upstream version this adapter's artifact formats
// track. Bump when re-aligning (Track 4.3 process).
const specKitAlignedWith = "GitHub Spec-Kit v0.10.x (2026-06)"

// AlignedWith reports the upstream version this adapter's artifact formats
// track (Track 4.3 process). Optional interface, not part of SpecStandard.
func (SpecKitAdapter) AlignedWith() string { return specKitAlignedWith }

// ID returns "spec-kit".
func (SpecKitAdapter) ID() string { return "spec-kit" }

// DisplayName returns "GitHub Spec-Kit".
func (SpecKitAdapter) DisplayName() string { return "GitHub Spec-Kit" }

// BootstrapArtifacts returns the Spec-Kit canonical files.
//
// Required (3): spec, plan, tasks — el núcleo del workflow Spec-Kit, bajo
// specs/<feature-id>/.
// Optional (4): constitution (project-level, .specify/memory/, skip-if-exists),
// research, data_model, quickstart.
func (SpecKitAdapter) BootstrapArtifacts() []service.SpecArtifact {
	feat := "specs/" + service.FeatureToken
	return []service.SpecArtifact{
		// Project-level constitution: lives outside the feature dir, generated
		// once, never clobbered on a per-feature re-run.
		{GuideName: "speckit_constitution", FileName: "constitution.md", Dir: ".specify/memory", Required: false, SkipIfExists: true},
		{GuideName: "speckit_spec", FileName: "spec.md", Dir: feat, Required: true},
		{GuideName: "speckit_plan", FileName: "plan.md", Dir: feat, Required: true},
		{GuideName: "speckit_tasks", FileName: "tasks.md", Dir: feat, Required: true},
		{GuideName: "speckit_research", FileName: "research.md", Dir: feat, Required: false},
		{GuideName: "speckit_data_model", FileName: "data-model.md", Dir: feat, Required: false},
		{GuideName: "speckit_quickstart", FileName: "quickstart.md", Dir: feat, Required: false},
	}
}

// TemplateDir returns "spec-kit". Templates viven en
// templates/{locale}/sdd/spec-kit/spec/.
func (SpecKitAdapter) TemplateDir() string { return "spec-kit" }

// SystemPromptHints returns Spec-Kit-specific guidance. Reinforces the
// per-feature layout, lowercase names, the constitution location, and the
// upstream clarification marker (audit SDD-2/SDD-3).
func (SpecKitAdapter) SystemPromptHints(locale string) string {
	if locale == "es" {
		return `<sdd_standard_hints>
Estándar activo: GitHub Spec-Kit.
Convenciones obligatorias:
- Los nombres de archivo son lowercase con guiones (spec.md, plan.md, data-model.md — NUNCA mayúsculas).
- spec/plan/tasks viven bajo specs/<feature-id>/ — NUNCA en la raíz de specs/.
- La constitution del proyecto vive en .specify/memory/constitution.md (NO bajo specs/<feature-id>/); contiene principios no negociables y es contra lo que el "Constitution Check" de plan.md se gatea.
- Marca lo no resuelto con "[NEEDS CLARIFICATION: <pregunta>]" (máximo 3 por spec) — NO inventes; ese es el marcador del estándar.
- research.md, data-model.md y quickstart.md son opcionales: emítelos solo si el contexto los justifica.
</sdd_standard_hints>
`
	}
	return `<sdd_standard_hints>
Active standard: GitHub Spec-Kit.
Required conventions:
- File names are lowercase with hyphens (spec.md, plan.md, data-model.md — NEVER uppercase).
- spec/plan/tasks live under specs/<feature-id>/ — NEVER at the root of specs/.
- The project constitution lives at .specify/memory/constitution.md (NOT under specs/<feature-id>/); it holds non-negotiable principles and is what plan.md's "Constitution Check" gates against.
- Mark unresolved points with "[NEEDS CLARIFICATION: <question>]" (max 3 per spec) — do NOT invent; that is the standard's marker.
- research.md, data-model.md, and quickstart.md are optional — emit only if the context warrants them.
</sdd_standard_hints>
`
}
