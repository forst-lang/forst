package typechecker

import (
	"strings"
	"testing"

	"forst/internal/parser"
)

func TestFieldPromotion_promotedAccess(t *testing.T) {
	t.Parallel()
	src := `package main

type Inner = {
  Value: Int
}

type Outer = {
  Inner
}

func main() {
  o := Outer{ Inner: { Value: 1 } }
  println(o.Value)
}
`
	log := setupTestLogger(nil)
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
}

func TestFieldPromotion_ambiguousPromotedField(t *testing.T) {
	t.Parallel()
	src := `package main

type A = {
  X: Int
}

type B = {
  X: Int
}

type Outer = {
  A
  B
}

func main() {
  o := Outer{ A: { X: 1 }, B: { X: 2 } }
  println(o.X)
}
`
	log := setupTestLogger(nil)
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil || !strings.Contains(err.Error(), "ambiguous selector") {
		t.Fatalf("got %v", err)
	}
}
