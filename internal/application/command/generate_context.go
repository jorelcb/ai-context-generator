package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jorelcb/codify-og/internal/application/dto"
	"github.com/jorelcb/codify-og/internal/domain/service"
)

// rootFiles are written to the project root, not the context/ subdirectory.
var rootFiles = map[string]bool{
	"AGENTS.md": true,
}

// GenerateContextCommand orchestrates LLM-based context file generation.
type GenerateContextCommand struct {
	llmProvider      service.LLMProvider
	fileWriter       service.FileWriter
	directoryManager service.DirectoryManager
}

// NewGenerateContextCommand creates a new GenerateContextCommand.
func NewGenerateContextCommand(
	llmProvider service.LLMProvider,
	fileWriter service.FileWriter,
	directoryManager service.DirectoryManager,
) *GenerateContextCommand {
	return &GenerateContextCommand{
		llmProvider:      llmProvider,
		fileWriter:       fileWriter,
		directoryManager: directoryManager,
	}
}

// Execute runs the full context generation pipeline:
// 1. Build generation request from config + templates
// 2. Call LLM provider
// 3. Create output directories
// 4. Write generated files to disk (AGENTS.md at root, rest in context/)
func (c *GenerateContextCommand) Execute(
	ctx context.Context,
	config *dto.ProjectConfig,
	templateGuides []service.TemplateGuide,
) (*dto.GenerationResult, error) {
	// 1. Build generation request
	req := service.GenerationRequest{
		ProjectDescription: config.Description,
		TemplateGuides:     templateGuides,
		Language:           config.Language,
		ProjectType:        config.Type,
		Architecture:       config.Architecture,
		Locale:             config.Locale,
		Mode:               config.Mode,
	}

	// 2. Call LLM provider
	response, err := c.llmProvider.GenerateContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// 3. Create output directories
	contextDir := filepath.Join(config.OutputPath, "context")
	if err := c.directoryManager.CreateDir(config.OutputPath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := c.directoryManager.CreateDir(contextDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create context directory: %w", err)
	}

	// 4. Write each generated file
	var generatedFiles []string
	for _, file := range response.Files {
		// AGENTS.md goes to project root, rest goes to context/
		var filePath string
		if rootFiles[file.Name] {
			filePath = filepath.Join(config.OutputPath, file.Name)
		} else {
			filePath = filepath.Join(contextDir, file.Name)
		}

		if err := c.fileWriter.WriteFile(filePath, []byte(file.Content), os.FileMode(0o644)); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
		}
		generatedFiles = append(generatedFiles, filePath)

		// Claude Code does not read AGENTS.md natively — it reads CLAUDE.md.
		// Emit the official bridge (a CLAUDE.md that imports @AGENTS.md plus a
		// space for Claude-specific notes) so the agent-agnostic AGENTS.md stays
		// the canonical root (ADR-0014 / D5-B) while Claude Code picks it up.
		// Never clobber an existing CLAUDE.md — on a brownfield `analyze` the
		// repo may already have a curated one.
		if file.Name == "AGENTS.md" {
			if bridge, err := c.writeClaudeBridge(config.OutputPath); err != nil {
				return nil, err
			} else if bridge != "" {
				generatedFiles = append(generatedFiles, bridge)
			}
		}
	}

	return &dto.GenerationResult{
		OutputPath:     config.OutputPath,
		GeneratedFiles: generatedFiles,
		Model:          response.Model,
		TokensIn:       response.TokensIn,
		TokensOut:      response.TokensOut,
	}, nil
}

// claudeBridge is the static CLAUDE.md that points Claude Code at the shared
// AGENTS.md. It is deterministic (no LLM) and intentionally minimal — the real
// context lives in AGENTS.md so every agent benefits; this file only adds the
// Claude-specific import plus room for Claude-only notes.
const claudeBridge = `# CLAUDE.md

This file gives Claude Code its working context. The shared, agent-agnostic
project instructions live in AGENTS.md (the open standard most coding agents
read); this file imports them and adds anything Claude-specific.

@AGENTS.md

## Claude-specific notes

<!-- Add instructions that apply only to Claude Code here (e.g. preferred
     skills, slash-command workflows). Keep shared project context in AGENTS.md
     so every agent — Claude Code, Cursor, Codex, Gemini CLI — benefits. -->
`

// writeClaudeBridge writes the CLAUDE.md bridge to outputPath when one does not
// already exist, returning the path written (or "" when skipped). Skipping an
// existing CLAUDE.md keeps brownfield `analyze` runs non-destructive.
func (c *GenerateContextCommand) writeClaudeBridge(outputPath string) (string, error) {
	bridgePath := filepath.Join(outputPath, "CLAUDE.md")
	exists, err := c.directoryManager.Exists(bridgePath)
	if err != nil {
		return "", fmt.Errorf("check existing CLAUDE.md: %w", err)
	}
	if exists {
		return "", nil
	}
	if err := c.fileWriter.WriteFile(bridgePath, []byte(claudeBridge), os.FileMode(0o644)); err != nil {
		return "", fmt.Errorf("write CLAUDE.md bridge: %w", err)
	}
	return bridgePath, nil
}
