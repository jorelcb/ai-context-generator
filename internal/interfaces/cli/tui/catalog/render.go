package catalog

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- estilos (lipgloss degrada solo bajo NO_COLOR / terminal sin color) ---

var (
	activeTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("63")).Padding(0, 1)
	inactiveTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1)
	headerInfoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	categoryStyle    = lipgloss.NewStyle().Bold(true)
	descStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cursorStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	installedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	footerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	warnStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// Result is what the selector returns to its caller.
type Result struct {
	Selected  []Selected
	Cancelled bool
}

// Model is the bubbletea wrapper around the pure-logic Catalog. It only
// translates key messages into Catalog calls and renders — all selection state
// lives in the (testable) Catalog.
type Model struct {
	cat       *Catalog
	glyphs    glyphSet
	width     int
	height    int
	warn      string // inline warning (e.g. empty selection on confirm)
	done      bool
	cancelled bool
}

// NewModel wraps a Catalog for rendering, choosing Unicode or ASCII glyphs from
// the environment (spec §Compatibilidad).
func NewModel(cat *Catalog) Model {
	return Model{cat: cat, width: 80, glyphs: pickGlyphs()}
}

func (m Model) Init() tea.Cmd { return nil }

// Update handles keys. Key map (spec §Contrato): tab/shift+tab switch tabs;
// arrows/jk move + expand/collapse; space toggles; a toggles-all-in-tab; enter
// confirms (warns if empty); q/esc/ctrl+c cancel.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		m.warn = ""
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "tab", "right", "l":
			// right/l expands a collapsed category first; only switches tab when
			// the cursor isn't on a collapsible-and-collapsed node.
			if msg.String() != "tab" && m.canExpandHere() {
				m.cat.Expand()
			} else {
				m.cat.NextTab()
			}
		case "shift+tab":
			m.cat.PrevTab()
		case "left", "h":
			m.cat.Collapse()
		case "up", "k":
			m.cat.CursorUp()
		case "down", "j":
			m.cat.CursorDown()
		case "enter":
			if m.cat.TotalChecked() == 0 {
				m.warn = "Selecciona al menos un paquete en alguna categoría"
			} else {
				m.done = true
				return m, tea.Quit
			}
		case " ":
			m.cat.ToggleCurrent()
		case "a":
			m.cat.ToggleAllInActiveTab()
		}
	}
	return m, nil
}

// canExpandHere reports whether the cursor is on a collapsible, currently
// collapsed category (so right-arrow expands instead of switching tab).
func (m Model) canExpandHere() bool {
	r, ok := m.cat.CurrentRow()
	if !ok || r.Kind != RowCategory || !r.HasCats {
		return false
	}
	return !m.cat.ActiveTab().Categories[r.Cat].Expanded
}

// Result returns the outcome after the program exits.
func (m Model) Result() Result {
	if m.cancelled {
		return Result{Cancelled: true}
	}
	return Result{Selected: m.cat.Collect()}
}

