package transformergo

import (
	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
)

func testingTGoTypeExpr() goast.Expr {
	return &goast.StarExpr{
		X: &goast.SelectorExpr{
			X:   goast.NewIdent("testing"),
			Sel: goast.NewIdent("T"),
		},
	}
}

func (t *Transformer) isGoTestFunction(fn ast.FunctionNode) bool {
	return t.TypeChecker.IsGoTestFunction(fn)
}

func (t *Transformer) testingTParamIdent(fn ast.FunctionNode) (ast.Identifier, bool) {
	if id, ok := ast.TestingTParamIdent(fn); ok {
		return id, true
	}
	if sig, ok := t.TypeChecker.Functions[fn.Ident.ID]; ok {
		for _, p := range fn.Params {
			sp, ok := p.(ast.SimpleParamNode)
			if !ok {
				continue
			}
			for _, ps := range sig.Parameters {
				if ps.Ident.ID != sp.Ident.ID {
					continue
				}
				if ast.IsTestingTParamType(ps.Type) {
					return sp.Ident.ID, true
				}
				if typechecker.IsGoTypesTestingT(t.TypeChecker.GoTypeForVariable(sp.Ident.ID)) {
					return sp.Ident.ID, true
				}
			}
		}
	}
	for _, p := range fn.Params {
		sp, ok := p.(ast.SimpleParamNode)
		if !ok {
			continue
		}
		if typechecker.IsGoTypesTestingT(t.TypeChecker.GoTypeForVariable(sp.Ident.ID)) {
			return sp.Ident.ID, true
		}
	}
	return "", false
}
