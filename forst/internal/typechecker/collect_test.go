package typechecker

import (
	"testing"

	"forst/internal/parser"
)

// TestCollect_importAndTypeDefsCheckTypes exercises collectExplicitTypes (imports, type defs, functions)
// through the real CheckTypes entrypoint.
func TestCollect_importAndTypeDefsCheckTypes(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

import "fmt"

type Label = String

func greet(): String {
	return "hi"
}

func main() {
	fmt.Println(greet())
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	if _, ok := tc.Defs["Label"]; !ok {
		t.Fatal("expected type alias Label in Defs after collect")
	}
}

// TestCollect_typeGuardRegisteredInDefs ensures type guards from collectExplicitTypes land in Defs.
func TestCollect_typeGuardRegisteredInDefs(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

is (n Int) Positive() {
	ensure n is GreaterThan(0)
}

func main() {
	v := 1
	ensure v is Positive() {
		println("bad")
	}
	println("ok")
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	if _, ok := tc.Defs["Positive"]; !ok {
		t.Fatal("expected type guard Positive in Defs")
	}
}
