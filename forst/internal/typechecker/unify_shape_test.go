package typechecker

import (
	"testing"

	"forst/internal/parser"
)

// TestUnifyShape_namedAliasWithStructLiteralReturn exercises shape unification through CheckTypes
// (register + infer paths used by unify_shape.go).
func TestUnifyShape_namedAliasWithStructLiteralReturn(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type Point = { x: Int, y: Int }

func origin(): Point {
	return { x: 0, y: 1 }
}

func main() {
	p := origin()
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
	if _, ok := tc.Defs["Point"]; !ok {
		t.Fatal("expected Point type in Defs")
	}
}

// TestUnifyShape_twoShapeLiteralsAssignable checks that compatible anonymous shapes unify for assignment.
func TestUnifyShape_twoShapeLiteralsAssignable(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func main() {
	a := { x: 1, y: 2 }
	b := { x: 3, y: 4 }
	a = b
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
		t.Fatalf("expected compatible shapes to unify: %v", err)
	}
}
