package transformergo

import (
	"fmt"

	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

func (t *Transformer) transformConstGroup(n ast.ConstGroupNode) (*goast.GenDecl, error) {
	specs := make([]goast.Spec, 0, len(n.Specs))
	for _, spec := range n.Specs {
		valueSpec := &goast.ValueSpec{
			Names: []*goast.Ident{goast.NewIdent(string(spec.Name.ID))},
		}
		if spec.Type != nil {
			typeExpr, err := t.transformType(*spec.Type)
			if err != nil {
				return nil, err
			}
			valueSpec.Type = typeExpr
		}
		if spec.Value != nil {
			val, err := t.transformExpression(spec.Value)
			if err != nil {
				return nil, err
			}
			valueSpec.Values = []goast.Expr{val}
		}
		specs = append(specs, valueSpec)
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("const group has no specs")
	}
	return &goast.GenDecl{Tok: token.CONST, Specs: specs}, nil
}
