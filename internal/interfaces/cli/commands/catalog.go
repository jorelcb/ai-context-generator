package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	root "github.com/jorelcb/codify"
	"github.com/jorelcb/codify/internal/application/command"
	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/infrastructure/llm"
	"github.com/jorelcb/codify/internal/infrastructure/packagesource"
	"github.com/jorelcb/codify/internal/infrastructure/targetinstaller"
)

// catalogParams groups the flags of `codify catalog`.
type catalogParams struct {
	ecosystem   string
	pkgType     string
	packages    string
	scope       string
	mode        string
	model       string
	context     string
	marketplace string
	list        bool
}

// defaultMarketplace is the trusted, Anthropic-managed plugin directory used
// when `--type plugin` is requested without an explicit `--marketplace`.
const defaultMarketplace = "anthropics/claude-plugins-official"

// typeToTarget maps a user-facing catalog type to a Claude Target.
var typeToTarget = map[string]catalog.Target{
	"skill":  catalog.TargetClaudeSkill,
	"hook":   catalog.TargetClaudeHook,
	"plugin": catalog.TargetClaudePlugin,
}

// NewCatalogCmd creates the `catalog` command — the unified surface for
// browsing and installing ecosystem packages (skills, hooks). See ADR-0010.
func NewCatalogCmd() *cobra.Command {
	var p catalogParams

	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Browse and install ecosystem packages (skills, hooks, plugins)",
		Long: `Browse and install packages from codify's catalog into your agent.

A package maps one-to-one to a native ecosystem primitive — a Claude skill,
hook, or plugin. Skills and hooks come from the built-in/local catalog; plugins
come from a Claude plugin marketplace (a '.claude-plugin/marketplace.json'),
default the Anthropic-managed directory.

Modes:
  Interactive (no flags, TTY): pick ecosystem → type → packages → scope.
  List:        codify catalog --list [--type skill|hook|plugin] [--scope project]
  Install:     codify catalog --type skill  --package ddd-entity,hexagonal-port --scope project
               codify catalog --type plugin --package gopls-lsp --scope workstation

Scopes:
  project      .claude/ in the current repository
  workstation  ~/.claude/ shared across all your projects (alias: global)

Plugins are installed by delegating to Claude Code's own plugin CLI (requires
'claude' on PATH); see --marketplace to point at another marketplace.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			explicit := make(map[string]bool)
			cmd.Flags().Visit(func(f *pflag.Flag) { explicit[f.Name] = true })
			return runCatalog(p, explicit)
		},
	}

	cmd.Flags().StringVar(&p.ecosystem, "ecosystem", "claude", "Target ecosystem (MVP: claude)")
	cmd.Flags().StringVar(&p.pkgType, "type", "", "Package type: skill, hook, or plugin")
	cmd.Flags().StringVar(&p.packages, "package", "", "Comma-separated package IDs to install")
	cmd.Flags().StringVar(&p.scope, "scope", "", "Install scope: project or workstation (alias: global)")
	cmd.Flags().StringVar(&p.mode, "mode", "static", "Skill mode: static (embedded template) or personalized (LLM-adapted)")
	cmd.Flags().StringVar(&p.model, "model", "", "LLM model for personalized mode (e.g. claude-sonnet-4-6)")
	cmd.Flags().StringVar(&p.context, "context", "", "Project context for personalized mode")
	cmd.Flags().StringVar(&p.marketplace, "marketplace", defaultMarketplace, "Plugin marketplace (owner/repo or marketplace.json URL); used with --type plugin")
	cmd.Flags().BoolVar(&p.list, "list", false, "List available packages instead of installing")

	return cmd
}

func runCatalog(p catalogParams, explicit map[string]bool) error {
	if p.ecosystem != "" && p.ecosystem != "claude" {
		return fmt.Errorf("ecosystem %q not supported yet (MVP: claude)", p.ecosystem)
	}

	ctx := context.Background()

	if p.list {
		return catalogList(ctx, p)
	}

	// Non-interactive install when the actionable flags are present.
	if explicit["type"] || explicit["package"] {
		return catalogInstallFromFlags(ctx, p)
	}

	if !isInteractive() {
		return fmt.Errorf("codify catalog needs a TTY for interactive mode; pass --type and --package for non-interactive install, or --list to browse")
	}
	return catalogInteractive(ctx, p)
}

// catalogRegistry wires the Claude installers: skills/hooks (filesystem) and
// plugins (delegated to the native plugin CLI). The registry routes by Target.
func catalogRegistry() *targetinstaller.Registry {
	return targetinstaller.NewRegistry(mustClaudeInstaller(), mustClaudePluginInstaller())
}

// pluginService browses + installs from a Claude plugin marketplace. Plugins
// install by delegating to the agent's plugin CLI (ClaudePluginInstaller).
func pluginService(marketplaceRef string) *command.CatalogService {
	source := packagesource.NewPluginMarketplaceSource(marketplaceRef)
	return command.NewCatalogService(source, catalogRegistry())
}

// browseService returns the read service for a type. Plugins read from the
// marketplace (network); skills/hooks read the static embedded+local catalog
// (no API key, no network).
func browseService(tp string, p catalogParams) *command.CatalogService {
	if tp == "plugin" {
		return pluginService(p.marketplace)
	}
	return embeddedService()
}

// localSourceRoot is the convention directory for personal/team packages,
// composed on top of the built-in catalog. Missing dir = no local packages.
func localSourceRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".codify", "sources")
	}
	return filepath.Join(home, ".codify", "sources")
}

// embeddedService is the static path: built-in embedded packages composed with
// any local packages under ~/.codify/sources/ (local overrides built-in on ID
// collision). Used for browsing (List/Installed) and static installs — no API
// key required.
func embeddedService() *command.CatalogService {
	embedded := packagesource.NewEmbeddedSource(root.TemplatesFS, codifyVersion)
	local := packagesource.NewLocalDirectorySource(localSourceRoot())
	source := packagesource.NewCompositeSource(embedded, local) // local overrides embedded
	return command.NewCatalogService(source, catalogRegistry())
}

// personalizedService is the LLM path: built-in skills are adapted to
// projectContext via the PersonalizingSource decorator, composed with local
// packages (which install statically — local packages are pre-authored, not
// LLM-adapted). Requires a usable model + API key.
func personalizedService(ctx context.Context, model, projectContext string) (*command.CatalogService, error) {
	apiKey, err := llm.ResolveAPIKey(model)
	if err != nil {
		return nil, err
	}
	provider, err := llm.NewProvider(ctx, model, apiKey, os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("create LLM provider: %w", err)
	}
	embedded := packagesource.NewEmbeddedSource(root.TemplatesFS, codifyVersion)
	personalizing := packagesource.NewPersonalizingSource(embedded, provider, projectContext, "en", "claude")
	local := packagesource.NewLocalDirectorySource(localSourceRoot())
	source := packagesource.NewCompositeSource(personalizing, local)
	return command.NewCatalogService(source, catalogRegistry()), nil
}

// mustClaudeInstaller builds a ClaudeInstaller against the real cwd/home. On
// the (rare) failure to resolve those, it falls back to relative roots so the
// command still runs rather than panicking.
func mustClaudeInstaller() *targetinstaller.ClaudeInstaller {
	inst, err := targetinstaller.NewClaudeInstaller()
	if err != nil {
		return targetinstaller.NewClaudeInstallerWithRoots(".", ".")
	}
	return inst
}

// mustClaudePluginInstaller builds a ClaudePluginInstaller. On failure to
// resolve home it falls back to a relative config dir; the actual install
// still surfaces a clear error if `claude` is absent.
func mustClaudePluginInstaller() *targetinstaller.ClaudePluginInstaller {
	inst, err := targetinstaller.NewClaudePluginInstaller()
	if err != nil {
		return targetinstaller.NewClaudePluginInstallerWith(targetinstaller.NewExecRunner(), ".claude")
	}
	return inst
}

func resolveScope(s string) (catalog.Scope, error) {
	switch s {
	case "project":
		return catalog.ScopeProject, nil
	case "workstation", "global":
		return catalog.ScopeWorkstation, nil
	default:
		return "", fmt.Errorf("invalid scope %q (use project or workstation)", s)
	}
}

// catalogList prints available packages, grouped by type, badged with their
// source, and marked when already installed in the resolved scope.
func catalogList(ctx context.Context, p catalogParams) error {
	scope := catalog.ScopeProject
	if p.scope != "" {
		s, err := resolveScope(p.scope)
		if err != nil {
			return err
		}
		scope = s
	}

	// Default browse is skill+hook (local/built-in, fast). Plugins require an
	// explicit `--type plugin` since listing them hits the marketplace network.
	types := []string{"skill", "hook"}
	if p.pkgType != "" {
		if _, ok := typeToTarget[p.pkgType]; !ok {
			return fmt.Errorf("invalid type %q (use skill, hook, or plugin)", p.pkgType)
		}
		types = []string{p.pkgType}
	}

	fmt.Println()
	fmt.Printf("Catalog · ecosystem: claude · scope: %s\n", scope)
	fmt.Println(strings.Repeat("─", 48))

	for _, tp := range types {
		target := typeToTarget[tp]
		svc := browseService(tp, p)
		available, err := svc.Available(ctx, target)
		if err != nil {
			return err
		}
		installed, err := svc.Installed(ctx, target, scope)
		if err != nil {
			return err
		}
		installedSet := make(map[string]bool, len(installed))
		for _, ip := range installed {
			installedSet[ip.ID] = true
		}

		sort.Slice(available, func(i, j int) bool { return available[i].ID < available[j].ID })

		fmt.Printf("\n%s (%d)\n", typeLabel(tp), len(available))
		for _, m := range available {
			marker := " "
			if installedSet[m.ID] {
				marker = "✓"
			}
			fmt.Printf("  [%s] %-26s [%s] %s\n", marker, m.ID, m.Source.Kind, m.Description)
		}
	}
	fmt.Println()
	fmt.Println("Install with: codify catalog --type <skill|hook|plugin> --package <id,...> --scope <project|workstation>")
	return nil
}

// installService returns the CatalogService for installing the given type.
// Plugins use the marketplace source (+ plugin installer). Skills in
// personalized mode use the LLM source; everything else the static
// embedded+local catalog. Reads mode/model/context/marketplace from p (the
// interactive flow stuffs its resolved values back into p first).
func installService(ctx context.Context, tp string, p catalogParams) (*command.CatalogService, error) {
	if tp == "plugin" {
		return pluginService(p.marketplace), nil
	}
	if p.mode == "personalized" {
		if typeToTarget[tp] != catalog.TargetClaudeSkill {
			fmt.Fprintf(os.Stderr, "→ personalized mode applies to skills only; installing %q statically\n", tp)
			return embeddedService(), nil
		}
		if p.context == "" {
			return nil, fmt.Errorf("personalized mode requires --context (a project description)")
		}
		if p.model == "" {
			return nil, fmt.Errorf("personalized mode requires --model (or run interactively to pick one)")
		}
		return personalizedService(ctx, p.model, p.context)
	}
	return embeddedService(), nil
}

func catalogInstallFromFlags(ctx context.Context, p catalogParams) error {
	if p.pkgType == "" {
		return fmt.Errorf("--type is required for install (skill, hook, or plugin)")
	}
	target, ok := typeToTarget[p.pkgType]
	if !ok {
		return fmt.Errorf("invalid type %q (use skill, hook, or plugin)", p.pkgType)
	}
	ids := splitCSV(p.packages)
	if len(ids) == 0 {
		return fmt.Errorf("--package is required for install (comma-separated IDs)")
	}
	scopeStr := p.scope
	if scopeStr == "" {
		scopeStr = "project"
	}
	scope, err := resolveScope(scopeStr)
	if err != nil {
		return err
	}

	svc, err := installService(ctx, p.pkgType, p)
	if err != nil {
		return err
	}
	out, err := svc.Install(ctx, command.InstallRequest{Target: target, IDs: ids, Scope: scope})
	printInstallOutcome(out, scope)
	return err
}

func catalogInteractive(ctx context.Context, p catalogParams) error {
	// Step 1 — ecosystem. MVP ships Claude only; auto-select with a notice
	// rather than a single-option menu.
	fmt.Fprintln(os.Stderr, "→ Ecosystem: claude (the only one supported in this version)")

	// Step 2 — type tab.
	tp, err := promptSelect("Package type", []selectOption{
		{"Skills (prompt-based behaviors)", "skill"},
		{"Hooks (deterministic guardrails)", "hook"},
		{"Plugins (Claude marketplace bundles)", "plugin"},
	}, "skill")
	if err != nil {
		return err
	}
	target := typeToTarget[tp]

	// Step 3 — mode (skills only; hooks are catalog-driven, no LLM).
	mode := "static"
	model := p.model
	projectContext := p.context
	if tp == "skill" {
		mode, err = promptSelect("Skill mode", []selectOption{
			{"Static (instant, embedded template)", "static"},
			{"Personalized (LLM-adapted to your project)", "personalized"},
		}, "static")
		if err != nil {
			return err
		}
		if mode == "personalized" {
			if projectContext == "" {
				projectContext, err = promptInput("Describe your project (stack, architecture, domain)", "")
				if err != nil {
					return err
				}
			}
			if projectContext == "" {
				return fmt.Errorf("personalized mode requires a project description")
			}
			if model == "" {
				model, err = promptModel()
				if err != nil {
					return err
				}
			}
		}
	}

	// Step 4 — scope (needed to mark installed state).
	scopeStr, err := promptSelect("Install scope", []selectOption{
		{"Project (.claude/ in this repo)", "project"},
		{"Workstation (~/.claude/, all projects)", "workstation"},
	}, "project")
	if err != nil {
		return err
	}
	scope, err := resolveScope(scopeStr)
	if err != nil {
		return err
	}

	// Step 5 — package multi-select, marking already-installed entries.
	// Browsing uses the type's read source (marketplace for plugins; static
	// embedded+local for skills/hooks — no API key needed to list).
	if tp == "plugin" {
		fmt.Fprintf(os.Stderr, "→ Fetching plugins from %s …\n", p.marketplace)
	}
	browse := browseService(tp, p)
	available, err := browse.Available(ctx, target)
	if err != nil {
		return err
	}
	if len(available) == 0 {
		fmt.Printf("No %s packages available.\n", tp)
		return nil
	}
	installed, err := browse.Installed(ctx, target, scope)
	if err != nil {
		return err
	}
	installedSet := make(map[string]bool, len(installed))
	for _, ip := range installed {
		installedSet[ip.ID] = true
	}
	sort.Slice(available, func(i, j int) bool { return available[i].ID < available[j].ID })

	options := make([]selectOption, 0, len(available))
	for _, m := range available {
		label := m.ID
		if installedSet[m.ID] {
			label += " (installed)"
		}
		if m.Description != "" {
			label += " — " + m.Description
		}
		options = append(options, selectOption{Label: label, Value: m.ID})
	}

	ids, err := promptMultiSelect(fmt.Sprintf("Select %ss to install", tp), options)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		fmt.Println("Nothing selected.")
		return nil
	}

	// Carry the interactively-resolved values into p so installService reads
	// them uniformly with the flag path.
	p.mode, p.model, p.context = mode, model, projectContext
	svc, err := installService(ctx, tp, p)
	if err != nil {
		return err
	}
	out, err := svc.Install(ctx, command.InstallRequest{Target: target, IDs: ids, Scope: scope})
	printInstallOutcome(out, scope)
	return err
}

func printInstallOutcome(out command.InstallOutcome, scope catalog.Scope) {
	if len(out.Installed) > 0 {
		fmt.Printf("\nInstalled %d package(s) into %s scope:\n", len(out.Installed), scope)
		for _, id := range out.Installed {
			fmt.Printf("  ✓ %s\n", id)
		}
	}
	if len(out.NotFound) > 0 {
		fmt.Printf("\nNot found in catalog (skipped):\n")
		for _, id := range out.NotFound {
			fmt.Printf("  ✗ %s\n", id)
		}
	}
}

// typeLabel returns the plural display label for a catalog type.
func typeLabel(tp string) string {
	switch tp {
	case "skill":
		return "Skills"
	case "hook":
		return "Hooks"
	case "plugin":
		return "Plugins"
	default:
		return tp
	}
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
