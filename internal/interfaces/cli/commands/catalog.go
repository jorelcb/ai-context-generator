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
	"github.com/jorelcb/codify/internal/infrastructure/desiredstate"
	"github.com/jorelcb/codify/internal/infrastructure/lockfile"
	"github.com/jorelcb/codify/internal/infrastructure/packagesource"
	"github.com/jorelcb/codify/internal/infrastructure/targetinstaller"
	tuicatalog "github.com/jorelcb/codify/internal/interfaces/cli/tui/catalog"
)

// catalogParams groups the flags of `codify catalog`.
type catalogParams struct {
	ecosystem   string
	pkgType     string
	packages    string
	scope       string
	marketplace string
	list        bool
	status      bool
	uninstall   bool
	sync        bool
	apply       bool
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
  Interactive (no flags, TTY): pick ecosystem → scope → rich selector (tabs + tree).
  List:        codify catalog --list [--type skill|hook|plugin] [--scope project]
  Status:      codify catalog --status [--scope project|workstation]
  Sync:        codify catalog --sync [--scope project|workstation]
  Apply:       codify catalog --apply [--scope project|workstation]
  Install:     codify catalog --type skill  --package ddd-entity,hexagonal-port --scope project
               codify catalog --type plugin --package gopls-lsp --scope workstation
  Uninstall:   codify catalog --uninstall --type skill --package ddd-entity --scope project

The interactive selector writes your picks to a committable desired-state file
(.codify/project.catalog.yml, ~/.codify/workstation.catalog.yml) — catalog-as-code.
Apply installs everything listed there, so a teammate reproduces your setup with
'codify catalog --apply' after checking out the repo.

Sync re-applies a scope's lockfile: it reinstalls every recorded package that
is missing on disk (onboarding a new machine/repo from a committed lockfile).
Plugins are skipped — their marketplace ref is not yet stored in the lockfile.

Status compares codify's lockfile (~/.codify/workstation.lock,
.codify/project.lock — what codify recorded installing) against what's
actually on disk, flagging packages that drifted (recorded but now missing).

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

	cmd.Flags().StringVar(&p.ecosystem, "ecosystem", "claude", "Target ecosystem: claude or antigravity")
	cmd.Flags().StringVar(&p.pkgType, "type", "", "Package type: skill, hook, or plugin")
	cmd.Flags().StringVar(&p.packages, "package", "", "Comma-separated package IDs to install")
	cmd.Flags().StringVar(&p.scope, "scope", "", "Install scope: project or workstation (alias: global)")
	cmd.Flags().StringVar(&p.marketplace, "marketplace", defaultMarketplace, "Plugin marketplace (owner/repo or marketplace.json URL); used with --type plugin")
	cmd.Flags().BoolVar(&p.list, "list", false, "List available packages instead of installing")
	cmd.Flags().BoolVar(&p.status, "status", false, "Show lockfile status: recorded installs vs live state (drift)")
	cmd.Flags().BoolVar(&p.uninstall, "uninstall", false, "Uninstall packages (with --type/--package/--scope) and drop them from the lockfile")
	cmd.Flags().BoolVar(&p.sync, "sync", false, "Re-apply the scope's lockfile: reinstall recorded packages missing on disk")
	cmd.Flags().BoolVar(&p.apply, "apply", false, "Install everything in the scope's desired-state file (.codify/*.catalog.yml)")

	return cmd
}

