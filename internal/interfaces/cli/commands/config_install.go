package commands

import (
	"context"
	"fmt"
	"path/filepath"

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

// PromptInstallPackages is the shared package-install entry for the bootstrap
// commands. It runs the SAME rich catalog selector as `codify catalog`
// (tabs + tree + accumulated cross-tab selection) at the given scope, so a user
// equips skills, hooks AND plugins in one pass during `config`/`init` — ADR-0010
// Decision 6 (one selector, reused), the reuse v3.0.0 dropped. Workflows are NOT
// here: they are a separate lifecycle concern (ADR-0010 §5), handled by
// promptInstallWorkflows.
//
// The selector is gated behind a yes/no so bootstrap isn't forced into a
// full-screen TUI (and a plugin marketplace fetch) unless the user opts in. The
// catalog supports Claude and Antigravity; other targets are skipped with a
// notice. Non-interactive sessions skip silently.
func PromptInstallPackages(target, scope string) error {
	if !isInteractive() {
		return nil
	}
	if target != "claude" && target != "antigravity" {
		fmt.Printf("\nPackages: skipped — the catalog supports Claude and Antigravity (target is %q).\n", target)
		return nil
	}

	fmt.Println()
	fmt.Printf("Packages (%s, optional)\n", scopeLabel(scope))
	fmt.Println("────────────────────────────")
	fmt.Println("Browse and install skills, hooks and plugins in one selector (tabs + tree).")
	fmt.Println("You can revisit anytime with 'codify catalog'.")

	ok, err := promptConfirm("Open the package selector now?", false)
	if err != nil || !ok {
		return err
	}

	sc, err := resolveScope(scope)
	if err != nil {
		return err
	}
	p := catalogParams{marketplace: defaultMarketplace}
	return runCatalogSelector(context.Background(), p, target, sc, string(sc))
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
