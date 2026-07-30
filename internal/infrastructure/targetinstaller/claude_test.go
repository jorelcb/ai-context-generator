package targetinstaller

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	root "github.com/jorelcb/codify-og"
	"github.com/jorelcb/codify-og/internal/domain/catalog"
	"github.com/jorelcb/codify-og/internal/infrastructure/packagesource"
)

// newTestInstaller points both scope roots at one temp dir so tests never
// touch the real cwd or $HOME.
func newTestInstaller(t *testing.T) (*ClaudeInstaller, string) {
	t.Helper()
	dir := t.TempDir()
	return NewClaudeInstallerWithRoots(dir, dir), dir
}

func TestClaudeInstaller_Handles(t *testing.T) {
	c := NewClaudeInstallerWithRoots("", "")
	cases := []struct {
		target catalog.Target
		want   bool
	}{
		{catalog.TargetClaudeSkill, true},
		{catalog.TargetClaudeHook, true},
		{catalog.TargetCodifySDDStandard, false},
		{catalog.TargetAntigravityWF, false},
	}
	for _, tc := range cases {
		if got := c.Handles(tc.target); got != tc.want {
			t.Errorf("Handles(%q) = %v, want %v", tc.target, got, tc.want)
		}
	}
}

func TestClaudeInstaller_InstallSkill_WritesSkillMd(t *testing.T) {
	c, dir := newTestInstaller(t)
	m := catalog.PackageManifest{ID: "ddd-entity", Target: catalog.TargetClaudeSkill}
	content := catalog.PackageContent{Files: map[string][]byte{"SKILL.md": []byte("---\nname: ddd-entity\n---\nbody")}}

	if err := c.Install(context.Background(), m, content, catalog.ScopeProject); err != nil {
		t.Fatalf("Install: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, ".claude", "skills", "ddd-entity", "SKILL.md"))
	if err != nil {
		t.Fatalf("read installed SKILL.md: %v", err)
	}
	if !strings.Contains(string(got), "name: ddd-entity") {
		t.Errorf("installed SKILL.md missing frontmatter, got: %q", string(got))
	}
}

func TestClaudeInstaller_InstallSkill_MissingSkillMd(t *testing.T) {
	c, _ := newTestInstaller(t)
	m := catalog.PackageManifest{ID: "x", Target: catalog.TargetClaudeSkill}
	err := c.Install(context.Background(), m, catalog.PackageContent{Files: map[string][]byte{}}, catalog.ScopeProject)
	if err == nil {
		t.Fatal("expected error when content has no SKILL.md")
	}
}

func TestClaudeInstaller_UninstallSkill_RemovesDir(t *testing.T) {
	c, dir := newTestInstaller(t)
	m := catalog.PackageManifest{ID: "ddd-entity", Target: catalog.TargetClaudeSkill}
	content := catalog.PackageContent{Files: map[string][]byte{"SKILL.md": []byte("x")}}
	if err := c.Install(context.Background(), m, content, catalog.ScopeProject); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if err := c.Uninstall(context.Background(), m, catalog.ScopeProject); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "ddd-entity")); !os.IsNotExist(err) {
		t.Errorf("skill dir should be gone, stat err = %v", err)
	}

	// Idempotent: uninstalling again is a no-op.
	if err := c.Uninstall(context.Background(), m, catalog.ScopeProject); err != nil {
		t.Errorf("second Uninstall should be a no-op, got: %v", err)
	}
}

