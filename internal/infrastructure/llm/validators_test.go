package llm

import (
	"strings"
	"testing"
)

func TestValidateOutput_DetectsDefineMarkers(t *testing.T) {
	body := `# Title

The currency is [DEFINE: ISO 4217 code], the timezone is [DEFINE].
Padding text to keep the body above the truncation threshold so we do not
trigger the length-based warning together with the marker assertion below.
`
	r := ValidateOutput(body, "generate", "AGENTS.md")
	if len(r.DefineMarkers) != 2 {
		t.Fatalf("DefineMarkers: got %d, want 2 (got: %v)", len(r.DefineMarkers), r.DefineMarkers)
	}
	if r.DefineMarkers[0].Text != "[DEFINE: ISO 4217 code]" {
		t.Fatalf("first marker text: got %q, want %q", r.DefineMarkers[0].Text, "[DEFINE: ISO 4217 code]")
	}
	if r.DefineMarkers[0].Line != 3 {
		t.Fatalf("first marker line: got %d, want 3", r.DefineMarkers[0].Line)
	}
	if r.DefineMarkers[1].Text != "[DEFINE]" {
		t.Fatalf("second marker text: got %q, want %q", r.DefineMarkers[1].Text, "[DEFINE]")
	}
	if r.DefineMarkers[1].Line != 3 {
		t.Fatalf("second marker line: got %d, want 3", r.DefineMarkers[1].Line)
	}
	if r.Fatal {
		t.Fatal("Fatal should be false; markers are a soft signal")
	}
}

func TestValidateOutput_FlagsUnbalancedFences(t *testing.T) {
	body := "# Title\n\n```go\nfmt.Println(\"hi\")\nbody is long enough to avoid the truncation warning so the only thing flagged is the unclosed code fence in this fixture, kept short on purpose."
	r := ValidateOutput(body, "generate", "DEVELOPMENT_GUIDE.md")
	found := false
	for _, w := range r.Warnings {
		if contains(w, "unbalanced code fences") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unbalanced fence warning, got %v", r.Warnings)
	}
}

func TestValidateOutput_RequiresFrontmatterForSkill(t *testing.T) {
	body := `# Skill body

No frontmatter here, but enough text to exceed the truncation threshold so the
only warning we want surfaced is about the missing YAML frontmatter that every
SKILL.md must declare at the top of the file before any markdown content.
`
	r := ValidateOutput(body, "skills", "SKILL.md")
	found := false
	for _, w := range r.Warnings {
		if contains(w, "expected YAML frontmatter") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected frontmatter warning, got %v", r.Warnings)
	}
}

func TestValidateOutput_EmptyIsFatal(t *testing.T) {
	r := ValidateOutput("   \n  ", "generate", "AGENTS.md")
	if !r.Fatal {
		t.Fatal("empty output must be Fatal")
	}
}

func TestValidateOutput_ShortSkillStubIsFlagged(t *testing.T) {
	// A stub with valid frontmatter used to slip past the truncation
	// heuristic because frontmatter files were exempted from the length check.
	body := "---\nname: x\ndescription: y\n---\n\nStub.\n"
	r := ValidateOutput(body, "skills", "SKILL.md")
	found := false
	for _, w := range r.Warnings {
		if contains(w, "suspiciously short") {
			found = true
		}
	}
	if !found {
		t.Fatalf("a 40-char SKILL.md stub must trigger the short-output warning, got %v", r.Warnings)
	}
}

func TestValidateOutput_FenceAtStartOfContentIsCounted(t *testing.T) {
	// A fence on the very first line is an opening too — the old "\n```"
	// counter only saw it via a special-case prefix check; the per-line
	// anchor must keep covering it.
	body := "```go\nfmt.Println(\"hi\")\nthe rest of this fixture only exists to push the body past the truncation threshold so the single warning surfaced is the unclosed fence that starts at offset zero of the content."
	r := ValidateOutput(body, "generate", "DEVELOPMENT_GUIDE.md")
	found := false
	for _, w := range r.Warnings {
		if contains(w, "unbalanced code fences") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unbalanced fence warning for fence at offset 0, got %v", r.Warnings)
	}
}

func TestValidateOutput_SizeGuards(t *testing.T) {
	bigBody := func(lines int) string {
		var sb strings.Builder
		for range lines {
			sb.WriteString("a line of actionable context that earns its place in the file\n")
		}
		return sb.String()
	}

	cases := []struct {
		name     string
		fileName string
		lines    int
		wantOver bool
	}{
		{"CLAUDE.md over 200", "CLAUDE.md", 250, true},
		{"CLAUDE.md under 200", "CLAUDE.md", 150, false},
		{"AGENTS.md over 500", "AGENTS.md", 600, true},
		{"AGENTS.md under 500", "AGENTS.md", 400, false},
		{"SKILL.md over 500", "SKILL.md", 600, true},
		{"unbudgeted file is never flagged", "CONTEXT.md", 5000, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := ValidateOutput(bigBody(tc.lines), "generate", tc.fileName)
			over := false
			for _, w := range r.Warnings {
				if contains(w, "over the") && contains(w, "line budget") {
					over = true
				}
			}
			if over != tc.wantOver {
				t.Errorf("size warning = %v, want %v (warnings: %v)", over, tc.wantOver, r.Warnings)
			}
		})
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
