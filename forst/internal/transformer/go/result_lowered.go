package transformergo

import (
	"strings"

	"forst/internal/ast"
	goast "go/ast"
)

// Lowered Result storage (shape fields and parallel local bindings) uses a fixed Go struct layout:
// success payload in field V, failure/error slot in field Err. These names must stay consistent
// across type emission, composite literals, selectors, and print/ensure lowering.
const (
	loweredResultValueFieldName = "V"
	loweredResultErrFieldName   = "Err"
)

// variablePathSegments splits a Forst variable path ("w.r", "a.b") or a simple name ("x").
func variablePathSegments(vn ast.VariableNode) []string {
	return strings.Split(string(vn.Ident.ID), ".")
}

// isDotQualifiedVariable is true when the variable refers to a field path (contains '.').
func isDotQualifiedVariable(vn ast.VariableNode) bool {
	return strings.Contains(string(vn.Ident.ID), ".")
}

func goLoweredResultValueSelector(base goast.Expr) *goast.SelectorExpr {
	return &goast.SelectorExpr{X: base, Sel: goast.NewIdent(loweredResultValueFieldName)}
}

func goLoweredResultErrSelector(base goast.Expr) *goast.SelectorExpr {
	return &goast.SelectorExpr{X: base, Sel: goast.NewIdent(loweredResultErrFieldName)}
}
