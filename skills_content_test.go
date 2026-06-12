package codify

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// Estas pruebas son las "fitness functions" del CONTENIDO del catálogo de
// skills (hallazgo X-5 de la auditoría 2026-06). La lección v3.0.0 ("pure
// logic testable") aplicada a los artefactos: en mayo 2026 una auditoría
// externa encontró que las skills instaladas eran "andamiajes vacíos"
// (wrapper "## Sections to Generate" pensado para un LLM generador,
// instalado verbatim como contenido). El retro-port de Track 2 eliminó el
// patrón; estos tests garantizan que no reaparezca y que el layout
// multi-archivo (v4.0.0) se mantenga íntegro.

// forbiddenScaffolding son los marcadores del patrón "plantilla de
// plantillas": texto dirigido a un LLM generador que jamás debe instalarse
// como contenido de una skill.
var forbiddenScaffolding = []string{
	"## Sections to Generate",
	"Describe scenarios:",
	"Sections to Generate",
}

// skillNameRE es la regla de la spec abierta de Agent Skills
// (agentskills.io): minúsculas, números y guiones, máximo 64 chars.
var skillNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// TestSkillsContent_NoGeneratorScaffolding falla si cualquier archivo del
// árbol de skills contiene texto meta de la generación anterior.
func TestSkillsContent_NoGeneratorScaffolding(t *testing.T) {
	err := fs.WalkDir(TemplatesFS, "templates/skills", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		data, err := fs.ReadFile(TemplatesFS, p)
		if err != nil {
			return err
		}
		for _, marker := range forbiddenScaffolding {
			if strings.Contains(string(data), marker) {
				t.Errorf("%s contains generator scaffolding %q — skills are final artifacts, not LLM prompts", p, marker)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk templates/skills: %v", err)
	}
}

// TestSkillsContent_CatalogLayout valida, para cada skill registrada en el
// catálogo, el contrato del layout multi-archivo:
//   - el directorio templates/skills/<TemplateDir>/<guide>/ existe
//   - contiene SKILL.md
//   - el cuerpo NO trae frontmatter YAML (el frontmatter es por-ecosistema y
//     lo genera catalog.GenerateFrontmatter)
//   - el nombre instalado (kebab) cumple la spec abierta de Agent Skills
//   - cada archivo se mantiene bajo 500 líneas (límite de la spec)
func TestSkillsContent_CatalogLayout(t *testing.T) {
	for _, cat := range catalog.Categories {
		for _, opt := range cat.Options {
			for skillDir, guide := range opt.TemplateMapping {
				dir := path.Join("templates", "skills", opt.TemplateDir, skillDir)

				entries, err := fs.ReadDir(TemplatesFS, dir)
				if err != nil {
					t.Errorf("skill %s/%s: missing directory %s: %v", cat.Name, guide, dir, err)
					continue
				}

				installedName := strings.ReplaceAll(guide, "_", "-")
				if !skillNameRE.MatchString(installedName) {
					t.Errorf("skill %s: installed name %q violates the Agent Skills name rule (lowercase/digits/hyphens, ≤64)", guide, installedName)
				}

				hasSkillMD := false
				for _, e := range entries {
					if e.IsDir() {
						continue
					}
					if e.Name() == "SKILL.md" {
						hasSkillMD = true
					}
					data, err := fs.ReadFile(TemplatesFS, path.Join(dir, e.Name()))
					if err != nil {
						t.Errorf("read %s/%s: %v", dir, e.Name(), err)
						continue
					}
					if strings.HasPrefix(strings.TrimSpace(string(data)), "---") {
						t.Errorf("%s/%s must not carry YAML frontmatter — it is generated per-ecosystem by the pipeline", dir, e.Name())
					}
					if lines := strings.Count(string(data), "\n") + 1; lines > 500 {
						t.Errorf("%s/%s has %d lines — exceeds the 500-line Agent Skills guideline", dir, e.Name(), lines)
					}
				}
				if !hasSkillMD {
					t.Errorf("skill %s: %s has no SKILL.md (catalog layout violation)", guide, dir)
				}
			}
		}
	}
}

// TestSkillsContent_NoOrphanSkillDirs detecta directorios de skill presentes
// en el árbol pero no registrados en el catálogo (quedarían inalcanzables e
// invisibles para el usuario).
func TestSkillsContent_NoOrphanSkillDirs(t *testing.T) {
	registered := map[string]bool{}
	for _, cat := range catalog.Categories {
		for _, opt := range cat.Options {
			for skillDir := range opt.TemplateMapping {
				registered[path.Join(opt.TemplateDir, skillDir)] = true
			}
		}
	}

	categoryDirs, err := fs.ReadDir(TemplatesFS, "templates/skills")
	if err != nil {
		t.Fatalf("read templates/skills: %v", err)
	}
	for _, catDir := range categoryDirs {
		if !catDir.IsDir() {
			continue
		}
		skillDirs, err := fs.ReadDir(TemplatesFS, path.Join("templates", "skills", catDir.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", catDir.Name(), err)
		}
		for _, sd := range skillDirs {
			if !sd.IsDir() {
				t.Errorf("templates/skills/%s/%s: loose file at category level — skills are directories since v4.0.0", catDir.Name(), sd.Name())
				continue
			}
			if !registered[path.Join(catDir.Name(), sd.Name())] {
				t.Errorf("templates/skills/%s/%s is not registered in any catalog.Categories mapping — orphan skill", catDir.Name(), sd.Name())
			}
		}
	}
}
