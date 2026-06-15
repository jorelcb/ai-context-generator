package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	root "github.com/jorelcb/codify"
	"github.com/jorelcb/codify/internal/application/command"
	"github.com/jorelcb/codify/internal/application/dto"
	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/domain/service"
	"github.com/jorelcb/codify/internal/infrastructure/filesystem"
	"github.com/jorelcb/codify/internal/infrastructure/llm"
	"github.com/jorelcb/codify/internal/infrastructure/scanner"
	"github.com/jorelcb/codify/internal/infrastructure/sdd"
	infratemplate "github.com/jorelcb/codify/internal/infrastructure/template"
)

const serverVersion = "4.0.0"

// validContextPresets enumerates accepted preset names for context generation
// (generate_context + analyze_project tools). The "default" alias was removed
// in v2.0 (ADR-001 phase 3); the new built-in default is "neutral".
var validContextPresets = map[string]bool{
	"clean-ddd":    true,
	"neutral":      true,
	"hexagonal":    true,
	"event-driven": true,
	"workflow":     true,
}

// normalizeContextPreset validates the preset name. In v2.0 this returns an
// error for "default" (removed) and unknown presets — MCP callers see the
// error in the response. The CLI side has equivalent error handling in
// resolvePreset (cli/commands/generate.go).
func normalizeContextPreset(preset string) (string, error) {
	if preset == "default" {
		return "", fmt.Errorf("preset 'default' was removed in Codify v2.0.0. Use preset='clean-ddd' to keep v1.x behavior, or preset='neutral' (the new default) for no architectural opinion")
	}
	if !validContextPresets[preset] {
		return "", fmt.Errorf("unknown preset %q. Valid presets: neutral, clean-ddd, hexagonal, event-driven", preset)
	}
	return preset, nil
}

// NewServer creates and configures the MCP server with all tools registered.
func NewServer() *server.MCPServer {
	s := server.NewMCPServer(
		"codify",
		serverVersion,
		server.WithToolCapabilities(true),
	)

	s.AddTools(
		generateContextTool(),
		generateSpecsTool(),
		analyzeProjectTool(),
		generateSkillsTool(),
		generateHooksTool(),
		commitGuidanceTool(),
		versionGuidanceTool(),
		getUsageTool(),
	)

	return s
}

