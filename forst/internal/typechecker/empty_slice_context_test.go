package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestInferExpressionTypeWithExpected_emptyStringSlice(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type Tags = []String

func takesTags(tags []String) {}

func main() {
	takesTags([])
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

func TestInferAssignmentTypes_emptySliceAppendWidensToString(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func main() {
	items := []
	items = append(items, "a")
	println(items[0])
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	ty, ok := tc.VariableTypes[ast.Identifier("items")]
	if !ok || len(ty) != 1 || ty[0].Ident != ast.TypeArray || len(ty[0].TypeParams) != 1 || ty[0].TypeParams[0].Ident != ast.TypeString {
		t.Fatalf("items type = %#v ok=%v", ty, ok)
	}
}
