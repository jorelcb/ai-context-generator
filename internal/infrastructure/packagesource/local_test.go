package packagesource

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// writePackage creates <root>/<id>/ with a manifest + the given content files.
func writePackage(t *testing.T, root, id, manifest string, files map[string]string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(dir, localManifestFile), []byte(manifest), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func TestLocalSource_MissingRoot_EmptyList(t *testing.T) {
	src := NewLocalDirectorySource(filepath.Join(t.TempDir(), "does-not-exist"))
	got, err := src.List(context.Background())
	if err != nil {
		t.Fatalf("List on missing root should not error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty list, got %d", len(got))
	}
}

func TestLocalSource_List_ParsesSkillAndHook(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "my-skill", `
id: my-skill
target: claude-skill
version: "1.2.0"
description: A custom skill
claude:
  userInvocable: true
  triggers: [foo, bar]
`, map[string]string{"SKILL.md": "---\nname: my-skill\n---\nbody"})
	writePackage(t, root, "my-hook", `
id: my-hook
target: claude-hook
description: A custom hook
claude:
  hookEvents: [PostToolUse]
`, map[string]string{
		"hooks.json": `{"hooks":{"PostToolUse":[]}}`,
		"guard.sh":   "#!/bin/bash",
	})
	// A non-package directory (no manifest) must be ignored.
	if err := os.MkdirAll(filepath.Join(root, "not-a-package"), 0o755); err != nil {
		t.Fatal(err)
	}

	src := NewLocalDirectorySource(root)
	got, err := src.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 packages, got %d (%+v)", len(got), got)
	}
	// Sorted by (Target, ID): claude-hook before claude-skill.
	if got[0].ID != "my-hook" || got[0].Target != catalog.TargetClaudeHook {
		t.Errorf("expected my-hook first, got %+v", got[0])
	}
	if got[1].ID != "my-skill" || got[1].Version != "1.2.0" {
		t.Errorf("expected my-skill v1.2.0, got %+v", got[1])
	}
	if got[1].Source.Kind != "local-fs" || got[1].Source.URI != root {
		t.Errorf("skill source not decorated: %+v", got[1].Source)
	}
	if got[1].Claude == nil || !got[1].Claude.UserInvocable || len(got[1].Claude.Triggers) != 2 {
		t.Errorf("skill claude metadata not mapped: %+v", got[1].Claude)
	}
	if got[1].SourceChecksum == "" {
		t.Error("checksum not computed")
	}
}

func TestLocalSource_Version_DefaultsToLocal(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "s", "id: s\ntarget: claude-skill\n", map[string]string{"SKILL.md": "x"})
	got, _ := NewLocalDirectorySource(root).List(context.Background())
	if len(got) != 1 || got[0].Version != "local" {
		t.Errorf("expected version 'local', got %+v", got)
	}
}

func TestLocalSource_MalformedManifest_Errors(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "bad", "id: [unclosed", map[string]string{})
	if _, err := NewLocalDirectorySource(root).List(context.Background()); err == nil {
		t.Fatal("expected error on malformed manifest")
	}
}

func TestLocalSource_UnsupportedTarget_Errors(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", "id: p\ntarget: gemini-extension\n", map[string]string{})
	if _, err := NewLocalDirectorySource(root).List(context.Background()); err == nil {
		t.Fatal("expected error on unsupported target")
	}
}

func TestLocalSource_DirMismatch_Errors(t *testing.T) {
	root := t.TempDir()
	// manifest id "other" lives in directory "p" — must error.
	writePackage(t, root, "p", "id: other\ntarget: claude-skill\n", map[string]string{"SKILL.md": "x"})
	if _, err := NewLocalDirectorySource(root).List(context.Background()); err == nil {
		t.Fatal("expected error when dir name != manifest id")
	}
}

func TestLocalSource_Fetch_Skill(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "my-skill", "id: my-skill\ntarget: claude-skill\n",
		map[string]string{"SKILL.md": "---\nname: my-skill\n---\nthe body"})

	src := NewLocalDirectorySource(root)
	m := catalog.PackageManifest{ID: "my-skill", Target: catalog.TargetClaudeSkill}
	content, err := src.Fetch(context.Background(), m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !strings.Contains(string(content.Files["SKILL.md"]), "the body") {
		t.Errorf("SKILL.md content wrong: %q", content.Files["SKILL.md"])
	}
	if len(content.SettingsFragment) != 0 {
		t.Error("skill should have no settings fragment")
	}
}

func TestLocalSource_Fetch_Hook_SeparatesScriptsAndSettings(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "my-hook", "id: my-hook\ntarget: claude-hook\n", map[string]string{
		"hooks.json": `{"hooks":{"PostToolUse":[{"matcher":"Edit"}]}}`,
		"guard.sh":   "#!/bin/bash\necho guard",
	})

	src := NewLocalDirectorySource(root)
	m := catalog.PackageManifest{ID: "my-hook", Target: catalog.TargetClaudeHook}
	content, err := src.Fetch(context.Background(), m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if _, ok := content.Files["guard.sh"]; !ok {
		t.Errorf("guard.sh missing from Files: %v", keysOf(content.Files))
	}
	if _, ok := content.Files["hooks.json"]; ok {
		t.Error("hooks.json should be in SettingsFragment, not Files")
	}
	if _, ok := content.Files[localManifestFile]; ok {
		t.Error("manifest file must be excluded from content")
	}
	if !strings.Contains(string(content.SettingsFragment), "PostToolUse") {
		t.Errorf("settings fragment wrong: %s", content.SettingsFragment)
	}
}

// End-to-end: the local source composes with the same ClaudeInstaller used by
// the embedded source — proving a personal package installs identically.
func TestLocalSource_Kind(t *testing.T) {
	if NewLocalDirectorySource("/x").Kind() != "local-fs" {
		t.Error("Kind() should be local-fs")
	}
}
