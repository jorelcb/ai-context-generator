// Package targetinstaller agrupa los adapters concretos del port
// catalog.TargetInstaller (ver ADR-0010 §9). Cada adapter conoce las rutas,
// formatos y convenciones de instalación de UN ecosystem y sabe instalar,
// desinstalar y enumerar los packages que le corresponden.
//
// El primer adapter es ClaudeInstaller (D.3). Gemini y Antigravity llegan en
// sub-hitos posteriores.
package targetinstaller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jorelcb/codify/internal/domain/catalog"
	"github.com/jorelcb/codify/internal/infrastructure/settings"
)

// Compile-time guard: ClaudeInstaller debe satisfacer el contrato
// catalog.TargetInstaller. Si la interfaz o este adapter divergen, el build
// falla acá. Todo installer futuro (Gemini, Antigravity) agrega la misma
// aserción.
var _ catalog.TargetInstaller = (*ClaudeInstaller)(nil)

// ClaudeInstaller instala packages del ecosystem Claude Code. Es
// per-ecosystem (no per-target): cubre TargetClaudeSkill y TargetClaudeHook,
// que comparten las raíces ~/.claude/ (workstation) y ./.claude/ (project) y
// la mecánica de settings.json. Ver ADR-0010 §9.
//
// Recetas por target:
//   - claude-skill: escribe un único SKILL.md bajo <root>/.claude/skills/<id>/.
//   - claude-hook: copia los scripts del bundle a <root>/.claude/hooks/ y
//     mergea el fragmento hooks.json en <root>/.claude/settings.json
//     (idempotente, con backup). En scope workstation reescribe las rutas de
//     los comandos a "$HOME" porque los scripts no viven en cada proyecto.
type ClaudeInstaller struct {
	// projectRoot es la base para ScopeProject (las rutas .claude/ cuelgan de
	// acá). Típicamente el cwd.
	projectRoot string
	// homeRoot es la base para ScopeWorkstation. Típicamente $HOME.
	homeRoot string
}

// NewClaudeInstaller construye un ClaudeInstaller con las raíces reales:
// projectRoot = cwd, homeRoot = home del usuario.
func NewClaudeInstaller() (*ClaudeInstaller, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("claude installer: resolve cwd: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("claude installer: resolve home: %w", err)
	}
	return &ClaudeInstaller{projectRoot: cwd, homeRoot: home}, nil
}

// NewClaudeInstallerWithRoots construye un ClaudeInstaller con raíces
// explícitas. Para tests (apuntar ambas a un temp dir) y para callers que ya
// resolvieron las bases.
func NewClaudeInstallerWithRoots(projectRoot, homeRoot string) *ClaudeInstaller {
	return &ClaudeInstaller{projectRoot: projectRoot, homeRoot: homeRoot}
}

// Handles reporta si este installer cubre el target dado.
func (c *ClaudeInstaller) Handles(t catalog.Target) bool {
	return t == catalog.TargetClaudeSkill || t == catalog.TargetClaudeHook
}

// Install escribe el package en disco según su Target y Scope.
func (c *ClaudeInstaller) Install(_ context.Context, m catalog.PackageManifest, content catalog.PackageContent, scope catalog.Scope) error {
	switch m.Target {
	case catalog.TargetClaudeSkill:
		return c.installSkill(m, content, scope)
	case catalog.TargetClaudeHook:
		return c.installHook(m, content, scope)
	default:
		return fmt.Errorf("claude installer: cannot install target %q", m.Target)
	}
}

// Uninstall remueve el package del scope dado. Idempotente.
func (c *ClaudeInstaller) Uninstall(_ context.Context, m catalog.PackageManifest, scope catalog.Scope) error {
	switch m.Target {
	case catalog.TargetClaudeSkill:
		return c.uninstallSkill(m, scope)
	case catalog.TargetClaudeHook:
		return c.uninstallHook(m, scope)
	default:
		return fmt.Errorf("claude installer: cannot uninstall target %q", m.Target)
	}
}

