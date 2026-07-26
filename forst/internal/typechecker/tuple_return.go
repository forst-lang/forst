package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// checkMultiValueReturnLegality validates `return a, b, ...` against the enclosing function signature.
func (tc *TypeChecker) checkMultiValueReturnLegality(ret ast.ReturnNode) error {
	if len(ret.Values) <= 1 {
		return nil
	}
	if tc.currentFunction == nil {
		return fmt.Errorf("multi-value return outside function body")
	}
	parsed := tc.currentFunction.ReturnTypes
	if len(parsed) == 1 && parsed[0].IsTupleType() {
		tup := parsed[0]
		if len(ret.Values) != len(tup.TypeParams) {
			return fmt.Errorf("return arity %d does not match Tuple return type %s", len(ret.Values), tup.String())
		}
		return nil
	}
	if len(parsed) == 1 && parsed[0].IsResultType() {
		return fmt.Errorf("multi-value return is not allowed for Result-declared functions; use a single success value")
	}
	if len(parsed) == len(ret.Values) {
		return nil
	}
	return fmt.Errorf("multiple return values are only supported for Tuple-declared return types")
}
