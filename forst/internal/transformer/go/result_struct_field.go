package transformergo

import (
	"errors"
	"fmt"

	"forst/internal/ast"
	goast "go/ast"
)

var errNotLoweredResultStructField = errors.New("transform: not a lowered Result(Error) struct field")

// compoundResultFieldStorageSelector returns the Go expression for the lowered struct value
// (e.g. w.r) when vn is w.r and the field stores Result(S, Error).
func (t *Transformer) compoundResultFieldStorageSelector(vn ast.VariableNode) (goast.Expr, bool) {
	if !t.compoundVarDeclaresResultField(vn) {
		return nil, false
	}
	parts := variablePathSegments(vn)
	var sel goast.Expr = goast.NewIdent(parts[0])
	for _, field := range parts[1:] {
		fn := field
		if t.ExportReturnStructFields {
			fn = capitalizeFirst(field)
		}
		sel = &goast.SelectorExpr{X: sel, Sel: goast.NewIdent(fn)}
	}
	return sel, true
}

// compoundVarDeclaresResultField reports whether vn is base.field and the shape definition
// gives that field type Result(S,F) with F = Error (struct-field lowering).
func (t *Transformer) compoundVarDeclaresResultField(vn ast.VariableNode) bool {
	parts := variablePathSegments(vn)
	if len(parts) < 2 {
		return false
	}
	wt, ok := t.TypeChecker.VariableTypes[ast.Identifier(parts[0])]
	if !ok || len(wt) != 1 {
		return false
	}
	ft, ok := t.TypeChecker.FieldTypeForNamedShape(wt[0].Ident, parts[1])
	if !ok || !ft.IsResultType() || len(ft.TypeParams) < 2 {
		return false
	}
	return ft.TypeParams[1].Ident == ast.TypeError
}

// appendResultStoragePayloadSelector appends .V or .Err to a selector that already points at the
// lowered struct field (e.g. w.r) when the Forst occurrence type is a narrowed Result payload.
func (t *Transformer) appendResultStoragePayloadSelector(vn ast.VariableNode, baseStructSel goast.Expr) goast.Expr {
	if !t.compoundVarDeclaresResultField(vn) {
		return baseStructSel
	}
	parts := variablePathSegments(vn)
	ft, _ := t.TypeChecker.FieldTypeForNamedShape(
		t.TypeChecker.VariableTypes[ast.Identifier(parts[0])][0].Ident,
		parts[1],
	)
	occ, ok := t.TypeChecker.InferredTypesForVariableNode(vn)
	if !ok || len(occ) != 1 {
		return baseStructSel
	}
	if occ[0].IsResultType() {
		return baseStructSel
	}
	succ := ft.TypeParams[0]
	fail := ft.TypeParams[1]
	if t.TypeChecker.IsTypeCompatible(occ[0], succ) {
		return goLoweredResultValueSelector(baseStructSel)
	}
	if t.TypeChecker.IsTypeCompatible(occ[0], fail) {
		return goLoweredResultErrSelector(baseStructSel)
	}
	return baseStructSel
}

// goResultErrIdentForCompoundField returns w.r.Err for subject w.r when the field stores lowered Result.
func (t *Transformer) goResultErrIdentForCompoundField(vn ast.VariableNode) (goast.Expr, error) {
	sel, ok := t.compoundResultFieldStorageSelector(vn)
	if !ok {
		return nil, errNotLoweredResultStructField
	}
	return goLoweredResultErrSelector(sel), nil
}

// goResultSuccessValueExprForOkDiscriminator returns the Go expression for the success payload in
// `left is Ok(...)` when left is w.r (struct-stored Result) or a plain split local x.
func (t *Transformer) goResultSuccessValueExprForOkDiscriminator(left ast.ExpressionNode) (goast.Expr, error) {
	vn, ok := left.(ast.VariableNode)
	if !ok {
		return nil, fmt.Errorf("if-is: Ok value compare requires variable subject")
	}
	parts := variablePathSegments(vn)
	if len(parts) == 1 {
		return t.transformExpression(left)
	}
	if t.compoundVarDeclaresResultField(vn) {
		sel, ok := t.compoundResultFieldStorageSelector(vn)
		if !ok {
			return nil, errNotLoweredResultStructField
		}
		return goLoweredResultValueSelector(sel), nil
	}
	return nil, fmt.Errorf("if-is: Result Ok/Err requires a simple variable or struct field subject")
}
