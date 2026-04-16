package transformergo

import (
	"fmt"
	"io"
	"testing"

	"forst/internal/parser"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// TestTransformForstFileToGo_manyMinimalPrograms runs parse → typecheck → Go emit for small sources.
func TestTransformForstFileToGo_manyMinimalPrograms(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	cases := []struct {
		name string
		src  string
	}{
		{
			"println_int",
			`package main
func main() {
	println(string(1))
}`,
		},
		{
			"if_else",
			`package main
func main() {
	if true {
		println("a")
	} else {
		println("b")
	}
}`,
		},
		{
			"shape_type",
			`package main
type T = { n: Int }
func main() {
	v := { n: 1 }
	println(string(v.n))
}`,
		},
		{
			"type_alias",
			`package main
type P = { x: Int }
func main() {
	v := { x: 2 }
	println(string(v.x))
}`,
		},
		{
			"two_funcs",
			`package main
func id(x Int) {
	println(string(x))
}
func main() {
	id(3)
}`,
		},
		{
			"mul_sub",
			`package main
func main() {
	println(string(2 * 3 - 1))
}`,
		},
		{
			"string_concat",
			`package main
func main() {
	println("a" + "b")
}`,
		},
		{
			"comparison",
			`package main
func main() {
	if 1 < 2 {
		println("lt")
	}
}`,
		},
		{
			"assign",
			`package main
func main() {
	x := 1
	x = x + 1
	println(string(x))
}`,
		},
		{
			"defer",
			`package main
func work() {}
func main() {
	defer work()
	println("d")
}`,
		},
		{
			"go_stmt",
			`package main
func work() {}
func main() {
	go work()
	println("g")
}`,
		},
		{
			"for_while",
			`package main
func main() {
	n := 0
	for n < 2 {
		n = n + 1
	}
	println("f")
}`,
		},
		{
			"for_three_clause_incdec",
			`package main
func main() {
	n := 2
	for i := 0; i < n; i++ {
		println("x")
	}
	println("f")
}`,
		},
		{
			"for_three_clause_post_assign",
			`package main
func main() {
	for i := 0; i < 2; i = i + 1 {
		println("a")
	}
}`,
		},
		{
			"for_three_clause_post_call_expr",
			`package main
func tick() {}
func main() {
	for ; ; tick() {
		break
	}
	println("ok")
}`,
		},
		{
			"go_builtins_append_copy_min",
			`package main
func main() {
	xs := [1]
	ys := append(xs, 2, 3)
	a := [1, 2]
	b := [0, 0]
	println(string(len(ys)))
	println(string(copy(b, a)))
	println(string(min(3, 2, 1)))
}`,
		},
		{
			"go_builtins_map_clear_recover",
			`package main
func main() {
	m := map[String]Int{ "a": 1 }
	delete(m, "a")
	sl := [1, 2]
	clear(sl)
	recover()
	complex(1.0, 2.0)
	println("ok")
}`,
		},
		{
			"if_short_decl_in_condition",
			`package main
func main() {
	if x := 1; x > 0 {
		println("y")
	}
}`,
		},
		{
			"slice_index",
			`package main
func main() {
	xs := [1, 2]
	println(string(xs[0]))
}`,
		},
		{
			"logical",
			`package main
func main() {
	if true && !false {
		println("ok")
	}
}`,
		},
		{
			"nested_if",
			`package main
func main() {
	if true {
		if false {
			println("a")
		} else {
			println("b")
		}
	}
}`,
		},
		{
			"pointer_field",
			`package main
type Box = { p: *Int }
func main() {
	n := 1
	v := { p: &n }
	println(string(*v.p))
}`,
		},
		{
			"range_over_slice",
			`package main
func main() {
	xs := [1, 2]
	for i := range xs {
		println(string(i))
	}
}`,
		},
		{
			"modulo",
			`package main
func main() {
	println(string(7 % 3))
}`,
		},
		{
			"result_if_ok",
			`package main
func one() {
	n := 1
	ensure n is GreaterThan(0)
	return n
}
func main() {
	x := one()
	if x is Ok() {
		println(x)
	}
}`,
		},
		{
			"ensure_min_then_print",
			`package main
func main() {
	x := 1
	ensure x is Min(1)
	println(string(x))
}
`,
		},
		{
			"if_else_if_else",
			`package main
func main() {
	n := 2
	if n > 10 {
		println("a")
	} else if n < 0 {
		println("b")
	} else {
		println("c")
	}
}
`,
		},
		{
			"for_break_continue",
			`package main
func main() {
	for {
		break
	}
	n := 0
	for n < 2 {
		n = n + 1
		continue
	}
	println("x")
}`,
		},
		{
			"for_range_no_var",
			`package main
func main() {
	xs := [1, 2]
	for range xs {
		println("r")
	}
}`,
		},
		{
			"map_literal_delete",
			`package main
func main() {
	m := map[String]Int{ "a": 1 }
	delete(m, "a")
	println("ok")
}`,
		},
		{
			"type_alias_field",
			`package main
type Row = { id: Int }
func main() {
	r := { id: 1 }
	println(string(r.id))
}`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := parser.NewTestParser(tc.src, log)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			chk := typechecker.New(log, false)
			if err := chk.CheckTypes(nodes); err != nil {
				t.Fatalf("typecheck: %v", err)
			}
			tr := New(chk, log)
			f, err := tr.TransformForstFileToGo(nodes)
			if err != nil {
				t.Fatalf("transform: %v", err)
			}
			if f == nil || len(f.Decls) == 0 {
				t.Fatal("empty output")
			}
		})
	}
}

