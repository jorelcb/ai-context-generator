package catalog

import (
	"sort"
	"strings"
)

// Este archivo expone funciones puras que convierten entradas de los
// catálogos legacy (SkillCategory para skills, SkillCategory para hooks)
// en []PackageManifest. Es el primer puente desde el modelo declarativo
// estático hacia el modelo de packages unificado de ADR-0010.
//
// Los converters NO populan Version, SourceChecksum, ni Source — esos son
// concerns del PackageSource que produce el manifest (ver D.1.c). Mantener
// los converters libres de esa info los hace deterministas y testeables
// sin dependencies adicionales.
//
// Workflows quedan fuera de este archivo: tienen complejidad multi-target
// (el mismo template se instala como Claude skill o Antigravity workflow
// según el flag --target), y la conversión vive más naturalmente en
// EmbeddedSource (D.1.c) donde se conoce el target deseado.

// ManifestsFromSkillsCategory convierte una categoría de skills en un
// PackageManifest por cada template registrado en alguna de sus opciones.
//
// El catálogo actual de skills usa la siguiente jerarquía:
//
//	SkillCategory ("architecture", "testing", "conventions", ...)
//	  └── SkillOption ("clean-ddd", "neutral", ...)        — un preset
//	        └── TemplateMapping {file: guideName}          — los templates
//
// Cada entry de TemplateMapping es UN skill instalable. Esta función
// emite UN manifest por cada entry. El SkillOption.Name (preset) NO se
// preserva como dimensión del manifest porque los presets son una
// agrupación del catálogo legacy, no un primitivo del ecosystem destino.
//
// El orden de salida es estable: ordenado por (option.Name, guideName).
// Esto facilita tests deterministas y evita churn diff en el lockfile.
func ManifestsFromSkillsCategory(category *SkillCategory) []PackageManifest {
	if category == nil {
		return nil
	}
	var out []PackageManifest
	for _, opt := range category.Options {
		out = append(out, manifestsForSkillOption(opt)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// manifestsForSkillOption emite un manifest por cada template del option.
// Llave de orden interno: el guide name. Determinista a pesar del
// recorrido de map (Go no garantiza orden de iteración de maps).
func manifestsForSkillOption(opt SkillOption) []PackageManifest {
	guides := make([]string, 0, len(opt.TemplateMapping))
	for _, guide := range opt.TemplateMapping {
		guides = append(guides, guide)
	}
	sort.Strings(guides)

	out := make([]PackageManifest, 0, len(guides))
	for _, guideName := range guides {
		out = append(out, manifestForSkill(guideName))
	}
	return out
}

// manifestForSkill construye el manifest de UN skill identificado por su
// guide name (e.g., "ddd_entity"). Lee SkillMetadata para Description y
// Triggers — la única fuente de verdad sobre la metadata del skill es ese
// map, que se preserva durante la migración.
func manifestForSkill(guideName string) PackageManifest {
	meta := SkillMetadata[guideName] // zero-value si no existe — no es error
	id := normalizeID(guideName)
	return PackageManifest{
		ID:          id,
		Label:       humanLabel(id),
		Description: meta.Description,
		Target:      TargetClaudeSkill,
		Claude: &ClaudeMetadata{
			Triggers: meta.Triggers,
			// UserInvocable mirrors the default applied by
			// catalog.GenerateFrontmatter for the claude target. Cuando
			// migremos a SkillMetadata enriquecido (futuro), esto pasaría
			// a salir de ahí.
			UserInvocable: true,
		},
	}
}

// ManifestsFromHooksCategory convierte la categoría "hooks" en un
// manifest por cada bundle (linting, security-guardrails, etc.).
//
// Diferencia frente a skills: cada SkillOption es un BUNDLE entero — un
// directorio con N scripts + un fragmento de settings.json — no una
// colección de items individuales. Por eso emitimos UN manifest por
// option, no uno por template.
//
// HookEvents se deja vacío: la información de qué eventos hookea cada
// bundle vive en el fragmento settings.json del bundle (PostToolUse,
// PreToolUse, etc.), que se carga al fetchear el contenido — no en la
// metadata estática del catálogo. Cuando D.1.c (EmbeddedSource) o
// el ClaudeInstaller (D.3) lean ese fragmento, podrán enriquecer el
// manifest con HookEvents.
func ManifestsFromHooksCategory(category *SkillCategory) []PackageManifest {
	if category == nil {
		return nil
	}
	out := make([]PackageManifest, 0, len(category.Options))
	for _, opt := range category.Options {
		out = append(out, manifestForHook(opt))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// manifestForHook arma el manifest de un bundle de hooks.
// Lee HookMetadata para la description.
func manifestForHook(opt SkillOption) PackageManifest {
	meta := HookMetadata[opt.Name] // zero-value si no existe
	return PackageManifest{
		ID:          opt.Name,
		Label:       opt.Label, // los hooks ya traen un label legible
		Description: meta.Description,
		Target:      TargetClaudeHook,
		Claude:      &ClaudeMetadata{
			// HookEvents es responsabilidad del Source / Installer, no
			// del converter — vive en el fragmento settings.json del
			// bundle, no en la metadata estática.
		},
	}
}

// normalizeID convierte un guide name interno (con underscores) en un ID
// kebab-case usable como nombre de directorio + machine identifier.
// Refleja la misma convención usada por catalog.GenerateFrontmatter para
// preservar continuidad de IDs durante la migración a manifests.
func normalizeID(guideName string) string {
	return strings.ReplaceAll(guideName, "_", "-")
}

// humanLabel transforma un ID kebab-case en un label legible para humanos.
// "ddd-entity" → "DDD Entity". Heurística mínima sin dependencias externas:
// title-case por palabra con un set chico de uppercase abbreviations
// preservadas (DDD, BDD, CQRS, TDD, API, MCP).
//
// Este label se muestra en la UI del catalog. Si en el futuro se quiere
// localización/customización por package, el label puede sobrescribirse
// vía un campo nuevo en SkillMetadata o vía manifest del source remoto.
func humanLabel(id string) string {
	parts := strings.Split(id, "-")
	for i, p := range parts {
		if upper, ok := preservedUppercase[strings.ToLower(p)]; ok {
			parts[i] = upper
			continue
		}
		if len(p) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// preservedUppercase mapea acrónimos comunes que deben quedar en
// mayúsculas en el label. Lookup en lowercase para case-insensitivity.
var preservedUppercase = map[string]string{
	"ddd":  "DDD",
	"bdd":  "BDD",
	"cqrs": "CQRS",
	"tdd":  "TDD",
	"api":  "API",
	"mcp":  "MCP",
	"sdd":  "SDD",
	"llm":  "LLM",
}
