package parser

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
)

// String comparisons combined with && must parse as (a != x) && (a != y), not a != (x && a) != y.
// Regression: flat binary parsing made the RHS of != absorb a following && chain.
func TestParseExpression_logicalAndBindsLooserThanCompare(t *testing.T) {
	src := `package main
func main() {
  s := "a"
  if s != "X" && s != "O" {
    println("ok")
  }
}`
	logger := ast.SetupTestLogger(nil)
	toks := lexer.New([]byte(src), "t.ft", logger).Lex()
	p := New(toks, "t.ft", logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := findMainFunction(t, nodes)
	ifStmt := firstIfInMain(t, fn)
	cond := ifStmt.Condition.(ast.BinaryExpressionNode)
	if cond.Operator != ast.TokenLogicalAnd {
		t.Fatalf("top of if condition should be &&, got %v", cond.Operator)
	}
	left := cond.Left.(ast.BinaryExpressionNode)
	right := cond.Right.(ast.BinaryExpressionNode)
	if left.Operator != ast.TokenNotEquals || right.Operator != ast.TokenNotEquals {
		t.Fatalf("expected (s != \"X\") && (s != \"O\"), got left op %v right op %v", left.Operator, right.Operator)
	}
}

func TestParseExpression_compareChainsWithAndInLineWinner(t *testing.T) {
	src := `package main
func f(x0 String, b String, c String): Bool {
  if x0 == b && x0 == c {
    return true
  }
  return false
}`
	logger := ast.SetupTestLogger(nil)
	toks := lexer.New([]byte(src), "t.ft", logger).Lex()
	p := New(toks, "t.ft", logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := findFunction(t, nodes, "f")
	ifStmt := fn.Body[0].(*ast.IfNode)
	cond := ifStmt.Condition.(ast.BinaryExpressionNode)
	if cond.Operator != ast.TokenLogicalAnd {
		t.Fatalf("top should be &&, got %v", cond.Operator)
	}
	l := cond.Left.(ast.BinaryExpressionNode)
	r := cond.Right.(ast.BinaryExpressionNode)
	if l.Operator != ast.TokenEquals || r.Operator != ast.TokenEquals {
		t.Fatalf("expected (x0 == b) && (x0 == c)")
	}
}

func findMainFunction(t *testing.T, nodes []ast.Node) ast.FunctionNode {
	t.Helper()
	return findFunction(t, nodes, "main")
}

func findFunction(t *testing.T, nodes []ast.Node, name string) ast.FunctionNode {
	t.Helper()
	for _, n := range nodes {
		if fn, ok := n.(ast.FunctionNode); ok && string(fn.Ident.ID) == name {
			return fn
		}
	}
	t.Fatalf("function %q not found", name)
	return ast.FunctionNode{}
}

func firstIfInMain(t *testing.T, fn ast.FunctionNode) *ast.IfNode {
	t.Helper()
	for _, st := range fn.Body {
		if ifn, ok := st.(*ast.IfNode); ok {
			return ifn
		}
	}
	t.Fatal("no if in main")
	return nil
}