func runCatalog(p catalogParams, explicit map[string]bool) error {
	if p.ecosystem == "" {
		p.ecosystem = "claude"
	}
	if p.ecosystem != "claude" && p.ecosystem != "antigravity" {
		return fmt.Errorf("ecosystem %q not supported (claude or antigravity)", p.ecosystem)
	}

	ctx := context.Background()

	if p.status {
		return catalogStatus(ctx, p)
	}

	if p.uninstall {
		return catalogUninstall(ctx, p)
	}

	if p.sync {
		return catalogSync(ctx, p)
	}

	if p.apply {
		return catalogApply(ctx, p)
	}

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

// recordDesiredState merges the just-selected packages into the scope's
// desired-state file (.codify/*.catalog.yml) — the user-authored "catalog-as-
// code" intent (distinct from the lockfile's install record). Best-effort: a
// write failure warns but never fails the install that already succeeded.
func recordDesiredState(scope catalog.Scope, ecosystem string, byType map[string][]string, marketplace string) {
	path, err := desiredstate.PathForScope(scope)
	if err != nil {
		fmt.Fprintf(os.Stderr, "→ desired-state skipped: %v\n", err)
		return
	}
	ds, err := desiredstate.Load(path, scope)
	if err != nil {
		fmt.Fprintf(os.Stderr, "→ desired-state skipped: %v\n", err)
		return
	}
	var pkgs []desiredstate.Package
	for tp, ids := range byType {
		for _, id := range ids {
			pk := desiredstate.Package{Ecosystem: ecosystem, ID: id, Type: tp}
			if tp == "plugin" {
				pk.Source = marketplace
			}
			pkgs = append(pkgs, pk)
		}
	}
	if ds.Add(pkgs...) == 0 {
		return // nothing new to record
	}
	if err := desiredstate.Save(path, ds); err != nil {
		fmt.Fprintf(os.Stderr, "→ could not write desired-state: %v\n", err)
		return
	}
	fmt.Printf("→ desired state updated: %s\n", path)
}

// catalogApply installs everything listed in the scope's desired-state file —
// the reproduce-an-environment path: a teammate checks out the repo and runs
// `codify catalog --apply` to install exactly what the committed
// .codify/project.catalog.yml declares. Groups by (ecosystem, type) and routes
// each group through the normal install path; plugins use their recorded source.
func catalogApply(ctx context.Context, p catalogParams) error {
	scopeStr := p.scope
	if scopeStr == "" {
		scopeStr = "project"
	}
	scope, err := resolveScope(scopeStr)
	if err != nil {
		return err
	}
	path, err := desiredstate.PathForScope(scope)
	if err != nil {
		return err
	}
	ds, err := desiredstate.Load(path, scope)
	if err != nil {
		return err
	}
	if len(ds.Packages) == 0 {
		fmt.Printf("\nNo desired packages for %s scope (%s).\n", scope, path)
		return nil
	}

	fmt.Println()
	fmt.Printf("Applying desired state · scope: %s · %s\n", scope, path)
	fmt.Println(strings.Repeat("─", 48))

	// Group ids by ecosystem → type, remembering a plugin source ref if present.
	type group struct {
		ids    []string
		source string
	}
	groups := map[string]map[string]*group{}
	for _, pk := range ds.Packages {
		eco := pk.Ecosystem
		if eco == "" {
			eco = desiredstate.DefaultEcosystem
		}
		if groups[eco] == nil {
			groups[eco] = map[string]*group{}
		}
		g := groups[eco][pk.Type]
		if g == nil {
			g = &group{}
			groups[eco][pk.Type] = g
		}
		g.ids = append(g.ids, pk.ID)
		if pk.Source != "" {
			g.source = pk.Source
		}
	}

	ecos := make([]string, 0, len(groups))
	for eco := range groups {
		ecos = append(ecos, eco)
	}
	sort.Strings(ecos)

	restored, missing := 0, 0
	for _, eco := range ecos {
		for _, tp := range typesForEcosystem(eco) {
			g := groups[eco][tp]
			if g == nil || len(g.ids) == 0 {
				continue
			}
			target, terr := targetFor(eco, tp)
			if terr != nil {
				fmt.Fprintf(os.Stderr, "→ skipping %s/%s: %v\n", eco, tp, terr)
				continue
			}
			ap := p
			if tp == "plugin" && g.source != "" {
				ap.marketplace = g.source
			}
			svc, serr := installService(ctx, eco, tp, ap)
			if serr != nil {
				return serr
			}
			out, ierr := svc.Install(ctx, command.InstallRequest{Target: target, IDs: g.ids, Scope: scope})
			printInstallOutcome(out, scope)
			restored += len(out.Installed)
			missing += len(out.NotFound)
			if ierr != nil {
				return ierr
			}
		}
	}
	fmt.Printf("\nApply complete: %d installed, %d not found.\n", restored, missing)
	return nil
}

// catalogRegistry wires every installer the catalog can route to: Claude
// skills/hooks (filesystem), Claude plugins (native CLI), Antigravity skills
// (filesystem), and the Antigravity plugin stub (R-8 — honest blocked error
// until agy exposes marketplace registration). The registry routes by Target.
func catalogRegistry() *targetinstaller.Registry {
	return targetinstaller.NewRegistry(
		mustClaudeInstaller(),
		mustClaudePluginInstaller(),
		mustAntigravityInstaller(),
		targetinstaller.NewAntigravityPluginInstaller(),
	)
}

// targetFor resolves the Target for an ecosystem + user-facing type.
// Antigravity supports skills only for now (ADR-0012 §5: its plugin
// marketplace model is immature).
func targetFor(ecosystem, tp string) (catalog.Target, error) {
	switch ecosystem {
	case "antigravity":
		if tp == "skill" {
			return catalog.TargetAntigravitySkill, nil
		}
		return "", fmt.Errorf("antigravity supports type 'skill' only for now (got %q)", tp)
	default: // claude
		if t, ok := typeToTarget[tp]; ok {
			return t, nil
		}
		return "", fmt.Errorf("invalid type %q (use skill, hook, or plugin)", tp)
	}
}

// pluginService browses + installs from a Claude plugin marketplace. Plugins
// install by delegating to the agent's plugin CLI (ClaudePluginInstaller).
func pluginService(marketplaceRef string) *command.CatalogService {
	source := packagesource.NewPluginMarketplaceSource(marketplaceRef)
	return command.NewCatalogService(source, catalogRegistry()).WithRecorder(lockfile.NewRecorder())
}

// antigravityService browses + installs Antigravity skills (the built-in
// skills re-framed with Antigravity frontmatter, written to the agy skill
// dirs). Static only — no LLM personalization for Antigravity v0.
func antigravityService() *command.CatalogService {
	embedded := packagesource.NewEmbeddedSource(root.TemplatesFS, codifyVersion)
	source := packagesource.NewAntigravitySkillSource(embedded)
	return command.NewCatalogService(source, catalogRegistry()).WithRecorder(lockfile.NewRecorder())
}

// browseService returns the read service for an ecosystem + type. Antigravity
// reads its skill source; Claude plugins read the marketplace (network);
// Claude skills/hooks read the static embedded+local catalog (no API key).
func browseService(ecosystem, tp string, p catalogParams) *command.CatalogService {
	if ecosystem == "antigravity" {
		return antigravityService()
	}
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
	return command.NewCatalogService(source, catalogRegistry()).WithRecorder(lockfile.NewRecorder())
}

// statusService builds the service used by `--status`: it needs the full
// installer registry (to query live state for every recorded target) and a
// lockfile reader. The source is unused by Status, so the embedded source is
// passed as a harmless placeholder.
func statusService() *command.CatalogService {
	embedded := packagesource.NewEmbeddedSource(root.TemplatesFS, codifyVersion)
	return command.NewCatalogService(embedded, catalogRegistry()).WithReader(lockfile.NewRecorder())
}

// catalogStatus prints, per scope, how codify's recorded installs compare to
// what's actually on disk — the drift check the lockfile exists to power.
// Without --scope it reports both workstation and project lockfiles.
func catalogStatus(ctx context.Context, p catalogParams) error {
	var scopes []catalog.Scope
	if p.scope != "" {
		s, err := resolveScope(p.scope)
		if err != nil {
			return err
		}
		scopes = []catalog.Scope{s}
	} else {
		scopes = []catalog.Scope{catalog.ScopeWorkstation, catalog.ScopeProject}
	}

	svc := statusService()
	fmt.Println()
	fmt.Println("Lockfile status · codify's recorded installs vs live state")
	fmt.Println(strings.Repeat("─", 58))

	anyDrift := false
	for _, scope := range scopes {
		report, err := svc.Status(ctx, scope)
		if err != nil {
			return err
		}
		path, _ := lockfile.PathForScope(scope)
		fmt.Printf("\n%s  (%s)\n", scope, path)
		if len(report.Entries) == 0 {
			fmt.Println("  (no packages recorded)")
			continue
		}
		for _, e := range report.Entries {
			marker, note := "✓", ""
			if e.Status == command.DriftMissing {
				marker, note = "✗", "  ← missing (recorded but not on disk)"
				anyDrift = true
			}
			ver := e.Package.Version
			if ver == "" {
				ver = "-"
			}
			fmt.Printf("  %s %-26s [%s] %s%s\n", marker, e.Package.ID, e.Package.Target, ver, note)
		}
	}

	fmt.Println()
	if anyDrift {
		fmt.Println("Drift detected: ✗ packages are recorded in the lockfile but missing on disk.")
		fmt.Println("Reinstall with: codify catalog --type <skill|hook|plugin> --package <id,...> --scope <scope>")
	} else {
		fmt.Println("In sync: every recorded package is present on disk.")
	}
	return nil
}

// uninstallService builds the service used by `--uninstall`: the full
// installer registry (to route the removal to the right ecosystem) plus a
// lockfile forgetter (to drop the removed entries). The source is unused by
// Uninstall, so the embedded source is a harmless placeholder.
func uninstallService() *command.CatalogService {
	embedded := packagesource.NewEmbeddedSource(root.TemplatesFS, codifyVersion)
	return command.NewCatalogService(embedded, catalogRegistry()).WithForgetter(lockfile.NewRecorder())
}

// catalogUninstall removes the requested packages from disk and drops them
// from the lockfile. Removal is idempotent, so it also serves to prune a
// drifted lockfile entry whose files are already gone.
func catalogUninstall(ctx context.Context, p catalogParams) error {
	if p.pkgType == "" {
		return fmt.Errorf("--type is required for uninstall (skill, hook, or plugin)")
	}
	target, err := targetFor(p.ecosystem, p.pkgType)
	if err != nil {
		return err
	}
	ids := splitCSV(p.packages)
	if len(ids) == 0 {
		return fmt.Errorf("--package is required for uninstall (comma-separated IDs)")
	}
	scopeStr := p.scope
	if scopeStr == "" {
		scopeStr = "project"
	}
	scope, err := resolveScope(scopeStr)
	if err != nil {
		return err
	}

	// Plugins need the marketplace ref to address the uninstall; pass it along
	// when the type is plugin (the installer treats it as optional).
	var meta map[string]string
	if p.pkgType == "plugin" && p.marketplace != "" {
		meta = map[string]string{catalog.MetaKeyMarketplace: p.marketplace}
	}

	svc := uninstallService()
	out, err := svc.Uninstall(ctx, command.UninstallRequest{
		Target:   target,
		IDs:      ids,
		Scope:    scope,
		Metadata: meta,
	})
	if len(out.Removed) > 0 {
		fmt.Printf("\nUninstalled %d package(s) from %s scope:\n", len(out.Removed), scope)
		for _, id := range out.Removed {
			fmt.Printf("  ✓ %s\n", id)
		}
	}
	return err
}

// ecoTypeForTarget reverses targetFor: it maps an installed Target back to the
// (ecosystem, user-facing type) the install services are keyed on, so sync can
// rebuild the right source/installer for a recorded package.
func ecoTypeForTarget(t catalog.Target) (ecosystem, tp string, ok bool) {
	switch t {
	case catalog.TargetClaudeSkill:
		return "claude", "skill", true
	case catalog.TargetClaudeHook:
		return "claude", "hook", true
	case catalog.TargetClaudePlugin:
		return "claude", "plugin", true
	case catalog.TargetAntigravitySkill:
		return "antigravity", "skill", true
	default:
		return "", "", false
	}
}

// catalogSync re-applies a scope's lockfile: it reinstalls every recorded
// package that is currently missing on disk. This is the reproducibility path —
// onboard a new machine or repo from a committed lockfile, or repair drift that
// `--status` flagged. It reuses Status (to find what's missing) and Install
// (per target), so the restore goes through the same paths as a fresh install
// and re-records the lockfile idempotently.
//
// Plugins are reinstalled from their recorded marketplace ref (Source.URI),
// sub-grouped by ref; entries from older lockfiles without one are skipped.
func catalogSync(ctx context.Context, p catalogParams) error {
	scopeStr := p.scope
	if scopeStr == "" {
		scopeStr = "project"
	}
	scope, err := resolveScope(scopeStr)
	if err != nil {
		return err
	}

	report, err := statusService().Status(ctx, scope)
	if err != nil {
		return err
	}

	// Collect missing entries grouped by target (deterministic order). Keep the
	// full package so plugins can be sub-grouped by their recorded source ref.
	missingByTarget := map[catalog.Target][]catalog.InstalledPackage{}
	for _, e := range report.Entries {
		if e.Status == command.DriftMissing {
			missingByTarget[e.Package.Target] = append(missingByTarget[e.Package.Target], e.Package)
		}
	}

	fmt.Println()
	fmt.Printf("Syncing lockfile · scope: %s\n", scope)
	fmt.Println(strings.Repeat("─", 48))
	if len(missingByTarget) == 0 {
		fmt.Println("Already in sync — every recorded package is present on disk.")
		return nil
	}

	targets := make([]catalog.Target, 0, len(missingByTarget))
	for t := range missingByTarget {
		targets = append(targets, t)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i] < targets[j] })

	restored, skipped := 0, 0
	for _, target := range targets {
		pkgs := missingByTarget[target]
		sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].ID < pkgs[j].ID })

		ecosystem, tp, ok := ecoTypeForTarget(target)
		if !ok {
			for _, pk := range pkgs {
				fmt.Printf("  ⊘ %-26s [%s] skipped (unsupported target)\n", pk.ID, target)
			}
			skipped += len(pkgs)
			continue
		}

		// Plugins re-install from their recorded marketplace ref (Source.URI),
		// sub-grouped by ref. Entries without one (older lockfiles) can't be
		// rebuilt, so they're skipped.
		if tp == "plugin" {
			r, s, err := syncPlugins(ctx, p, target, pkgs, scope)
			if err != nil {
				return err
			}
			restored += r
			skipped += s
			continue
		}

		// Static restore path for skills/hooks (built-in/local catalog).
		svc, err := installService(ctx, ecosystem, tp, p)
		if err != nil {
			return err
		}
		ids := make([]string, len(pkgs))
		for i, pk := range pkgs {
			ids[i] = pk.ID
		}
		r, s, err := syncInstall(ctx, svc, target, ids, scope)
		if err != nil {
			return err
		}
		restored += r
		skipped += s
	}

	fmt.Println()
	fmt.Printf("Sync complete: %d restored, %d skipped.\n", restored, skipped)
	return nil
}