// InstalledList enumera los packages Claude presentes en el scope. Las skills
// se detectan escaneando el directorio (fiable); los hooks por presencia de
// sus scripts característicos (heurística, hasta que el lockfile de D.9
// registre el estado instalado de forma genérica).
func (c *ClaudeInstaller) InstalledList(_ context.Context, scope catalog.Scope) ([]catalog.InstalledPackage, error) {
	var out []catalog.InstalledPackage

	skills, err := c.installedSkills(scope)
	if err != nil {
		return nil, err
	}
	out = append(out, skills...)

	hooks, err := c.installedHooks(scope)
	if err != nil {
		return nil, err
	}
	out = append(out, hooks...)

	sort.Slice(out, func(i, j int) bool {
		if out[i].Target != out[j].Target {
			return out[i].Target < out[j].Target
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// --- path resolution ---

func (c *ClaudeInstaller) claudeRoot(scope catalog.Scope) (string, error) {
	switch scope {
	case catalog.ScopeProject:
		return c.projectRoot, nil
	case catalog.ScopeWorkstation:
		return c.homeRoot, nil
	default:
		return "", fmt.Errorf("claude installer: unsupported scope %q", scope)
	}
}

func (c *ClaudeInstaller) skillsDir(scope catalog.Scope) (string, error) {
	root, err := c.claudeRoot(scope)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".claude", "skills"), nil
}

func (c *ClaudeInstaller) hooksDir(scope catalog.Scope) (string, error) {
	root, err := c.claudeRoot(scope)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".claude", "hooks"), nil
}

func (c *ClaudeInstaller) settingsPath(scope catalog.Scope) (string, error) {
	root, err := c.claudeRoot(scope)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".claude", "settings.json"), nil
}

// --- skills ---

func (c *ClaudeInstaller) installSkill(m catalog.PackageManifest, content catalog.PackageContent, scope catalog.Scope) error {
	if _, ok := content.Files["SKILL.md"]; !ok {
		return fmt.Errorf("claude installer: skill %q content has no SKILL.md", m.ID)
	}
	base, err := c.skillsDir(scope)
	if err != nil {
		return err
	}
	dir := filepath.Join(base, m.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("claude installer: create skill dir %s: %w", dir, err)
	}
	// Multi-file skills (v4.0.0): SKILL.md + progressive-disclosure sidecars
	// (reference.md, examples.md) — every fetched file lands in the skill
	// dir, mirroring how the hooks path installs full bundles. Dropping
	// extra files silently was the SK-4 audit finding.
	for name, data := range content.Files {
		dst := filepath.Join(dir, name)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("claude installer: write %s: %w", dst, err)
		}
	}
	return nil
}

func (c *ClaudeInstaller) uninstallSkill(m catalog.PackageManifest, scope catalog.Scope) error {
	base, err := c.skillsDir(scope)
	if err != nil {
		return err
	}
	dir := filepath.Join(base, m.ID)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("claude installer: remove skill dir %s: %w", dir, err)
	}
	return nil
}

func (c *ClaudeInstaller) installedSkills(scope catalog.Scope) ([]catalog.InstalledPackage, error) {
	base, err := c.skillsDir(scope)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("claude installer: read skills dir %s: %w", base, err)
	}
	var out []catalog.InstalledPackage
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillMd := filepath.Join(base, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillMd); err != nil {
			continue // a directory without SKILL.md is not one of ours
		}
		out = append(out, catalog.InstalledPackage{
			ID:     e.Name(),
			Target: catalog.TargetClaudeSkill,
			Scope:  scope,
		})
	}
	return out, nil
}

// --- hooks ---

// claudeHookScripts mapea cada hook package conocido a sus scripts. Hasta que
// el lockfile (D.9) registre los archivos instalados de forma genérica, el
// installer carga este conocimiento del catálogo para soportar Uninstall e
// InstalledList de hooks (que reciben solo el manifest, sin el contenido).
var claudeHookScripts = map[string][]string{
	"linting":                {"lint.sh"},
	"security-guardrails":    {"block-dangerous-commands.sh", "protect-sensitive-files.sh"},
	"convention-enforcement": {"validate-commit-message.sh", "check-protected-branches.sh"},
}

