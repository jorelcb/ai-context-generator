package llm

import (
	"fmt"
	"os"
	"strings"

	"github.com/jorelcb/codify-og/internal/domain/service"
)

// PromptBuilder constructs prompts for the LLM from templates and project description.
type PromptBuilder struct{}

// NewPromptBuilder creates a new PromptBuilder.
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// fileOutputNames maps template guide names to output file names.
var fileOutputNames = map[string]string{
	"agents":            "AGENTS.md",
	"context":           "CONTEXT.md",
	"interactions":      "INTERACTIONS_LOG.md",
	"development_guide": "DEVELOPMENT_GUIDE.md",
	"idioms":            "IDIOMS.md",
	// Spec command output files
	"constitution": "CONSTITUTION.md",
	"spec":         "SPEC.md",
	"plan":         "PLAN.md",
	"tasks":        "TASKS.md",
	// Skills command output files (all produce SKILL.md in separate directories)
	"ddd_entity":          "SKILL.md",
	"clean_arch_layer":    "SKILL.md",
	"bdd_scenario":        "SKILL.md",
	"cqrs_command":        "SKILL.md",
	"hexagonal_port":      "SKILL.md",
	"code_review":         "SKILL.md",
	"test_strategy":       "SKILL.md",
	"refactor_safely":     "SKILL.md",
	"api_design":          "SKILL.md",
	"conventional_commit": "SKILL.md",
	"semantic_versioning": "SKILL.md",
	// Testing skills
	"test_foundational": "SKILL.md",
	"test_tdd":          "SKILL.md",
	"test_bdd":          "SKILL.md",
	// Lifecycle skills (ex-workflows, ADR-0015) — all produce SKILL.md in
	// separate directories like every other skill.
	"bug_fix":         "SKILL.md",
	"release_cycle":   "SKILL.md",
	"speckit_specify": "SKILL.md",
	"speckit_clarify": "SKILL.md",
	"speckit_plan":    "SKILL.md",
	"speckit_tasks":   "SKILL.md",
	"speckit_analyze": "SKILL.md",
	"spec_propose":    "SKILL.md",
	"spec_apply":      "SKILL.md",
	"spec_archive":    "SKILL.md",
}

// localeLanguageNames maps locale codes to their language name for the LLM directive.
var localeLanguageNames = map[string]string{
	"en": "English",
	"es": "Spanish",
}

// FileOutputName returns the output file name for a given template guide name.
// Used as a fallback when no per-guide override is available.
//
// Prefer GuideOutputName(guide) at call sites that have a TemplateGuide in
// hand — it respects the optional guide.OutputFileName override which lets
// SpecStandard adapters dispatch file names without mutating this global
// map (see ADR-0011).
func FileOutputName(guideName string) string {
	if name, ok := fileOutputNames[guideName]; ok {
		return name
	}
	return guideName + ".md"
}

// GuideOutputName resuelve el nombre de archivo para un TemplateGuide.
// Prioriza guide.OutputFileName (cuando lo setea el caller — típicamente la
// spec command para variar SPEC.md vs spec.md por SpecStandard); si está
// vacío, cae al mapping global.
func GuideOutputName(guide service.TemplateGuide) string {
	if guide.OutputFileName != "" {
		return guide.OutputFileName
	}
	return FileOutputName(guide.Name)
}

// outputLanguageName returns the language name for the given locale (defaults to English).
//
// If the locale is non-empty but unknown, a warning is emitted to stderr so the
// caller can detect silent fallbacks during development. The function never
// errors — the LLM still receives a valid language directive.
func outputLanguageName(locale string) string {
	if name, ok := localeLanguageNames[locale]; ok {
		return name
	}
	if locale != "" {
		_, _ = fmt.Fprintf(os.Stderr, "warning: locale %q is not supported, defaulting to English\n", locale)
	}
	return "English"
}

