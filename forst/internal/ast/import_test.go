package ast

import (
	"strings"
	"testing"
)

func TestImportGroupNode_string_and_kind(t *testing.T) {
	g := ImportGroupNode{Imports: []ImportNode{{Path: "a"}, {Path: "b"}}}
	if g.Kind() != NodeKindImportGroup {
		t.Fatal(g.Kind())
	}
	s := g.String()
	if !strings.Contains(s, "a") || !strings.Contains(s, "b") {
		t.Fatalf("ImportGroupNode.String() should include both imports: %q", s)
	}
}

func TestImportNode_alias_format(t *testing.T) {
	alias := Ident{ID: "f"}
	n := ImportNode{Path: "fmt", Alias: &alias}
	s := n.String()
	if !strings.Contains(s, "as") || !strings.Contains(s, "f") {
		t.Fatalf("ImportNode.String() = %q", s)
	}
}

func TestImportNode_IsGrouped_placeholder(t *testing.T) {
	n := ImportNode{Path: "fmt"}
	if n.IsGrouped() {
		t.Fatal("ImportNode.IsGrouped() placeholder returns false")
	}
	alias := Ident{ID: "f"}
	withAlias := ImportNode{Path: "fmt", Alias: &alias}
	if withAlias.IsGrouped() {
		t.Fatal("ImportNode.IsGrouped() placeholder returns false")
	}
}

func TestImportNode_path_only_kind(t *testing.T) {
	n := ImportNode{Path: "os"}
	if n.Kind() != NodeKindImport || !strings.Contains(n.String(), "os") {
		t.Fatal(n.Kind(), n.String())
	}
}
