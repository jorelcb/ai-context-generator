package catalog

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- estilos (lipgloss degrada solo bajo NO_COLOR / terminal sin color) ---

var (
	activeTabStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("63")).Padding(0, 1)
	inactiveTabStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1)
	headerInfoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	categoryStyle        = lipgloss.NewStyle().Bold(true)
	descStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cursorStyle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	installedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	footerStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	warnStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dividerStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	detailTitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
	detailKeyStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	filterActiveStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("63"))
	filterCommittedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
)

// sideBySideMin is the minimum terminal width at which the detail panel sits to
// the right of the tree; below it, the panel stacks under the tree.
const (
	sideBySideMin = 96
	detailWidth   = 34
)

// Result is what the selector returns to its caller.
type Result struct {
	Selected  []Selected
	Cancelled bool
}

// Model is the bubbletea wrapper around the pure-logic Catalog. It only
// translates key messages into Catalog calls and renders — all selection and
// filter state lives in the (testable) Catalog.
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

// Update handles keys. Key map (spec §Contrato + R-6): tab/shift+tab switch
// tabs; arrows/jk move + expand/collapse; space toggles; a toggles-all-in-tab;
// `/` opens the fuzzy filter; enter confirms (warns if empty); esc clears an
// active filter or otherwise cancels; q/ctrl+c cancel.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		m.warn = ""
		if m.cat.Filtering {
			return m.updateFilter(msg)
		}
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "esc":
			// esc clears an applied filter first; only cancels when there's none.
			if m.cat.Query != "" {
				m.cat.ClearFilter()
			} else {
				m.cancelled = true
				m.done = true
				return m, tea.Quit
			}
		case "/":
			m.cat.StartFilter()
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

// updateFilter handles keys while the fuzzy-filter input is active: printable
// runes edit the query live; esc clears; enter/arrows commit and return focus to
// the tree; tab still switches tabs.
func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.cancelled = true
		m.done = true
		return m, tea.Quit
	case "esc":
		m.cat.ClearFilter()
	case "enter":
		m.cat.CommitFilter()
	case "down":
		m.cat.CommitFilter()
		m.cat.CursorDown()
	case "up":
		m.cat.CommitFilter()
		m.cat.CursorUp()
	case "tab":
		m.cat.CommitFilter()
		m.cat.NextTab()
	case "shift+tab":
		m.cat.CommitFilter()
		m.cat.PrevTab()
	case "backspace":
		m.cat.FilterBackspace()
	default:
		if len(msg.Runes) == 1 {
			m.cat.FilterInput(msg.Runes[0])
		}
	}
	return m, nil
}

// canExpandHere reports whether the cursor is on a collapsible, currently
// collapsed category (so right-arrow expands instead of switching tab). A live
// filter ignores collapse state, so expansion is never offered while filtering.
func (m Model) canExpandHere() bool {
	if m.cat.Query != "" {
		return false
	}
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
	b.WriteString("\n")

	// --- filter bar (only when filtering or a filter is applied) ---
	if bar := m.renderFilterBar(); bar != "" {
		b.WriteString(bar + "\n")
	}
	b.WriteString("\n")

	// --- body: tree + always-visible detail panel ---
	tree := m.renderTree()
	detail := m.renderDetail()
	b.WriteString(m.composeBody(tree, detail))

	// --- footer ---
	b.WriteString("\n")
	if m.warn != "" {
		b.WriteString(warnStyle.Render("! "+m.warn) + "\n")
	}
	b.WriteString(footerStyle.Render(m.footerHints()))
	return b.String()
}