// --- Shared prompt fragments (private helpers) ----------------------------------
//
// These helpers consolidate previously duplicated text across the six prompt
// builders. They produce identical XML blocks regardless of which prompt invokes
// them, so editing the rule body only requires touching one place.

// groundingRulesForGeneratedContent returns the anti-hallucination block used
// by file-generation modes (generate, analyze, spec). It distinguishes
// technical framework choices (LLM may opine) from domain logic (LLM must
// only echo what was stated).
func groundingRulesForGeneratedContent() string {
	return fmt.Sprintf(`<grounding_rules>
CRITICAL — Distinguish between two types of content:

1. TECHNICAL FRAMEWORK (you may opine freely): architectural patterns, project
   structure, code conventions, testing strategy, observability, layer
   responsibilities. This is the template's value.

2. DOMAIN LOGIC (only what the user/context stated): business rules, specific
   validations, default values, edge cases, data formats, concrete behaviors,
   error messages.

%s
</grounding_rules>`, domainLogicGroundingRules())
}

// domainLogicGroundingRules returns the "For domain logic:" bullet block shared
// by the generate and analyze grounding rules. Both prompts diverge in how much
// they trust the technical signals (analyze treats scan data as ground truth)
// but the domain-logic discipline is identical — keeping it here prevents the
// two copies from drifting apart.
func domainLogicGroundingRules() string {
	return `For domain logic:
- Only include what is EXPLICITLY in the project description or scanned context
- DO NOT invent validation rules, default values, formats, or behaviors
- DO NOT generate speculative edge cases or error scenarios
- If a template section asks for domain details the input does not cover,
  mark "[DEFINE: short hint of what is needed]" instead of inventing an answer`
}

// commonOutputRules returns the closing rules block, always last in every
// system prompt. It enforces the response shape (raw markdown, no wrappers,
// no commentary) and the output language.
func commonOutputRules(locale string) string {
	return fmt.Sprintf(`<rules>
- Respond ONLY with the requested content (markdown body or full file)
- DO NOT wrap the response in code blocks
- DO NOT add explanations before or after the content
- Content must be in %s
</rules>`, outputLanguageName(locale))
}

// BuildSystemPromptForFile returns a system prompt for generating a single context file.
//
// The target file name deliberately does NOT appear here — it travels in the
// user message (the file attribute of <template_guide>). Keeping the system
// prompt byte-identical across the per-guide calls of one run is what makes
// the Anthropic prompt cache hit from the second file onward (caching is
// prefix-exact; one interpolated name at the top used to defeat it entirely).
func (b *PromptBuilder) BuildSystemPromptForFile(locale string) string {
	return fmt.Sprintf(`<role>
You are a senior software architect and expert technical writer.
Your task is to generate context files optimized for AI-assisted software development.
The files you generate will be consumed by AI agents as working context.
</role>

<task>
Generate the content for exactly ONE context file per request.
The user message provides the project description and a structural template guide; the target file name is the file attribute of the <template_guide> tag.
</task>

%s

<workflow>
1. Analyze the project description: identify language, architecture, type, key capabilities
2. Read the template guide provided in the user message
3. Mentally separate: what is technical framework (opine freely) vs domain logic (only what was stated)
4. For each template section, generate SPECIFIC and ACTIONABLE content for the described project
5. Where the template uses variables like {{VARIABLE}}, generate real content ONLY if the description supports it; otherwise emit a labelled placeholder of the form [DEFINE: <what is missing>] (always include the colon and a concrete hint — never a bare "[DEFINE]"). Example in context: "Currency handling: [DEFINE: ISO 4217 currency code — USD, EUR, other?]"
6. Verify that no business rule or specific behavior was invented
</workflow>

<output_quality>
- Maximum 200 lines per generated file
- Zero filler sentences or generic boilerplate
- Structured formats (YAML, lists, tables) over prose for configuration and specs
- Every sentence must be actionable and useful for a consuming AI agent
- Critical information at the beginning and end of the file (attention-aware ordering)
- Commands must be exact and copy-pasteable, not generic placeholders
- Use the template guide as structural reference, NOT as a variable replacement template
</output_quality>

%s`, groundingRulesForGeneratedContent(), commonOutputRules(locale))
}

