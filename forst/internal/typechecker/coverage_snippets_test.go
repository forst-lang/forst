package typechecker

import (
	"testing"

	"forst/internal/parser"
)

// TestCheckTypes_manyMinimalPrograms exercises common CheckTypes paths for coverage (parse → collect → infer).
func TestCheckTypes_manyMinimalPrograms(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		src  string
	}{
		{
			"if_comparison",
			`package main
func main() {
	if 1 < 2 {
		println("a")
	}
}`,
		},
		{
			"else_if_chain",
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
}`,
		},
		{
			"for_while_style",
			`package main
func main() {
	n := 0
	for n < 2 {
		n = n + 1
	}
	println("x")
}`,
		},
		{
			"slice_literal_and_index",
			`package main
func main() {
	xs := [1, 2]
	println(string(xs[0]))
}`,
		},
		{
			"type_alias_shape",
			`package main
type P = { x: Int }
func main() {
	v := { x: 1 }
	println(string(v.x))
}`,
		},
		{
			"result_ok_branch",
			`package main
func f(): Result(Int, Error) {
	return 1
}
func main() {
	x := f()
	if x is Ok() {
		println(x)
	}
}`,
		},
		{
			"type_guard_call",
			`package main
type Password = String
is (p Password) Long() {
	ensure p is Min(3)
}
func main() {
	s: Password = "abc"
	ensure s is Long() {
		println("no")
	}
	println("done")
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
			"negation",
			`package main
func main() {
	if !false {
		println("t")
	}
}`,
		},
		{
			"comparison_eq",
			`package main
func main() {
	if 1 == 1 {
		println("eq")
	}
}`,
		},
		{
			"short_decl_and_reassign",
			`package main
func main() {
	x := 1
	x = 2
	println(string(x))
}`,
		},
		{
			"nested_block_scope",
			`package main
func main() {
	if true {
		y := 1
		println(string(y))
	}
	println("out")
}`,
		},
		{
			"logical_or",
			`package main
func main() {
	if false || true {
		println("y")
	}
}`,
		},
		{
			"less_than_compare",
			`package main
func main() {
	a := 1
	b := 2
	if a < b {
		println("lt")
	}
}`,
		},
		{
			"defer_style_not",
			`package main
func work() {}
func main() {
	defer work()
	println("z")
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
			"multiple_returns_void",
			`package main
func early() {
	return
}
func main() {
	early()
	println("m")
}`,
		},
		{
			"string_len_builtin",
			`package main
func main() {
	println(len("hi"))
}`,
		},
		{
			"int_comparison_chain",
			`package main
func main() {
	a := 1
	b := 2
	if a < b && b > 0 {
		println("ok")
	}
}`,
		},
		{
			"shape_field_access",
			`package main
type T = { n: Int }
func main() {
	v := { n: 3 }
	println(string(v.n))
}`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			log := setupTestLogger(nil)
			p := parser.NewTestParser(tc.src, log)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			chk := New(log, false)
			if err := chk.CheckTypes(nodes); err != nil {
				t.Fatalf("CheckTypes: %v", err)
			}
		})
	}
}
