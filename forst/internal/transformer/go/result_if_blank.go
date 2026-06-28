package transformergo

import (
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

// blankUnusedResultSuccessForIfCondition emits `_ = x` when an `if x is Err(...)` branch may not use the success slot.
func (t *Transformer) blankUnusedResultSuccessForIfCondition(cond ast.ExpressionNode) goast.Stmt {
	bin, ok := cond.(ast.BinaryExpressionNode)
	if !ok || bin.Operator != ast.TokenIs {
		return nil
	}
	vn, ok := bin.Left.(ast.VariableNode)
	if !ok || isDotQualifiedVariable(vn) {
		return nil
	}
	asn, ok := bin.Right.(ast.AssertionNode)
	if !ok {
		return nil
	}
	hasErr := false
	for _, c := range asn.Constraints {
		if c.Name == "Err" {
			hasErr = true
			break
		}
	}
	if !hasErr {
		return nil
	}
	name := string(vn.Ident.ID)
	if t.resultLocalSplit == nil {
		return nil
	}
	split, ok := t.resultLocalSplit[name]
	if !ok || len(split.successGoNames) == 0 {
		return nil
	}
	return &goast.AssignStmt{
		Lhs: []goast.Expr{goast.NewIdent("_")},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{goast.NewIdent(split.successGoNames[0])},
	}
}