// BuildAnalyzeSystemPromptForFile returns a system prompt optimized for analyze mode.
// Unlike the generate prompt, this treats scan data as factual ground truth from real code.
//
// Same caching contract as BuildSystemPromptForFile: the target file name
// lives in the user message so this prompt is identical across the run.
func (b *PromptBuilder) BuildAnalyzeSystemPromptForFile(locale string) string {
	return fmt.Sprintf(`<role>
You are a senior software architect and expert technical writer.
Your task is to generate context files optimized for AI-assisted software development.
The files you generate will be consumed by AI agents as working context.
</role>

<task>
Generate the content for exactly ONE context file per request.
You will receive a project analysis AUTO-SCANNED from an existing codebase and a structural template guide; the target file name is the file attribute of the <template_guide> tag.
</task>

<scan_trust>
The project description was extracted by scanning a real codebase. The following signals are FACTUAL:
- Language and framework: detected from manifest files (go.mod, package.json, etc.)
- Dependencies: parsed from the actual dependency manifest
- Directory structure: read from the real filesystem
- README content: extracted from the project's README file
- Infrastructure signals: detected from real config files (Dockerfile, CI workflows, etc.)
- Build targets: parsed from actual Makefile/Taskfile if present
- Existing context files: read verbatim from the project
- Testing patterns: detected from real test files and framework dependencies
- CI/CD pipelines: summarized from actual workflow definitions

Trust these signals fully. Generate content that matches the REAL state of the codebase.
</scan_trust>

<grounding_rules>
CRITICAL RULE — Distinguish between two types of content:

1. TECHNICAL FRAMEWORK + SCANNED SIGNALS (generate with confidence): architectural patterns,
   project structure, code conventions, testing strategy, build commands, CI/CD pipeline,
   dependencies. These are backed by real scan data — use them directly.

2. DOMAIN LOGIC (only what is explicitly stated): business rules, specific validations,
   default values, edge cases, data formats, concrete behaviors, error messages.

For scanned signals:
- Use detected language, framework, and dependencies as ground truth
- Generate exact commands from build targets (make build, task test, etc.)
- Reference the actual directory structure when describing the project layout
- Incorporate existing context files to maintain continuity with prior decisions
- Describe the real CI/CD pipeline, not a hypothetical one

%s
</grounding_rules>

<workflow>
1. Analyze the scanned project data: language, framework, dependencies, structure, infrastructure
2. Read existing context files if present — they represent prior architectural decisions
3. Read the template guide provided in the user message
4. For each template section, generate SPECIFIC content grounded in the scan data
5. For commands sections, use EXACT build targets detected (not generic placeholders)
6. Where the template asks for domain details not in the scan, emit a labelled placeholder "[DEFINE: <what is missing>]" — always include the colon and a concrete hint, never a bare "[DEFINE]". Example in context: "Currency handling: [DEFINE: ISO 4217 currency code — USD, EUR, other?]"
</workflow>

<output_quality>
- Maximum 200 lines per generated file
- Zero filler sentences or generic boilerplate
- Structured formats (YAML, lists, tables) over prose for configuration and specs
- Every sentence must be actionable and useful for a consuming AI agent
- Critical information at the beginning and end of the file (attention-aware ordering)
- Commands must be exact and copy-pasteable, derived from actual build targets
- Use the template guide as structural reference, NOT as a variable replacement template
</output_quality>

%s`, domainLogicGroundingRules(), commonOutputRules(locale))
}

