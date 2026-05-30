package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	root "github.com/jorelcb/codify"
	"github.com/jorelcb/codify/internal/application/command"
	"github.com/jorelcb/codify/internal/application/dto"
	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/infrastructure/filesystem"
	infratemplate "github.com/jorelcb/codify/internal/infrastructure/template"
)

// scopeLabel humanizes a dto.InstallScope* constant for prompt copy.
func scopeLabel(scope string) string {
	if scope == dto.InstallScopeGlobal {
		return "globally"
	}
	return "for this project"
}

// promptInstallSkills offers the user a chance to install skills at the given
// scope (global or project), one catalog category at a time. Each prompt has
// "skip" as the default — running through with all skips installs nothing.
//
// Skills install through the catalog (static mode, no LLM, no API key). The
// catalog surface is Claude-only for now, so non-Claude targets are skipped
// with a notice — they return when the catalog gains those ecosystems. Power
// users who want personalized skills can run `codify catalog --mode
// personalized`.
func promptInstallSkills(target, scope string) error {
	if !isInteractive() {
		return nil
	}
	if target != "claude" {
		fmt.Printf("\nSkills: skipped — the catalog supports Claude only for now (target is %q).\n", target)
		return nil
	}

	fmt.Println()
	fmt.Printf("Skills (%s, optional)\n", scopeLabel(scope))
	fmt.Println("─────────────────────────────")
	fmt.Println("Each preset is a curated bundle of related SKILL.md files installed under .claude/skills/.")
	fmt.Println("Pick one preset per category, or skip. You can revisit later with 'codify catalog'.")

	for _, cat := range catalog.Categories {
		preset, err := promptSelect(
			fmt.Sprintf("Skills — %s", cat.Label),
			buildCategoryPresetOptions(cat),
			"skip",
		)
		if err != nil {
			return err
		}
		if preset == "skip" {
			continue
		}
		if err := installSkillsViaCatalog(cat.Name, preset, scope); err != nil {
			fmt.Printf("  ✗ %s/%s install failed: %v\n", cat.Name, preset, err)
			continue
		}
	}
	return nil
}

// buildCategoryPresetOptions builds a select-list of presets for a skill
// category, with "Skip" as the first (and default) option. Each label is
// annotated with the file count so the user knows how many SKILL.md files
// the bundle contains.
func buildCategoryPresetOptions(cat catalog.SkillCategory) []selectOption {
	options := []selectOption{
		{"Skip — don't install this category now", "skip"},
	}
	for _, opt := range cat.Options {
		count := len(opt.TemplateMapping)
		label := fmt.Sprintf("%s — %d skill(s)", opt.Label, count)
		options = append(options, selectOption{label, opt.Name})
	}
	if !cat.Exclusive {
		options = append(options, selectOption{"All presets in this category", "all"})
	}
	return options
}

// installSkillsViaCatalog resolves a category preset to its skill package IDs
// and installs them through the catalog (static, Claude). Replaces the bespoke
// template-load + DeliverStaticSkillsCommand path — the catalog's
// EmbeddedSource + ClaudeInstaller now own skill delivery.
func installSkillsViaCatalog(categoryName, preset, scope string) error {
	cat, err := catalog.FindCategory(categoryName)
	if err != nil {
		return err
	}
	selection, err := cat.Resolve(preset)
	if err != nil {
		return err
	}
	ids := skillIDsFromSelection(selection)

	sc, err := resolveScope(scope)
	if err != nil {
		return err
	}
	out, err := embeddedService().Install(context.Background(),
		command.InstallRequest{Target: catalog.TargetClaudeSkill, IDs: ids, Scope: sc})
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ %s/%s installed (%d skill(s))\n", categoryName, preset, len(out.Installed))
	return nil
}

// skillIDsFromSelection derives the catalog package IDs from a resolved
// selection. Package IDs are the guide names with underscores turned to
// hyphens (the D.1.b normalization), so the converse of the catalog's guide
// naming. All skill presets carry an explicit TemplateMapping, so the values
// are the guide names.
func skillIDsFromSelection(sel *catalog.ResolvedSelection) []string {
	ids := make([]string, 0, len(sel.TemplateMapping))
	for _, guide := range sel.TemplateMapping {
		ids = append(ids, strings.ReplaceAll(guide, "_", "-"))
	}
	sort.Strings(ids)
	return ids
}

