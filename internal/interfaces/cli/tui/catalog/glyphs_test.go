package catalog

import (
	"strings"
	"testing"
)

func TestAsciiOnly_EnvOverride(t *testing.T) {
	t.Setenv("CODIFY_ASCII", "1")
	if !asciiOnly() {
		t.Error("CODIFY_ASCII=1 should force ASCII")
	}
	t.Setenv("CODIFY_ASCII", "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "en_US.UTF-8")
	if asciiOnly() {
		t.Error("UTF-8 locale should not be ASCII-only")
	}
	t.Setenv("LANG", "C")
	if !asciiOnly() {
		t.Error("non-UTF-8 locale (C) should be ASCII-only")
	}
}

func TestView_ASCIIFallbackUsesAsciiGlyphs(t *testing.T) {
	t.Setenv("CODIFY_ASCII", "1")
	c := withDescriptions()
	c.Active = 0
	c.Tabs[0].Categories[0].Expanded = true
	c.markByID("ddd-entity")
	out := NewModel(c).View()

	// No Unicode arrows/cursor/check in ASCII mode.
	for _, bad := range []string{"❯", "▸", "▾", "↑↓", "⇥", "⏎"} {
		if strings.Contains(out, bad) {
			t.Errorf("ASCII view still contains Unicode glyph %q", bad)
		}
	}
	// ASCII markers present.
	if !strings.Contains(out, "> ") { // cursor
		t.Errorf("ASCII cursor '>' missing:\n%s", out)
	}
	if !strings.Contains(out, "- ") { // expanded category arrow
		t.Errorf("ASCII expanded arrow '-' missing:\n%s", out)
	}
	// Tri-state checkboxes are ASCII in both modes.
	if !strings.Contains(out, "[~]") {
		t.Errorf("partial checkbox missing:\n%s", out)
	}
}

func TestRenderTabBar_OverflowDegradesToBullets(t *testing.T) {
	c := withDescriptions()
	// Mark items in two tabs so they carry counts.
	c.Active = 0
	c.markByID("ddd-entity")
	c.Active = 1
	c.markByID("format-on-save")

	m := NewModel(c)
	m.width = 200 // wide: full "(n)" counts
	wide := m.renderTabBar()
	if !strings.Contains(wide, "(1)") {
		t.Errorf("wide tab bar should show numeric counts:\n%s", wide)
	}

	m.width = 12 // very narrow: must degrade. Active tab keeps its full label
	narrow := m.renderTabBar()
	// Inactive marked tab (Skills) drops its "(n)" for the compact bullet, and
	// inactive names truncate — proving the overflow tiers fired. The active tab
	// legitimately keeps its count (spec: active tab always shown in full).
	if !strings.Contains(narrow, m.glyphs.bullet) {
		t.Errorf("narrow tab bar should use the compact bullet, got:\n%s", narrow)
	}
	if !strings.Contains(narrow, m.glyphs.ellipsis) {
		t.Errorf("narrow tab bar should truncate inactive names, got:\n%s", narrow)
	}
}
