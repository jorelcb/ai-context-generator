package llm

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"google.golang.org/genai"

	"github.com/jorelcb/codify/internal/domain/service"
)

const defaultGeminiModel = "gemini-3.1-pro-preview"

// GeminiProvider implements service.LLMProvider using the Google Gemini API.
type GeminiProvider struct {
	client        *genai.Client
	model         string
	promptBuilder *PromptBuilder
	progressOut   io.Writer
}

// NewGeminiProvider creates a new GeminiProvider.
// If apiKey is empty, the SDK will use the GEMINI_API_KEY or GOOGLE_API_KEY env var.
// If model is empty, defaults to gemini-3.1-pro-preview.
// If progressOut is non-nil, progress messages will be written to it.
func NewGeminiProvider(ctx context.Context, apiKey string, model string, progressOut io.Writer) (*GeminiProvider, error) {
	config := &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
	}
	if apiKey != "" {
		config.APIKey = apiKey
	}

	client, err := genai.NewClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	if model == "" {
		model = defaultGeminiModel
	}

	return &GeminiProvider{
		client:        client,
		model:         model,
		promptBuilder: NewPromptBuilder(),
		progressOut:   progressOut,
	}, nil
}

// GenerateContext generates all context files by making one API call per file.
func (p *GeminiProvider) GenerateContext(ctx context.Context, req service.GenerationRequest) (*service.GenerationResponse, error) {
	start := time.Now()
	var files []service.GeneratedFile
	var totalIn, totalOut int
	success := true

	for i, guide := range req.TemplateGuides {
		outputName := GuideOutputName(guide)

		if p.progressOut != nil {
			fmt.Fprintf(p.progressOut, "  [%d/%d] Generating %s...", i+1, len(req.TemplateGuides), outputName)
		}

		content, tokensIn, tokensOut, err := p.generateSingleFile(ctx, req, guide)
		totalIn += tokensIn
		totalOut += tokensOut

		if err != nil {
			success = false
			recordUsage("gemini", p.model, commandFromMode(req.Mode), totalIn, totalOut, time.Since(start), false)
			return nil, fmt.Errorf("failed to generate %s: %w", outputName, err)
		}

		if p.progressOut != nil {
			fmt.Fprintf(p.progressOut, " done (%d tokens)\n", tokensOut)
		}

		validation := ValidateOutput(content, req.Mode, outputName)
		if validation.Fatal {
			// One retry per file: fatal shape failures (empty output, stubs)
			// are usually transient, and aborting here used to throw away every
			// file already generated in this run.
			if p.progressOut != nil {
				fmt.Fprintf(p.progressOut, "  [%d/%d] %s rejected (%v) — retrying...", i+1, len(req.TemplateGuides), outputName, validation.Warnings)
			}
			retryContent, retryIn, retryOut, retryErr := p.generateSingleFile(ctx, req, guide)
			totalIn += retryIn
			totalOut += retryOut
			if retryErr != nil {
				recordUsage("gemini", p.model, commandFromMode(req.Mode), totalIn, totalOut, time.Since(start), false)
				return nil, fmt.Errorf("failed to regenerate %s after validator rejection: %w", outputName, retryErr)
			}
			validation = ValidateOutput(retryContent, req.Mode, outputName)
			if validation.Fatal {
				recordUsage("gemini", p.model, commandFromMode(req.Mode), totalIn, totalOut, time.Since(start), false)
				return nil, fmt.Errorf("output for %s was rejected by validator twice: %v", outputName, validation.Warnings)
			}
			content = retryContent
			if p.progressOut != nil {
				fmt.Fprintf(p.progressOut, " done (%d tokens)\n", retryOut)
			}
		}
		emitValidationFeedback(p.progressOut, outputName, validation)

		files = append(files, service.GeneratedFile{
			Name:    outputName,
			Content: content,
		})
	}

	recordUsage("gemini", p.model, commandFromMode(req.Mode), totalIn, totalOut, time.Since(start), success)

	return &service.GenerationResponse{
		Files:     files,
		Model:     p.model,
		TokensIn:  totalIn,
		TokensOut: totalOut,
	}, nil
}

