package targetinstaller

import (
	"context"
	"fmt"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// Compile-time guard: AntigravityPluginInstaller satisfies catalog.TargetInstaller.
var _ catalog.TargetInstaller = (*AntigravityPluginInstaller)(nil)

// AntigravityPluginInstaller es el stub R-8 para plugins Antigravity. La
// instalación funcional está BLOQUEADA: el spike HOME-sandboxed (2026-05-31,
// agy v1.0.0) probó que `agy` NO expone `marketplace add` — los marketplaces
// conocidos son backend-served, no registrables por CLI para un `owner/repo`
// arbitrario (ADR-0012 §5). El flujo de codify ("agregar un marketplace.json
// arbitrario → instalar de él") no es reproducible hoy en Antigravity.
//
// Este installer existe para que la abstracción quede completa (el registry
// rutea antigravity-plugin acá) y el error sea explícito y accionable, en vez
// de un "no installer registered" genérico. Cuando `agy` exponga registro de
// marketplaces, Install pasa a delegar al CLI con el mismo patrón que
// ClaudePluginInstaller (B-CLI).
type AntigravityPluginInstaller struct{}

// NewAntigravityPluginInstaller builds the stub installer.
func NewAntigravityPluginInstaller() *AntigravityPluginInstaller {
	return &AntigravityPluginInstaller{}
}

// Handles reports that this installer covers Antigravity plugin packages.
func (a *AntigravityPluginInstaller) Handles(t catalog.Target) bool {
	return t == catalog.TargetAntigravityPlugin
}

// errAgyBlocked is the single, honest blocked message (ADR-0012 §5).
const errAgyBlocked = "antigravity plugin installer: BLOCKED — agy v1.0.0 has no `marketplace add` (known marketplaces are backend-served), so codify cannot register an arbitrary marketplace.json and install from it (ADR-0012 §5). Re-check when agy exposes marketplace registration"

// Install is blocked until agy supports arbitrary-marketplace registration.
func (a *AntigravityPluginInstaller) Install(_ context.Context, m catalog.PackageManifest, _ catalog.PackageContent, _ catalog.Scope) error {
	if m.Target != catalog.TargetAntigravityPlugin {
		return fmt.Errorf("antigravity plugin installer: cannot install target %q", m.Target)
	}
	return fmt.Errorf("%s (package %q)", errAgyBlocked, m.ID)
}

// Uninstall is blocked for the same reason as Install.
func (a *AntigravityPluginInstaller) Uninstall(_ context.Context, m catalog.PackageManifest, _ catalog.Scope) error {
	return fmt.Errorf("%s (package %q)", errAgyBlocked, m.ID)
}

// InstalledList reports no installed plugins: codify has never been able to
// install one, and agy's installed-state file location/schema is unverified
// (D.2.b). Returning empty (not an error) keeps --status/--list usable.
func (a *AntigravityPluginInstaller) InstalledList(_ context.Context, _ catalog.Scope) ([]catalog.InstalledPackage, error) {
	return nil, nil
}