// syncPlugins reinstalls missing plugin packages, sub-grouped by their recorded
// marketplace ref so each group rebuilds the right marketplace source. Entries
// without a recorded ref are skipped.
func syncPlugins(ctx context.Context, p catalogParams, target catalog.Target, pkgs []catalog.InstalledPackage, scope catalog.Scope) (restored, skipped int, err error) {
	byRef := map[string][]string{}
	for _, pk := range pkgs {
		if pk.SourceURI == "" {
			fmt.Printf("  ⊘ %-26s [%s] skipped (no marketplace ref recorded)\n", pk.ID, target)
			skipped++
			continue
		}
		byRef[pk.SourceURI] = append(byRef[pk.SourceURI], pk.ID)
	}

	refs := make([]string, 0, len(byRef))
	for ref := range byRef {
		refs = append(refs, ref)
	}
	sort.Strings(refs)

	for _, ref := range refs {
		ids := byRef[ref]
		sort.Strings(ids)
		sp := p
		sp.marketplace = ref
		svc, serr := installService(ctx, "claude", "plugin", sp)
		if serr != nil {
			return restored, skipped, serr
		}
		r, s, ierr := syncInstall(ctx, svc, target, ids, scope)
		if ierr != nil {
			return restored + r, skipped + s, ierr
		}
		restored += r
		skipped += s
	}
	return restored, skipped, nil
}

