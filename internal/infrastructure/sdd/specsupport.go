package sdd

import (
	"path/filepath"

	"github.com/jorelcb/codify/internal/domain/service"
)

// SpecTemplatePath devuelve la ruta (en el embed FS) de las plantillas de spec
// del estándar activo: templates/{locale}/sdd/{TemplateDir}/spec.
//
// Centralizar esta ruta aquí evita que CLI y MCP deriven por su cuenta y se
// desincronicen: la regresión de v3.0.0 fue exactamente eso — el MCP cargaba
// la ruta legacy templates/{locale}/spec (eliminada en v2.2.0) mientras el CLI
// usaba la ruta SDD correcta. Con un único builder compartido, un cambio de
// layout se hace en un solo lugar.
func SpecTemplatePath(locale string, std service.SpecStandard) string {
	return filepath.Join("templates", locale, "sdd", std.TemplateDir(), "spec")
}

// SpecTemplateMapping construye el mapping {"<guide>.template" → "<guide>"} a
// partir de los bootstrap artifacts del estándar activo.
func SpecTemplateMapping(std service.SpecStandard) map[string]string {
	artifacts := std.BootstrapArtifacts()
	m := make(map[string]string, len(artifacts))
	for _, a := range artifacts {
		m[a.GuideName+".template"] = a.GuideName
	}
	return m
}

// ApplySpecOutputNames anota cada TemplateGuide con el nombre de archivo de
// salida específico del estándar (el mismo guide "spec" es "SPEC.md" en OpenSpec
// y "spec.md" en Spec-Kit).
func ApplySpecOutputNames(guides []service.TemplateGuide, std service.SpecStandard) []service.TemplateGuide {
	byGuide := make(map[string]string, len(std.BootstrapArtifacts()))
	for _, a := range std.BootstrapArtifacts() {
		byGuide[a.GuideName] = a.FileName
	}
	out := make([]service.TemplateGuide, 0, len(guides))
	for _, g := range guides {
		if name, ok := byGuide[g.Name]; ok {
			g.OutputFileName = name
		}
		out = append(out, g)
	}
	return out
}
