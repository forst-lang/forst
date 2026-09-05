package typechecker

import (
	"forst/internal/ast"
)

// inferReturnValueTypes returns types for a return expression, preferring types inferred
// during the body pass (while lexical scopes for loops, if branches, etc. were active).
func (tc *TypeChecker) inferReturnValueTypes(value ast.ExpressionNode) ([]ast.TypeNode, error) {
	if cached, err := tc.LookupInferredType(value, false); err != nil {
		return nil, err
	} else if cached != nil {
		return cached, nil
	}
	return tc.inferExpressionType(value)
}

// functionEnsureImpliesResultReturn reports whether ensure statements in fn should promote the
// function's inferred return to Result(S, Error). Pure Result discriminators (`ensure x is Ok()` /
// `Err()` on a Result binding) only narrow locals and do not change the function return type.
func (tc *TypeChecker) functionEnsureImpliesResultReturn(fn ast.FunctionNode) bool {
	for _, stmt := range fn.Body {
		if stmt.Kind() != ast.NodeKindEnsure {
			continue
		}
		ensureNode, ok := stmt.(ast.EnsureNode)
		if !ok {
			if ptr, ok := stmt.(*ast.EnsureNode); ok && ptr != nil {
				ensureNode = *ptr
			} else {
				return true
			}
		}
		if ensureLooksLikeResultDiscriminator(ensureNode) {
			continue
		}
		return true
	}
	return false
}

// ensureLooksLikeResultDiscriminator reports `ensure x is Ok()` / `Err()` shape (narrowing only).
func ensureLooksLikeResultDiscriminator(n ast.EnsureNode) bool {
	if n.Error != nil || n.Assertion.BaseType != nil || len(n.Assertion.Constraints) != 1 {
		return false
	}
	c := n.Assertion.Constraints[0].Name
	return c == "Ok" || c == "Err"
}

// Helper: isNilableType checks if a type can be assigned nil
func isNilableType(tc *TypeChecker, t ast.TypeNode) bool {
	base := t
	chain := tc.GetTypeAliasChain(t)
	if len(chain) > 0 {
		base = chain[len(chain)-1]
	}

	switch base.Ident {
	case ast.TypePointer, ast.TypeError, ast.TypeMap, ast.TypeArray:
		return true
	}

	switch string(base.Ident) {
	case "Pointer", "Error", "Map", "Array":
		return true
	}

	return false
}
