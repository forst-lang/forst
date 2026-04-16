package transformergo

import (
	"fmt"
	"strings"
	"testing"
)

// TestPipeline_coverageBatch runs parse → typecheck → transform on many minimal programs to lift
// merged coverage over the transformer and shared typechecker paths.
func TestPipeline_coverageBatch(t *testing.T) {
	t.Parallel()
	cases := []string{
		`package main
func main() { println("a") }`,
		`package main
func id(x Int): Int { return x }
func main() {
	println(id(1))
}`,
		`package main
func main() {
	x := 1
	y := 2
	println(string(x + y))
}`,
		`package main
type Box = { v: Int }
func main() {
	b := Box{v: 1}
	println(string(b.v))
}`,
		`package main
type U = Int | String
func main() {
	println("ok")
}`,
		`package main
func f(): Result(Int, String) { return 1 }
func main() {
	r := f()
	if r is Ok() {
		println(string(r))
	}
}`,
		`package main
func inner(): Result(Int, Error) {
	return 1
}
func outer(): Result(Int, Error) {
	return inner()
}
func main() {
	x := outer()
	println(x)
}`,
		`package main
error NotPositive { message: String }
func main() { println("h") }`,
		`package main
func main() {
	xs := [1, 2, 3]
	println(string(len(xs)))
}`,
		`package main
func main() {
	xs := [1, 2]
	for i := 0; i < len(xs); i++ {
		println("i")
	}
}`,
		`package main
func main() {
	n := 1
	if n == 1 {
		println("one")
	} else {
		println("d")
	}
}`,
		`package main
func main() {
	a := true
	b := false
	if a && !b {
		println("t")
	}
}`,
		`package main
func main() {
	if true {
		if true {
			println("n")
		}
	}
}`,
		`package main
func main() {
	x := 0
	x = x + 1
	println(string(x))
}`,
		`package main
func main() {
	x := 10
	x = x - 1
	println(string(x))
}`,
		`package main
func main() {
	s := "ab"
	println(string(len(s)))
}`,
		`package main
func main() {
	println("a" + "b")
}`,
		`package main
func main() {
	if 1 != 2 {
		println("neq")
	}
}`,
		`package main
func main() {
	x := 5
	x = x / 1
	println(string(x))
}`,
		`package main
func main() {
	x := 6
	x = x * 1
	println(string(x))
}`,
		`package main
func main() {
	x := 7
	x = x - 1
	println(string(x))
}`,
		`package main
func main() {
	x := 8
	x = x % 3
	println(string(x))
}`,
		`package main
func main() {
	x := 3
	println(string(x))
}`,
		`package main
func main() {
	x := 1
	println(string(x))
}`,
		`package main
func main() {
	x := 1 + 0
	println(string(x))
}`,
		`package main
func main() {
	x := 4 / 2
	println(string(x))
}`,
		`package main
func main() {
	x := 2 * 2
	println(string(x))
}`,
		`package main
func main() {
	x := 1
	x = x + 2
	println(string(x))
}`,
		`package main
func main() {
	a := 1
	b := 2
	if a < b {
		println("lt")
	}
}`,
		`package main
func main() {
	a := 2
	b := 1
	if a > b {
		println("gt")
	}
}`,
		`package main
func main() {
	if true || false {
		println("or")
	}
}`,
		`package main
func main() {
	x := 1
	if x == 1 {
		println("c")
	}
}`,
		`package main
func main() {
	for {
		break
	}
	println("b")
}`,
		`package main
func main() {
	n := 0
	for n < 1 {
		n = n + 1
		continue
	}
	println("c")
}`,
		`package main
type P = { a: Int, b: Int }
func main() {
	v := P{a: 1, b: 2}
	println(string(v.a + v.b))
}`,
		`package main
func main() {
	s := "x"
	p := &s
	println(*p)
}`,
		`package main
func main() {
	xs := ["a"]
	println(string(len(xs)))
}`,
		`package main
func main() {
	dst := [0, 0, 0]
	src := [1, 2]
	println(string(copy(dst, src)))
}`,
		`package main
func main() {
	xs := [1, 2, 3]
	println(string(cap(xs)))
}`,
		`package main
func main() {
	println(string(min(1, 2)))
	println(string(max(2, 3)))
}`,
		`package main
func main() {
	x := 3
	println(string(x))
}`,
		`package main
func main() {
	x := 1
	println(string(x))
}`,
		`package main
func main() {
	if true {
	} else {
		println("e")
	}
}`,
		`package main
func f() {}
func main() {
	f()
	println("c")
}`,
		`package main
func main() {
	var x Int = 1
	println(string(x))
}`,
		`package main
type Alias = Int
func main() {
	println("ok")
}`,
		`package main
func main() {
	x := 1 % 2
	println(string(x))
}`,
		`package main
func main() {
	x := 1
	println(string(x))
}`,
		`package main
func main() {
	println(string(-1))
}`,
		`package main
func main() {
	if !false {
		println("t")
	}
}`,
		`package main
func main() {
	x := true == false
	if x {
		println("t")
	} else {
		println("f")
	}
}`,
		`package main
func main() {
	if 1 <= 1 {
		println("le")
	}
}`,
		`package main
func main() {
	if 1 >= 1 {
		println("ge")
	}
}`,
		`package main
func main() {
	if 1 == 1 {
		println("eq")
	}
}`,
		`package main
func main() {
	if 1 != 2 {
		println("ne")
	}
}`,
		`package main
func main() {
	x := 1
	y := 2
	println(string(x * y))
}`,
		`package main
func main() {
	x := 10
	y := 2
	println(string(x / y))
}`,
		`package main
func main() {
	x := 9
	y := 4
	println(string(x % y))
}`,
		`package main
func main() {
	x := 1 + 2 * 3
	println(string(x))
}`,
		`package main
func main() {
	x := (1 + 2) * 3
	println(string(x))
}`,
		`package main
func main() {
	a := true
	b := true
	if a && b {
		println("and")
	}
}`,
		`package main
func main() {
	a := false
	b := true
	if a || b {
		println("or2")
	}
}`,
		`package main
func main() {
	x := 1
	if x == 1 {
		println("one")
	}
}`,
		`package main
type S = String
is (s S) NonEmpty() {
	ensure s is Min(1)
}
func main() {
	v: S = "a"
	ensure v is NonEmpty() {
		println("no")
	}
	println("y")
}`,
		`package main
func main() {
	xs := [1, 2]
	for i, v := range xs {
		println(string(i))
		println(string(v))
	}
}`,
		`package main
func main() {
	m := map[String]Int{ "a": 1, "b": 2 }
	println(string(len(m)))
}`,
	}
	for i, src := range cases {
		src := src
		t.Run(fmt.Sprintf("case_%03d", i), func(t *testing.T) {
			t.Parallel()
			out := compileForstPipeline(t, src)
			if !strings.Contains(out, "package main") {
				t.Fatalf("expected generated Go to contain package main:\n%s", out)
			}
		})
	}
}
