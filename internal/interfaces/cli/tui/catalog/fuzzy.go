package catalog

import "strings"

// Este archivo es parte del NÚCLEO DE LÓGICA PURA (sin imports de terminal):
// el matcher difuso que alimenta el filtro `/` del selector (R-6 / ADR-0010
// open-question §1). In-house a propósito — no añadimos `sahilm/fuzzy` ni otra
// dependencia: misma higiene que fijó D6 (YAML sobre TOML) y E1 (preferir lo ya
// transitivo). El algoritmo es una coincidencia de subsecuencia case-insensitive
// (las letras del query aparecen en orden, no necesariamente contiguas), que es
// el comportamiento esperado de un filtro tipo fzf/Ctrl-P.

// fuzzyMatch reports whether every rune of query appears in target in order
// (case-insensitive subsequence). An empty query always matches.
func fuzzyMatch(query, target string) bool {
	if query == "" {
		return true
	}
	q := []rune(strings.ToLower(query))
	t := []rune(strings.ToLower(target))
	qi := 0
	for ti := 0; ti < len(t) && qi < len(q); ti++ {
		if t[ti] == q[qi] {
			qi++
		}
	}
	return qi == len(q)
}

// leafMatches reports whether a leaf satisfies the query, scanning its ID, Label
// and Description so a user can filter by any of them.
func leafMatches(l *Leaf, query string) bool {
	if query == "" {
		return true
	}
	return fuzzyMatch(query, l.ID) ||
		fuzzyMatch(query, l.Label) ||
		fuzzyMatch(query, l.Desc)
}
