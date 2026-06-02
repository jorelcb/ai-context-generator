// Package catalog implementa el selector interactivo rico del catálogo
// (tabs por tipo + árbol categoría→paquete + checkbox tri-estado + selección
// acumulada cross-tab), decidido en ADR-0013.
//
// Este archivo es el NÚCLEO DE LÓGICA PURA: no importa bubbletea/lipgloss ni
// ninguna librería de terminal. Toda la mecánica de selección/navegación vive
// aquí para que sea unit-testeable sin terminal — la lección directa de la
// regresión de v3.0.0 ("ningún test lo atrapó"). El render (model_bubbletea.go)
// es una capa delgada que traduce teclas a llamadas de este motor y dibuja.
package catalog

// TriState describe el estado de selección de un nodo padre (categoría),
// derivado de sus hijos.
type TriState int

const (
	// None — ningún hijo marcado: [ ]
	None TriState = iota
	// Some — algunos hijos marcados: [~]
	Some
	// All — todos los hijos marcados: [x]
	All
)

// Glyph devuelve el checkbox ASCII del estado (fallback universal; el render
// puede sustituir por glifos Unicode/colores).
func (t TriState) Glyph() string {
	switch t {
	case All:
		return "[x]"
	case Some:
		return "[~]"
	default:
		return "[ ]"
	}
}

// Leaf es un paquete instalable (hoja del árbol).
type Leaf struct {
	ID        string
	Desc      string
	Checked   bool // marcado para instalar en esta sesión
	Installed bool // ya instalado en el scope activo (marca informativa)
}

// Category agrupa paquetes (parte del árbol). Para tabs sin categorías reales
// (Hooks, Plugins) se usa una única Category con HasCategories=false en el Tab.
type Category struct {
	Name     string
	Expanded bool
	Leaves   []Leaf
}

// State deriva el tri-estado de la categoría a partir de sus hijos. O(n) sobre
// los hijos de la categoría (no del árbol entero).
func (c *Category) State() TriState {
	if len(c.Leaves) == 0 {
		return None
	}
	checked := 0
	for _, l := range c.Leaves {
		if l.Checked {
			checked++
		}
	}
	switch {
	case checked == 0:
		return None
	case checked == len(c.Leaves):
		return All
	default:
		return Some
	}
}

// CheckedCount / Total para el contador (n/total) del render.
func (c *Category) CheckedCount() int {
	n := 0
	for _, l := range c.Leaves {
		if l.Checked {
			n++
		}
	}
	return n
}

// SetAll marca/desmarca todos los hijos de la categoría.
func (c *Category) SetAll(v bool) {
	for i := range c.Leaves {
		c.Leaves[i].Checked = v
	}
}

// Tab es una pestaña por tipo de primitivo del ecosystem (Skills/Hooks/Plugins).
// Type es el identificador canónico del tipo ("skill", "hook", "plugin") usado
// al instalar; Name es la etiqueta visible.
type Tab struct {
	Name          string
	Type          string
	Categories    []Category
	HasCategories bool // false => árbol degenera a lista plana (spec §"una sola categoría")
}

// CheckedCount cuenta los marcados en toda la tab (para el contador del header).
func (t *Tab) CheckedCount() int {
	n := 0
	for ci := range t.Categories {
		n += t.Categories[ci].CheckedCount()
	}
	return n
}

// Catalog es el estado raíz del selector: las tabs, la tab activa, el cursor
// dentro del árbol, y el contexto (scope/ecosystem) que el render muestra en el
// header. La selección es GLOBAL — vive en los Leaf.Checked de todas las tabs y
// persiste al cambiar de tab.
type Catalog struct {
	Tabs      []Tab
	Active    int // índice de la tab activa
	Cursor    int // índice de fila visible dentro de la tab activa
	Scope     string
	Ecosystem string
}

// RowKind distingue filas de categoría (padre) y de paquete (hoja) en el
// aplanado visible del árbol.
type RowKind int

const (
	RowCategory RowKind = iota
	RowLeaf
)

// Row es una fila visible del árbol (resultado de aplanar con colapso). Cat/Leaf
// son índices dentro de la tab activa.
type Row struct {
	Kind    RowKind
	Cat     int
	Leaf    int
	Depth   int
	HasCats bool
}

// ActiveTab devuelve un puntero a la tab activa.
func (c *Catalog) ActiveTab() *Tab { return &c.Tabs[c.Active] }

// VisibleRows aplana la tab activa a filas visibles, respetando el colapso de
// cada categoría. Para tabs sin categorías (HasCategories=false) la fila de
// categoría existe pero el render la oculta y muestra las hojas directamente.
func (c *Catalog) VisibleRows() []Row {
	t := c.ActiveTab()
	var rows []Row
	for ci := range t.Categories {
		cat := &t.Categories[ci]
		rows = append(rows, Row{Kind: RowCategory, Cat: ci, Depth: 0, HasCats: t.HasCategories})
		// Sin categorías reales: las hojas siempre visibles (lista plana).
		// Con categorías: solo si está expandida.
		if !t.HasCategories || cat.Expanded {
			for li := range cat.Leaves {
				rows = append(rows, Row{Kind: RowLeaf, Cat: ci, Leaf: li, Depth: 1, HasCats: t.HasCategories})
			}
		}
	}
	return rows
}

