package transformergo

import (
	"testing"

	"forst/internal/ast"
	goast "go/ast"
)

func TestBlankUnusedResultSuccess_emitsAssignBlank(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	tr.resultLocalSplit = map[string]resultLocalSplit{
		"x": {successGoNames: []string{"xOk"}},
	}
	cond := ast.BinaryExpressionNode{
		Left:     ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		Operator: ast.TokenIs,
		Right: ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{Name: "Err"}},
		},
	}
	stmt := tr.blankUnusedResultSuccessForIfCondition(cond)
	if stmt == nil {
		t.Fatal("expected blank assign stmt")
	}
	assign, ok := stmt.(*goast.AssignStmt)
	if !ok || len(assign.Rhs) != 1 {
		t.Fatalf("stmt = %#v", stmt)
	}
	if id, ok := assign.Rhs[0].(*goast.Ident); !ok || id.Name != "xOk" {
		t.Fatalf("rhs = %#v", assign.Rhs[0])
	}
	if id, ok := assign.Lhs[0].(*goast.Ident); !ok || id.Name != "_" {
		t.Fatalf("lhs = %#v", assign.Lhs[0])
	}
}

func TestBlankUnusedResultSuccess_skipsOkBinding(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	tr.resultLocalSplit = map[string]resultLocalSplit{
		"x": {successGoNames: []string{"xOk"}},
	}
	cond := ast.BinaryExpressionNode{
		Left:     ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		Operator: ast.TokenIs,
		Right: ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{Name: "Ok"}},
		},
	}
	if stmt := tr.blankUnusedResultSuccessForIfCondition(cond); stmt != nil {
		t.Fatalf("expected nil, got %#v", stmt)
	}
}

func TestBlankUnusedResultSuccess_skipsDotQualified(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	tr.resultLocalSplit = map[string]resultLocalSplit{
		"w.r": {successGoNames: []string{"wROk"}},
	}
	cond := ast.BinaryExpressionNode{
		Left:     ast.VariableNode{Ident: ast.Ident{ID: "w.r"}},
		Operator: ast.TokenIs,
		Right: ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{Name: "Err"}},
		},
	}
	if stmt := tr.blankUnusedResultSuccessForIfCondition(cond); stmt != nil {
		t.Fatalf("expected nil, got %#v", stmt)
	}
}

func TestBlankUnusedResultSuccess_skipsWhenSplitMissing(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	cond := ast.BinaryExpressionNode{
		Left:     ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		Operator: ast.TokenIs,
		Right: ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{Name: "Err"}},
		},
	}
	if stmt := tr.blankUnusedResultSuccessForIfCondition(cond); stmt != nil {
		t.Fatalf("expected nil, got %#v", stmt)
	}
}
