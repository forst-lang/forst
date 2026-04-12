package typechecker

import (
	"strings"
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

// TestUnifyIs_typeGuardCallInIfCondition_invalidLeftHandSide exercises unifyIsOperator when getLeftmostVariable fails (non-variable LHS of `is`).
func TestUnifyIs_typeGuardCallInIfCondition_invalidLeftHandSide(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type N = Int

is (v N) Ok() {
	ensure v is GreaterThan(0)
}

func main() {
	if 1 is Ok() {
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
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error for non-variable left-hand side of is")
	}
	if !strings.Contains(err.Error(), "invalid left-hand side of 'is' operator") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnifyIs_nominalErrorBaseType_rejected(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

error ParseError { code: Int }
type ErrKind = ParseError

func main() {
	var e: ErrKind = { code: 1 }
	if e is ParseError {
		println(e)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: nominal error cannot be bare `is` guard")
	}
	if !strings.Contains(err.Error(), "cannot be used as an `is` guard") || !strings.Contains(err.Error(), "Err()") {
		t.Fatalf("unexpected error: %v", err)
	}
}