// NextTab / PrevTab rotan la tab activa y reposicionan el cursor al inicio (un
// índice de cursor de la tab anterior no es válido en la nueva).
func (c *Catalog) NextTab() {
	if len(c.Tabs) == 0 {
		return
	}
	c.Active = (c.Active + 1) % len(c.Tabs)
	c.Cursor = 0
}

func (c *Catalog) PrevTab() {
	if len(c.Tabs) == 0 {
		return
	}
	c.Active = (c.Active - 1 + len(c.Tabs)) % len(c.Tabs)
	c.Cursor = 0
}

// CursorUp / CursorDown mueven el cursor entre filas visibles (sin entrar a
// hijos colapsados, porque VisibleRows ya los excluye).
func (c *Catalog) CursorUp() {
	if c.Cursor > 0 {
		c.Cursor--
	}
}

func (c *Catalog) CursorDown() {
	if c.Cursor < len(c.VisibleRows())-1 {
		c.Cursor++
	}
}

// CurrentRow devuelve la fila bajo el cursor (ok=false si no hay filas).
func (c *Catalog) CurrentRow() (Row, bool) {
	rows := c.VisibleRows()
	if c.Cursor < 0 || c.Cursor >= len(rows) {
		return Row{}, false
	}
	return rows[c.Cursor], true
}

// Expand expande la categoría bajo el cursor (no-op si es hoja, ya expandida o
// si la tab no tiene categorías).
func (c *Catalog) Expand() {
	r, ok := c.CurrentRow()
	if !ok || r.Kind != RowCategory || !r.HasCats {
		return
	}
	c.ActiveTab().Categories[r.Cat].Expanded = true
}

// Collapse colapsa la categoría bajo el cursor; si el cursor está sobre una
// hoja, mueve el cursor a su categoría padre y la colapsa.
func (c *Catalog) Collapse() {
	r, ok := c.CurrentRow()
	if !ok || !r.HasCats {
		return
	}
	tab := c.ActiveTab()
	tab.Categories[r.Cat].Expanded = false
	// Reposicionar el cursor sobre la fila de la categoría colapsada.
	for i, rr := range c.VisibleRows() {
		if rr.Kind == RowCategory && rr.Cat == r.Cat {
			c.Cursor = i
			break
		}
	}
}

// ToggleCurrent alterna la marca bajo el cursor:
//   - hoja: invierte su Checked (el padre recalcula su estado automáticamente).
//   - categoría: si no está toda marcada, marca todos sus hijos; si lo está, los
//     desmarca (spec §Selección).
func (c *Catalog) ToggleCurrent() {
	r, ok := c.CurrentRow()
	if !ok {
		return
	}
	cat := &c.ActiveTab().Categories[r.Cat]
	switch r.Kind {
	case RowLeaf:
		cat.Leaves[r.Leaf].Checked = !cat.Leaves[r.Leaf].Checked
	case RowCategory:
		cat.SetAll(cat.State() != All)
	}
}

// ToggleAllInActiveTab implementa la tecla `a`: si hay ≥1 marcado en la tab
// activa, desmarca todo; si no, marca todo. No afecta otras tabs (spec §`a`).
func (c *Catalog) ToggleAllInActiveTab() {
	t := c.ActiveTab()
	target := t.CheckedCount() == 0 // si nada marcado => marcar todo
	for ci := range t.Categories {
		t.Categories[ci].SetAll(target)
	}
}

// Selected es un paquete marcado, con el contexto necesario para instalarlo.
type Selected struct {
	TabType  string // "skill" | "hook" | "plugin"
	Category string
	ID       string
}

// Collect devuelve TODOS los paquetes marcados, de TODAS las tabs (selección
// global acumulada). Las categorías/tabs sin marcas no aportan entradas.
func (c *Catalog) Collect() []Selected {
	var out []Selected
	for ti := range c.Tabs {
		t := &c.Tabs[ti]
		for ci := range t.Categories {
			cat := &t.Categories[ci]
			for _, l := range cat.Leaves {
				if l.Checked {
					out = append(out, Selected{TabType: t.Type, Category: cat.Name, ID: l.ID})
				}
			}
		}
	}
	return out
}

// TotalChecked es el conteo global de marcados (para el header "N elegidos").
func (c *Catalog) TotalChecked() int {
	n := 0
	for ti := range c.Tabs {
		n += c.Tabs[ti].CheckedCount()
	}
	return n
}