func (m Model) View() string {
	var b strings.Builder

	// --- header: tab bar + scope/total ---
	scope := m.cat.Scope
	if scope == "" {
		scope = "project"
	}
	info := headerInfoStyle.Render(fmt.Sprintf("Scope: %s · %d elegidos", scope, m.cat.TotalChecked()))
	b.WriteString(m.renderTabBar())
	b.WriteString("   ")
	b.WriteString(info)
	b.WriteString("\n\n")

	// --- body: tree ---
	tab := m.cat.ActiveTab()
	rows := m.cat.VisibleRows()
	if len(rows) == 0 {
		b.WriteString(descStyle.Render("  (esta categoría no tiene paquetes)\n"))
	}
	for i, r := range rows {
		cat := &tab.Categories[r.Cat]
		var line string
		switch r.Kind {
		case RowCategory:
			if !r.HasCats {
				continue // tab plana (Hooks/Plugins): no se muestra el header sintético
			}
			arrow := m.glyphs.collapsed
			counter := ""
			if cat.Expanded {
				arrow = m.glyphs.expanded
				counter = fmt.Sprintf("   (%d/%d)", cat.CheckedCount(), len(cat.Leaves))
			} else if cat.State() == Some {
				counter = fmt.Sprintf("   (%d/%d)", cat.CheckedCount(), len(cat.Leaves))
			}
			line = fmt.Sprintf("%s %s %s%s", cat.State().Glyph(), arrow, categoryStyle.Render(cat.Name), headerInfoStyle.Render(counter))
		case RowLeaf:
			leaf := cat.Leaves[r.Leaf]
			box := "[ ]"
			if leaf.Checked {
				box = "[x]"
			}
			indent := ""
			if r.HasCats {
				indent = "  "
			}
			extra := ""
			if leaf.Installed {
				extra += installedStyle.Render(" " + m.glyphs.installed + " instalado")
			}
			if leaf.Desc != "" {
				extra += "  " + descStyle.Render(leaf.Desc)
			}
			line = fmt.Sprintf("%s%s %s%s", indent, box, leaf.ID, extra)
		}
		if i == m.cat.Cursor {
			b.WriteString(cursorStyle.Render(m.glyphs.cursor+" ") + line + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}

	// --- footer ---
	b.WriteString("\n")
	if m.warn != "" {
		b.WriteString(warnStyle.Render("! "+m.warn) + "\n")
	}
	b.WriteString(footerStyle.Render(m.footerHints()))
	return b.String()
}

func (m Model) footerHints() string {
	g := m.glyphs
	r, ok := m.cat.CurrentRow()
	if ok && r.Kind == RowCategory && r.HasCats {
		if m.cat.ActiveTab().Categories[r.Cat].Expanded {
			return fmt.Sprintf("%s moverse · %s colapsar · espacio marcar-categoría · %s tab · %s instalar · q salir", g.updown, g.left, g.tab, g.enter)
		}
		return fmt.Sprintf("%s moverse · %s expandir · espacio marcar-categoría · %s tab · %s instalar · q salir", g.updown, g.right, g.tab, g.enter)
	}
	return fmt.Sprintf("%s moverse · espacio marcar · a marcar-todo · %s cambiar tab · %s instalar · q salir", g.updown, g.tab, g.enter)
}

// renderTabBar draws the tab strip, degrading on overflow (spec §Tabs overflow):
// tier 1 replaces inactive "(n)" counts with a compact bullet; tier 2 truncates
// inactive tab names. The active tab is always shown in full.
func (m Model) renderTabBar() string {
	full := m.styledTabs(false, 0)
	if m.width <= 0 || lipgloss.Width(full) <= m.width {
		return full
	}
	if bullets := m.styledTabs(true, 0); lipgloss.Width(bullets) <= m.width {
		return bullets
	}
	return m.styledTabs(true, 6) // truncate inactive names
}

// styledTabs builds the tab row. bullet=true renders inactive marked tabs with a
// compact bullet instead of "(n)"; trunc>0 truncates inactive names to trunc
// runes + an ellipsis.
func (m Model) styledTabs(bullet bool, trunc int) string {
	tabs := make([]string, 0, len(m.cat.Tabs))
	for i := range m.cat.Tabs {
		t := &m.cat.Tabs[i]
		active := i == m.cat.Active
		name := t.Name
		if trunc > 0 && !active {
			if rs := []rune(name); len(rs) > trunc {
				name = string(rs[:trunc]) + m.glyphs.ellipsis
			}
		}
		label := name
		if n := t.CheckedCount(); n > 0 {
			if bullet && !active {
				label += " " + m.glyphs.bullet
			} else {
				label += fmt.Sprintf(" (%d)", n)
			}
		}
		if active {
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(label))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// Run launches the selector and returns the user's selection. The caller must
// ensure a TTY (non-TTY callers use the flag/declarative path instead).
func Run(cat *Catalog) (Result, error) {
	m := NewModel(cat)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return Result{Cancelled: true}, err
	}
	return final.(Model).Result(), nil
}
