package targetinstaller

import (
	"fmt"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// Registry routes a package to the installer that handles its Target. The
// catalog command (D.4) builds one Registry with every ecosystem installer
// it supports and calls For(target) to dispatch installs/uninstalls.
//
// Installers are consulted in registration order; the first one whose
// Handles(target) returns true wins. Registering two installers that both
// handle the same Target is a configuration error the caller controls — the
// first registered takes precedence.
type Registry struct {
	installers []catalog.TargetInstaller
}

// NewRegistry builds a Registry from the given installers, in priority order.
func NewRegistry(installers ...catalog.TargetInstaller) *Registry {
	return &Registry{installers: installers}
}

// For returns the installer that handles the given Target, or an error if no
// registered installer claims it.
func (r *Registry) For(t catalog.Target) (catalog.TargetInstaller, error) {
	for _, inst := range r.installers {
		if inst.Handles(t) {
			return inst, nil
		}
	}
	return nil, fmt.Errorf("targetinstaller: no installer registered for target %q", t)
}
