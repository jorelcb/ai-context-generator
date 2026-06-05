package catalog

import "testing"

func TestFuzzyMatch_Subsequence(t *testing.T) {
	cases := []struct {
		query, target string
		want          bool
	}{
		{"", "anything", true},         // empty query matches all
		{"ddd", "ddd-entity", true},    // prefix
		{"dent", "ddd-entity", true},   // non-contiguous subsequence
		{"DDD", "ddd-entity", true},    // case-insensitive
		{"entity", "ddd-entity", true}, // contiguous mid-string
		{"xyz", "ddd-entity", false},   // missing letters
		{"yttne", "ddd-entity", false}, // right letters, wrong order
		{"cqrs", "cqrs-command", true}, // full prefix word
		{"cmd", "cqrs-command", true},  // scattered
		{"commandx", "command", false}, // query longer than target
	}
	for _, c := range cases {
		if got := fuzzyMatch(c.query, c.target); got != c.want {
			t.Errorf("fuzzyMatch(%q, %q)=%v, want %v", c.query, c.target, got, c.want)
		}
	}
}

func TestLeafMatches_ScansIDLabelDesc(t *testing.T) {
	l := &Leaf{ID: "ddd-entity", Label: "DDD Entity", Desc: "Model domain entities"}
	if !leafMatches(l, "entity") {
		t.Error("should match on ID")
	}
	if !leafMatches(l, "DDD Ent") {
		t.Error("should match on Label")
	}
	if !leafMatches(l, "domain") {
		t.Error("should match on Desc")
	}
	if leafMatches(l, "zzz") {
		t.Error("should not match unrelated query")
	}
	if !leafMatches(l, "") {
		t.Error("empty query matches")
	}
}