func TestClaudeInstaller_InstallHook_CopiesScriptsAndMergesSettings(t *testing.T) {
	c, dir := newTestInstaller(t)
	m := catalog.PackageManifest{ID: "linting", Target: catalog.TargetClaudeHook}
	content := catalog.PackageContent{
		Files:            map[string][]byte{"lint.sh": []byte("#!/bin/bash\necho lint")},
		SettingsFragment: []byte(`{"hooks":{"PostToolUse":[{"matcher":"Edit|Write","hooks":[{"type":"command","command":"\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/lint.sh"}]}]}}`),
	}

	if err := c.Install(context.Background(), m, content, catalog.ScopeProject); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Script copied.
	if _, err := os.Stat(filepath.Join(dir, ".claude", "hooks", "lint.sh")); err != nil {
		t.Errorf("lint.sh not copied: %v", err)
	}
	// settings.json contains the merged hook.
	s, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}
	if !strings.Contains(string(s), "PostToolUse") || !strings.Contains(string(s), "lint.sh") {
		t.Errorf("settings.json missing merged hook, got: %s", string(s))
	}
	// Project scope keeps $CLAUDE_PROJECT_DIR (no rewrite).
	if !strings.Contains(string(s), "CLAUDE_PROJECT_DIR") {
		t.Errorf("project-scope command should keep $CLAUDE_PROJECT_DIR, got: %s", string(s))
	}
}

func TestClaudeInstaller_InstallHook_Idempotent(t *testing.T) {
	c, dir := newTestInstaller(t)
	m := catalog.PackageManifest{ID: "linting", Target: catalog.TargetClaudeHook}
	content := catalog.PackageContent{
		Files:            map[string][]byte{"lint.sh": []byte("x")},
		SettingsFragment: []byte(`{"hooks":{"PostToolUse":[{"matcher":"Edit|Write","hooks":[{"type":"command","command":"\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/lint.sh"}]}]}}`),
	}
	for i := range 2 {
		if err := c.Install(context.Background(), m, content, catalog.ScopeProject); err != nil {
			t.Fatalf("Install run %d: %v", i, err)
		}
	}
	s, _ := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if n := strings.Count(string(s), "lint.sh"); n != 1 {
		t.Errorf("expected exactly one lint.sh handler after two installs, got %d", n)
	}
}

func TestClaudeInstaller_InstallHook_WorkstationRewritesToHome(t *testing.T) {
	c, dir := newTestInstaller(t)
	m := catalog.PackageManifest{ID: "linting", Target: catalog.TargetClaudeHook}
	content := catalog.PackageContent{
		Files:            map[string][]byte{"lint.sh": []byte("x")},
		SettingsFragment: []byte(`{"hooks":{"PostToolUse":[{"matcher":"Edit|Write","hooks":[{"type":"command","command":"\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/lint.sh"}]}]}}`),
	}
	if err := c.Install(context.Background(), m, content, catalog.ScopeWorkstation); err != nil {
		t.Fatalf("Install: %v", err)
	}
	s, _ := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if strings.Contains(string(s), "CLAUDE_PROJECT_DIR") {
		t.Errorf("workstation command should be rewritten to $HOME, got: %s", string(s))
	}
	if !strings.Contains(string(s), "$HOME") || !strings.Contains(string(s), "/.claude/hooks/lint.sh") {
		t.Errorf("workstation command should reference $HOME and the hooks path, got: %s", string(s))
	}
}

func TestClaudeInstaller_UninstallHook_RemovesScriptsAndHandlers(t *testing.T) {
	c, dir := newTestInstaller(t)
	m := catalog.PackageManifest{ID: "linting", Target: catalog.TargetClaudeHook}
	content := catalog.PackageContent{
		Files:            map[string][]byte{"lint.sh": []byte("x")},
		SettingsFragment: []byte(`{"hooks":{"PostToolUse":[{"matcher":"Edit|Write","hooks":[{"type":"command","command":"\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/lint.sh"}]}]}}`),
	}
	if err := c.Install(context.Background(), m, content, catalog.ScopeProject); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if err := c.Uninstall(context.Background(), m, catalog.ScopeProject); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "hooks", "lint.sh")); !os.IsNotExist(err) {
		t.Errorf("lint.sh should be removed, stat err = %v", err)
	}
	s, _ := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if strings.Contains(string(s), "lint.sh") {
		t.Errorf("settings.json should no longer reference lint.sh, got: %s", string(s))
	}

	// Idempotent.
	if err := c.Uninstall(context.Background(), m, catalog.ScopeProject); err != nil {
		t.Errorf("second Uninstall should be a no-op, got: %v", err)
	}
}

