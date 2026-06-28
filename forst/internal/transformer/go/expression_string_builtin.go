package transformergo

import (
	"forst/internal/ast"
	goast "go/ast"
)

// transformStringBuiltinCall lowers Forst string() for types Go's predeclared string() does not handle.
// Returns ok=false when the call should use default lowering (e.g. string(int) rune conversion).
func (t *Transformer) transformStringBuiltinCall(arg ast.ExpressionNode) (goast.Expr, bool, error) {
	ts, err := t.TypeChecker.LookupInferredType(arg, false)
	if err != nil || len(ts) != 1 {
		return nil, false, nil
	}
	switch ts[0].Ident {
	case ast.TypeBool:
		argExpr, err := t.transformExpression(arg)
		if err != nil {
			return nil, true, err
		}
		t.Output.EnsureImport("strconv")
		return &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   goast.NewIdent("strconv"),
				Sel: goast.NewIdent("FormatBool"),
			},
			Args: []goast.Expr{argExpr},
		}, true, nil
	default:
		return nil, false, nil
	}
}