// promptInstallHooks offers the user a chance to install Claude Code hook
// bundles at the given scope (global or project). Skipping installs nothing.
//
// Hooks are Claude-only (Codex/Antigravity have no equivalent), so callers
// should gate this on target == "claude" before invoking. Install goes
// through the catalog (EmbeddedSource + ClaudeInstaller).
func promptInstallHooks(scope string) error {
	if !isInteractive() {
		return nil
	}

	settingsPath := "~/.claude/settings.json"
	hooksDir := "~/.claude/hooks/"
	if scope == dto.InstallScopeProject {
		settingsPath = ".claude/settings.json"
		hooksDir = ".claude/hooks/"
	}

	fmt.Println()
	fmt.Printf("Hooks (%s, Claude Code only, optional)\n", scopeLabel(scope))
	fmt.Println("──────────────────────────────────────────")
	fmt.Println("Hooks are deterministic guardrails (linting, security checks, commit conventions).")
	fmt.Printf("They merge into %s + copy scripts to %s.\n", settingsPath, hooksDir)

	preset, err := promptSelect("Hook bundle to install", []selectOption{
		{"Skip — don't install hooks now", "skip"},
		{"linting (auto-format on Edit/Write)", "linting"},
		{"security-guardrails (block dangerous commands)", "security-guardrails"},
		{"convention-enforcement (validate commits + protect main)", "convention-enforcement"},
		{"all (linting + security-guardrails + convention-enforcement)", "all"},
	}, "skip")
	if err != nil {
		return err
	}
	if preset == "skip" {
		return nil
	}

	sc, err := resolveScope(scope)
	if err != nil {
		return err
	}
	out, err := embeddedService().Install(context.Background(),
		command.InstallRequest{Target: catalog.TargetClaudeHook, IDs: hookIDsForPreset(preset), Scope: sc})
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ hooks/%s installed (%d bundle(s)) → %s\n", preset, len(out.Installed), settingsPath)
	return nil
}

// hookIDsForPreset maps a hook preset name to its package IDs. "all" expands
// to the three bundles; any other value is a single bundle ID.
func hookIDsForPreset(preset string) []string {
	if preset == "all" {
		return []string{"linting", "security-guardrails", "convention-enforcement"}
	}
	return []string{preset}
}

// promptInstallWorkflows offers the user a chance to install workflow bundles
// at the given scope. Skipping installs nothing. Workflows currently target
// Claude Code (`.claude/skills/`, `~/.claude/skills/`) and Antigravity
// (`.agent/workflows/`, `~/.gemini/antigravity/global_workflows/`); Codex is
// not supported, so callers should gate on target before invoking.
func promptInstallWorkflows(target, locale, scope string) error {
	if !isInteractive() {
		return nil
	}
	if target != "claude" && target != "antigravity" {
		return nil
	}

	output := workflowsPathForScope(target, scope)

	fmt.Println()
	fmt.Printf("Workflows (%s, optional)\n", scopeLabel(scope))
	fmt.Println("────────────────────────────────")
	fmt.Printf("Workflows are multi-step lifecycle skills (bug-fix, release-cycle, spec-driven-change).\n")
	fmt.Printf("They install to %s.\n", output)

	cat := &catalog.WorkflowCategories[0] // single category: "workflows"
	options := []selectOption{
		{"Skip — don't install workflows now", "skip"},
	}
	for _, opt := range cat.Options {
		count := len(opt.TemplateMapping)
		label := fmt.Sprintf("%s — %d workflow(s)", opt.Label, count)
		options = append(options, selectOption{label, opt.Name})
	}
	options = append(options, selectOption{"All workflows (bug-fix + release-cycle + spec-driven-change)", "all"})

	preset, err := promptSelect("Workflow bundle to install", options, "skip")
	if err != nil {
		return err
	}
	if preset == "skip" {
		return nil
	}

	if err := installWorkflow(target, locale, preset, scope); err != nil {
		fmt.Printf("  ✗ workflows/%s install failed: %v\n", preset, err)
	}
	return nil
}

// installWorkflow executes a static-mode workflows install at the given scope.
func installWorkflow(target, locale, preset, scope string) error {
	cat, err := catalog.FindWorkflowCategory("workflows")
	if err != nil {
		return err
	}

	var selection *catalog.ResolvedSelection
	if preset == "all" {
		selection = catalog.ResolveAllWorkflows()
	} else {
		selection, err = cat.Resolve(preset)
		if err != nil {
			return err
		}
	}

	// Workflow templates live at templates/{locale}/workflows/ — selection.TemplateDir
	// is the literal "workflows" directory. Same path for claude and antigravity;
	// the deliver command handles target-specific frontmatter rendering.
	templatePath := filepath.Join("templates", locale, selection.TemplateDir)
	loader := infratemplate.NewFileSystemTemplateLoaderWithMapping(root.TemplatesFS, templatePath, selection.TemplateMapping)
	guides, err := loader.LoadAll()
	if err != nil {
		return fmt.Errorf("load workflow templates: %w", err)
	}

	output := workflowsPathForScope(target, scope)
	config := &dto.WorkflowConfig{
		Category:   "workflows",
		Preset:     preset,
		Mode:       dto.SkillModeStatic,
		Target:     target,
		Locale:     locale,
		OutputPath: output,
		Install:    scope,
	}

	fileWriter := filesystem.NewFileWriter()
	dirManager := filesystem.NewDirectoryManager()
	deliver := command.NewDeliverStaticWorkflowsCommand(fileWriter, dirManager)

	result, err := deliver.Execute(config, guides)
	if err != nil {
		return err
	}

	fmt.Printf("  ✓ workflows/%s installed (%d file(s)) → %s\n", preset, len(result.GeneratedFiles), result.OutputPath)
	return nil
}

func workflowsPathForScope(target, scope string) string {
	if scope == dto.InstallScopeGlobal {
		return globalWorkflowsPath(target)
	}
	return defaultWorkflowsPath(target)
}
