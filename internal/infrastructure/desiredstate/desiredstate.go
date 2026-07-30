// Package desiredstate persists the user's DESIRED catalog packages as a
// human-readable, committable file — the "catalog-as-code" backbone of ADR-0013.
//
// It is distinct from the lockfile (internal/infrastructure/lockfile):
//   - the lockfile is codify's RECORD of what it installed (auto, JSON, for drift
//     detection and reproduction);
//   - this file is the user's INTENT (hand-editable, YAML, committed to the repo)
//     — the source of truth a teammate runs `catalog --apply` against to get the
//     same setup.
//
// Format/location decisions (divergence from ADR-0013's tentative
// `codify.catalog.toml` mock, for consistency + dependency hygiene):
//   - YAML, not TOML: the project already depends on gopkg.in/yaml.v3 (config)
//     and adds no TOML library — the same dep-hygiene reasoning ADR-0013 used to
//     pick bubbletea over a net-new tcell subtree.
//   - Per-scope files mirroring the lockfile, not one file with sections:
//     workstation -> ~/.codify/workstation.catalog.yml
//     project     -> <cwd>/.codify/project.catalog.yml
package desiredstate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/jorelcb/codify-og/internal/domain/catalog"
)

// SchemaVersion is the on-disk format version.
const SchemaVersion = 1

// DesiredState is the parsed desired-catalog document for one scope.
type DesiredState struct {
	Version  int       `yaml:"version"`
	Scope    string    `yaml:"scope"`
	Packages []Package `yaml:"packages"`
}

// DefaultEcosystem is assumed when a package omits one (the dominant case).
const DefaultEcosystem = "claude"

// Package is one desired install. Ecosystem is "claude" | "antigravity"
// (defaulted to claude when empty). Type is the user-facing package type
// ("skill" | "hook" | "plugin"). Source is the locator needed to rebuild a
// package's source on apply — the marketplace ref for plugins; empty for
// embedded skills/hooks.
type Package struct {
	Ecosystem string `yaml:"ecosystem,omitempty"`
	ID        string `yaml:"id"`
	Type      string `yaml:"type"`
	Source    string `yaml:"source,omitempty"`
}

// eco returns the package's ecosystem, defaulting to claude.
func (p Package) eco() string {
	if p.Ecosystem == "" {
		return DefaultEcosystem
	}
	return p.Ecosystem
}

// key uniquely identifies a desired package within a scope.
func (p Package) key() string { return p.eco() + "\x00" + p.Type + "\x00" + p.ID }

// PathForScope returns the desired-state file path for a scope, mirroring the
// lockfile layout: ~/.codify/workstation.catalog.yml (workstation) and
// <cwd>/.codify/project.catalog.yml (project).
func PathForScope(scope catalog.Scope) (string, error) {
	switch scope {
	case catalog.ScopeWorkstation:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("desiredstate: resolve home: %w", err)
		}
		return filepath.Join(home, ".codify", "workstation.catalog.yml"), nil
	case catalog.ScopeProject:
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("desiredstate: resolve cwd: %w", err)
		}
		return filepath.Join(cwd, ".codify", "project.catalog.yml"), nil
	default:
		return "", fmt.Errorf("desiredstate: unsupported scope %q", scope)
	}
}

// Load reads the desired-state file at path. A missing file is not an error —
// it returns an empty, well-formed state (the common first-run case).
func Load(path string, scope catalog.Scope) (*DesiredState, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &DesiredState{Version: SchemaVersion, Scope: string(scope)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("desiredstate: read %s: %w", path, err)
	}
	var ds DesiredState
	if err := yaml.Unmarshal(data, &ds); err != nil {
		return nil, fmt.Errorf("desiredstate: parse %s: %w", path, err)
	}
	if ds.Version == 0 {
		ds.Version = SchemaVersion
	}
	if ds.Scope == "" {
		ds.Scope = string(scope)
	}
	return &ds, nil
}

// Save writes the desired-state file atomically (temp + rename), creating the
// .codify directory if needed. Packages are sorted for a stable, diff-friendly
// file.
func Save(path string, ds *DesiredState) error {
	ds.sort()
	if ds.Version == 0 {
		ds.Version = SchemaVersion
	}
	out, err := yaml.Marshal(ds)
	if err != nil {
		return fmt.Errorf("desiredstate: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("desiredstate: mkdir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("desiredstate: write temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("desiredstate: rename: %w", err)
	}
	return nil
}

// Add merges packages into the state, de-duplicating by (type, id). A later
// entry with the same key updates the Source. Returns the number of newly-added
// packages (existing keys don't count).
func (ds *DesiredState) Add(pkgs ...Package) int {
	index := make(map[string]int, len(ds.Packages))
	for i, p := range ds.Packages {
		index[p.key()] = i
	}
	added := 0
	for _, p := range pkgs {
		if i, ok := index[p.key()]; ok {
			if p.Source != "" {
				ds.Packages[i].Source = p.Source
			}
			continue
		}
		index[p.key()] = len(ds.Packages)
		ds.Packages = append(ds.Packages, p)
		added++
	}
	return added
}

// Remove drops a package by (type, id). Returns true if it was present.
func (ds *DesiredState) Remove(typ, id string) bool {
	want := Package{Type: typ, ID: id}.key()
	for i, p := range ds.Packages {
		if p.key() == want {
			ds.Packages = append(ds.Packages[:i], ds.Packages[i+1:]...)
			return true
		}
	}
	return false
}

// ByType returns the desired package IDs for a given type, sorted.
func (ds *DesiredState) ByType(typ string) []string {
	var ids []string
	for _, p := range ds.Packages {
		if p.Type == typ {
			ids = append(ids, p.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func (ds *DesiredState) sort() {
	sort.Slice(ds.Packages, func(i, j int) bool {
		a, b := ds.Packages[i], ds.Packages[j]
		if a.eco() != b.eco() {
			return a.eco() < b.eco()
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.ID < b.ID
	})
}