// renderTree builds the tree body (tab bar excluded). Filtered rows already come
// pre-flattened from the Catalog; rendering is identical to the unfiltered tree.
func (m Model) renderTree() string {
	var b strings.Builder
	tab := m.cat.ActiveTab()
	rows := m.cat.VisibleRows()
	if len(rows) == 0 {
		if m.cat.Query != "" {
			return descStyle.Render(fmt.Sprintf("  (sin coincidencias para «%s»)", m.cat.Query))
		}
		return descStyle.Render("  (esta categoría no tiene paquetes)")
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
			// Con filtro activo la categoría se muestra siempre abierta.
			expanded := cat.Expanded || m.cat.Query != ""
			if expanded {
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
	return strings.TrimRight(b.String(), "\n")
}

// renderDetail builds the always-visible detail panel for the package under the
// cursor (R-6 / ADR-0010 open-question §2): id/label, version, source, tags,
// install+select state and a wrapped description.
func (m Model) renderDetail() string {
	var b strings.Builder
	b.WriteString(detailTitleStyle.Render("Detalle"))
	b.WriteString("\n")

	leaf, ok := m.cat.CurrentLeaf()
	if !ok {
		b.WriteString(descStyle.Render("(mueve el cursor a un paquete)"))
		return b.String()
	}

	b.WriteString(cursorStyle.Render(leaf.ID) + "\n")
	if leaf.Label != "" && leaf.Label != leaf.ID {
		b.WriteString(descStyle.Render(leaf.Label) + "\n")
	}

	field := func(k, v string) {
		if v == "" {
			return
		}
		b.WriteString(detailKeyStyle.Render(fmt.Sprintf("%-9s", k)) + " " + v + "\n")
	}
	field("versión:", leaf.Version)
	field("source:", leaf.Source)
	field("tags:", leaf.Tags)
	field("estado:", m.detailStatus(leaf))

	if leaf.Desc != "" {
		b.WriteString("\n")
		b.WriteString(descStyle.Width(m.detailContentWidth()).Render(leaf.Desc))
	}
	return b.String()
}

// detailStatus renders the install/select state line for the detail panel.
func (m Model) detailStatus(leaf *Leaf) string {
	g := m.glyphs
	switch {
	case leaf.Checked && leaf.Installed:
		return cursorStyle.Render(g.check+" marcado") + " · " + installedStyle.Render(g.installed+" instalado")
	case leaf.Checked:
		return cursorStyle.Render(g.check + " marcado para instalar")
	case leaf.Installed:
		return installedStyle.Render(g.installed + " instalado")
	default:
		return descStyle.Render("disponible")
	}
}

// detailContentWidth is the wrap width for the panel's description, bounded so
// it reads well whether the panel is to the side or stacked below.
func (m Model) detailContentWidth() int {
	if m.width >= sideBySideMin {
		return detailWidth
	}
	w := m.width
	if w <= 0 || w > 72 {
		w = 72
	}
	return w
}

// composeBody lays out the tree and the detail panel: side by side on wide
// terminals (a vertical rule between them), stacked otherwise (a horizontal
// rule between them).
func (m Model) composeBody(tree, detail string) string {
	if m.width >= sideBySideMin {
		treeW := m.width - detailWidth - 3
		left := lipgloss.NewStyle().MaxWidth(treeW).Render(tree)
		h := lineCount(left)
		if d := lineCount(detail); d > h {
			h = d
		}
		sep := m.verticalRule(h)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, " "+sep+" ", detail)
	}
	return tree + "\n" + m.horizontalRule() + "\n" + detail
}

func lineCount(s string) int { return strings.Count(s, "\n") + 1 }

func (m Model) verticalRule(n int) string {
	if n < 1 {
		n = 1
	}
	rows := make([]string, n)
	for i := range rows {
		rows[i] = dividerStyle.Render(m.glyphs.vline)
	}
	return strings.Join(rows, "\n")
}

func (m Model) horizontalRule() string {
	w := m.width
	if w <= 0 || w > 60 {
		w = 60
	}
	return dividerStyle.Render(strings.Repeat(m.glyphs.hline, w))
}

// renderFilterBar shows the live filter input (with a caret) or, once committed,
// the applied filter. Empty when no filter is in play.
func (m Model) renderFilterBar() string {
	if m.cat.Filtering {
		return filterActiveStyle.Render(" /"+m.cat.Query+"_ ") +
			headerInfoStyle.Render("  escribe para filtrar · ⏎ aplica · esc limpia")
	}
	if m.cat.Query != "" {
		return headerInfoStyle.Render("filtro ") +
			filterCommittedStyle.Render("/"+m.cat.Query) +
			headerInfoStyle.Render("   / edita · esc limpia")
	}
	return ""
}

func (m Model) footerHints() string {
	g := m.glyphs
	if m.cat.Filtering {
		return fmt.Sprintf("escribe para filtrar · %s aplica · esc limpia", g.enter)
	}
	r, ok := m.cat.CurrentRow()
	if ok && r.Kind == RowCategory && r.HasCats && m.cat.Query == "" {
		if m.cat.ActiveTab().Categories[r.Cat].Expanded {
			return fmt.Sprintf("%s moverse · %s colapsar · espacio marcar-categoría · / filtrar · %s tab · %s instalar · q salir", g.updown, g.left, g.tab, g.enter)
		}
		return fmt.Sprintf("%s moverse · %s expandir · espacio marcar-categoría · / filtrar · %s tab · %s instalar · q salir", g.updown, g.right, g.tab, g.enter)
	}
	return fmt.Sprintf("%s moverse · espacio marcar · / filtrar · a marcar-todo · %s cambiar tab · %s instalar · q salir", g.updown, g.tab, g.enter)
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