// syncInstall runs one Install batch for sync and prints per-package results,
// returning (restored, skipped) counts.
func syncInstall(ctx context.Context, svc *command.CatalogService, target catalog.Target, ids []string, scope catalog.Scope) (restored, skipped int, err error) {
	out, ierr := svc.Install(ctx, command.InstallRequest{Target: target, IDs: ids, Scope: scope})
	if ierr != nil {
		printInstallOutcome(out, scope)
		return len(out.Installed), 0, ierr
	}
	for _, id := range out.Installed {
		fmt.Printf("  ✓ %-26s [%s] restored\n", id, target)
	}
	for _, id := range out.NotFound {
		fmt.Printf("  ⊘ %-26s [%s] skipped (no longer in catalog)\n", id, target)
	}
	return len(out.Installed), len(out.NotFound), nil
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

// mustAntigravityInstaller builds an AntigravityInstaller against the real
// home/cwd, falling back to relative roots on failure.
func mustAntigravityInstaller() *targetinstaller.AntigravityInstaller {
	inst, err := targetinstaller.NewAntigravityInstaller()
	if err != nil {
		return targetinstaller.NewAntigravityInstallerWithRoots(".", ".")
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

	// Default browse types per ecosystem. Antigravity is skills-only;
	// Claude defaults to skill+hook (plugins require explicit --type plugin
	// since listing them hits the marketplace network).
	types := []string{"skill", "hook"}
	if p.ecosystem == "antigravity" {
		types = []string{"skill"}
	}
	if p.pkgType != "" {
		if _, err := targetFor(p.ecosystem, p.pkgType); err != nil {
			return err
		}
		types = []string{p.pkgType}
	}

	fmt.Println()
	fmt.Printf("Catalog · ecosystem: %s · scope: %s\n", p.ecosystem, scope)
	fmt.Println(strings.Repeat("─", 48))

	for _, tp := range types {
		target, err := targetFor(p.ecosystem, tp)
		if err != nil {
			return err
		}
		svc := browseService(p.ecosystem, tp, p)
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
// Plugins use the marketplace source (+ plugin installer); everything else the
// static embedded+local catalog. Skills are static-only since v4.0.0 — the
// LLM personalization mode was dropped (Track 2 D4: it added no value over
// well-authored static skills).
func installService(_ context.Context, ecosystem, tp string, p catalogParams) (*command.CatalogService, error) {
	if ecosystem == "antigravity" {
		return antigravityService(), nil // skills only
	}
	if tp == "plugin" {
		return pluginService(p.marketplace), nil
	}
	return embeddedService(), nil
}

func catalogInstallFromFlags(ctx context.Context, p catalogParams) error {
	if p.pkgType == "" {
		return fmt.Errorf("--type is required for install (skill, hook, or plugin)")
	}
	target, err := targetFor(p.ecosystem, p.pkgType)
	if err != nil {
		return err
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

	svc, err := installService(ctx, p.ecosystem, p.pkgType, p)
	if err != nil {
		return err
	}
	out, err := svc.Install(ctx, command.InstallRequest{Target: target, IDs: ids, Scope: scope})
	printInstallOutcome(out, scope)
	return err
}

// typesForEcosystem returns the package types shown as tabs for an ecosystem.
// Antigravity is skills-only for now (ADR-0012 §5).
func typesForEcosystem(ecosystem string) []string {
	if ecosystem == "antigravity" {
		return []string{"skill"}
	}
	return []string{"skill", "hook", "plugin"}
}

// catalogInteractive runs the rich TUI selector (ADR-0013): pick ecosystem +
// scope, then a tabs+tree+checkbox selector that accumulates a cross-tab
// selection, then install grouped per type. Replaces the v3.0.0 sequential
// huh wizard. Only reached with a TTY (runCatalog guards non-TTY → flags).
func catalogInteractive(ctx context.Context, p catalogParams) error {
	// Pre-step 1 — ecosystem (ADR-0010 Decision 3: ecosystem first).
	ecosystem, err := promptSelect("Target ecosystem", []selectOption{
		{"Claude Code", "claude"},
		{"Antigravity CLI", "antigravity"},
	}, "claude")
	if err != nil {
		return err
	}
	p.ecosystem = ecosystem

	// Pre-step 2 — scope (needed to mark installed state and to install).
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

	return runCatalogSelector(ctx, p, ecosystem, scope, scopeStr)
}

// runCatalogSelector builds the tabs from live catalog data, runs the rich
// selector, and installs the accumulated cross-tab selection grouped per type.
// It is the single selector surface shared by `catalog` (standalone), `config`
// (workstation) and `init` (project) — ADR-0010 Decision 6. Callers resolve
// ecosystem + scope first.
func runCatalogSelector(ctx context.Context, p catalogParams, ecosystem string, scope catalog.Scope, scopeStr string) error {
	// Build one tab per package type, best-effort: a type that fails to list
	// (e.g. plugins offline) is skipped with a warning rather than aborting.
	specs := buildTabSpecs(ctx, p, ecosystem, scope)
	if len(specs) == 0 {
		return fmt.Errorf("no packages available for ecosystem %q", ecosystem)
	}

	// Rich selector — tabs + tree + accumulated cross-tab selection.
	result, err := tuicatalog.Run(tuicatalog.Build(scopeStr, ecosystem, specs))
	if err != nil {
		return err
	}
	if result.Cancelled || len(result.Selected) == 0 {
		fmt.Println("Nothing selected.")
		return nil
	}

	// Group the accumulated selection by type for per-target install.
	byType := make(map[string][]string)
	for _, s := range result.Selected {
		byType[s.TabType] = append(byType[s.TabType], s.ID)
	}

	// Install per type, deterministic order.
	for _, tp := range typesForEcosystem(ecosystem) {
		ids := byType[tp]
		if len(ids) == 0 {
			continue
		}
		target, _ := targetFor(ecosystem, tp)
		svc, serr := installService(ctx, ecosystem, tp, p)
		if serr != nil {
			return serr
		}
		out, ierr := svc.Install(ctx, command.InstallRequest{Target: target, IDs: ids, Scope: scope})
		printInstallOutcome(out, scope)
		if ierr != nil {
			return ierr
		}
	}

	// Record the selection as desired state (catalog-as-code backbone, ADR-0013):
	// the committable file a teammate can `codify catalog --apply` to reproduce.
	recordDesiredState(scope, ecosystem, byType, p.marketplace)
	return nil
}

// buildTabSpecs reads each package type for the ecosystem and maps the manifests
// to the TUI's domain-agnostic Pkg specs, marking already-installed packages and
// carrying the source-declared category for tree grouping. Per-type failures are
// skipped with a warning (offline plugins shouldn't kill the whole selector).
func buildTabSpecs(ctx context.Context, p catalogParams, ecosystem string, scope catalog.Scope) []tuicatalog.TabSpec {
	var specs []tuicatalog.TabSpec
	for _, tp := range typesForEcosystem(ecosystem) {
		target, terr := targetFor(ecosystem, tp)
		if terr != nil {
			continue
		}
		if tp == "plugin" {
			fmt.Fprintf(os.Stderr, "→ Fetching plugins from %s …\n", p.marketplace)
		}
		svc := browseService(ecosystem, tp, p)
		available, aerr := svc.Available(ctx, target)
		if aerr != nil {
			fmt.Fprintf(os.Stderr, "→ skipping %s tab: %v\n", tp, aerr)
			continue
		}
		installed, _ := svc.Installed(ctx, target, scope)
		instSet := make(map[string]bool, len(installed))
		for _, ip := range installed {
			instSet[ip.ID] = true
		}
		pkgs := make([]tuicatalog.Pkg, 0, len(available))
		for _, m := range available {
			pkgs = append(pkgs, tuicatalog.Pkg{
				ID:        m.ID,
				Label:     m.Label,
				Desc:      m.Description,
				Version:   m.Version,
				Source:    m.Source.Kind,
				Tags:      m.Metadata[catalog.MetaKeyTags],
				Category:  m.Metadata[catalog.MetaKeyCategory],
				Installed: instSet[m.ID],
			})
		}
		specs = append(specs, tuicatalog.TabSpec{Name: typeLabel(tp), Type: tp, Pkgs: pkgs})
	}
	return specs
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