func (c *ClaudeInstaller) installHook(m catalog.PackageManifest, content catalog.PackageContent, scope catalog.Scope) error {
	// 1. Copiar scripts del bundle.
	dir, err := c.hooksDir(scope)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("claude installer: create hooks dir %s: %w", dir, err)
	}
	names := make([]string, 0, len(content.Files))
	for name := range content.Files {
		names = append(names, name)
	}
	sort.Strings(names) // determinismo
	for _, name := range names {
		dst := filepath.Join(dir, name)
		if err := os.WriteFile(dst, content.Files[name], 0o755); err != nil {
			return fmt.Errorf("claude installer: write hook script %s: %w", dst, err)
		}
	}

	// 2. Mergear el fragmento de settings.json (si lo hay).
	if len(content.SettingsFragment) == 0 {
		return nil
	}
	var block map[string]any
	if err := json.Unmarshal(content.SettingsFragment, &block); err != nil {
		return fmt.Errorf("claude installer: parse hook settings fragment for %q: %w", m.ID, err)
	}
	if scope == catalog.ScopeWorkstation {
		// Los scripts viven en ~/.claude/hooks/, no en cada proyecto: reescribir
		// las rutas de los comandos a "$HOME" para que el handler funcione desde
		// cualquier proyecto activo.
		rewriteHookCommandsToHome(block)
	}

	sp, err := c.settingsPath(scope)
	if err != nil {
		return err
	}
	s, err := settings.Load(sp)
	if err != nil {
		return fmt.Errorf("claude installer: load settings %s: %w", sp, err)
	}
	if _, _, err := s.MergeHooks(block); err != nil {
		return fmt.Errorf("claude installer: merge hooks into %s: %w", sp, err)
	}
	if _, err := s.Save(""); err != nil {
		return fmt.Errorf("claude installer: save settings %s: %w", sp, err)
	}
	return nil
}

func (c *ClaudeInstaller) uninstallHook(m catalog.PackageManifest, scope catalog.Scope) error {
	scripts, known := claudeHookScripts[m.ID]
	if !known {
		return fmt.Errorf("claude installer: unknown hook package %q (no script signature)", m.ID)
	}

	// 1. Remover los handlers de settings.json cuyo comando referencia alguno
	//    de los scripts del package.
	sp, err := c.settingsPath(scope)
	if err != nil {
		return err
	}
	s, err := settings.Load(sp)
	if err != nil {
		return fmt.Errorf("claude installer: load settings %s: %w", sp, err)
	}
	removed := s.RemoveHooksMatching(func(cmd string) bool {
		for _, script := range scripts {
			if strings.Contains(cmd, "/.claude/hooks/"+script) {
				return true
			}
		}
		return false
	})
	if len(removed) > 0 {
		if _, err := s.Save(""); err != nil {
			return fmt.Errorf("claude installer: save settings %s: %w", sp, err)
		}
	}

	// 2. Remover los scripts del disco (idempotente).
	dir, err := c.hooksDir(scope)
	if err != nil {
		return err
	}
	for _, script := range scripts {
		if err := os.Remove(filepath.Join(dir, script)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("claude installer: remove hook script %s: %w", script, err)
		}
	}
	return nil
}

func (c *ClaudeInstaller) installedHooks(scope catalog.Scope) ([]catalog.InstalledPackage, error) {
	dir, err := c.hooksDir(scope)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(claudeHookScripts))
	for id := range claudeHookScripts {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var out []catalog.InstalledPackage
	for _, id := range ids {
		scripts := claudeHookScripts[id]
		// Un hook package se considera instalado si TODOS sus scripts están
		// presentes en el hooks dir.
		allPresent := true
		for _, script := range scripts {
			if _, err := os.Stat(filepath.Join(dir, script)); err != nil {
				allPresent = false
				break
			}
		}
		if allPresent {
			out = append(out, catalog.InstalledPackage{
				ID:     id,
				Target: catalog.TargetClaudeHook,
				Scope:  scope,
			})
		}
	}
	return out, nil
}

// rewriteHookCommandsToHome muta in-place el block reemplazando
// `"$CLAUDE_PROJECT_DIR"/.claude/hooks/` por `"$HOME"/.claude/hooks/` en cada
// handler. Necesario para scope workstation. Espejo del helper homónimo en
// internal/application/command/install_hooks.go — se consolidan cuando el
// comando `hooks` se reescriba sobre este installer (rewire posterior).
func rewriteHookCommandsToHome(doc map[string]any) {
	hooks, ok := doc["hooks"].(map[string]any)
	if !ok {
		return
	}
	for _, eventVal := range hooks {
		matchers, ok := eventVal.([]any)
		if !ok {
			continue
		}
		for _, m := range matchers {
			matcher, ok := m.(map[string]any)
			if !ok {
				continue
			}
			handlers, ok := matcher["hooks"].([]any)
			if !ok {
				continue
			}
			for _, h := range handlers {
				handler, ok := h.(map[string]any)
				if !ok {
					continue
				}
				cmd, ok := handler["command"].(string)
				if !ok {
					continue
				}
				handler["command"] = strings.ReplaceAll(
					cmd,
					`"$CLAUDE_PROJECT_DIR"/.claude/hooks/`,
					`"$HOME"/.claude/hooks/`,
				)
			}
		}
	}
}
