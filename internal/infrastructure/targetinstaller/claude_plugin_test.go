package targetinstaller

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// fakeRunner records calls and returns canned availability/output/error.
type fakeRunner struct {
	available bool
	runErr    error
	calls     [][]string
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	if f.available {
		return "/usr/bin/" + name, nil
	}
	return "", errors.New("not found")
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	return "ok", f.runErr
}

func pluginManifest() catalog.PackageManifest {
	return catalog.PackageManifest{
		ID:       "code-formatter",
		Target:   catalog.TargetClaudePlugin,
		Source:   catalog.SourceRef{Kind: "claude-marketplace", URI: "acme/plugins"},
		Metadata: map[string]string{catalog.MetaKeyMarketplace: "acme-tools"},
	}
}

func TestClaudePluginInstaller_Handles(t *testing.T) {
	c := NewClaudePluginInstallerWith(&fakeRunner{}, "")
	if !c.Handles(catalog.TargetClaudePlugin) {
		t.Error("should handle claude-plugin")
	}
	if c.Handles(catalog.TargetClaudeSkill) {
		t.Error("should not handle claude-skill")
	}
}

func TestClaudePluginInstaller_Install_DelegatesToCLI(t *testing.T) {
	fr := &fakeRunner{available: true}
	c := NewClaudePluginInstallerWith(fr, t.TempDir())

	if err := c.Install(context.Background(), pluginManifest(), catalog.PackageContent{}, catalog.ScopeWorkstation); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(fr.calls) != 2 {
		t.Fatalf("expected 2 CLI calls, got %d: %v", len(fr.calls), fr.calls)
	}
	// 1) marketplace add with an https source (not owner/repo, not SSH).
	add := strings.Join(fr.calls[0], " ")
	if !strings.Contains(add, "plugin marketplace add https://github.com/acme/plugins") {
		t.Errorf("marketplace add call wrong: %q", add)
	}
	// 2) install <id>@<mkt> -s user (workstation → user).
	inst := strings.Join(fr.calls[1], " ")
	if !strings.Contains(inst, "plugin install code-formatter@acme-tools -s user") {
		t.Errorf("install call wrong: %q", inst)
	}
}

func TestClaudePluginInstaller_Install_ProjectScope(t *testing.T) {
	fr := &fakeRunner{available: true}
	c := NewClaudePluginInstallerWith(fr, t.TempDir())
	if err := c.Install(context.Background(), pluginManifest(), catalog.PackageContent{}, catalog.ScopeProject); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !strings.Contains(strings.Join(fr.calls[1], " "), "-s project") {
		t.Errorf("expected -s project, got %v", fr.calls[1])
	}
}

// TestClaudePluginInstaller_Install_ClaudeMissing_B2Fallback verifies that with
// `claude` absent, Install does NOT error but falls back to B2: it declares the
// marketplace + plugin in the scope's settings.json (no CLI calls), so Claude
// Code materializes it later (ADR-0012 §3).
func TestClaudePluginInstaller_Install_ClaudeMissing_B2Fallback(t *testing.T) {
	dir := t.TempDir()
	fr := &fakeRunner{available: false}
	c := NewClaudePluginInstallerWith(fr, dir)

	if err := c.Install(context.Background(), pluginManifest(), catalog.PackageContent{}, catalog.ScopeWorkstation); err != nil {
		t.Fatalf("B2 fallback should not error: %v", err)
	}
	if len(fr.calls) != 0 {
		t.Errorf("no CLI calls should happen when claude is missing, got %v", fr.calls)
	}

	// settings.json now declares the marketplace + enabled plugin.
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("settings.json not written: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("settings.json not valid JSON: %v", err)
	}
	mkts, _ := raw["extraKnownMarketplaces"].(map[string]any)
	if _, ok := mkts["acme-tools"]; !ok {
		t.Errorf("extraKnownMarketplaces missing acme-tools: %v", raw["extraKnownMarketplaces"])
	}
	plugins, _ := raw["enabledPlugins"].(map[string]any)
	if enabled, _ := plugins["code-formatter@acme-tools"].(bool); !enabled {
		t.Errorf("enabledPlugins missing code-formatter@acme-tools=true: %v", raw["enabledPlugins"])
	}
}

