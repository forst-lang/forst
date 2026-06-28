package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// inferNominalErrorConstructorCall types `NotFound({ id: "x" })` as a nominal error value.
func (tc *TypeChecker) inferNominalErrorConstructorCall(e ast.FunctionCallNode, argTypes [][]ast.TypeNode) ([]ast.TypeNode, bool, error) {
	def, ok := tc.Defs[ast.TypeIdent(e.Function.ID)].(ast.TypeDefNode)
	if !ok {
		return nil, false, nil
	}
	errEx, ok := def.Expr.(ast.TypeDefErrorExpr)
	if !ok {
		return nil, false, nil
	}
	if len(e.Arguments) != 1 {
		return nil, true, diagnosticf(e.CallSpan, "call-arity", "%s(...) expects one payload argument", e.Function.ID)
	}
	shape, ok := e.Arguments[0].(ast.ShapeNode)
	if !ok {
		return nil, true, diagnosticf(e.CallSpan, "call-type", "%s(...) argument must be a shape literal", e.Function.ID)
	}
	payload := errEx.Payload
	inferred, err := tc.inferShapeType(shape, &ast.TypeNode{Ident: def.Ident})
	if err != nil {
		return nil, true, err
	}
	if !tc.shapesHaveSameStructure(payload, shape) && !tc.IsTypeCompatible(inferred, ast.TypeNode{Ident: def.Ident}) {
		return nil, true, fmt.Errorf("%s payload does not match %s", e.Function.ID, def.Ident)
	}
	_ = argTypes
	return []ast.TypeNode{{Ident: def.Ident}}, true, nil
}
