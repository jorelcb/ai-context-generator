package commands

import (
	"context"
	"fmt"

	"github.com/jorelcb/codify-og/internal/application/dto"
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
