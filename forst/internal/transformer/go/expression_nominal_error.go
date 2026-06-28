package transformergo

import (
	"fmt"

	"forst/internal/ast"
	goast "go/ast"
)

// transformNominalErrorConstructorCall lowers `E({ field: value })` to a Go composite literal.
func (t *Transformer) transformNominalErrorConstructorCall(e ast.FunctionCallNode) (goast.Expr, bool, error) {
	def, ok := t.TypeChecker.Defs[ast.TypeIdent(e.Function.ID)].(ast.TypeDefNode)
	if !ok {
		return nil, false, nil
	}
	if _, ok := def.Expr.(ast.TypeDefErrorExpr); !ok {
		return nil, false, nil
	}
	if len(e.Arguments) != 1 {
		return nil, true, fmt.Errorf("%s(...) expects one payload argument", e.Function.ID)
	}
	shape, ok := e.Arguments[0].(ast.ShapeNode)
	if !ok {
		return nil, true, fmt.Errorf("%s(...) argument must be a shape literal", e.Function.ID)
	}
	expected := &ast.TypeNode{Ident: def.Ident}
	lit, err := t.transformShapeNodeWithExpectedType(&shape, expected)
	if err != nil {
		return nil, true, err
	}
	return lit, true, nil
}
