package transformergo

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestTransformFunctionLiteral_emitsGoFuncLit(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	n := 1
	f := func(x Int): Int { return x + n }
	func() { println("now") }()
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{})
	if !strings.Contains(out, "func(x int) int") {
		t.Fatalf("expected func literal signature in output, got:\n%s", out)
	}
	if !strings.Contains(out, `func() {
		println("now")
	}()`) && !strings.Contains(out, `func() { println("now") }()`) {
		t.Fatalf("expected IIFE in output, got:\n%s", out)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "out.go", out, 0); err != nil {
		t.Fatalf("generated Go does not parse: %v\n%s", err, out)
	}
}

func TestTransformFunctionLiteral_iifeCall(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	println(func(x Int): Int { return x }(42))
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{})
	if !strings.Contains(out, "func(x int) int") {
		t.Fatalf("expected func literal, got:\n%s", out)
	}
}