// generateContextTool defines the generate_context MCP tool.
func generateContextTool() server.ServerTool {
	tool := mcp.NewTool("generate_context",
		mcp.WithDescription("Generate AI-optimized context files for a software project from a description"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Project name")),
		mcp.WithString("description", mcp.Required(), mcp.Description("Project description")),
		mcp.WithString("language", mcp.Description("Programming language (go, python, javascript, etc.)")),
		mcp.WithString("project_type", mcp.Description("Project type hint (api, cli, library, service, webapp, etc.) — grounds the generated content")),
		mcp.WithString("architecture", mcp.Description("Architecture hint (clean, hexagonal, mvc, microservices, etc.) — grounds the generated content")),
		mcp.WithString("preset", mcp.Description("Template preset for context. Options: neutral (default — no architectural opinion), clean-ddd (DDD + Clean Architecture), hexagonal (Ports & Adapters), event-driven (CQRS + Event Sourcing + Sagas), workflow."), mcp.Enum("neutral", "clean-ddd", "hexagonal", "event-driven", "workflow"), mcp.DefaultString("neutral")),
		mcp.WithString("locale", mcp.Description("Output language: en (English) or es (Spanish)"), mcp.DefaultString("en")),
		mcp.WithString("model", mcp.Description("Claude model to use"), mcp.DefaultString("claude-sonnet-4-6")),
		mcp.WithBoolean("with_specs", mcp.Description("Also generate SDD spec files after context generation")),
		mcp.WithString("sdd_standard", mcp.Description("SDD standard for with_specs: spec-kit (default) or openspec"), mcp.Enum("spec-kit", "openspec"), mcp.DefaultString("spec-kit")),
	)

	return server.ServerTool{Tool: tool, Handler: handleGenerateContext}
}

// generateSpecsTool defines the generate_specs MCP tool.
func generateSpecsTool() server.ServerTool {
	tool := mcp.NewTool("generate_specs",
		mcp.WithDescription("Generate SDD specification files from existing context files"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Project name")),
		mcp.WithString("from_context", mcp.Required(), mcp.Description("Path to existing output directory with context files")),
		mcp.WithString("output", mcp.Description("Output directory for the spec files (default: from_context)")),
		mcp.WithString("locale", mcp.Description("Output language: en or es"), mcp.DefaultString("en")),
		mcp.WithString("model", mcp.Description("Claude model to use"), mcp.DefaultString("claude-sonnet-4-6")),
		mcp.WithString("sdd_standard", mcp.Description("SDD standard: spec-kit (default) or openspec"), mcp.Enum("spec-kit", "openspec"), mcp.DefaultString("spec-kit")),
	)

	return server.ServerTool{Tool: tool, Handler: handleGenerateSpecs}
}

// analyzeProjectTool defines the analyze_project MCP tool.
func analyzeProjectTool() server.ServerTool {
	tool := mcp.NewTool("analyze_project",
		mcp.WithDescription("Scan an existing project directory and generate AI context files from its structure, dependencies, and README"),
		mcp.WithString("project_path", mcp.Required(), mcp.Description("Path to the project directory to analyze")),
		mcp.WithString("name", mcp.Description("Project name (defaults to directory name)")),
		mcp.WithString("language", mcp.Description("Override detected language")),
		mcp.WithString("preset", mcp.Description("Template preset for context. Options: neutral (default — no architectural opinion), clean-ddd (DDD + Clean Architecture), hexagonal (Ports & Adapters), event-driven (CQRS + Event Sourcing + Sagas)."), mcp.Enum("neutral", "clean-ddd", "hexagonal", "event-driven"), mcp.DefaultString("neutral")),
		mcp.WithString("locale", mcp.Description("Output language: en or es"), mcp.DefaultString("en")),
		mcp.WithString("model", mcp.Description("Claude model to use"), mcp.DefaultString("claude-sonnet-4-6")),
		mcp.WithBoolean("with_specs", mcp.Description("Also generate SDD spec files after context generation")),
		mcp.WithString("sdd_standard", mcp.Description("SDD standard for with_specs: spec-kit (default) or openspec"), mcp.Enum("spec-kit", "openspec"), mcp.DefaultString("spec-kit")),
	)

	return server.ServerTool{Tool: tool, Handler: handleAnalyzeProject}
}

// generateSkillsTool defines the generate_skills MCP tool. Skills are
// static-only since v4.0.0 (D4: the LLM personalization mode was dropped) and
// multi-file: each skill ships SKILL.md plus progressive-disclosure sidecars
// (reference.md, examples.md) when the skill provides them.
func generateSkillsTool() server.ServerTool {
	tool := mcp.NewTool("generate_skills",
		mcp.WithDescription("Deliver curated AI agent skills by category and preset. Each skill is a directory with SKILL.md plus reference/examples companions. Instant, offline — no API key needed."),
		mcp.WithString("category", mcp.Required(), mcp.Description("Skill category"), mcp.Enum(catalog.CategoryNames()...)),
		mcp.WithString("preset", mcp.Required(), mcp.Description("Preset within category (or 'all' where supported). architecture: clean-ddd, hexagonal, event-driven, neutral. testing: foundational, tdd, bdd. conventions: conventional-commit, semantic-versioning, all"), mcp.Enum(catalog.AllSkillPresetNames()...)),
		mcp.WithString("target", mcp.Description("Target ecosystem"), mcp.Enum("claude", "codex", "antigravity"), mcp.DefaultString("claude")),
		mcp.WithString("output", mcp.Description("Output directory (default: ecosystem-specific, e.g. .claude/skills/)")),
	)

	return server.ServerTool{Tool: tool, Handler: handleGenerateSkills}
}

// generateHooksTool defines the generate_hooks MCP tool.
func generateHooksTool() server.ServerTool {
	tool := mcp.NewTool("generate_hooks",
		mcp.WithDescription("Activate Claude Code hook bundles. Default install_scope is 'preview' (writes a standalone bundle to 'output' for the user to merge). Set install_scope to 'global' or 'project' to auto-merge into settings.json + copy scripts into the agent's hooks directory. Static-only, Claude Code-only."),
		mcp.WithString("preset", mcp.Required(), mcp.Description("Hook preset"), mcp.Enum(catalog.HookPresetNames()...)),
		mcp.WithString("install_scope", mcp.Description("Activation mode: global (auto-merge into ~/.claude), project (auto-merge into .claude), or preview (write bundle to output, no settings change)"), mcp.Enum("global", "project", "preview"), mcp.DefaultString("preview")),
		mcp.WithString("output", mcp.Description("Output directory (required for install_scope=preview; default: ./codify-hooks)")),
	)

	return server.ServerTool{Tool: tool, Handler: handleGenerateHooks}
}

// commitGuidanceTool defines the commit_guidance MCP knowledge tool.
func commitGuidanceTool() server.ServerTool {
	tool := mcp.NewTool("commit_guidance",
		mcp.WithDescription("Conventional Commits behavioral context. Returns the spec and instructions for generating proper commit messages. No API key needed."),
	)

	return server.ServerTool{Tool: tool, Handler: handleCommitGuidance}
}

// versionGuidanceTool defines the version_guidance MCP knowledge tool.
func versionGuidanceTool() server.ServerTool {
	tool := mcp.NewTool("version_guidance",
		mcp.WithDescription("Semantic Versioning behavioral context. Returns the spec and instructions for determining version bumps from conventional commits. No API key needed."),
	)

	return server.ServerTool{Tool: tool, Handler: handleVersionGuidance}
}

// --- Tool Handlers ---

func handleGenerateContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := stringArg(request, "name")
	description := stringArg(request, "description")
	language := stringArg(request, "language")
	projectType := stringArg(request, "project_type")
	architecture := stringArg(request, "architecture")
	preset := stringArgDefault(request, "preset", "neutral")
	locale := stringArgDefault(request, "locale", "en")
	model := stringArgDefault(request, "model", "")
	withSpecs := boolArg(request, "with_specs")
	sddStandard := stringArgDefault(request, "sdd_standard", "")

	result, err := executeGenerate(ctx, name, description, language, projectType, architecture, preset, locale, model)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Generation failed: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Context files generated for '%s'\n", name))
	sb.WriteString(fmt.Sprintf("Output: %s\n", result.OutputPath))
	sb.WriteString(fmt.Sprintf("Model: %s\n", result.Model))
	sb.WriteString(fmt.Sprintf("Tokens: %d in / %d out\n", result.TokensIn, result.TokensOut))
	sb.WriteString("\nGenerated files:\n")
	for _, f := range result.GeneratedFiles {
		sb.WriteString(fmt.Sprintf("  - %s\n", f))
	}
	sb.WriteString(validationSummary(result.GeneratedFiles, "generate"))

	if withSpecs {
		specResult, err := executeSpecs(ctx, name, result.OutputPath, result.OutputPath, locale, model, sddStandard)
		if err != nil {
			sb.WriteString(fmt.Sprintf("\nSpec generation failed: %v\n", err))
		} else {
			sb.WriteString(fmt.Sprintf("\nSpec files generated\n"))
			sb.WriteString(fmt.Sprintf("Tokens: %d in / %d out\n", specResult.TokensIn, specResult.TokensOut))
			sb.WriteString("\nSpec files:\n")
			for _, f := range specResult.GeneratedFiles {
				sb.WriteString(fmt.Sprintf("  - %s\n", f))
			}
			sb.WriteString(validationSummary(specResult.GeneratedFiles, "spec"))
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleGenerateSpecs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := stringArg(request, "name")
	fromContext := stringArg(request, "from_context")
	output := stringArgDefault(request, "output", "")
	locale := stringArgDefault(request, "locale", "en")
	model := stringArgDefault(request, "model", "")
	sddStandard := stringArgDefault(request, "sdd_standard", "")

	result, err := executeSpecs(ctx, name, fromContext, output, locale, model, sddStandard)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Spec generation failed: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Spec files generated for '%s'\n", name))
	sb.WriteString(fmt.Sprintf("Output: %s\n", result.OutputPath))
	sb.WriteString(fmt.Sprintf("Model: %s\n", result.Model))
	sb.WriteString(fmt.Sprintf("Tokens: %d in / %d out\n", result.TokensIn, result.TokensOut))
	sb.WriteString("\nGenerated files:\n")
	for _, f := range result.GeneratedFiles {
		sb.WriteString(fmt.Sprintf("  - %s\n", f))
	}
	sb.WriteString(validationSummary(result.GeneratedFiles, "spec"))

	return mcp.NewToolResultText(sb.String()), nil
}

func handleAnalyzeProject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectPath := stringArg(request, "project_path")
	name := stringArg(request, "name")
	language := stringArg(request, "language")
	preset := stringArgDefault(request, "preset", "neutral")
	locale := stringArgDefault(request, "locale", "en")
	model := stringArgDefault(request, "model", "")
	withSpecs := boolArg(request, "with_specs")
	sddStandard := stringArgDefault(request, "sdd_standard", "")

	// Resolve path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path: %v", err)), nil
	}

	if name == "" {
		name = filepath.Base(absPath)
	}

	// Scan project
	s := scanner.NewProjectScanner()
	scanResult, err := s.Scan(absPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Scan failed: %v", err)), nil
	}

	// Use detected language if not overridden
	if language == "" && scanResult.Language != "" {
		language = normalizeLanguageFlag(scanResult.Language)
	}

	// Format scan as description and generate with analyze mode
	description := scanResult.FormatAsDescription()
	result, err := executeGenerateWithMode(ctx, name, description, language, "", "", preset, locale, model, "analyze")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Generation failed: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Project analyzed and context generated for '%s'\n", name))
	sb.WriteString(fmt.Sprintf("Detected: %s", scanResult.Language))
	if scanResult.Framework != "" {
		sb.WriteString(fmt.Sprintf(" / %s", scanResult.Framework))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Output: %s\n", result.OutputPath))
	sb.WriteString(fmt.Sprintf("Model: %s\n", result.Model))
	sb.WriteString(fmt.Sprintf("Tokens: %d in / %d out\n", result.TokensIn, result.TokensOut))
	sb.WriteString("\nGenerated files:\n")
	for _, f := range result.GeneratedFiles {
		sb.WriteString(fmt.Sprintf("  - %s\n", f))
	}
	sb.WriteString(validationSummary(result.GeneratedFiles, "analyze"))

	if withSpecs {
		specResult, err := executeSpecs(ctx, name, result.OutputPath, result.OutputPath, locale, model, sddStandard)
		if err != nil {
			sb.WriteString(fmt.Sprintf("\nSpec generation failed: %v\n", err))
		} else {
			sb.WriteString(fmt.Sprintf("\nSpec files generated\n"))
			for _, f := range specResult.GeneratedFiles {
				sb.WriteString(fmt.Sprintf("  - %s\n", f))
			}
			sb.WriteString(validationSummary(specResult.GeneratedFiles, "spec"))
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleGenerateSkills(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	categoryName := stringArg(request, "category")
	preset := stringArg(request, "preset")
	target := stringArgDefault(request, "target", "claude")
	output := stringArg(request, "output")
	if output == "" {
		output = defaultSkillsPath(target)
	}

	// Resolver categoría y preset desde el catálogo
	cat, err := catalog.FindCategory(categoryName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid category: %v", err)), nil
	}

	selection, err := cat.Resolve(preset)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid preset: %v", err)), nil
	}

	config := &dto.SkillsConfig{
		Category:   cat.Name,
		Preset:     preset,
		Target:     target,
		OutputPath: output,
	}

	result, err := executeStaticSkillsMCP(config, selection)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Skills generation failed: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Agent skills delivered (category: %s, preset: %s, target: %s)\n", categoryName, preset, target))
	sb.WriteString(fmt.Sprintf("Output: %s\n", result.OutputPath))
	sb.WriteString("\nGenerated skills:\n")
	for _, f := range result.GeneratedFiles {
		sb.WriteString(fmt.Sprintf("  - %s\n", f))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleGenerateHooks(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	preset := stringArg(request, "preset")
	output := stringArg(request, "output")
	scope := stringArgDefault(request, "install_scope", "preview")

	if !dto.ValidHookPresets[preset] {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid hook preset: %s (valid: linting, security-guardrails, convention-enforcement, all)", preset)), nil
	}

	switch scope {
	case "preview":
		if output == "" {
			output = filepath.Join(".", "codify-hooks")
		}
		config := &dto.HookConfig{
			Category:   "hooks",
			Preset:     preset,
			OutputPath: output,
		}
		result, err := executePreviewHooksMCP(config)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Hook bundle generation failed: %v", err)), nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Claude Code hook bundle written (preset: %s, mode: preview)\n", preset))
		sb.WriteString(fmt.Sprintf("Output: %s\n", result.OutputPath))
		sb.WriteString("\nGenerated files:\n")
		for _, f := range result.GeneratedFiles {
			sb.WriteString(fmt.Sprintf("  - %s\n", f))
		}
		sb.WriteString("\nThis is preview mode — settings.json was NOT modified.\n")
		sb.WriteString("Re-run with install_scope=global|project to auto-activate.\n")
		return mcp.NewToolResultText(sb.String()), nil

	case dto.InstallScopeGlobal, dto.InstallScopeProject:
		config := &dto.HookConfig{
			Category: "hooks",
			Preset:   preset,
			Install:  scope,
		}
		result, err := executeInstallHooksMCP(config)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Hook activation failed: %v", err)), nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Claude Code hooks activated (preset: %s, scope: %s)\n", preset, scope))
		sb.WriteString(fmt.Sprintf("Settings: %s\n", result.SettingsPath))
		if result.BackupPath != "" {
			sb.WriteString(fmt.Sprintf("Backup:   %s\n", result.BackupPath))
		}
		sb.WriteString(fmt.Sprintf("Hooks dir: %s\n", result.HooksDir))
		if total := sumIntMap(result.HandlersAdded); total > 0 {
			sb.WriteString(fmt.Sprintf("Added:    %d handler(s) across %d event(s)\n", total, len(result.HandlersAdded)))
		}
		if total := sumIntMap(result.HandlersSkipped); total > 0 {
			sb.WriteString(fmt.Sprintf("Skipped:  %d handler(s) already present (idempotent)\n", total))
		}
		if len(result.ScriptsCopied) > 0 {
			sb.WriteString(fmt.Sprintf("Scripts copied: %d\n", len(result.ScriptsCopied)))
		}
		if len(result.ScriptsConflict) > 0 {
			sb.WriteString(fmt.Sprintf("Scripts in conflict: %d (existing differs — not overwritten)\n", len(result.ScriptsConflict)))
		}
		return mcp.NewToolResultText(sb.String()), nil

	default:
		return mcp.NewToolResultError(fmt.Sprintf("Invalid install_scope: %s (must be 'global', 'project', or 'preview')", scope)), nil
	}
}

func sumIntMap(m map[string]int) int {
	t := 0
	for _, v := range m {
		t += v
	}
	return t
}

func handleCommitGuidance(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := loadKnowledgeSkill("conventions", "conventional_commit")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load commit guidance: %v", err)), nil
	}
	return mcp.NewToolResultText(content), nil
}

func handleVersionGuidance(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := loadKnowledgeSkill("conventions", "semantic_versioning")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load version guidance: %v", err)), nil
	}
	return mcp.NewToolResultText(content), nil
}

