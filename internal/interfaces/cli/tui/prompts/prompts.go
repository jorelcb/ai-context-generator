// Package prompts implementa los tres prompts interactivos básicos del CLI
// (select / input / confirm) sobre bubbletea + lipgloss, reemplazando a
// charmbracelet/huh (huh-drop: consolidar en un solo stack TUI, el mismo del
// selector rico de catalog — ADR-0013).
//
// Cada prompt es un tea.Model mínimo e inline (sin alt-screen). Los modelos
// son testeables sin TTY: Update se alimenta con tea.KeyMsg directamente y
// View() retorna string — el mismo enfoque de tui/catalog (la lección de
// v3.0.0). La cancelación (esc/ctrl+c) se reporta como Cancelled=true y los
// callers la traducen a error, conservando el contrato de los wrappers
// promptSelect/promptInput/promptConfirm.
package prompts

import (
	"os"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	cursorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	faintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	doneStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

// cursorGlyph devuelve el marcador de fila activa, honrando CODIFY_ASCII y
// locales sin UTF-8 (mismas reglas que tui/catalog/glyphs.go, en miniatura).
func cursorGlyph() string {
	if asciiOnly() {
		return ">"
	}
	return "❯"
}

func asciiOnly() bool {
	switch strings.ToLower(os.Getenv("CODIFY_ASCII")) {
	case "1", "true", "yes":
		return true
	}
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := os.Getenv(k)
		if v == "" {
			continue
		}
		up := strings.ToUpper(v)
		return !(strings.Contains(up, "UTF-8") || strings.Contains(up, "UTF8"))
	}
	return runtime.GOOS == "windows"
}
