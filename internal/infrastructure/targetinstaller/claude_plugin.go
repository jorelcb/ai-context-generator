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
)

// Compile-time guard: ClaudePluginInstaller satisfies catalog.TargetInstaller.
var _ catalog.TargetInstaller = (*ClaudePluginInstaller)(nil)

// ClaudePluginInstaller installs Claude plugins by delegating to Claude
// Code's own plugin CLI (the "B-CLI" mechanic, validated by the 2026-05-30
// spike — see ADR-0012 §3). It is registered alongside ClaudeInstaller; the
// installer registry routes claude-plugin packages here.
//
// Why delegate instead of writing files: a Claude plugin lives in a versioned
// cache (~/.claude/plugins/cache/...) that Claude Code manages. The native
// `claude plugin install` does the clone, namespacing, version pinning and
// state tracking; re-implementing that is brittle. Delegation is immediate,
// headless, and lets the agent own its evolving internals.
type ClaudePluginInstaller struct {
	runner CommandRunner
	// configDir is the agent config root (~/.claude) whose
	// plugins/installed_plugins.json InstalledList reads. Injectable for tests.
	configDir string
}

// NewClaudePluginInstaller builds an installer that shells out to the real
// `claude` binary and reads ~/.claude for installed state.
func NewClaudePluginInstaller() (*ClaudePluginInstaller, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("claude plugin installer: resolve home: %w", err)
	}
	return &ClaudePluginInstaller{runner: NewExecRunner(), configDir: filepath.Join(home, ".claude")}, nil
}

// NewClaudePluginInstallerWith injects a CommandRunner and config dir (tests).
func NewClaudePluginInstallerWith(runner CommandRunner, configDir string) *ClaudePluginInstaller {
	return &ClaudePluginInstaller{runner: runner, configDir: configDir}
}

// Handles reports that this installer covers Claude plugin packages.
func (c *ClaudePluginInstaller) Handles(t catalog.Target) bool {
	return t == catalog.TargetClaudePlugin
}

// Install registers the plugin's marketplace and installs the plugin via the
// native CLI. Idempotent: `marketplace add` and `install` are safe to re-run.
func (c *ClaudePluginInstaller) Install(ctx context.Context, m catalog.PackageManifest, _ catalog.PackageContent, scope catalog.Scope) error {
	if m.Target != catalog.TargetClaudePlugin {
		return fmt.Errorf("claude plugin installer: cannot install target %q", m.Target)
	}
	mkt := m.Metadata[catalog.MetaKeyMarketplace]
	if mkt == "" {
		return fmt.Errorf("claude plugin installer: manifest %q has no marketplace metadata", m.ID)
	}
	if _, err := c.runner.LookPath("claude"); err != nil {
		return fmt.Errorf("claude plugin installer: `claude` not found on PATH — install Claude Code, or add the marketplace and plugin manually (`claude plugin marketplace add %s` then `claude plugin install %s@%s`)", marketplaceAddArg(m.Source.URI), m.ID, mkt)
	}
	scopeFlag, err := pluginScopeFlag(scope)
	if err != nil {
		return err
	}

	// 1. Register the marketplace (idempotent). Force an https source so the
	//    clone does not depend on the user's SSH setup (the spike showed
	//    `marketplace add owner/repo` defaulting to SSH).
	if out, err := c.runner.Run(ctx, "claude", "plugin", "marketplace", "add", marketplaceAddArg(m.Source.URI)); err != nil {
		return fmt.Errorf("claude plugin installer: marketplace add: %w\n%s", err, out)
	}
	// 2. Install the plugin at the requested scope.
	if out, err := c.runner.Run(ctx, "claude", "plugin", "install", m.ID+"@"+mkt, "-s", scopeFlag); err != nil {
		return fmt.Errorf("claude plugin installer: install %s@%s: %w\n%s", m.ID, mkt, err, out)
	}
	return nil
}

// Uninstall removes the plugin via the native CLI. Idempotent.
func (c *ClaudePluginInstaller) Uninstall(ctx context.Context, m catalog.PackageManifest, _ catalog.Scope) error {
	if m.Target != catalog.TargetClaudePlugin {
		return fmt.Errorf("claude plugin installer: cannot uninstall target %q", m.Target)
	}
	mkt := m.Metadata[catalog.MetaKeyMarketplace]
	target := m.ID
	if mkt != "" {
		target = m.ID + "@" + mkt
	}
	if _, err := c.runner.LookPath("claude"); err != nil {
		return fmt.Errorf("claude plugin installer: `claude` not found on PATH (run `claude plugin uninstall %s`)", target)
	}
	if out, err := c.runner.Run(ctx, "claude", "plugin", "uninstall", target); err != nil {
		return fmt.Errorf("claude plugin installer: uninstall %s: %w\n%s", target, err, out)
	}
	return nil
}

// installedPluginsFile is the agent's plugin state file (schema v2).
type installedPluginsFile struct {
	Version int                          `json:"version"`
	Plugins map[string][]installedPlugin `json:"plugins"`
}

type installedPlugin struct {
	Scope   string `json:"scope"`
	Version string `json:"version"`
}

// InstalledList reads the agent's installed_plugins.json and returns the
// plugins present at the given scope. The on-disk state is the source of
// truth (the agent owns it); codify does not guess.
func (c *ClaudePluginInstaller) InstalledList(_ context.Context, scope catalog.Scope) ([]catalog.InstalledPackage, error) {
	path := filepath.Join(c.configDir, "plugins", "installed_plugins.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("claude plugin installer: read %s: %w", path, err)
	}
	var doc installedPluginsFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("claude plugin installer: parse %s: %w", path, err)
	}
	want, err := pluginScopeFlag(scope)
	if err != nil {
		return nil, err
	}

	var out []catalog.InstalledPackage
	for key, entries := range doc.Plugins {
		// key is "<id>@<marketplace>"; the catalog manifest ID is just <id>.
		id := key
		if at := strings.Index(key, "@"); at >= 0 {
			id = key[:at]
		}
		for _, e := range entries {
			if e.Scope != want {
				continue
			}
			out = append(out, catalog.InstalledPackage{
				ID:      id,
				Version: e.Version,
				Target:  catalog.TargetClaudePlugin,
				Scope:   scope,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// pluginScopeFlag maps a codify Scope to the agent CLI `-s` value. The agent
// also has a "local" scope; codify does not surface it.
func pluginScopeFlag(scope catalog.Scope) (string, error) {
	switch scope {
	case catalog.ScopeWorkstation:
		return "user", nil
	case catalog.ScopeProject:
		return "project", nil
	default:
		return "", fmt.Errorf("claude plugin installer: unsupported scope %q", scope)
	}
}

// marketplaceAddArg normalizes a marketplace ref into an https git source for
// `claude plugin marketplace add`. A bare `owner/repo` becomes
// `https://github.com/owner/repo` (avoids the SSH-clone default); a full URL
// or path is passed through.
func marketplaceAddArg(ref string) string {
	if ref == "" {
		return ref
	}
	if strings.Contains(ref, "://") || strings.HasPrefix(ref, "git@") || strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, ".") {
		return ref
	}
	// owner/repo shorthand.
	if strings.Count(ref, "/") == 1 {
		return "https://github.com/" + ref
	}
	return ref
}
