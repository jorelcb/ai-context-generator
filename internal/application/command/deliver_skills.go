package command

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jorelcb/codify/internal/application/dto"
	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/domain/service"
)

// DeliverStaticSkillsCommand entrega skills estáticas multi-archivo desde el
// catálogo embebido, sin LLM. Desde v4.0.0 las skills son artefactos puros
// (D4: la personalización LLM fue eliminada): un directorio por skill con
// SKILL.md (cuerpo sin frontmatter — el frontmatter es por-ecosistema y se
// genera acá) más sidecars opcionales de progressive disclosure
// (reference.md, examples.md).
type DeliverStaticSkillsCommand struct {
	fileWriter       service.FileWriter
	directoryManager service.DirectoryManager
}

// NewDeliverStaticSkillsCommand crea un nuevo comando de entrega estática.
func NewDeliverStaticSkillsCommand(
	fileWriter service.FileWriter,
	directoryManager service.DirectoryManager,
) *DeliverStaticSkillsCommand {
	return &DeliverStaticSkillsCommand{
		fileWriter:       fileWriter,
		directoryManager: directoryManager,
	}
}

// Execute entrega las skills de la selección: para cada (skillDir → guide)
// lee templates/skills/<TemplateDir>/<skillDir>/ del filesystem dado y
// escribe <output>/<nombre-kebab>/ con SKILL.md (frontmatter según
// config.Target + body) y los sidecars verbatim.
//
// El target Antigravity NO pasa por acá (usa AntigravitySkillSource, que
// inlinea los sidecars en su layout plano); este comando sirve los targets
// con layout de directorio (claude, codex).
func (c *DeliverStaticSkillsCommand) Execute(
	config *dto.SkillsConfig,
	fsys fs.FS,
	selection *catalog.ResolvedSelection,
) (*dto.GenerationResult, error) {
	// Orden determinista — mapas de Go no lo garantizan y los tests y el
	// output de progreso sí lo necesitan.
	skillDirs := make([]string, 0, len(selection.TemplateMapping))
	for dir := range selection.TemplateMapping {
		skillDirs = append(skillDirs, dir)
	}
	sort.Strings(skillDirs)

	var generatedFiles []string
	for _, skillDir := range skillDirs {
		guideName := selection.TemplateMapping[skillDir]
		srcDir := path.Join("templates", "skills", selection.TemplateDir, skillDir)
		entries, err := fs.ReadDir(fsys, srcDir)
		if err != nil {
			return nil, fmt.Errorf("read skill dir %s: %w", srcDir, err)
		}

		outDirName := strings.ReplaceAll(guideName, "_", "-")
		outDir := filepath.Join(config.OutputPath, outDirName)
		if err := c.directoryManager.CreateDir(outDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create skill directory %s: %w", outDirName, err)
		}

		wroteSkillMD := false
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := fs.ReadFile(fsys, path.Join(srcDir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("read skill file %s/%s: %w", srcDir, entry.Name(), err)
			}
			if entry.Name() == "SKILL.md" {
				frontmatter := catalog.GenerateFrontmatter(guideName, config.Target)
				data = []byte(frontmatter + "\n" + string(data))
				wroteSkillMD = true
			}
			filePath := filepath.Join(outDir, entry.Name())
			if err := c.fileWriter.WriteFile(filePath, data, os.FileMode(0o644)); err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", filePath, err)
			}
			generatedFiles = append(generatedFiles, filePath)
		}
		if !wroteSkillMD {
			return nil, fmt.Errorf("skill %q: %s has no SKILL.md (catalog layout violation)", guideName, srcDir)
		}
	}

	return &dto.GenerationResult{
		OutputPath:     config.OutputPath,
		GeneratedFiles: generatedFiles,
		Model:          "static",
	}, nil
}
