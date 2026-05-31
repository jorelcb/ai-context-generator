package targetinstaller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// Compile-time guard: AntigravityInstaller satisfies catalog.TargetInstaller.
var _ catalog.TargetInstaller = (*AntigravityInstaller)(nil)

// AntigravityInstaller installs Antigravity CLI (`agy`) packages. v0 covers
// skills only (ADR-0012 §5): an Agent-Skills markdown file written flat into
// Antigravity's skill directories — a filesystem write, no CLI delegation
// (unlike Claude plugins). Antigravity plugins are deferred until `agy`'s
// marketplace model matures.
//
// Skill paths (from the official agy docs):
//   - workstation (global):  ~/.gemini/antigravity-cli/skills/<id>.md
//   - project (workspace):   <project>/.agents/skills/<id>.md
type AntigravityInstaller struct {
	homeRoot    string // base for ScopeWorkstation; typically $HOME
	projectRoot string // base for ScopeProject; typically cwd
}

// NewAntigravityInstaller builds an installer against the real home + cwd.
func NewAntigravityInstaller() (*AntigravityInstaller, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("antigravity installer: resolve home: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("antigravity installer: resolve cwd: %w", err)
	}
	return &AntigravityInstaller{homeRoot: home, projectRoot: cwd}, nil
}

// NewAntigravityInstallerWithRoots injects roots (tests).
func NewAntigravityInstallerWithRoots(homeRoot, projectRoot string) *AntigravityInstaller {
	return &AntigravityInstaller{homeRoot: homeRoot, projectRoot: projectRoot}
}

// Handles reports that this installer covers Antigravity skills.
func (a *AntigravityInstaller) Handles(t catalog.Target) bool {
	return t == catalog.TargetAntigravitySkill
}

// skillsDir resolves the Antigravity skills directory for a scope.
func (a *AntigravityInstaller) skillsDir(scope catalog.Scope) (string, error) {
	switch scope {
	case catalog.ScopeWorkstation:
		return filepath.Join(a.homeRoot, ".gemini", "antigravity-cli", "skills"), nil
	case catalog.ScopeProject:
		return filepath.Join(a.projectRoot, ".agents", "skills"), nil
	default:
		return "", fmt.Errorf("antigravity installer: unsupported scope %q", scope)
	}
}

// Install writes the skill markdown flat as <id>.md.
func (a *AntigravityInstaller) Install(_ context.Context, m catalog.PackageManifest, content catalog.PackageContent, scope catalog.Scope) error {
	if m.Target != catalog.TargetAntigravitySkill {
		return fmt.Errorf("antigravity installer: cannot install target %q", m.Target)
	}
	body, ok := content.Files["SKILL.md"]
	if !ok {
		return fmt.Errorf("antigravity installer: skill %q content has no SKILL.md", m.ID)
	}
	dir, err := a.skillsDir(scope)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("antigravity installer: create skills dir %s: %w", dir, err)
	}
	dst := filepath.Join(dir, m.ID+".md")
	if err := os.WriteFile(dst, body, 0o644); err != nil {
		return fmt.Errorf("antigravity installer: write %s: %w", dst, err)
	}
	return nil
}

// Uninstall removes the skill file. Idempotent.
func (a *AntigravityInstaller) Uninstall(_ context.Context, m catalog.PackageManifest, scope catalog.Scope) error {
	if m.Target != catalog.TargetAntigravitySkill {
		return fmt.Errorf("antigravity installer: cannot uninstall target %q", m.Target)
	}
	dir, err := a.skillsDir(scope)
	if err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(dir, m.ID+".md")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("antigravity installer: remove %s.md: %w", m.ID, err)
	}
	return nil
}

// InstalledList enumerates the `.md` skills present in the scope's skills dir.
func (a *AntigravityInstaller) InstalledList(_ context.Context, scope catalog.Scope) ([]catalog.InstalledPackage, error) {
	dir, err := a.skillsDir(scope)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("antigravity installer: read skills dir %s: %w", dir, err)
	}
	var out []catalog.InstalledPackage
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		out = append(out, catalog.InstalledPackage{
			ID:     strings.TrimSuffix(e.Name(), ".md"),
			Target: catalog.TargetAntigravitySkill,
			Scope:  scope,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