// validationSummary re-validates the written files and renders a block for
// the MCP tool result. The CLI surfaces this same information interactively
// (validation feedback on stderr + the resolve flow); MCP used to discard it
// (provider wired with progressOut=nil), so files with unresolved
// [DEFINE: ...] markers were reported as plain success. The calling agent is
// exactly the consumer that can act on these findings, so they belong in the
// tool result. Returns "" when every file is clean.
func validationSummary(files []string, mode string) string {
	var sb strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		r := llm.ValidateOutput(string(data), mode, filepath.Base(f))
		if len(r.DefineMarkers) == 0 && len(r.Warnings) == 0 {
			continue
		}
		if sb.Len() == 0 {
			sb.WriteString("\nValidation findings (review before relying on these files):\n")
		}
		for _, m := range r.DefineMarkers {
			sb.WriteString(fmt.Sprintf("  - %s L%d: unresolved %s\n", f, m.Line, m.Text))
		}
		for _, w := range r.Warnings {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", f, w))
		}
	}
	return sb.String()
}

// loadKnowledgeSkill reads an embedded skill body and returns its content as
// behavioral context. Skills are English-only by design, so the path is
// locale-free. Since the v4.0.0 multi-file layout each skill is a directory;
// the knowledge tools serve the SKILL.md body (the sidecars are
// progressive-disclosure material for installed skills, not needed here).
func loadKnowledgeSkill(category, skill string) (string, error) {
	path := filepath.Join("templates", "skills", category, skill, "SKILL.md")
	data, err := root.TemplatesFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("skill not found: %s", path)
	}
	return string(data), nil
}

