package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseExpression_okErrAndTypedShapeLiteral(t *testing.T) {
	t.Parallel()
	src := `package main

type User = { name: String }

func f(): Result(String, Error) {
	ok := Ok("hi")
	err := Err("boom")
	u := User{ name: "Ann" }
	return ok
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[2], "ast.FunctionNode")
	if len(fn.Body) < 3 {
		t.Fatalf("body len = %d", len(fn.Body))
	}
}

func TestParseExpression_unaryMinusAndNestedParens(t *testing.T) {
	t.Parallel()
	p := NewTestParser(`-(-1)`, ast.SetupTestLogger(nil))
	expr := p.parseExpression()
	if _, ok := expr.(ast.UnaryExpressionNode); !ok {
		t.Fatalf("got %T", expr)
	}
}

func TestParseExpression_okWrongArityErrors(t *testing.T) {
	t.Parallel()
	p := NewTestParser(`Ok()`, ast.SetupTestLogger(nil))
	defer func() {
		if recover() == nil {
			t.Fatal("expected Ok arity error")
		}
	}()
	_ = p.parseExpression()
}

func TestParseExpression_errWrongArityErrors(t *testing.T) {
	t.Parallel()
	p := NewTestParser(`Err(1, 2)`, ast.SetupTestLogger(nil))
	defer func() {
		if recover() == nil {
			t.Fatal("expected Err arity error")
		}
	}()
	_ = p.parseExpression()
}

func TestParseExpression_derefNonValueErrors(t *testing.T) {
	t.Parallel()
	p := NewTestParser(`*f()`, ast.SetupTestLogger(nil))
	defer func() {
		if recover() == nil {
			t.Fatal("expected deref operand error")
		}
	}()
	_ = p.parseExpression()
}

func TestParseExpr_isShapeLiteralBranch(t *testing.T) {
	t.Parallel()
	p := NewTestParser(`x is { n: 1 }`, ast.SetupTestLogger(nil))
	expr := p.parseExpression()
	bin, ok := expr.(ast.BinaryExpressionNode)
	if !ok || bin.Operator != ast.TokenIs {
		t.Fatalf("got %#v", expr)
	}
}

func TestParseExpr_isShapeCallBranch(t *testing.T) {
	t.Parallel()
	p := NewTestParser(`x is Shape({ n: 1 })`, ast.SetupTestLogger(nil))
	expr := p.parseExpression()
	bin, ok := expr.(ast.BinaryExpressionNode)
	if !ok || bin.Operator != ast.TokenIs {
		t.Fatalf("got %#v", expr)
	}
}

func TestParseExpression_isErrGuardPattern(t *testing.T) {
	t.Parallel()
	src := `package main

error ParseError {
	code: Int,
}

type ErrKind = ParseError

func mk(): Result(Int, ErrKind) {
	return 0
}

func demo() {
	x := mk()
	if x is Err(ParseError) {
		return
	}
}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}

func TestParseFunction_receiverWithResultReturn(t *testing.T) {
	t.Parallel()
	src := `package main

type Box = { n: Int }

func (b Box) read(): Result(Int, Error) {
	return Ok(b.n)
}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}

func TestParseEnsure_negatedAndOrError(t *testing.T) {
	t.Parallel()
	src := `package main

func f(x Int, err Error) {
	ensure !x
	ensure err is Error.NotNil() or err
}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}

func TestParseImportGroup_mixedImports(t *testing.T) {
	t.Parallel()
	src := `package main

import (
	"fmt"
	json "encoding/json"
	_ "database/sql"
)

func main() {}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := nodes[1].(ast.ImportGroupNode); !ok {
		t.Fatalf("got %T", nodes[1])
	}
}

func TestParseShapeType_andTypeGuardInFile(t *testing.T) {
	t.Parallel()
	src := `package main

is (n Int) Positive ensure n is GreaterThan(0)

func main() {}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}

func TestParseVarStatement_andMultipleReturnsError(t *testing.T) {
	t.Parallel()
	src := `package main

func f(): Int {
	var n: Int = 1
	return n
}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}

func TestParseTupleType_emptyArgsError(t *testing.T) {
	t.Parallel()
	p := NewTestParser(`Tuple()`, ast.SetupTestLogger(nil))
	defer func() {
		if recover() == nil {
			t.Fatal("expected tuple parse error")
		}
	}()
	_ = p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
}

func TestParseFunction_multiReturnTypeError(t *testing.T) {
	t.Parallel()
	src := `package main

func bad(): (Int, String) {
	return 1
}
`
	if err := parseShouldFail(src); err == nil {
		t.Fatal("expected multi-return error")
	}
}

func TestParseExpression_isComparisonAndCallArgs(t *testing.T) {
	t.Parallel()
	src := `package main

func g(x Int, y Int): Bool {
	return x is GreaterThan(y) && helper(1, 2, 3)
}

func helper(a Int, b Int, c Int): Bool {
	return true
}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}
