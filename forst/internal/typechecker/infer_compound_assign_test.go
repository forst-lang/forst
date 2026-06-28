package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestInferAssignmentTypes_rejectsCompoundShortDecl(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	chk := New(log, false)
	assign := ast.AssignmentNode{
		LValues:    []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "n"}}},
		RValues:    []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		IsShort:    true,
		CompoundOp: ast.TokenPlusEq,
	}
	if err := chk.inferAssignmentTypes(assign); err == nil {
		t.Fatal("expected error for compound assign with :=")
	}
}
