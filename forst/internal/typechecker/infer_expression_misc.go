package typechecker

import (
	"forst/internal/ast"
)

func (tc *TypeChecker) inferExpressionMisc(expr ast.Node) ([]ast.TypeNode, bool, error) {
	switch e := expr.(type) {
	case ast.FunctionLiteralNode:

		ret, err := tc.inferFunctionLiteral(e, expr)
		if err != nil {
			return nil, true, err
		}
		tc.storeInferredType(e, ret)
		return ret, true, nil
	case ast.TypeExpressionNode:

		tc.storeInferredType(e, []ast.TypeNode{e.Type})
		return []ast.TypeNode{e.Type}, true, nil
	case ast.OkExprNode:

		sp := e.Span
		if !sp.IsSet() {
			sp = spanOfExpression(e.Value)
		}
		return nil, true, reportf(sp, "result-value-constructor",
			"Ok(...) is not a value constructor",
			"`Ok(...)` is a Result discriminant for `is` / `ensure`, not a runtime value constructor.",
			"use `if r is Ok()` / `ensure r is Ok()`, or return a plain success value of type S for Result(S, F)")
	case ast.ErrExprNode:

		sp := e.Span
		if !sp.IsSet() {
			sp = spanOfExpression(e.Value)
		}
		return nil, true, reportf(sp, "result-value-constructor",
			"Err(...) is not a value constructor",
			"`Err(...)` is a Result discriminant for `is` / `ensure`, not a runtime value constructor.",
			"use `is Err()` / `ensure ... is Err()` and FFI/interop for failure values")
	}
	return nil, false, nil
}
