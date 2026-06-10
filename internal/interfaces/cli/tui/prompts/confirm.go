package prompts

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ConfirmModel es el tea.Model de confirmación sí/no. y/n responde directo,
// ←→/h/l/tab alternan, enter confirma, esc/ctrl+c cancela.
type ConfirmModel struct {
	Title     string
	Value     bool
	Done      bool
	Cancelled bool
}

// NewConfirm builds a yes/no prompt preset to defaultVal.
func NewConfirm(title string, defaultVal bool) ConfirmModel {
	return ConfirmModel{Title: title, Value: defaultVal}
}

func (m ConfirmModel) Init() tea.Cmd { return nil }

func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "esc":
		m.Cancelled = true
		m.Done = true
		return m, tea.Quit
	case "y", "Y":
		m.Value = true
		m.Done = true
		return m, tea.Quit
	case "n", "N":
		m.Value = false
		m.Done = true
		return m, tea.Quit
	case "left", "right", "h", "l", "tab":
		m.Value = !m.Value
	case "enter":
		m.Done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m ConfirmModel) View() string {
	var b strings.Builder
	if m.Done {
		if !m.Cancelled {
			answer := "No"
			if m.Value {
				answer = "Sí"
			}
			fmt.Fprintf(&b, "%s %s\n", titleStyle.Render(m.Title), doneStyle.Render(answer))
		}
		return b.String()
	}
	yes, no := "  Sí  ", "  No  "
	if m.Value {
		yes = cursorStyle.Render("[ Sí ]")
	} else {
		no = cursorStyle.Render("[ No ]")
	}
	fmt.Fprintf(&b, "%s  %s %s\n", titleStyle.Render(m.Title), yes, no)
	b.WriteString(faintStyle.Render("y/n responder · ←→ alternar · enter confirmar · esc cancelar") + "\n")
	return b.String()
}