// TestClaudePluginInstaller_B2Fallback_PreservesAndIsIdempotent checks the
// fallback preserves unrelated settings keys and adds nothing on a second pass.
func TestClaudePluginInstaller_B2Fallback_PreservesAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	// Pre-existing settings with an unrelated key.
	seed := `{"model":"opus","extraKnownMarketplaces":{"other":{"source":"x/y"}}}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewClaudePluginInstallerWith(&fakeRunner{available: false}, dir)

	for i := 0; i < 2; i++ {
		if err := c.Install(context.Background(), pluginManifest(), catalog.PackageContent{}, catalog.ScopeWorkstation); err != nil {
			t.Fatalf("pass %d: %v", i, err)
		}
	}
	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("settings.json not valid JSON: %v", err)
	}
	if raw["model"] != "opus" {
		t.Errorf("unrelated key 'model' not preserved: %v", raw["model"])
	}
	mkts, _ := raw["extraKnownMarketplaces"].(map[string]any)
	if _, ok := mkts["other"]; !ok {
		t.Errorf("pre-existing marketplace 'other' clobbered: %v", mkts)
	}
	if _, ok := mkts["acme-tools"]; !ok {
		t.Errorf("declared marketplace 'acme-tools' missing: %v", mkts)
	}
}

func TestClaudePluginInstaller_Install_MissingMarketplace(t *testing.T) {
	fr := &fakeRunner{available: true}
	c := NewClaudePluginInstallerWith(fr, t.TempDir())
	m := pluginManifest()
	m.Metadata = nil
	if err := c.Install(context.Background(), m, catalog.PackageContent{}, catalog.ScopeWorkstation); err == nil {
		t.Fatal("expected error when manifest has no marketplace metadata")
	}
}

func TestClaudePluginInstaller_Uninstall(t *testing.T) {
	fr := &fakeRunner{available: true}
	c := NewClaudePluginInstallerWith(fr, t.TempDir())
	if err := c.Uninstall(context.Background(), pluginManifest(), catalog.ScopeWorkstation); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !strings.Contains(strings.Join(fr.calls[0], " "), "plugin uninstall code-formatter@acme-tools") {
		t.Errorf("uninstall call wrong: %v", fr.calls[0])
	}
}

func TestClaudePluginInstaller_Install_PropagatesCLIError(t *testing.T) {
	fr := &fakeRunner{available: true, runErr: errors.New("exit 1")}
	c := NewClaudePluginInstallerWith(fr, t.TempDir())
	if err := c.Install(context.Background(), pluginManifest(), catalog.PackageContent{}, catalog.ScopeWorkstation); err == nil {
		t.Fatal("expected install to propagate the CLI error")
	}
}

func TestClaudePluginInstaller_InstalledList(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	state := `{
	  "version": 2,
	  "plugins": {
	    "gopls-lsp@claude-plugins-official": [{"scope":"user","version":"1.0.0"}],
	    "deployer@acme-tools": [{"scope":"project","version":"2.0.0"}]
	  }
	}`
	if err := os.WriteFile(filepath.Join(dir, "plugins", "installed_plugins.json"), []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewClaudePluginInstallerWith(&fakeRunner{}, dir)

	user, err := c.InstalledList(context.Background(), catalog.ScopeWorkstation)
	if err != nil {
		t.Fatalf("InstalledList(user): %v", err)
	}
	if len(user) != 1 || user[0].ID != "gopls-lsp" || user[0].Version != "1.0.0" || user[0].Target != catalog.TargetClaudePlugin {
		t.Errorf("user-scope list wrong: %+v", user)
	}

	proj, err := c.InstalledList(context.Background(), catalog.ScopeProject)
	if err != nil {
		t.Fatalf("InstalledList(project): %v", err)
	}
	if len(proj) != 1 || proj[0].ID != "deployer" {
		t.Errorf("project-scope list wrong: %+v", proj)
	}
}

func TestClaudePluginInstaller_InstalledList_MissingFile(t *testing.T) {
	c := NewClaudePluginInstallerWith(&fakeRunner{}, t.TempDir())
	got, err := c.InstalledList(context.Background(), catalog.ScopeWorkstation)
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty list, got %v", got)
	}
}

func TestMarketplaceAddArg(t *testing.T) {
	cases := map[string]string{
		"acme/plugins":                "https://github.com/acme/plugins",
		"https://example.com/m.json":  "https://example.com/m.json",
		"git@github.com:acme/plugins": "git@github.com:acme/plugins",
		"/local/path":                 "/local/path",
		"./rel":                       "./rel",
	}
	for in, want := range cases {
		if got := marketplaceAddArg(in); got != want {
			t.Errorf("marketplaceAddArg(%q) = %q, want %q", in, got, want)
		}
	}
}
