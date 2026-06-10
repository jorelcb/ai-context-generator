package prompts

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Option es una opción de un menú de selección.
type Option struct {
	Label string
	Value string
}

// SelectModel es el tea.Model de selección única. Navegación ↑↓/jk, enter
// confirma, esc/q/ctrl+c cancela.
type SelectModel struct {
	Title     string
	Options   []Option
	Cursor    int
	Done      bool
	Cancelled bool
}

// NewSelect builds a select prompt with the cursor on defaultVal (first option
// if absent).
func NewSelect(title string, options []Option, defaultVal string) SelectModel {
	cursor := 0
	for i, o := range options {
		if o.Value == defaultVal {
			cursor = i
			break
		}
	}
	return SelectModel{Title: title, Options: options, Cursor: cursor}
}

func (m SelectModel) Init() tea.Cmd { return nil }

func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "esc", "q":
		m.Cancelled = true
		m.Done = true
		return m, tea.Quit
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < len(m.Options)-1 {
			m.Cursor++
		}
	case "enter":
		m.Done = true
		return m, tea.Quit
	}
	return m, nil
}

// Value returns the selected option's value ("" if none).
func (m SelectModel) Value() string {
	if len(m.Options) == 0 || m.Cursor < 0 || m.Cursor >= len(m.Options) {
		return ""
	}
	return m.Options[m.Cursor].Value
}

func (m SelectModel) View() string {
	var b strings.Builder
	if m.Done {
		// Línea compacta final (como deja huh): título + valor elegido.
		if !m.Cancelled {
			fmt.Fprintf(&b, "%s %s\n", titleStyle.Render(m.Title+":"), doneStyle.Render(m.Options[m.Cursor].Label))
		}
		return b.String()
	}
	b.WriteString(titleStyle.Render(m.Title) + "\n")
	g := cursorGlyph()
	for i, o := range m.Options {
		if i == m.Cursor {
			fmt.Fprintf(&b, "%s %s\n", cursorStyle.Render(g), o.Label)
		} else {
			fmt.Fprintf(&b, "  %s\n", o.Label)
		}
	}
	b.WriteString(faintStyle.Render("↑↓ moverse · enter elegir · esc cancelar") + "\n")
	return b.String()
}