// generateSingleFile makes one streaming API call to generate a single context file.
func (p *GeminiProvider) generateSingleFile(
	ctx context.Context,
	req service.GenerationRequest,
	guide service.TemplateGuide,
) (content string, tokensIn int, tokensOut int, err error) {
	var systemPrompt string
	var userMessage string
	switch req.Mode {
	case "spec":
		systemPrompt = p.promptBuilder.BuildSpecSystemPrompt(req.ExistingContext, req.Locale, req.SDDStandardHints)
		userMessage = p.promptBuilder.BuildSpecUserMessage(guide)
	case "analyze":
		systemPrompt = p.promptBuilder.BuildAnalyzeSystemPromptForFile(req.Locale)
		userMessage = p.promptBuilder.BuildUserMessageForFile(req, guide)
	default:
		if req.Mode != "" && req.Mode != "generate" {
			return "", 0, 0, fmt.Errorf("unknown generation mode: %q", req.Mode)
		}
		systemPrompt = p.promptBuilder.BuildSystemPromptForFile(req.Locale)
		userMessage = p.promptBuilder.BuildUserMessageForFile(req, guide)
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		MaxOutputTokens:   16000,
	}

	var textBuilder strings.Builder
	var inTokens, outTokens int32
	var finishReason genai.FinishReason

	for resp, err := range p.client.Models.GenerateContentStream(
		ctx,
		p.model,
		genai.Text(userMessage),
		config,
	) {
		if err != nil {
			return "", int(inTokens), int(outTokens), fmt.Errorf("streaming failed: %w", err)
		}

		if resp.UsageMetadata != nil {
			inTokens = resp.UsageMetadata.PromptTokenCount
			outTokens = resp.UsageMetadata.CandidatesTokenCount
		}

		for _, candidate := range resp.Candidates {
			if candidate.FinishReason != "" {
				finishReason = candidate.FinishReason
			}
			if candidate.Content != nil {
				for _, part := range candidate.Content.Parts {
					if part.Text != "" {
						textBuilder.WriteString(part.Text)
					}
				}
			}
		}
	}

	text := textBuilder.String()
	if text == "" {
		return "", int(inTokens), int(outTokens), fmt.Errorf("empty response from LLM")
	}
	// Truncated output must never reach disk: a MAX_TOKENS finish means the
	// file is incomplete even though the stream ended without error.
	if finishReason == genai.FinishReasonMaxTokens {
		return "", int(inTokens), int(outTokens), fmt.Errorf("response truncated: max output tokens hit after %d tokens — the file is incomplete and was discarded", outTokens)
	}

	return text, int(inTokens), int(outTokens), nil
}

// EvaluatePrompt implements service.LLMProvider for one-shot prompt evaluation.
// Mirrors the AnthropicProvider implementation: sends a single non-streaming
// request and returns the raw text plus token counts. Records usage via the
// shared shim.
func (p *GeminiProvider) EvaluatePrompt(ctx context.Context, req service.EvaluationRequest) (*service.EvaluationResponse, error) {
	start := time.Now()
	maxTokens := int32(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 4000
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(req.SystemPrompt, genai.RoleUser),
		MaxOutputTokens:   maxTokens,
	}
	if req.OutputSchema != nil {
		// Native structured output: constrain the response to the JSON schema
		// so the returned text is guaranteed valid JSON (no fences, no prose).
		config.ResponseMIMEType = "application/json"
		config.ResponseJsonSchema = req.OutputSchema
	}

	var textBuilder strings.Builder
	var inTokens, outTokens int32
	var streamErr error

	for resp, err := range p.client.Models.GenerateContentStream(
		ctx,
		p.model,
		genai.Text(req.UserPrompt),
		config,
	) {
		if err != nil {
			streamErr = err
			break
		}
		if resp.UsageMetadata != nil {
			inTokens = resp.UsageMetadata.PromptTokenCount
			outTokens = resp.UsageMetadata.CandidatesTokenCount
		}
		for _, candidate := range resp.Candidates {
			if candidate.Content != nil {
				for _, part := range candidate.Content.Parts {
					if part.Text != "" {
						textBuilder.WriteString(part.Text)
					}
				}
			}
		}
	}

	cmd := req.Command
	if cmd == "" {
		cmd = "evaluate"
	}

	if streamErr != nil {
		recordUsage("gemini", p.model, cmd, int(inTokens), int(outTokens), time.Since(start), false)
		return nil, fmt.Errorf("streaming failed: %w", streamErr)
	}

	text := textBuilder.String()
	recordUsage("gemini", p.model, cmd, int(inTokens), int(outTokens), time.Since(start), text != "")
	if text == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	return &service.EvaluationResponse{
		Text:      text,
		Model:     p.model,
		TokensIn:  int(inTokens),
		TokensOut: int(outTokens),
	}, nil
}