// --- Execution helpers (shared by all handlers) ---

func executeGenerate(ctx context.Context, name, description, language, projectType, architecture, preset, locale, model string) (*dto.GenerationResult, error) {
	return executeGenerateWithMode(ctx, name, description, language, projectType, architecture, preset, locale, model, "")
}

func executeGenerateWithMode(ctx context.Context, name, description, language, projectType, architecture, preset, locale, model, mode string) (*dto.GenerationResult, error) {
	apiKey, err := llm.ResolveAPIKey(model)
	if err != nil {
		return nil, err
	}

	preset, err = normalizeContextPreset(preset)
	if err != nil {
		return nil, err
	}

	templatePath := filepath.Join("templates", locale, preset)
	localeBase := filepath.Join("templates", locale)

	var templateLoader service.TemplateLoader
	if language != "" {
		templateLoader = infratemplate.NewFileSystemTemplateLoaderWithLanguage(root.TemplatesFS, templatePath, localeBase, language)
	} else {
		templateLoader = infratemplate.NewFileSystemTemplateLoader(root.TemplatesFS, templatePath)
	}

	guides, err := templateLoader.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	provider, err := llm.NewProvider(ctx, model, apiKey, nil) // no stdout in MCP mode
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}
	fileWriter := filesystem.NewFileWriter()
	dirManager := filesystem.NewDirectoryManager()

	generateCmd := command.NewGenerateContextCommand(provider, fileWriter, dirManager)

	config := &dto.ProjectConfig{
		Name:         name,
		Description:  description,
		Language:     language,
		Type:         projectType,
		Architecture: architecture,
		Model:        model,
		OutputPath:   ".",
		Locale:       locale,
		Mode:         mode,
	}

	return generateCmd.Execute(ctx, config, guides)
}

