// Package lockfile records the packages codify installed, per scope, so the
// installed set is reproducible and drift-checkable (ADR-0010 §6, D.9).
//
// Two lockfiles exist:
//   - workstation: ~/.codify/workstation.lock
//   - project:     <cwd>/.codify/project.lock
//
// The lockfile is codify's own record of what it installed — distinct from the
// agent's live state (e.g. ~/.claude/plugins/installed_plugins.json). It is the
// "intended state" used to reproduce an environment and to detect drift.
package lockfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// SchemaVersion is the lockfile format version.
const SchemaVersion = 1

// Lockfile is the parsed `.codify/*.lock` document plus the path it came from.
type Lockfile struct {
	Version  int     `json:"version"`
	Scope    string  `json:"scope"`
	Packages []Entry `json:"packages"`

	path string // not serialized
}

// Entry is one recorded package install.
type Entry struct {
	ID          string `json:"id"`
	Target      string `json:"target"`
	Version     string `json:"version,omitempty"`
	SourceKind  string `json:"sourceKind,omitempty"`
	Checksum    string `json:"checksum,omitempty"`
	InstalledAt string `json:"installedAt"`
}

// Load reads the lockfile at path. A missing file yields an empty lockfile
// (not an error) so callers can Upsert + Save to create it. A malformed file
// is an explicit error rather than a silent overwrite.
func Load(path string) (*Lockfile, error) {
	l := &Lockfile{Version: SchemaVersion, path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return l, nil
		}
		return nil, fmt.Errorf("lockfile: read %s: %w", path, err)
	}
	if len(data) == 0 {
		return l, nil
	}
	if err := json.Unmarshal(data, l); err != nil {
		return nil, fmt.Errorf("lockfile: parse %s: %w (refusing to overwrite)", path, err)
	}
	l.path = path
	if l.Version == 0 {
		l.Version = SchemaVersion
	}
	return l, nil
}

// Upsert inserts or replaces the entry matched by (ID, Target).
func (l *Lockfile) Upsert(e Entry) {
	for i := range l.Packages {
		if l.Packages[i].ID == e.ID && l.Packages[i].Target == e.Target {
			l.Packages[i] = e
			return
		}
	}
	l.Packages = append(l.Packages, e)
}

// Remove deletes the entry matched by (id, target). Returns true if one was
// removed.
func (l *Lockfile) Remove(id, target string) bool {
	for i := range l.Packages {
		if l.Packages[i].ID == id && l.Packages[i].Target == target {
			l.Packages = append(l.Packages[:i], l.Packages[i+1:]...)
			return true
		}
	}
	return false
}

// Save writes the lockfile to its path atomically (tmp + rename), creating the
// parent directory if needed. Entries are sorted by (target, id) for stable
// diffs.
func (l *Lockfile) Save() error {
	if l.path == "" {
		return errors.New("lockfile: no path set")
	}
	if l.Version == 0 {
		l.Version = SchemaVersion
	}
	sort.Slice(l.Packages, func(i, j int) bool {
		if l.Packages[i].Target != l.Packages[j].Target {
			return l.Packages[i].Target < l.Packages[j].Target
		}
		return l.Packages[i].ID < l.Packages[j].ID
	})

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return fmt.Errorf("lockfile: ensure dir: %w", err)
	}
	out, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("lockfile: marshal: %w", err)
	}
	out = append(out, '\n')
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("lockfile: write tmp: %w", err)
	}
	if err := os.Rename(tmp, l.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("lockfile: rename: %w", err)
	}
	return nil
}

// PathForScope returns the lockfile path for a scope: ~/.codify/workstation.lock
// for workstation, <cwd>/.codify/project.lock for project.
func PathForScope(scope catalog.Scope) (string, error) {
	switch scope {
	case catalog.ScopeWorkstation:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("lockfile: resolve home: %w", err)
		}
		return filepath.Join(home, ".codify", "workstation.lock"), nil
	case catalog.ScopeProject:
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("lockfile: resolve cwd: %w", err)
		}
		return filepath.Join(cwd, ".codify", "project.lock"), nil
	default:
		return "", fmt.Errorf("lockfile: unsupported scope %q", scope)
	}
}
