package llm

import (
	"strings"
	"testing"

	"github.com/jorelcb/codify/internal/domain/service"
)

func TestPromptBuilder_BuildSystemPromptForFile(t *testing.T) {
	builder := NewPromptBuilder()

	prompt := builder.BuildSystemPromptForFile("en")

	if prompt == "" {
		t.Error("BuildSystemPromptForFile() returned empty string")
	}
	// Verify XML tag structure
	if !strings.Contains(prompt, "<role>") {
		t.Error("BuildSystemPromptForFile() should contain <role> XML tag")
	}
	if !strings.Contains(prompt, "<workflow>") {
		t.Error("BuildSystemPromptForFile() should contain <workflow> XML tag")
	}
	if !strings.Contains(prompt, "<output_quality>") {
		t.Error("BuildSystemPromptForFile() should contain <output_quality> XML tag")
	}
	// The target file travels in the user message — the system prompt must
	// reference the <template_guide> file attribute instead of naming a file.
	if !strings.Contains(prompt, "<template_guide>") {
		t.Error("BuildSystemPromptForFile() should point the model at the <template_guide> file attribute")
	}
	for _, fileName := range []string{"AGENTS.md", "CONTEXT.md", "INTERACTIONS_LOG.md"} {
		if strings.Contains(prompt, fileName) {
			t.Errorf("BuildSystemPromptForFile() must not name %s — an interpolated file name breaks prompt-cache prefix reuse", fileName)
		}
	}
}

// TestPromptBuilder_SystemPromptsStableAcrossRun pins the caching contract:
// the system prompt of every per-file mode must be byte-identical across the
// guides of one run, because Anthropic prompt caching is prefix-exact.
func TestPromptBuilder_SystemPromptsStableAcrossRun(t *testing.T) {
	builder := NewPromptBuilder()

	if a, b := builder.BuildSystemPromptForFile("en"), builder.BuildSystemPromptForFile("en"); a != b {
		t.Error("generate system prompt must be identical across calls of one run")
	}
	if a, b := builder.BuildAnalyzeSystemPromptForFile("en"), builder.BuildAnalyzeSystemPromptForFile("en"); a != b {
		t.Error("analyze system prompt must be identical across calls of one run")
	}
	if a, b := builder.BuildSpecSystemPrompt("ctx", "en", ""), builder.BuildSpecSystemPrompt("ctx", "en", ""); a != b {
		t.Error("spec system prompt must be identical across the artifacts of one run")
	}
}

func TestPromptBuilder_BuildUserMessageForFile(t *testing.T) {
	builder := NewPromptBuilder()

	req := service.GenerationRequest{
		ProjectDescription: "API REST de gestion de inventarios en Go",
		Language:           "go",
		ProjectType:        "api",
		Architecture:       "clean",
	}

	guide := service.TemplateGuide{
		Name:    "agents",
		Content: "# Agents template content here",
	}

	msg := builder.BuildUserMessageForFile(req, guide)

	if msg == "" {
		t.Error("BuildUserMessageForFile() returned empty string")
	}

	// Verify description is included in XML tags
	if !strings.Contains(msg, "API REST de gestion de inventarios en Go") {
		t.Error("BuildUserMessageForFile() missing project description")
	}
	if !strings.Contains(msg, "<project_description>") {
		t.Error("BuildUserMessageForFile() should use <project_description> XML tag")
	}

	// Verify optional fields in metadata
	if !strings.Contains(msg, "go") {
		t.Error("BuildUserMessageForFile() missing language")
	}
	if !strings.Contains(msg, "<project_metadata>") {
		t.Error("BuildUserMessageForFile() should use <project_metadata> XML tag")
	}

	// Verify template content in XML tag
	if !strings.Contains(msg, "# Agents template content here") {
		t.Error("BuildUserMessageForFile() missing template content")
	}
	if !strings.Contains(msg, "<template_guide") {
		t.Error("BuildUserMessageForFile() should use <template_guide> XML tag")
	}
}

func TestPromptBuilder_BuildUserMessageForFile_WithoutOptionalFields(t *testing.T) {
	builder := NewPromptBuilder()

	req := service.GenerationRequest{
		ProjectDescription: "A simple project",
	}

	guide := service.TemplateGuide{
		Name:    "context",
		Content: "# Template",
	}

	msg := builder.BuildUserMessageForFile(req, guide)

	if strings.Contains(msg, "<project_metadata>") {
		t.Error("should not include project_metadata when all fields are empty")
	}
}