// BuildUserMessageForFile constructs the user message for generating a single file.
func (b *PromptBuilder) BuildUserMessageForFile(req service.GenerationRequest, guide service.TemplateGuide) string {
	var sb strings.Builder

	sb.WriteString("<project_description>\n")
	sb.WriteString(req.ProjectDescription)
	sb.WriteString("\n</project_description>\n\n")

	hasMetadata := req.Language != "" || req.ProjectType != "" || req.Architecture != ""
	if hasMetadata {
		sb.WriteString("<project_metadata>\n")
		if req.Language != "" {
			sb.WriteString(fmt.Sprintf("- Language: %s\n", req.Language))
		}
		if req.ProjectType != "" {
			sb.WriteString(fmt.Sprintf("- Project type: %s\n", req.ProjectType))
		}
		if req.Architecture != "" {
			sb.WriteString(fmt.Sprintf("- Architecture: %s\n", req.Architecture))
		}
		sb.WriteString("</project_metadata>\n\n")
	}

	sb.WriteString(fmt.Sprintf("<template_guide file=\"%s\">\n", GuideOutputName(guide)))
	sb.WriteString(guide.Content)
	sb.WriteString("\n</template_guide>\n")

	return sb.String()
}

// BuildSpecSystemPrompt returns a system prompt for generating spec files from existing context.
//
// standardHints es el bloque que el SpecStandard activo aporta para reforzar
// convenciones específicas del estándar (layout, naming, etc.). Vacío para
// OpenSpec — la base ya describe ese formato. No-vacío para Spec-Kit
// (lowercase filenames, per-feature dir, etc.).
//
// El contexto del proyecto viaja ÚNICAMENTE acá (<existing_context>); el user
// message de spec lleva solo el template guide (BuildSpecUserMessage). Antes
// se enviaba duplicado en ambos lados — miles de tokens repetidos por cada
// artefacto generado (auditoría PR-1).
func (b *PromptBuilder) BuildSpecSystemPrompt(existingContext string, locale string, standardHints string) string {
	hints := strings.TrimSpace(standardHints)
	if hints != "" {
		hints = "\n\n" + hints
	}
	return fmt.Sprintf(`<role>
You are a senior software architect specialized in technical specifications.
Your task is to generate SDD (Spec-Driven Development) specification documents from an existing project context.
The context you receive was previously generated and contains the project's architecture, patterns, and decisions.
</role>

<task>
Generate actionable specification documents based on the existing project context.
The complete project context is in <existing_context> below; the user message provides the template guide for the specific file to generate (its file attribute names the target file).
</task>

<existing_context>
%s
</existing_context>

%s%s

<workflow>
1. Deeply analyze the existing context: architecture, stack, patterns, constraints
2. Read the template guide provided in the user message
3. Identify which domain information is EXPLICITLY in the context and which you would have to invent
4. Generate CONCRETE specifications COHERENT with the existing context
5. For domain details not covered in the context, emit "[DEFINE: <what is missing>]" instead of inventing — always include the colon and a concrete hint, never a bare "[DEFINE]"
6. Each specification must be implementable and verifiable
7. Maintain total coherence with already documented architectural decisions
</workflow>

<output_quality>
- Actionable specifications, not generic
- Verifiable acceptance criteria based on real context information
- Total coherence with existing context
- Structured formats (lists, tables, YAML) over prose
- Maximum 200 lines per file
- Zero invented business rules — if not in the context, emit "[DEFINE: <what is missing>]" with a concrete hint
- Base ALL content on the existing context provided
</output_quality>

%s`, existingContext, groundingRulesForGeneratedContent(), hints, commonOutputRules(locale))
}

// BuildSpecUserMessage constructs the user message for generating a single
// spec artifact. Deliberadamente lleva SOLO el template guide: el contexto del
// proyecto ya viaja en el <existing_context> del system prompt y repetirlo acá
// duplicaba los tokens de entrada de cada llamada de spec (auditoría PR-1) —
// además de no ser cacheable, al variar el user message por artefacto.
func (b *PromptBuilder) BuildSpecUserMessage(guide service.TemplateGuide) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<template_guide file=%q>\n", GuideOutputName(guide)))
	sb.WriteString(guide.Content)
	sb.WriteString("\n</template_guide>\n")
	return sb.String()
}
