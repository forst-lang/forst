package typechecker

import (
	"fmt"
	"testing"

	"forst/internal/parser"
)

// TestCheckTypes_combinatorialMain exercises many distinct arithmetic and comparison forms
// to cover infer/unify branches (sequential subtests to keep -race stable under load).
func TestCheckTypes_combinatorialMain(t *testing.T) {
	t.Parallel()
	ops := []struct {
		op string
		ok func(_, b int) bool
	}{
		{"+", func(_, _ int) bool { return true }},
		{"-", func(_, _ int) bool { return true }},
		{"*", func(_, _ int) bool { return true }},
		{"/", func(_, b int) bool { return b != 0 }},
		{"%", func(_, b int) bool { return b != 0 }},
	}
	cmp := []string{"<", ">", "<=", ">=", "==", "!="}
	for _, o := range ops {
		for a := range 8 {
			for b := range 8 {
				if !o.ok(a, b) {
					continue
				}
				src := fmt.Sprintf(`package main
func main() {
	x := %d %s %d
	println(string(x))
}`, a, o.op, b)
				runSnippetSequential(t, src)
			}
		}
	}
	for _, c := range cmp {
		for a := range 6 {
			for b := range 6 {
				src := fmt.Sprintf(`package main
func main() {
	if %d %s %d {
		println("y")
	}
}`, a, c, b)
				runSnippetSequential(t, src)
			}
		}
	}
}

func TestCheckTypes_combinatorialShapes(t *testing.T) {
	t.Parallel()
	for n := range 5 {
		src := fmt.Sprintf(`package main
type T = { n: Int }
func main() {
	v := T{n: %d}
	println(string(v.n))
}`, n)
		runSnippetSequential(t, src)
	}
}

func TestCheckTypes_combinatorialSlices(t *testing.T) {
	t.Parallel()
	for _, lit := range []string{"[1]", "[1, 2]", "[1, 2, 3]"} {
		src := fmt.Sprintf(`package main
func main() {
	xs := %s
	println(string(len(xs)))
}`, lit)
		runSnippetSequential(t, src)
	}
}

func runSnippetSequential(t *testing.T, src string) {
	t.Helper()
	log := setupTestLogger(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, src)
	}
	chk := New(log, false)
	if err := chk.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v\n%s", err, src)
	}
}
