package transformergo

import (
	"forst/internal/ast"
	goast "go/ast"
)

// coerceGoStringAliasExpr wraps named string aliases with Go string() for stdlib string APIs.
func (t *Transformer) coerceGoStringAliasExpr(expr goast.Expr, node ast.Node) goast.Expr {
	types, err := t.TypeChecker.LookupInferredType(node, false)
	if err != nil || len(types) != 1 {
		return expr
	}
	return t.coerceGoStringAliasExprForType(expr, types[0])
}

func (t *Transformer) typeIsStringAlias(ident ast.TypeIdent) bool {
	if ident == ast.TypeString {
		return true
	}
	return t.TypeChecker.UnderlyingBuiltinTypeOfAliasAssertion(ident) == ast.TypeString
}

func (t *Transformer) coerceGoStringAliasExprForType(expr goast.Expr, tn ast.TypeNode) goast.Expr {
	if tn.Ident == ast.TypeString {
		return expr
	}
	if t.typeIsStringAlias(tn.Ident) {
		return &goast.CallExpr{Fun: goast.NewIdent("string"), Args: []goast.Expr{expr}}
	}
	if tn.Assertion != nil && tn.Assertion.BaseType != nil {
		if t.typeIsStringAlias(*tn.Assertion.BaseType) {
			return &goast.CallExpr{Fun: goast.NewIdent("string"), Args: []goast.Expr{expr}}
		}
	}
	return expr
}

func (at *AssertionTransformer) transformStringBuiltinVariable(variable ast.VariableNode) (goast.Expr, error) {
	expr, err := at.transformer.transformExpression(variable)
	if err != nil {
		return nil, err
	}
	return at.transformer.coerceGoStringAliasExpr(expr, variable), nil
}
