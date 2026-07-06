package transformergo

import (
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

// coerceGoStringConcatOperands wraps named string-alias operands with Go string() for + with untyped strings.
func (t *Transformer) coerceGoStringConcatOperands(left, right ast.ExpressionNode, leftGo, rightGo goast.Expr) (goast.Expr, goast.Expr) {
	lt, errL := t.TypeChecker.LookupInferredType(left, false)
	rt, errR := t.TypeChecker.LookupInferredType(right, false)
	if errL != nil || errR != nil || len(lt) != 1 || len(rt) != 1 {
		return leftGo, rightGo
	}
	stringify := func(expr goast.Expr, tn ast.TypeNode) goast.Expr {
		return t.coerceGoStringAliasExprForType(expr, tn)
	}
	if lt[0].Ident == ast.TypeString || rt[0].Ident == ast.TypeString {
		leftGo = stringify(leftGo, lt[0])
		rightGo = stringify(rightGo, rt[0])
	}
	return leftGo, rightGo
}

// coerceGoStringPlusAlias ensures `"text" + alias` lowers to Go string concatenation (not typed alias +).
func (t *Transformer) coerceGoStringPlusAlias(left, right ast.ExpressionNode, leftGo, rightGo goast.Expr) (goast.Expr, goast.Expr) {
	if bl, ok := rightGo.(*goast.BasicLit); ok && bl.Kind == token.STRING {
		if lt, err := t.TypeChecker.LookupInferredType(left, false); err == nil && len(lt) == 1 {
			leftGo = t.coerceGoStringAliasExprForType(leftGo, lt[0])
		}
	}
	if bl, ok := leftGo.(*goast.BasicLit); ok && bl.Kind == token.STRING {
		if rt, err := t.TypeChecker.LookupInferredType(right, false); err == nil && len(rt) == 1 {
			rightGo = t.coerceGoStringAliasExprForType(rightGo, rt[0])
		}
	}
	return leftGo, rightGo
}
