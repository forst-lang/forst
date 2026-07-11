package parser

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestParseExpression_parseIdentifierPrimaryBranches(t *testing.T) {
	t.Parallel()
	logger := ast.SetupTestLogger(nil)

	t.Run("selector_chain", func(t *testing.T) {
		t.Parallel()
		p := NewTestParser(`pkg.fn()`, logger)
		expr := p.parseExpression()
		call, ok := expr.(ast.FunctionCallNode)
		if !ok {
			t.Fatalf("got %T", expr)
		}
		if call.Function.ID != "pkg.fn" {
			t.Fatalf("got %#v", call)
		}
	})

	t.Run("index_and_call_chain", func(t *testing.T) {
		t.Parallel()
		p := NewTestParser(`xs[0]()`, logger)
		if _, err := p.ParseFile(); err == nil {
			// expression-only parse may succeed at expr level
		}
		expr := p.parseExpression()
		if expr == nil {
			t.Fatal("nil expr")
		}
	})

	t.Run("nested_expression_depth", func(t *testing.T) {
		t.Parallel()
		var b strings.Builder
		for i := 0; i < 25; i++ {
			b.WriteByte('(')
		}
		b.WriteString("1")
		for i := 0; i < 25; i++ {
			b.WriteByte(')')
		}
		p := NewTestParser(b.String(), logger)
		defer func() {
			if recover() == nil {
				t.Fatal("expected depth error")
			}
		}()
		_ = p.parseExpression()
	})
}

func TestParseFunction_returnShapeType(t *testing.T) {
	t.Parallel()
	src := `package main

func f(): { ok: Bool } {
	return { ok: true }
}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}

func TestParseFor_threeClauseLoop(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
	for i := 0; i < 3; i++ {
		println(i)
	}
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
	loop := assertNodeType[*ast.ForNode](t, fn.Body[0], "*ast.ForNode")
	if loop.Init == nil || loop.Post == nil || loop.Cond == nil {
		t.Fatalf("three-clause for: init=%v post=%v cond=%v", loop.Init, loop.Post, loop.Cond)
	}
}

func TestParseValue_shapeAndArrayLiterals(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
	_ = { x: 1, y: 2 }
	_ = [1, 2, 3]
}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}

func TestParseFile_rejectsTypeConversionInt(t *testing.T) {
	t.Parallel()
	src := `package main

func f(c String): Int {
	return Int(c)
}
`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error for Int(c)")
	}
	if !strings.Contains(err.Error(), "not a conversion") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseFile_postfixMethodCallOnExpression(t *testing.T) {
	t.Parallel()
	src := `package main

import "time"

func f(): String {
	return time.Now().Format("20060102")
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[2], "ast.FunctionNode")
	ret := fn.Body[0].(ast.ReturnNode)
	mc, ok := ret.Values[0].(ast.MethodCallNode)
	if !ok || mc.Method.ID != "Format" {
		t.Fatalf("want Format method call, got %#v", ret.Values[0])
	}
}