func TestClaudeInstaller_InstalledList(t *testing.T) {
	c, _ := newTestInstaller(t)
	ctx := context.Background()

	// Empty scope: nothing installed.
	got, err := c.InstalledList(ctx, catalog.ScopeProject)
	if err != nil {
		t.Fatalf("InstalledList (empty): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty list, got %d", len(got))
	}

	// Install one skill + one hook.
	_ = c.Install(ctx, catalog.PackageManifest{ID: "ddd-entity", Target: catalog.TargetClaudeSkill},
		catalog.PackageContent{Files: map[string][]byte{"SKILL.md": []byte("x")}}, catalog.ScopeProject)
	_ = c.Install(ctx, catalog.PackageManifest{ID: "linting", Target: catalog.TargetClaudeHook},
		catalog.PackageContent{Files: map[string][]byte{"lint.sh": []byte("x")}}, catalog.ScopeProject)

	got, err = c.InstalledList(ctx, catalog.ScopeProject)
	if err != nil {
		t.Fatalf("InstalledList: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 installed packages, got %d (%+v)", len(got), got)
	}
	// Sorted by (Target, ID): claude-hook < claude-skill.
	if got[0].ID != "linting" || got[0].Target != catalog.TargetClaudeHook {
		t.Errorf("expected linting hook first, got %+v", got[0])
	}
	if got[1].ID != "ddd-entity" || got[1].Target != catalog.TargetClaudeSkill {
		t.Errorf("expected ddd-entity skill second, got %+v", got[1])
	}
}

// TestClaudeInstaller_EndToEnd_WithEmbeddedSource proves the read/write pair:
// fetch real content from EmbeddedSource, then install it via ClaudeInstaller.
func TestClaudeInstaller_EndToEnd_WithEmbeddedSource(t *testing.T) {
	ctx := context.Background()
	src := packagesource.NewEmbeddedSource(root.TemplatesFS, "test")
	manifests, err := src.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	var skill, hook *catalog.PackageManifest
	for i := range manifests {
		m := &manifests[i]
		if skill == nil && m.Target == catalog.TargetClaudeSkill {
			skill = m
		}
		if m.ID == "linting" && m.Target == catalog.TargetClaudeHook {
			hook = m
		}
	}
	if skill == nil || hook == nil {
		t.Fatalf("missing fixtures: skill=%v hook=%v", skill, hook)
	}

	c, dir := newTestInstaller(t)

	skillContent, err := src.Fetch(ctx, *skill)
	if err != nil {
		t.Fatalf("Fetch skill: %v", err)
	}
	if err := c.Install(ctx, *skill, skillContent, catalog.ScopeProject); err != nil {
		t.Fatalf("Install skill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", skill.ID, "SKILL.md")); err != nil {
		t.Errorf("skill SKILL.md not installed: %v", err)
	}

	hookContent, err := src.Fetch(ctx, *hook)
	if err != nil {
		t.Fatalf("Fetch hook: %v", err)
	}
	if err := c.Install(ctx, *hook, hookContent, catalog.ScopeProject); err != nil {
		t.Fatalf("Install hook: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "hooks", "lint.sh")); err != nil {
		t.Errorf("hook lint.sh not installed: %v", err)
	}

	// InstalledList should now report both.
	installed, err := c.InstalledList(ctx, catalog.ScopeProject)
	if err != nil {
		t.Fatalf("InstalledList: %v", err)
	}
	if len(installed) != 2 {
		t.Errorf("expected 2 installed, got %d (%+v)", len(installed), installed)
	}
}