// loadSpecGuides loads the spec template guides for the active SDD standard from
// the embedded FS, stamped with the standard's per-standard output file names.
// It is the single template-loading path for the MCP spec flow and is covered by
// a regression test (TestLoadSpecGuides...) — the v3.0.0 break was precisely a
// wrong, removed template path loaded here while no test exercised it.
func loadSpecGuides(locale string, standard service.SpecStandard) ([]service.TemplateGuide, error) {
	loader := infratemplate.NewFileSystemTemplateLoaderWithMapping(
		root.TemplatesFS, sdd.SpecTemplatePath(locale, standard), sdd.SpecTemplateMapping(standard),
	)
	guides, err := loader.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to load spec templates for standard %q: %w", standard.ID(), err)
	}
	return sdd.ApplySpecOutputNames(guides, standard), nil
}

// slugifySpecFeatureID derives a filesystem-safe feature id from a project name,
// used for the FeatureGrouped layout (Spec-Kit) where specs live under
// specs/<feature-id>/. Unused for the flat OpenSpec layout.
func slugifySpecFeatureID(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// resolveSpecOutputPath defaults the spec output directory to the context
// directory when no explicit output is given — mirroring the CLI, where
// --output defaults to --from-context (R-4r: CLI↔MCP SDD parity).
func resolveSpecOutputPath(fromContext, output string) string {
	if output == "" {
		return fromContext
	}
	return output
}

func executeSpecs(ctx context.Context, name, fromContextPath, outputPath, locale, model, sddStandard string) (*dto.GenerationResult, error) {
	outputPath = resolveSpecOutputPath(fromContextPath, outputPath)
	apiKey, err := llm.ResolveAPIKey(model)
	if err != nil {
		return nil, err
	}

	// Resolve the active SDD standard (explicit arg wins; default OpenSpec).
	standard, err := sdd.NewDefaultRegistry().Resolve(sddStandard, "", "")
	if err != nil {
		return nil, err
	}

	contextReader := filesystem.NewContextReader()
	existingContext, err := contextReader.ReadExistingContext(fromContextPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing context: %w", err)
	}

	guides, err := loadSpecGuides(locale, standard)
	if err != nil {
		return nil, err
	}

	provider, err := llm.NewProvider(ctx, model, apiKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}
	fileWriter := filesystem.NewFileWriter()
	dirManager := filesystem.NewDirectoryManager()

	specCmd := command.NewGenerateSpecCommand(provider, fileWriter, dirManager)

	config := &dto.SpecConfig{
		ProjectName:     name,
		FromContextPath: fromContextPath,
		OutputPath:      outputPath,
		Model:           model,
		Locale:          locale,
		FeatureID:       slugifySpecFeatureID(name),
		StandardID:      standard.ID(),
		StandardHints:   standard.SystemPromptHints(locale),
		Artifacts:       standard.BootstrapArtifacts(),
	}

	result, err := specCmd.Execute(ctx, config, existingContext, guides)
	if err != nil {
		return nil, err
	}

	// Update AGENTS.md with a specs reference, reusing the resolved standard so
	// the listed paths match what was actually generated. The section body
	// comes from the shared sdd helper (single source of truth for both
	// interfaces).
	agentsPath := filepath.Join(outputPath, "AGENTS.md")
	content, readErr := os.ReadFile(agentsPath)
	if readErr == nil && !strings.Contains(string(content), "specs/") {
		specsRef := sdd.SpecsReferenceSection(locale, standard, config.FeatureID)
		_ = os.WriteFile(agentsPath, []byte(string(content)+specsRef), 0o644)
	}

	return result, nil
}

func executeStaticSkillsMCP(config *dto.SkillsConfig, selection *catalog.ResolvedSelection) (*dto.GenerationResult, error) {
	fileWriter := filesystem.NewFileWriter()
	dirManager := filesystem.NewDirectoryManager()
	cmd := command.NewDeliverStaticSkillsCommand(fileWriter, dirManager)
	return cmd.Execute(config, root.TemplatesFS, selection)
}

func executePreviewHooksMCP(config *dto.HookConfig) (*dto.GenerationResult, error) {
	fileWriter := filesystem.NewFileWriter()
	dirManager := filesystem.NewDirectoryManager()
	cmd := command.NewDeliverHooksCommand(fileWriter, dirManager, root.TemplatesFS)
	return cmd.Execute(config)
}

func executeInstallHooksMCP(config *dto.HookConfig) (*command.InstallResult, error) {
	fileWriter := filesystem.NewFileWriter()
	dirManager := filesystem.NewDirectoryManager()
	deliverer := command.NewDeliverHooksCommand(fileWriter, dirManager, root.TemplatesFS)
	installer := command.NewInstallHooksCommand(deliverer, fileWriter, dirManager)
	return installer.Execute(config)
}

// --- Argument helpers ---

func stringArg(request mcp.CallToolRequest, name string) string {
	if v, ok := request.GetArguments()[name].(string); ok {
		return v
	}
	return ""
}

func stringArgDefault(request mcp.CallToolRequest, name, defaultVal string) string {
	if v := stringArg(request, name); v != "" {
		return v
	}
	return defaultVal
}

func boolArg(request mcp.CallToolRequest, name string) bool {
	if v, ok := request.GetArguments()[name].(bool); ok {
		return v
	}
	return false
}

// normalizeLanguageFlag maps detected language names to CLI flag values.
func normalizeLanguageFlag(detected string) string {
	mapping := map[string]string{
		"Go":                    "go",
		"JavaScript/TypeScript": "javascript",
		"TypeScript":            "typescript",
		"Python":                "python",
		"Rust":                  "rust",
		"Java":                  "java",
		"Ruby":                  "ruby",
		"Elixir":                "elixir",
		"PHP":                   "php",
		"Swift":                 "swift",
		"C#/.NET":               "csharp",
	}
	if flag, ok := mapping[detected]; ok {
		return flag
	}
	return ""
}

// defaultSkillsPath returns the ecosystem-specific default skills directory.
func defaultSkillsPath(target string) string {
	switch target {
	case "codex", "antigravity":
		return filepath.Join(".agents", "skills")
	default: // claude
		return filepath.Join(".claude", "skills")
	}
}
