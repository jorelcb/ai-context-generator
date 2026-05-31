package command

import (
	"context"
	"errors"
	"fmt"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// InstallerRegistry routes a Target to the TargetInstaller that handles it.
// Defined here (not imported from infrastructure) so CatalogService depends
// only on a behavior, not on a concrete registry — the infra
// targetinstaller.Registry satisfies it structurally.
type InstallerRegistry interface {
	For(t catalog.Target) (catalog.TargetInstaller, error)
}

// InstallRecorder records successfully-installed packages into a per-scope
// lockfile (ADR-0010 §6, D.9). Defined here as a behavior so CatalogService
// stays decoupled from the concrete lockfile; the infra lockfile.Recorder
// satisfies it. Optional — a nil recorder simply skips lockfile bookkeeping.
type InstallRecorder interface {
	Record(ctx context.Context, scope catalog.Scope, installed []catalog.PackageManifest) error
}

// LockfileReader reads the per-scope lockfile back into the domain's
// InstalledPackage shape — codify's *intended* state (what it recorded
// installing). Defined here as a behavior so CatalogService stays decoupled
// from the concrete lockfile; the infra lockfile.Recorder satisfies it.
// Optional — Status requires it, but install/list do not.
type LockfileReader interface {
	Recorded(ctx context.Context, scope catalog.Scope) ([]catalog.InstalledPackage, error)
}

// CatalogService orchestrates the read/write package ports behind the
// `catalog` command (ADR-0010): it lists what a PackageSource offers, reports
// what a TargetInstaller has on disk, and installs selected packages by
// fetching their content and routing to the right installer.
//
// It is the single use case the CLI (interactive + flags), and later config
// and init, drive — so the orchestration is tested once, here, independent
// of the presentation layer.
type CatalogService struct {
	source   catalog.PackageSource
	registry InstallerRegistry
	recorder InstallRecorder // optional; records installs into the lockfile
	reader   LockfileReader  // optional; reads the lockfile back for Status
}

// NewCatalogService wires the source (where packages come from) and the
// installer registry (where they go).
func NewCatalogService(source catalog.PackageSource, registry InstallerRegistry) *CatalogService {
	return &CatalogService{source: source, registry: registry}
}

// WithRecorder attaches a lockfile recorder so successful installs are
// recorded per scope. Returns the service for chaining.
func (s *CatalogService) WithRecorder(r InstallRecorder) *CatalogService {
	s.recorder = r
	return s
}

// WithReader attaches a lockfile reader so Status can compare codify's
// recorded installs against live state. Returns the service for chaining.
func (s *CatalogService) WithReader(r LockfileReader) *CatalogService {
	s.reader = r
	return s
}

// Available returns the packages the source offers. When target is non-empty,
// the list is filtered to that Target (the active type tab); when empty, all
// packages are returned.
func (s *CatalogService) Available(ctx context.Context, target catalog.Target) ([]catalog.PackageManifest, error) {
	all, err := s.source.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("catalog: list source %s: %w", s.source.Kind(), err)
	}
	if target == "" {
		return all, nil
	}
	filtered := make([]catalog.PackageManifest, 0, len(all))
	for _, m := range all {
		if m.Target == target {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// Installed reports the packages currently on disk for target's ecosystem in
// the given scope. Used to mark already-installed entries in the UI.
func (s *CatalogService) Installed(ctx context.Context, target catalog.Target, scope catalog.Scope) ([]catalog.InstalledPackage, error) {
	inst, err := s.registry.For(target)
	if err != nil {
		return nil, err
	}
	return inst.InstalledList(ctx, scope)
}

// InstallRequest describes a batch install: every ID must be a package of
// Target, installed into Scope.
type InstallRequest struct {
	Target catalog.Target
	IDs    []string
	Scope  catalog.Scope
}

// InstallOutcome summarizes a batch install. Installed lists the IDs written;
// NotFound lists requested IDs that the source does not offer for the target.
type InstallOutcome struct {
	Installed []string
	NotFound  []string
}

// Install fetches and installs each requested package. It fails loud on the
// first fetch/install error (returning the partial outcome so the caller can
// report what landed), but unknown IDs are collected into NotFound rather
// than aborting — a typo in one package shouldn't hide which others exist.
func (s *CatalogService) Install(ctx context.Context, req InstallRequest) (InstallOutcome, error) {
	var outcome InstallOutcome

	inst, err := s.registry.For(req.Target)
	if err != nil {
		return outcome, err
	}

	all, err := s.source.List(ctx)
	if err != nil {
		return outcome, fmt.Errorf("catalog: list source %s: %w", s.source.Kind(), err)
	}
	index := make(map[string]catalog.PackageManifest, len(all))
	for _, m := range all {
		if m.Target == req.Target {
			index[m.ID] = m
		}
	}

	var installed []catalog.PackageManifest
	for _, id := range req.IDs {
		m, ok := index[id]
		if !ok {
			outcome.NotFound = append(outcome.NotFound, id)
			continue
		}
		content, err := s.source.Fetch(ctx, m)
		if err != nil {
			return outcome, fmt.Errorf("catalog: fetch %q: %w", id, err)
		}
		if err := inst.Install(ctx, m, content, req.Scope); err != nil {
			return outcome, fmt.Errorf("catalog: install %q: %w", id, err)
		}
		outcome.Installed = append(outcome.Installed, id)
		installed = append(installed, m)
	}

	// Record the installs into the scope's lockfile (best-effort bookkeeping —
	// the packages are already on disk, so a lockfile-write failure is
	// surfaced but does not undo the install).
	if s.recorder != nil && len(installed) > 0 {
		if err := s.recorder.Record(ctx, req.Scope, installed); err != nil {
			return outcome, fmt.Errorf("catalog: record lockfile: %w", err)
		}
	}

	return outcome, nil
}

// DriftStatus classifies a recorded package against the ecosystems' live
// state.
type DriftStatus string

const (
	// DriftInSync — recorded in the lockfile and present on disk.
	DriftInSync DriftStatus = "in-sync"
	// DriftMissing — recorded in the lockfile but absent on disk (removed by
	// hand, or never finished installing).
	DriftMissing DriftStatus = "missing"
)

// DriftEntry pairs a recorded package with its current drift status.
type DriftEntry struct {
	Package catalog.InstalledPackage
	Status  DriftStatus
}

// DriftReport is the result of comparing a scope's lockfile (codify's intended
// state) against the installers' live state.
type DriftReport struct {
	Scope   catalog.Scope
	Entries []DriftEntry
}

// Missing returns the count of recorded packages absent on disk.
func (r DriftReport) Missing() int {
	n := 0
	for _, e := range r.Entries {
		if e.Status == DriftMissing {
			n++
		}
	}
	return n
}

// Status compares the scope's lockfile against the live install state and
// reports drift per recorded package. It reads only what codify recorded
// (the lockfile is the source of truth for intent); a package present on disk
// but never recorded is out of scope here — Status answers "is everything I
// installed still there?", the reproducibility question.
func (s *CatalogService) Status(ctx context.Context, scope catalog.Scope) (DriftReport, error) {
	report := DriftReport{Scope: scope}
	if s.reader == nil {
		return report, errors.New("catalog: no lockfile reader configured")
	}
	recorded, err := s.reader.Recorded(ctx, scope)
	if err != nil {
		return report, fmt.Errorf("catalog: read lockfile: %w", err)
	}

	// Query each distinct recorded target's installer once and index its live
	// packages by (target, id).
	type key struct {
		target catalog.Target
		id     string
	}
	live := make(map[key]bool)
	queried := make(map[catalog.Target]bool)
	for _, r := range recorded {
		if queried[r.Target] {
			continue
		}
		queried[r.Target] = true
		inst, err := s.registry.For(r.Target)
		if err != nil {
			continue // no installer for the target → its packages read as missing
		}
		list, err := inst.InstalledList(ctx, scope)
		if err != nil {
			return report, fmt.Errorf("catalog: live state for %s: %w", r.Target, err)
		}
		for _, p := range list {
			live[key{p.Target, p.ID}] = true
		}
	}

	for _, r := range recorded {
		st := DriftMissing
		if live[key{r.Target, r.ID}] {
			st = DriftInSync
		}
		report.Entries = append(report.Entries, DriftEntry{Package: r, Status: st})
	}
	return report, nil
}
