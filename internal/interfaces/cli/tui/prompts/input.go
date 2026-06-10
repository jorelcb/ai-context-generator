package prompts

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// InputModel es el tea.Model de entrada de texto en línea. Runas imprimibles
// editan, backspace borra, ctrl+u limpia, enter confirma, esc/ctrl+c cancela.
type InputModel struct {
	Title       string
	Placeholder string // mostrado tenue cuando el valor está vacío (el default)
	Runes       []rune
	Done        bool
	Cancelled   bool
}

// NewInput builds a text-input prompt; placeholder is the default value shown
// faint while the field is empty.
func NewInput(title, placeholder string) InputModel {
	return InputModel{Title: title, Placeholder: placeholder}
}

func (m InputModel) Init() tea.Cmd { return nil }

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "esc":
		m.Cancelled = true
		m.Done = true
		return m, tea.Quit
	case "enter":
		m.Done = true
		return m, tea.Quit
	case "backspace":
		if len(m.Runes) > 0 {
			m.Runes = m.Runes[:len(m.Runes)-1]
		}
	case "ctrl+u":
		m.Runes = nil
	default:
		if key.Type == tea.KeyRunes {
			m.Runes = append(m.Runes, key.Runes...)
		} else if key.Type == tea.KeySpace {
			m.Runes = append(m.Runes, ' ')
		}
	}
	return m, nil
}

// Value returns the typed text (empty if nothing typed).
func (m InputModel) Value() string { return string(m.Runes) }

func (m InputModel) View() string {
	var b strings.Builder
	if m.Done {
		if !m.Cancelled {
			shown := m.Value()
			if shown == "" {
				shown = m.Placeholder
			}
			fmt.Fprintf(&b, "%s %s\n", titleStyle.Render(m.Title+":"), doneStyle.Render(shown))
		}
		return b.String()
	}
	field := m.Value()
	if field == "" && m.Placeholder != "" {
		field = faintStyle.Render(m.Placeholder)
	}
	fmt.Fprintf(&b, "%s %s_\n", titleStyle.Render(m.Title+":"), field)
	b.WriteString(faintStyle.Render("enter confirmar · esc cancelar") + "\n")
	return b.String()
}
