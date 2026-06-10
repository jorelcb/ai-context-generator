package prompts

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// key feeds a key string ("enter", "esc", "down", …) through Update.
func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func opts() []Option {
	return []Option{{"English", "en"}, {"Spanish", "es"}, {"French", "fr"}}
}

func TestSelect_DefaultPositionsCursor(t *testing.T) {
	m := NewSelect("Lang", opts(), "es")
	if m.Cursor != 1 || m.Value() != "es" {
		t.Errorf("default 'es' should set cursor=1, got cursor=%d value=%q", m.Cursor, m.Value())
	}
	if NewSelect("Lang", opts(), "nope").Cursor != 0 {
		t.Error("unknown default should fall back to first option")
	}
}

func TestSelect_NavigateAndConfirm(t *testing.T) {
	var m tea.Model = NewSelect("Lang", opts(), "en")
	m, _ = m.Update(key("down"))
	m, _ = m.Update(key("down"))
	m, _ = m.Update(key("down")) // clamped at last
	m, _ = m.Update(key("enter"))
	sm := m.(SelectModel)
	if !sm.Done || sm.Cancelled {
		t.Fatalf("enter should finish without cancel: %+v", sm)
	}
	if sm.Value() != "fr" {
		t.Errorf("value=%q, want fr", sm.Value())
	}
}

func TestSelect_EscCancels(t *testing.T) {
	var m tea.Model = NewSelect("Lang", opts(), "en")
	m, _ = m.Update(key("esc"))
	if sm := m.(SelectModel); !sm.Cancelled {
		t.Error("esc should cancel")
	}
}

func TestInput_TypeBackspaceConfirm(t *testing.T) {
	var m tea.Model = NewInput("Name", "default")
	for _, r := range "abz" {
		m, _ = m.Update(key(string(r)))
	}
	m, _ = m.Update(key("backspace"))
	m, _ = m.Update(key("enter"))
	im := m.(InputModel)
	if !im.Done || im.Cancelled {
		t.Fatalf("enter should finish: %+v", im)
	}
	if im.Value() != "ab" {
		t.Errorf("value=%q, want ab", im.Value())
	}
}

func TestInput_EmptyKeepsPlaceholderSemantics(t *testing.T) {
	var m tea.Model = NewInput("Name", "default")
	m, _ = m.Update(key("enter"))
	im := m.(InputModel)
	if im.Value() != "" {
		t.Errorf("untouched input should return empty (caller applies default), got %q", im.Value())
	}
}

func TestInput_SpaceIsTyped(t *testing.T) {
	var m tea.Model = NewInput("Name", "")
	m, _ = m.Update(key("a"))
	m, _ = m.Update(key("space"))
	m, _ = m.Update(key("b"))
	if im := m.(InputModel); im.Value() != "a b" {
		t.Errorf("value=%q, want \"a b\"", im.Value())
	}
}

func TestConfirm_YesNoAndToggle(t *testing.T) {
	// 'n' answers directly.
	var m tea.Model = NewConfirm("Proceed?", true)
	m, _ = m.Update(key("n"))
	if cm := m.(ConfirmModel); !cm.Done || cm.Value {
		t.Errorf("'n' should finish with false: %+v", cm)
	}
	// arrow toggles then enter.
	m = NewConfirm("Proceed?", false)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m, _ = m.Update(key("enter"))
	if cm := m.(ConfirmModel); !cm.Value {
		t.Errorf("toggle+enter should yield true: %+v", cm)
	}
}

func TestConfirm_EscCancelsKeepingDefault(t *testing.T) {
	var m tea.Model = NewConfirm("Proceed?", true)
	m, _ = m.Update(key("esc"))
	if cm := m.(ConfirmModel); !cm.Cancelled {
		t.Error("esc should cancel")
	}
}

func TestViews_RenderTitleOptionsAndDoneLine(t *testing.T) {
	s := NewSelect("Language", opts(), "en")
	v := s.View()
	for _, want := range []string{"Language", "English", "Spanish", "cancelar"} {
		if !strings.Contains(v, want) {
			t.Errorf("select view missing %q:\n%s", want, v)
		}
	}
	s.Done = true
	if !strings.Contains(s.View(), "English") {
		t.Error("done view should print the chosen label")
	}
	s.Cancelled = true
	if s.View() != "" {
		t.Error("cancelled view should render nothing")
	}

	i := NewInput("Name", "dflt")
	if !strings.Contains(i.View(), "dflt") {
		t.Error("input view should show the placeholder while empty")
	}

	c := NewConfirm("Sure?", true)
	if v := c.View(); !strings.Contains(v, "Sí") || !strings.Contains(v, "No") {
		t.Errorf("confirm view should show both choices:\n%s", v)
	}
}
