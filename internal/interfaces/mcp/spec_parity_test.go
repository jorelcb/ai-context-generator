package mcp

import "testing"

// TestResolveSpecOutputPath_DefaultsToContext guards R-4r: the MCP spec path now
// honors an explicit output directory, defaulting to the context directory when
// none is given — parity with the CLI's `--output` (defaults to `--from-context`).
func TestResolveSpecOutputPath_DefaultsToContext(t *testing.T) {
	cases := []struct {
		name        string
		fromContext string
		output      string
		want        string
	}{
		{"empty output falls back to context", "/proj/ctx", "", "/proj/ctx"},
		{"explicit output is honored", "/proj/ctx", "/proj/specs", "/proj/specs"},
		{"explicit output equal to context", "/proj/ctx", "/proj/ctx", "/proj/ctx"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolveSpecOutputPath(c.fromContext, c.output); got != c.want {
				t.Errorf("resolveSpecOutputPath(%q, %q)=%q, want %q", c.fromContext, c.output, got, c.want)
			}
		})
	}
}

// TestGenerateSpecsTool_ExposesCLIParity asserts the generate_specs MCP tool
// exposes every knob the CLI `codify spec` command does (R-4r). The CLI flags
// are: <name>, --from-context, --output, --locale, --model, --sdd-standard.
// `output` is the param added by R-4r; this test fails if it regresses.
func TestGenerateSpecsTool_ExposesCLIParity(t *testing.T) {
	tool := generateSpecsTool().Tool
	props := tool.InputSchema.Properties

	for _, want := range []string{"name", "from_context", "output", "locale", "model", "sdd_standard"} {
		if _, ok := props[want]; !ok {
			t.Errorf("generate_specs is missing the %q parameter (CLI↔MCP parity gap)", want)
		}
	}

	// name + from_context stay required; output is optional (defaults to context).
	required := map[string]bool{}
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}
	if !required["name"] || !required["from_context"] {
		t.Errorf("name and from_context must stay required; got required=%v", tool.InputSchema.Required)
	}
	if required["output"] {
		t.Errorf("output must be optional (defaults to from_context), but it is marked required")
	}
}
