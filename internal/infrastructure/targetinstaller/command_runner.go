package targetinstaller

import (
	"context"
	"os/exec"
	"strings"
)

// CommandRunner abstracts executing an external binary so installers that
// delegate to a native CLI (e.g. ClaudePluginInstaller shelling out to
// `claude plugin install`) can be unit-tested with a fake.
type CommandRunner interface {
	// LookPath reports whether name resolves to an executable on PATH,
	// returning its resolved path or an error.
	LookPath(name string) (string, error)
	// Run executes name with args and returns combined stdout+stderr. A
	// non-zero exit is returned as an error (with the output preserved in the
	// returned string for diagnostics).
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// execRunner is the production CommandRunner backed by os/exec.
type execRunner struct{}

// NewExecRunner returns a CommandRunner that shells out via os/exec.
func NewExecRunner() CommandRunner { return execRunner{} }

func (execRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (execRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
