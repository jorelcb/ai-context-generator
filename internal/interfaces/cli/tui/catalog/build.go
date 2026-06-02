package catalog

import "sort"

// Pkg is the domain-agnostic input describing one installable package for the
// selector. The CLI maps a domain PackageManifest -> Pkg (reading the
// source-declared Metadata[category]); keeping this package free of domain
// imports keeps its logic trivially testable.
type Pkg struct {
	ID        string
	Label     string
	Desc      string
	Category  string // source-declared; empty => ungrouped
	Installed bool
}

// TabSpec describes one tab (a package type) and its packages.
type TabSpec struct {
	Name string // visible label, e.g. "Skills"
	Type string // canonical type, e.g. "skill"
	Pkgs []Pkg
}

// generalGroup holds packages with no declared category inside an
// otherwise-categorized tab.
const generalGroup = "General"

// Build assembles a Catalog from per-tab package specs. Within a tab, packages
// group by their SOURCE-DECLARED Category; if no package in the tab declares
// one, the tab renders as a flat list (HasCategories=false). In a categorized
// tab, packages without a category fall into "General". Categories and leaves
// are sorted for deterministic output (the iteration order of the bucket map is
// not relied upon).
func Build(scope, ecosystem string, specs []TabSpec) *Catalog {
	c := &Catalog{Scope: scope, Ecosystem: ecosystem}
	for _, spec := range specs {
		c.Tabs = append(c.Tabs, buildTab(spec))
	}
	return c
}

func buildTab(spec TabSpec) Tab {
	hasCats := false
	for _, p := range spec.Pkgs {
		if p.Category != "" {
			hasCats = true
			break
		}
	}

	t := Tab{Name: spec.Name, Type: spec.Type, HasCategories: hasCats}

	if !hasCats {
		t.Categories = []Category{{Name: "", Leaves: leavesSorted(spec.Pkgs)}}
		return t
	}

	buckets := map[string][]Pkg{}
	for _, p := range spec.Pkgs {
		key := p.Category
		if key == "" {
			key = generalGroup
		}
		buckets[key] = append(buckets[key], p)
	}
	names := make([]string, 0, len(buckets))
	for k := range buckets {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Categories = append(t.Categories, Category{Name: name, Leaves: leavesSorted(buckets[name])})
	}
	return t
}

func leavesSorted(pkgs []Pkg) []Leaf {
	cp := append([]Pkg(nil), pkgs...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].ID < cp[j].ID })
	leaves := make([]Leaf, 0, len(cp))
	for _, p := range cp {
		leaves = append(leaves, Leaf{ID: p.ID, Desc: p.Desc, Installed: p.Installed})
	}
	return leaves
}
