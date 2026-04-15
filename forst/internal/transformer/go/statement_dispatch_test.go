package transformergo

import (
	"go/token"
	"strings"
	"testing"

	"forst/internal/ast"
	goast "go/ast"
)

type unsupportedStatementNode struct{}

func (unsupportedStatementNode) Kind() ast.NodeKind { return ast.NodeKind("UnsupportedStatementNode") }
func (unsupportedStatementNode) String() string     { return "UnsupportedStatementNode" }

func TestTransformStatement_commentReturnsEmptyStmt(t *testing.T) {
	t.Parallel()
	transformer := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))

	statement, err := transformer.transformStatement(ast.CommentNode{Text: "noop"})
	if err != nil {
		t.Fatalf("transformStatement(comment): %v", err)
	}
	if _, ok := statement.(*goast.EmptyStmt); !ok {
		t.Fatalf("expected *goast.EmptyStmt, got %T", statement)
	}
}

func TestTransformStatement_unknownNodeFallsBackToEmptyStmt(t *testing.T) {
	t.Parallel()
	transformer := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))

	statement, err := transformer.transformStatement(unsupportedStatementNode{})
	if err != nil {
		t.Fatalf("transformStatement(unsupported): %v", err)
	}
	if _, ok := statement.(*goast.EmptyStmt); !ok {
		t.Fatalf("expected *goast.EmptyStmt fallback, got %T", statement)
	}
}

func TestTransformStatement_deferAndGoRequireCallExprAfterTransform(t *testing.T) {
	t.Parallel()
	transformer := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))

	_, err := transformer.transformStatement(&ast.DeferNode{Call: ast.IntLiteralNode{Value: 1}})
	if err == nil || !strings.Contains(err.Error(), "defer: internal error, expected call expression") {
		t.Fatalf("expected defer call-expression error, got %v", err)
	}

	_, err = transformer.transformStatement(&ast.GoStmtNode{Call: ast.IntLiteralNode{Value: 1}})
	if err == nil || !strings.Contains(err.Error(), "go: internal error, expected call expression") {
		t.Fatalf("expected go call-expression error, got %v", err)
	}
}

func TestTransformStatement_breakAndContinuePreserveLabels(t *testing.T) {
	t.Parallel()
	transformer := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))

	breakStatement, err := transformer.transformStatement(&ast.BreakNode{Label: &ast.Ident{ID: "outer"}})
	if err != nil {
		t.Fatalf("transformStatement(break): %v", err)
	}
	breakBranch, ok := breakStatement.(*goast.BranchStmt)
	if !ok || breakBranch.Tok != token.BREAK || breakBranch.Label == nil || breakBranch.Label.Name != "outer" {
		t.Fatalf("unexpected break stmt: %#v", breakStatement)
	}

	continueStatement, err := transformer.transformStatement(&ast.ContinueNode{Label: &ast.Ident{ID: "outer"}})
	if err != nil {
		t.Fatalf("transformStatement(continue): %v", err)
	}
	continueBranch, ok := continueStatement.(*goast.BranchStmt)
	if !ok || continueBranch.Tok != token.CONTINUE || continueBranch.Label == nil || continueBranch.Label.Name != "outer" {
		t.Fatalf("unexpected continue stmt: %#v", continueStatement)
	}
}

func TestTransformStatement_assignmentExplicitTypeRequiresVariableLHS(t *testing.T) {
	t.Parallel()
	transformer := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))

	assignment := ast.AssignmentNode{
		LValues: []ast.ExpressionNode{
			ast.IndexExpressionNode{
				Target: ast.VariableNode{Ident: ast.Ident{ID: "values"}},
				Index:  ast.IntLiteralNode{Value: 0},
			},
		},
		RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		ExplicitTypes: []*ast.TypeNode{
			{Ident: ast.TypeInt},
		},
	}

	_, err := transformer.transformStatement(assignment)
	if err == nil || !strings.Contains(err.Error(), "explicit type requires a simple variable on the left") {
		t.Fatalf("expected explicit type LHS validation error, got %v", err)
	}
}
