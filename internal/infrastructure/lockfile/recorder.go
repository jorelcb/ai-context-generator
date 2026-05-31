package lockfile

import (
	"context"
	"time"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// Recorder writes install/uninstall records into the per-scope lockfile. It
// satisfies the application's InstallRecorder port.
type Recorder struct {
	pathForScope func(catalog.Scope) (string, error)
	now          func() time.Time
}

// NewRecorder returns a Recorder using the real per-scope paths and clock.
func NewRecorder() *Recorder {
	return &Recorder{pathForScope: PathForScope, now: time.Now}
}

// NewRecorderWith injects the path resolver and clock (tests).
func NewRecorderWith(pathForScope func(catalog.Scope) (string, error), now func() time.Time) *Recorder {
	return &Recorder{pathForScope: pathForScope, now: now}
}

// Record upserts the given installed manifests into the scope's lockfile,
// stamping each with the install time. Idempotent per (id, target).
func (r *Recorder) Record(_ context.Context, scope catalog.Scope, installed []catalog.PackageManifest) error {
	if len(installed) == 0 {
		return nil
	}
	path, err := r.pathForScope(scope)
	if err != nil {
		return err
	}
	lf, err := Load(path)
	if err != nil {
		return err
	}
	lf.Scope = string(scope)
	ts := r.now().UTC().Format(time.RFC3339)
	for _, m := range installed {
		lf.Upsert(Entry{
			ID:          m.ID,
			Target:      string(m.Target),
			Version:     m.Version,
			SourceKind:  m.Source.Kind,
			SourceURI:   m.Source.URI,
			Checksum:    m.SourceChecksum,
			InstalledAt: ts,
		})
	}
	return lf.Save()
}

// Recorded reads the scope's lockfile back into the domain's InstalledPackage
// shape — codify's intended state, for drift comparison against live state.
// A missing lockfile yields an empty slice (not an error). It satisfies the
// application's LockfileReader port.
func (r *Recorder) Recorded(_ context.Context, scope catalog.Scope) ([]catalog.InstalledPackage, error) {
	path, err := r.pathForScope(scope)
	if err != nil {
		return nil, err
	}
	lf, err := Load(path)
	if err != nil {
		return nil, err
	}
	out := make([]catalog.InstalledPackage, 0, len(lf.Packages))
	for _, e := range lf.Packages {
		out = append(out, catalog.InstalledPackage{
			ID:                e.ID,
			Version:           e.Version,
			Target:            catalog.Target(e.Target),
			Scope:             scope,
			SourceURI:         e.SourceURI,
			InstalledAt:       e.InstalledAt,
			InstalledChecksum: e.Checksum,
		})
	}
	return out, nil
}

// Forget removes the given package IDs of a target from the scope's lockfile
// (used by uninstall). Saving only when something changed.
func (r *Recorder) Forget(_ context.Context, scope catalog.Scope, ids []string, target catalog.Target) error {
	if len(ids) == 0 {
		return nil
	}
	path, err := r.pathForScope(scope)
	if err != nil {
		return err
	}
	lf, err := Load(path)
	if err != nil {
		return err
	}
	changed := false
	for _, id := range ids {
		if lf.Remove(id, string(target)) {
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return lf.Save()
}