func TestTransformForstFileToGo_emitsPackageKeyword(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	src := `package main
func main() {
	println("x")
}`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	chk := typechecker.New(log, false)
	if err := chk.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	tr := New(chk, log)
	f, err := tr.TransformForstFileToGo(nodes)
	if err != nil {
		t.Fatal(err)
	}
	if f.Name == nil || f.Name.Name != "main" {
		t.Fatalf("package name: %+v", f.Name)
	}
}

// TestTransformForstFileToGo_combinatorialArithmetic exercises binary/unary emit paths (see typechecker combinatorial_operators_test).
func TestTransformForstFileToGo_combinatorialArithmetic(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	ops := []struct {
		op string
		ok func(a, b int) bool
	}{
		{"+", func(_, _ int) bool { return true }},
		{"-", func(_, _ int) bool { return true }},
		{"*", func(_, _ int) bool { return true }},
		{"/", func(_, b int) bool { return b != 0 }},
		{"%", func(_, b int) bool { return b != 0 }},
	}
	for _, o := range ops {
		for a := 0; a < 8; a++ {
			for b := 0; b < 8; b++ {
				if !o.ok(a, b) {
					continue
				}
				src := fmt.Sprintf(`package main
func main() {
	println(string(%d %s %d))
}`, a, o.op, b)
				p := parser.NewTestParser(src, log)
				nodes, err := p.ParseFile()
				if err != nil {
					t.Fatalf("parse %s: %v", src, err)
				}
				chk := typechecker.New(log, false)
				if err := chk.CheckTypes(nodes); err != nil {
					t.Fatalf("typecheck %s: %v", src, err)
				}
				tr := New(chk, log)
				if _, err := tr.TransformForstFileToGo(nodes); err != nil {
					t.Fatalf("transform %s: %v", src, err)
				}
			}
		}
	}
}

// TestTransformForstFileToGo_combinatorialComparison mirrors typechecker combinatorial comparison tests.
func TestTransformForstFileToGo_combinatorialComparison(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	cmp := []string{"<", ">", "<=", ">=", "==", "!="}
	for _, c := range cmp {
		for a := 0; a < 6; a++ {
			for b := 0; b < 6; b++ {
				src := fmt.Sprintf(`package main
func main() {
	if %d %s %d {
		println("y")
	}
}`, a, c, b)
				p := parser.NewTestParser(src, log)
				nodes, err := p.ParseFile()
				if err != nil {
					t.Fatalf("parse: %v", err)
				}
				chk := typechecker.New(log, false)
				if err := chk.CheckTypes(nodes); err != nil {
					t.Fatalf("typecheck: %v", err)
				}
				tr := New(chk, log)
				if _, err := tr.TransformForstFileToGo(nodes); err != nil {
					t.Fatalf("transform: %v", err)
				}
			}
		}
	}
}
