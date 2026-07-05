package transformergo

import (
	"fmt"

	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

func (t *Transformer) transformPackageVarDecl(n ast.AssignmentNode) (*goast.GenDecl, error) {
	if len(n.LValues) != 1 || len(n.RValues) != 1 {
		return nil, fmt.Errorf("package var expects one name and one value")
	}
	vn, ok := n.LValues[0].(ast.VariableNode)
	if !ok {
		return nil, fmt.Errorf("package var name must be a simple identifier")
	}
	val, err := t.transformExpression(n.RValues[0])
	if err != nil {
		return nil, err
	}
	spec := &goast.ValueSpec{
		Names:  []*goast.Ident{goast.NewIdent(string(vn.Ident.ID))},
		Values: []goast.Expr{val},
	}
	if len(n.ExplicitTypes) > 0 && n.ExplicitTypes[0] != nil {
		typeExpr, err := t.transformType(*n.ExplicitTypes[0])
		if err != nil {
			return nil, err
		}
		spec.Type = typeExpr
	}
	return &goast.GenDecl{Tok: token.VAR, Specs: []goast.Spec{spec}}, nil
}
