package prompts

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
)

// ErrCancelled is returned when the user aborts a prompt (esc / ctrl+c).
var ErrCancelled = errors.New("prompt cancelled")

// RunSelect launches a select prompt and returns the chosen value. The caller
// must ensure a TTY (the commands layer guards with isInteractive()).
func RunSelect(title string, options []Option, defaultVal string) (string, error) {
	final, err := tea.NewProgram(NewSelect(title, options, defaultVal)).Run()
	if err != nil {
		return "", err
	}
	m := final.(SelectModel)
	if m.Cancelled {
		return "", ErrCancelled
	}
	return m.Value(), nil
}

// RunInput launches a text-input prompt and returns the typed text ("" if the
// user just pressed enter — the caller applies its default).
func RunInput(title, placeholder string) (string, error) {
	final, err := tea.NewProgram(NewInput(title, placeholder)).Run()
	if err != nil {
		return "", err
	}
	m := final.(InputModel)
	if m.Cancelled {
		return "", ErrCancelled
	}
	return m.Value(), nil
}

// RunConfirm launches a yes/no prompt and returns the answer.
func RunConfirm(title string, defaultVal bool) (bool, error) {
	final, err := tea.NewProgram(NewConfirm(title, defaultVal)).Run()
	if err != nil {
		return defaultVal, err
	}
	m := final.(ConfirmModel)
	if m.Cancelled {
		return defaultVal, ErrCancelled
	}
	return m.Value, nil
}
