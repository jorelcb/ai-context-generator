package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jorelcb/codify/internal/application/dto"
	"github.com/jorelcb/codify/internal/domain/service"
)

// GenerateSpecCommand orchestrates LLM-based spec file generation from existing context.
type GenerateSpecCommand struct {
	llmProvider      service.LLMProvider
	fileWriter       service.FileWriter
	directoryManager service.DirectoryManager
}

// NewGenerateSpecCommand creates a new GenerateSpecCommand.
func NewGenerateSpecCommand(
	llmProvider service.LLMProvider,
	fileWriter service.FileWriter,
	directoryManager service.DirectoryManager,
) *GenerateSpecCommand {
	return &GenerateSpecCommand{
		llmProvider:      llmProvider,
		fileWriter:       fileWriter,
		directoryManager: directoryManager,
	}
}

// Execute runs the spec generation pipeline:
// 1. Build generation request with existing context and spec mode
// 2. Call LLM provider
// 3. Place each generated file at its artifact's resolved directory
func (c *GenerateSpecCommand) Execute(
	ctx context.Context,
	config *dto.SpecConfig,
	existingContext string,
	templateGuides []service.TemplateGuide,
) (*dto.GenerationResult, error) {
	// 1. Build generation request in spec mode. SDDStandardHints carries
	//    the active standard's prompt addendum so the LLM respects per-standard
	//    conventions (file naming, layout, etc.).
	//    ExistingContext viaja una sola vez (al <existing_context> del system
	//    prompt); NO se duplica en ProjectDescription — el user message de spec
	//    lleva solo el template guide (auditoría PR-1).
	req := service.GenerationRequest{
		TemplateGuides:   templateGuides,
		ExistingContext:  existingContext,
		Mode:             "spec",
		Locale:           config.Locale,
		SDDStandardHints: config.StandardHints,
	}

	// 2. Call LLM provider
	response, err := c.llmProvider.GenerateContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM spec generation failed: %w", err)
	}

	// 3. Place each generated file at its artifact's directory. The directory
	//    comes from the active SpecStandard (SpecArtifact.Dir), with the
	//    {feature} token expanded to the feature/capability slug — so Spec-Kit
	//    lands under specs/<feature>/, OpenSpec under openspec/specs/<cap>/, and
	//    a project-level artifact (constitution) under .specify/memory/.
	byFile := make(map[string]service.SpecArtifact, len(config.Artifacts))
	for _, a := range config.Artifacts {
		byFile[a.FileName] = a
	}

	var generatedFiles []string
	rootForResult := config.OutputPath
	for _, file := range response.Files {
		artifact, ok := byFile[file.Name]
		if !ok {
			// No artifact metadata — write at the output root (defensive).
			artifact = service.SpecArtifact{FileName: file.Name}
		}
		dir := strings.ReplaceAll(artifact.Dir, service.FeatureToken, config.FeatureID)
		targetDir := filepath.Join(config.OutputPath, dir)
		if err := c.directoryManager.CreateDir(targetDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", targetDir, err)
		}
		filePath := filepath.Join(targetDir, file.Name)

		// SkipIfExists protects project-level artifacts (e.g. the constitution)
		// from being clobbered on a per-feature re-run.
		if artifact.SkipIfExists {
			if exists, err := c.directoryManager.Exists(filePath); err != nil {
				return nil, fmt.Errorf("check existing %s: %w", filePath, err)
			} else if exists {
				continue
			}
		}

		if err := c.fileWriter.WriteFile(filePath, []byte(file.Content), os.FileMode(0o644)); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
		}
		generatedFiles = append(generatedFiles, filePath)
	}

	return &dto.GenerationResult{
		OutputPath:     rootForResult,
		GeneratedFiles: generatedFiles,
		Model:          response.Model,
		TokensIn:       response.TokensIn,
		TokensOut:      response.TokensOut,
	}, nil
}