func TestFileOutputName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"agents", "AGENTS.md"},
		{"context", "CONTEXT.md"},
		{"interactions", "INTERACTIONS_LOG.md"},
		{"development_guide", "DEVELOPMENT_GUIDE.md"},
		{"idioms", "IDIOMS.md"},
		{"constitution", "CONSTITUTION.md"},
		{"spec", "SPEC.md"},
		{"plan", "PLAN.md"},
		{"tasks", "TASKS.md"},
		// Skills output files
		{"ddd_entity", "SKILL.md"},
		{"clean_arch_layer", "SKILL.md"},
		{"bdd_scenario", "SKILL.md"},
		{"cqrs_command", "SKILL.md"},
		{"hexagonal_port", "SKILL.md"},
		{"code_review", "SKILL.md"},
		{"test_strategy", "SKILL.md"},
		{"refactor_safely", "SKILL.md"},
		{"api_design", "SKILL.md"},
		{"unknown", "unknown.md"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FileOutputName(tt.input)
			if got != tt.want {
				t.Errorf("FileOutputName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPromptBuilder_BuildAnalyzeSystemPromptForFile(t *testing.T) {
	builder := NewPromptBuilder()

	prompt := builder.BuildAnalyzeSystemPromptForFile("en")

	if prompt == "" {
		t.Error("BuildAnalyzeSystemPromptForFile() returned empty string")
	}

	// Verify analyze-specific XML tags
	if !strings.Contains(prompt, "<scan_trust>") {
		t.Error("BuildAnalyzeSystemPromptForFile() should contain <scan_trust> XML tag")
	}
	if !strings.Contains(prompt, "AUTO-SCANNED") {
		t.Error("BuildAnalyzeSystemPromptForFile() should mention AUTO-SCANNED")
	}
	if !strings.Contains(prompt, "FACTUAL") {
		t.Error("BuildAnalyzeSystemPromptForFile() should mention FACTUAL")
	}

	// Verify shared structure tags
	if !strings.Contains(prompt, "<role>") {
		t.Error("BuildAnalyzeSystemPromptForFile() should contain <role> XML tag")
	}
	if !strings.Contains(prompt, "<workflow>") {
		t.Error("BuildAnalyzeSystemPromptForFile() should contain <workflow> XML tag")
	}
	if !strings.Contains(prompt, "<output_quality>") {
		t.Error("BuildAnalyzeSystemPromptForFile() should contain <output_quality> XML tag")
	}

	// The target file travels in the user message (caching contract).
	for _, fileName := range []string{"AGENTS.md", "CONTEXT.md", "INTERACTIONS_LOG.md"} {
		if strings.Contains(prompt, fileName) {
			t.Errorf("BuildAnalyzeSystemPromptForFile() must not name %s — an interpolated file name breaks prompt-cache prefix reuse", fileName)
		}
	}

	// Verify it does NOT contain the aspirational grounding rules
	if strings.Contains(prompt, "only what the user stated") {
		t.Error("BuildAnalyzeSystemPromptForFile() should NOT contain aspirational grounding language")
	}
}

func TestPromptBuilder_BuildAnalyzeSystemPromptForFile_Locale(t *testing.T) {
	builder := NewPromptBuilder()

	promptEN := builder.BuildAnalyzeSystemPromptForFile("en")
	promptES := builder.BuildAnalyzeSystemPromptForFile("es")

	if !strings.Contains(promptEN, "English") {
		t.Error("English locale prompt should contain 'English'")
	}
	if !strings.Contains(promptES, "Spanish") {
		t.Error("Spanish locale prompt should contain 'Spanish'")
	}
}

func TestPromptBuilder_BuildSpecSystemPrompt(t *testing.T) {
	builder := NewPromptBuilder()

	existingContext := "# My Project Context\n\nThis is the architecture description."
	prompt := builder.BuildSpecSystemPrompt(existingContext, "en", "")

	if prompt == "" {
		t.Error("BuildSpecSystemPrompt() returned empty string")
	}
	if !strings.Contains(prompt, existingContext) {
		t.Error("BuildSpecSystemPrompt() should embed existing context")
	}
	if !strings.Contains(prompt, "<existing_context>") {
		t.Error("BuildSpecSystemPrompt() should use <existing_context> XML tag")
	}
	if !strings.Contains(prompt, "<role>") {
		t.Error("BuildSpecSystemPrompt() should contain <role> XML tag")
	}
	if !strings.Contains(prompt, "SDD") {
		t.Error("BuildSpecSystemPrompt() should mention SDD")
	}
}

func TestPromptBuilder_BuildSpecSystemPrompt_HintsSeparator(t *testing.T) {
	builder := NewPromptBuilder()

	prompt := builder.BuildSpecSystemPrompt("ctx", "en", "<sdd_standard_hints>\nlowercase filenames\n</sdd_standard_hints>")

	if strings.Contains(prompt, "</grounding_rules><sdd_standard_hints>") {
		t.Error("hints block must be separated from grounding rules by a blank line, not glued on the same line")
	}
	if !strings.Contains(prompt, "</grounding_rules>\n\n<sdd_standard_hints>") {
		t.Error("hints block should follow grounding rules after a blank line")
	}
}

// TestPromptBuilder_BuildSpecUserMessage pins the single-copy contract: the
// project context travels ONLY in the system prompt's <existing_context>;
// the spec user message carries just the template guide. Duplicating the
// context doubled the input tokens of every spec call (audit PR-1).
func TestPromptBuilder_BuildSpecUserMessage(t *testing.T) {
	builder := NewPromptBuilder()

	guide := service.TemplateGuide{Name: "spec", Content: "# Spec template body"}
	msg := builder.BuildSpecUserMessage(guide)

	if !strings.Contains(msg, `<template_guide file="SPEC.md">`) {
		t.Error("BuildSpecUserMessage() should carry the template guide with its file attribute")
	}
	if !strings.Contains(msg, "# Spec template body") {
		t.Error("BuildSpecUserMessage() should include the guide content")
	}
	if strings.Contains(msg, "<project_description>") || strings.Contains(msg, "<existing_context>") {
		t.Error("BuildSpecUserMessage() must NOT duplicate the project context — it already travels in the system prompt")
	}

	override := service.TemplateGuide{Name: "spec", Content: "body", OutputFileName: "spec.md"}
	if msg := builder.BuildSpecUserMessage(override); !strings.Contains(msg, `<template_guide file="spec.md">`) {
		t.Error("BuildSpecUserMessage() should honor the per-standard OutputFileName override")
	}
}
