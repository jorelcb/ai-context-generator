package commands

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	root "github.com/jorelcb/codify"
	"github.com/jorelcb/codify/internal/application/command"
	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/infrastructure/packagesource"
	"github.com/jorelcb/codify/internal/infrastructure/targetinstaller"
)

// catalogParams groups the flags of `codify catalog`.
type catalogParams struct {
	ecosystem string
	pkgType   string
	packages  string
	scope     string
	list      bool
}

// typeToTarget maps a user-facing catalog type to a Claude Target. MVP covers
// the two shipped types; plugins/slash-commands join as they materialize.
var typeToTarget = map[string]catalog.Target{
	"skill": catalog.TargetClaudeSkill,
	"hook":  catalog.TargetClaudeHook,
}

// NewCatalogCmd creates the `catalog` command — the unified surface for
// browsing and installing ecosystem packages (skills, hooks). See ADR-0010.
func NewCatalogCmd() *cobra.Command {
	var p catalogParams

	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Browse and install ecosystem packages (skills, hooks)",
		Long: `Browse and install packages from codify's catalog into your agent.

A package maps one-to-one to a native ecosystem primitive — a Claude skill or
a Claude hook today. The catalog reads from a package source (the built-in
embedded catalog for now) and installs through the per-ecosystem installer.

Modes:
  Interactive (no flags, TTY): pick ecosystem → type → packages → scope.
  List:        codify catalog --list [--type skill] [--scope project]
  Install:     codify catalog --type skill --package ddd-entity,hexagonal-port --scope project

Scopes:
  project      .claude/ in the current repository
  workstation  ~/.claude/ shared across all your projects (alias: global)

Note: this is the new home for skills/hooks selection (ADR-0010). The standalone
'codify skills' and 'codify hooks' commands remain for now and are folded in
incrementally.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			explicit := make(map[string]bool)
			cmd.Flags().Visit(func(f *pflag.Flag) { explicit[f.Name] = true })
			return runCatalog(p, explicit)
		},
	}

	cmd.Flags().StringVar(&p.ecosystem, "ecosystem", "claude", "Target ecosystem (MVP: claude)")
	cmd.Flags().StringVar(&p.pkgType, "type", "", "Package type: skill or hook")
	cmd.Flags().StringVar(&p.packages, "package", "", "Comma-separated package IDs to install")
	cmd.Flags().StringVar(&p.scope, "scope", "", "Install scope: project or workstation (alias: global)")
	cmd.Flags().BoolVar(&p.list, "list", false, "List available packages instead of installing")

	return cmd
}

func runCatalog(p catalogParams, explicit map[string]bool) error {
	if p.ecosystem != "" && p.ecosystem != "claude" {
		return fmt.Errorf("ecosystem %q not supported yet (MVP: claude)", p.ecosystem)
	}

	svc := newCatalogService()
	ctx := context.Background()

	if p.list {
		return catalogList(ctx, svc, p)
	}

	// Non-interactive install when the actionable flags are present.
	if explicit["type"] || explicit["package"] {
		return catalogInstallFromFlags(ctx, svc, p)
	}

	if !isInteractive() {
		return fmt.Errorf("codify catalog needs a TTY for interactive mode; pass --type and --package for non-interactive install, or --list to browse")
	}
	return catalogInteractive(ctx, svc)
}

// newCatalogService wires the embedded source + Claude installer registry.
func newCatalogService() *command.CatalogService {
	source := packagesource.NewEmbeddedSource(root.TemplatesFS, codifyVersion)
	registry := targetinstaller.NewRegistry(mustClaudeInstaller())
	return command.NewCatalogService(source, registry)
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
func catalogList(ctx context.Context, svc *command.CatalogService, p catalogParams) error {
	scope := catalog.ScopeProject
	if p.scope != "" {
		s, err := resolveScope(p.scope)
		if err != nil {
			return err
		}
		scope = s
	}

	types := []string{"skill", "hook"}
	if p.pkgType != "" {
		if _, ok := typeToTarget[p.pkgType]; !ok {
			return fmt.Errorf("invalid type %q (use skill or hook)", p.pkgType)
		}
		types = []string{p.pkgType}
	}

	fmt.Println()
	fmt.Printf("Catalog · ecosystem: claude · scope: %s\n", scope)
	fmt.Println(strings.Repeat("─", 48))

	for _, tp := range types {
		target := typeToTarget[tp]
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
	fmt.Println("Install with: codify catalog --type <skill|hook> --package <id,...> --scope <project|workstation>")
	return nil
}

func catalogInstallFromFlags(ctx context.Context, svc *command.CatalogService, p catalogParams) error {
	if p.pkgType == "" {
		return fmt.Errorf("--type is required for install (skill or hook)")
	}
	target, ok := typeToTarget[p.pkgType]
	if !ok {
		return fmt.Errorf("invalid type %q (use skill or hook)", p.pkgType)
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

	out, err := svc.Install(ctx, command.InstallRequest{Target: target, IDs: ids, Scope: scope})
	printInstallOutcome(out, scope)
	return err
}

func catalogInteractive(ctx context.Context, svc *command.CatalogService) error {
	// Step 1 — ecosystem. MVP ships Claude only; auto-select with a notice
	// rather than a single-option menu.
	fmt.Fprintln(os.Stderr, "→ Ecosystem: claude (the only one supported in this version)")

	// Step 2 — type tab.
	tp, err := promptSelect("Package type", []selectOption{
		{"Skills (prompt-based behaviors)", "skill"},
		{"Hooks (deterministic guardrails)", "hook"},
	}, "skill")
	if err != nil {
		return err
	}
	target := typeToTarget[tp]

	// Step 3 — scope (needed to mark installed state).
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

	// Step 4 — package multi-select, marking already-installed entries.
	available, err := svc.Available(ctx, target)
	if err != nil {
		return err
	}
	if len(available) == 0 {
		fmt.Printf("No %s packages available.\n", tp)
		return nil
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
