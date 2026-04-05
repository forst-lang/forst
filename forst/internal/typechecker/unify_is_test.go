package typechecker

import (
	"testing"

	"forst/internal/parser"
)

// TestUnifyIs_typeGuardCallInIfCondition exercises unifyIsOperator via real CheckTypes (see unify_is.go).
func TestUnifyIs_typeGuardCallInIfCondition(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type N = Int

is (v N) Ok() {
	ensure v is GreaterThan(0)
}

func main() {
	x := 1
	if x is Ok() {
		println("ok")
	}
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
