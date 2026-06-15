package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jorelcb/codify/internal/application/dto"
	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/infrastructure/filesystem"
)

func TestDeliverStaticSkills_WritesMultiFileSkillDirs(t *testing.T) {
	tmp := t.TempDir()
	cmd := NewDeliverStaticSkillsCommand(filesystem.NewFileWriter(), filesystem.NewDirectoryManager())

	// In-memory skills tree following the v4.0.0 layout: one directory per
	// skill with SKILL.md + optional progressive-disclosure sidecars.
	fsys := fstest.MapFS{
		"templates/skills/clean-ddd/ddd_entity/SKILL.md":     {Data: []byte("# DDD entity body")},
		"templates/skills/clean-ddd/ddd_entity/reference.md": {Data: []byte("# Reference")},
		"templates/skills/clean-ddd/ddd_entity/examples.md":  {Data: []byte("# Examples")},
		"templates/skills/clean-ddd/clean_arch_layer/SKILL.md": {
			Data: []byte("# Clean arch layer body"),
		},
	}

	cfg := &dto.SkillsConfig{
		Category:   "architecture",
		Preset:     "clean-ddd",
		Locale:     "en",
		Target:     "claude",
		OutputPath: tmp,
	}
	selection := &catalog.ResolvedSelection{
		TemplateDir: "clean-ddd",
		TemplateMapping: map[string]string{
			"ddd_entity":       "ddd_entity",
			"clean_arch_layer": "clean_arch_layer",
		},
	}

	result, err := cmd.Execute(cfg, fsys, selection)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result.GeneratedFiles) != 4 {
		t.Fatalf("GeneratedFiles: got %d, want 4 (2 SKILL.md + 2 sidecars)", len(result.GeneratedFiles))
	}

	// Underscores in guide names map to hyphens in directory names; SKILL.md
	// gets frontmatter; sidecars are copied verbatim.
	data, err := os.ReadFile(filepath.Join(tmp, "ddd-entity", "SKILL.md"))
	if err != nil {
		t.Fatalf("missing SKILL.md: %v", err)
	}
	if !strings.HasPrefix(string(data), "---") {
		t.Errorf("SKILL.md missing frontmatter delimiter")
	}
	for _, sidecar := range []string{"reference.md", "examples.md"} {
		raw, err := os.ReadFile(filepath.Join(tmp, "ddd-entity", sidecar))
		if err != nil {
			t.Errorf("missing sidecar %s: %v", sidecar, err)
			continue
		}
		if strings.HasPrefix(string(raw), "---") {
			t.Errorf("sidecar %s must be verbatim (no frontmatter)", sidecar)
		}
	}
	if _, err := os.Stat(filepath.Join(tmp, "clean-arch-layer", "SKILL.md")); err != nil {
		t.Errorf("missing clean-arch-layer/SKILL.md: %v", err)
	}
}

func TestDeliverStaticSkills_MissingSkillMDIsError(t *testing.T) {
	tmp := t.TempDir()
	cmd := NewDeliverStaticSkillsCommand(filesystem.NewFileWriter(), filesystem.NewDirectoryManager())

	fsys := fstest.MapFS{
		"templates/skills/neutral/code_review/reference.md": {Data: []byte("# only a sidecar")},
	}
	cfg := &dto.SkillsConfig{Category: "architecture", Preset: "neutral", Target: "claude", OutputPath: tmp}
	selection := &catalog.ResolvedSelection{
		TemplateDir:     "neutral",
		TemplateMapping: map[string]string{"code_review": "code_review"},
	}

	if _, err := cmd.Execute(cfg, fsys, selection); err == nil {
		t.Fatal("a skill dir without SKILL.md must be a catalog layout error")
	}
}
