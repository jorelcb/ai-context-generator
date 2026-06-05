package catalog

import (
	"os"
	"runtime"
	"strings"
)

// glyphSet holds the symbols the selector draws, with a Unicode set and an ASCII
// fallback for terminals that can't render box/arrow glyphs (spec §Compatibilidad).
type glyphSet struct {
	cursor    string // active-row marker
	collapsed string // collapsed category
	expanded  string // expanded category
	installed string // already-installed marker
	bullet    string // compact "has marks" indicator on overflowed tabs
	ellipsis  string // truncation marker
	check     string // marca de una hoja seleccionada (panel de detalle)
	vline     string // regla vertical (separador del panel lateral)
	hline     string // regla horizontal (separador del panel apilado)
	// footer key tokens
	updown string
	left   string
	right  string
	tab    string
	enter  string
}

var unicodeGlyphs = glyphSet{
	cursor: "❯", collapsed: "▸", expanded: "▾", installed: "✓", bullet: "•", ellipsis: "…",
	check: "●", vline: "│", hline: "─",
	updown: "↑↓", left: "←", right: "→", tab: "⇥", enter: "⏎",
}

var asciiGlyphs = glyphSet{
	cursor: ">", collapsed: "+", expanded: "-", installed: "*", bullet: "*", ellipsis: "...",
	check: "*", vline: "|", hline: "-",
	updown: "up/dn", left: "left", right: "right", tab: "tab", enter: "enter",
}

// pickGlyphs chooses the Unicode or ASCII set from the environment. Override with
// CODIFY_ASCII=1. Otherwise: a UTF-8 locale => Unicode; a set non-UTF-8 locale =>
// ASCII; no locale info => ASCII on Windows (legacy consoles), Unicode elsewhere.
func pickGlyphs() glyphSet {
	if asciiOnly() {
		return asciiGlyphs
	}
	return unicodeGlyphs
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
