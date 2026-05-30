package packagesource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// Compile-time guard: LocalDirectorySource satisfies the frozen
// catalog.PackageSource contract.
var _ catalog.PackageSource = (*LocalDirectorySource)(nil)

// localManifestFile is the per-package manifest file name a LocalDirectorySource
// looks for in each immediate subdirectory of its root.
const localManifestFile = "codify-package.yaml"

// LocalDirectorySource is a catalog.PackageSource over a local directory of
// hand-authored or internally-distributed packages (ADR-0010 §8). It lets
// users drop personal/team skills + hooks into a directory (e.g.
// ~/.codify/sources/<name>/) and have them appear in the catalog alongside
// the built-in embedded packages.
//
// On-disk layout: each immediate subdirectory of the root is one package and
// must contain a `codify-package.yaml` manifest plus its content files:
//
//	<root>/
//	  my-skill/
//	    codify-package.yaml      # id, target: claude-skill, description, claude:{...}
//	    SKILL.md                 # the skill body (frontmatter included by the author)
//	  my-hook/
//	    codify-package.yaml      # id, target: claude-hook, claude:{hookEvents:[...]}
//	    hooks.json               # settings fragment (separated to SettingsFragment)
//	    guard.sh                 # script(s) copied verbatim
//
// v0 covers claude-skill and claude-hook (the targets the ClaudeInstaller
// handles). Other targets in a manifest are rejected so authors get a clear
// error rather than a silently-skipped package.
type LocalDirectorySource struct {
	root string
}

// NewLocalDirectorySource builds a source rooted at dir. The directory need
// not exist yet — List treats a missing root as "no packages" so callers can
// register the source unconditionally.
func NewLocalDirectorySource(root string) *LocalDirectorySource {
	return &LocalDirectorySource{root: root}
}

// Kind identifies this source for UI badges and diagnostics.
func (s *LocalDirectorySource) Kind() string { return "local-fs" }

// List parses every package manifest under the root and returns the decorated
// manifests. A missing root yields an empty list (not an error); a present but
// malformed manifest is an error (fail loud) so authoring mistakes surface.
func (s *LocalDirectorySource) List(ctx context.Context) ([]catalog.PackageManifest, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("local source: read root %s: %w", s.root, err)
	}

	var out []catalog.PackageManifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(s.root, e.Name(), localManifestFile)
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			if os.IsNotExist(err) {
				// A subdirectory without a manifest is not a package — skip it.
				continue
			}
			return nil, fmt.Errorf("local source: read %s: %w", manifestPath, err)
		}
		m, err := s.parseManifest(data, manifestPath)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Target != out[j].Target {
			return out[i].Target < out[j].Target
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// Fetch resolves a package's installable content from disk. The manifest's
// directory is derived from its ID (the subdir name).
func (s *LocalDirectorySource) Fetch(ctx context.Context, m catalog.PackageManifest) (catalog.PackageContent, error) {
	dir := filepath.Join(s.root, m.ID)
	switch m.Target {
	case catalog.TargetClaudeSkill:
		return s.fetchSkill(dir)
	case catalog.TargetClaudeHook:
		return s.fetchHook(dir)
	default:
		return catalog.PackageContent{}, fmt.Errorf("local source: target %q not supported (claude-skill + claude-hook only)", m.Target)
	}
}

// localManifest is the on-disk YAML shape, mapped to catalog.PackageManifest.
type localManifest struct {
	ID          string       `yaml:"id"`
	Target      string       `yaml:"target"`
	Version     string       `yaml:"version"`
	Description string       `yaml:"description"`
	Claude      *localClaude `yaml:"claude"`
}

type localClaude struct {
	Triggers      []string `yaml:"triggers"`
	AllowedTools  []string `yaml:"allowedTools"`
	UserInvocable bool     `yaml:"userInvocable"`
	HookEvents    []string `yaml:"hookEvents"`
}

// parseManifest decodes a manifest, validates it, and decorates it with the
// source's identity + checksum.
func (s *LocalDirectorySource) parseManifest(data []byte, path string) (catalog.PackageManifest, error) {
	var lm localManifest
	if err := yaml.Unmarshal(data, &lm); err != nil {
		return catalog.PackageManifest{}, fmt.Errorf("local source: parse %s: %w", path, err)
	}
	if lm.ID == "" {
		return catalog.PackageManifest{}, fmt.Errorf("local source: %s: missing required field 'id'", path)
	}
	target := catalog.Target(lm.Target)
	if target != catalog.TargetClaudeSkill && target != catalog.TargetClaudeHook {
		return catalog.PackageManifest{}, fmt.Errorf("local source: %s: unsupported target %q (use claude-skill or claude-hook)", path, lm.Target)
	}
	// The directory name must match the ID — Fetch derives the dir from ID.
	if dirName := filepath.Base(filepath.Dir(path)); dirName != lm.ID {
		return catalog.PackageManifest{}, fmt.Errorf("local source: %s: directory %q must match id %q", path, dirName, lm.ID)
	}

	version := lm.Version
	if version == "" {
		version = "local"
	}

	m := catalog.PackageManifest{
		ID:          lm.ID,
		Version:     version,
		Description: lm.Description,
		Target:      target,
		Source:      catalog.SourceRef{Kind: "local-fs", URI: s.root},
	}
	if lm.Claude != nil {
		m.Claude = &catalog.ClaudeMetadata{
			Triggers:      lm.Claude.Triggers,
			AllowedTools:  lm.Claude.AllowedTools,
			UserInvocable: lm.Claude.UserInvocable,
			HookEvents:    lm.Claude.HookEvents,
		}
	}
	m.SourceChecksum = computeManifestChecksum(m)
	return m, nil
}

func (s *LocalDirectorySource) fetchSkill(dir string) (catalog.PackageContent, error) {
	skillMd := filepath.Join(dir, "SKILL.md")
	body, err := os.ReadFile(skillMd)
	if err != nil {
		return catalog.PackageContent{}, fmt.Errorf("local source: read %s: %w", skillMd, err)
	}
	return catalog.PackageContent{
		Files: map[string][]byte{"SKILL.md": body},
	}, nil
}

func (s *LocalDirectorySource) fetchHook(dir string) (catalog.PackageContent, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return catalog.PackageContent{}, fmt.Errorf("local source: read hook dir %s: %w", dir, err)
	}
	out := catalog.PackageContent{Files: make(map[string][]byte)}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == localManifestFile {
			continue // the manifest is not part of the installable content
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return catalog.PackageContent{}, fmt.Errorf("local source: read hook file %s: %w", name, err)
		}
		// hooks.json / settings.json are settings fragments, not scripts.
		if name == "hooks.json" || name == "settings.json" {
			out.SettingsFragment = data
			continue
		}
		out.Files[name] = data
	}
	return out, nil
}
