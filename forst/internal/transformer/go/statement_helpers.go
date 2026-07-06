package transformergo

import (
	"fmt"

	"forst/internal/ast"
	goast "go/ast"
)

// transformEnsureErrorFallback lowers `ensure … or Bad("msg")` / `or errVar` to a Go expression.
func (t *Transformer) transformEnsureErrorFallback(errorNode ast.EnsureErrorNode) (goast.Expr, error) {
	switch e := errorNode.(type) {
	case ast.EnsureErrorCall:
		if len(e.ErrorArgs) == 1 {
			if def, ok := t.TypeChecker.Defs[ast.TypeIdent(e.ErrorType)].(ast.TypeDefNode); ok {
				if _, ok := def.Expr.(ast.TypeDefErrorExpr); ok {
					if shape, ok := e.ErrorArgs[0].(ast.ShapeNode); ok {
						expected := &ast.TypeNode{Ident: def.Ident}
						return t.transformShapeNodeWithExpectedType(&shape, expected)
					}
				}
			}
		}
		args := make([]goast.Expr, len(e.ErrorArgs))
		for i, arg := range e.ErrorArgs {
			ex, err := t.transformExpression(arg)
			if err != nil {
				return nil, fmt.Errorf("ensure error fallback arg %d: %w", i, err)
			}
			args[i] = ex
		}
		return &goast.CallExpr{
			Fun:  goast.NewIdent(e.ErrorType),
			Args: args,
		}, nil
	case ast.EnsureErrorVar:
		return goast.NewIdent(string(e)), nil
	default:
		return nil, fmt.Errorf("unsupported ensure error fallback: %T", errorNode)
	}
}

func (t *Transformer) defaultAssertionErrorExpr(stmt ast.EnsureNode) goast.Expr {
	t.Output.EnsureImport("errors")
	subjectLabel, assertionLabel, wantHint := t.ensureFailureMessage(stmt)
	msg := fmt.Sprintf("ensure %s is %s: want %s", subjectLabel, assertionLabel, wantHint)
	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   goast.NewIdent("errors"),
			Sel: goast.NewIdent("New"),
		},
		Args: []goast.Expr{
			goQuotedStringLit(msg),
		},
	}
}

// ensureFailureErrorExpr returns the Go expression used on ensure failure (custom or generic).
func (t *Transformer) ensureFailureErrorExpr(stmt ast.EnsureNode) (goast.Expr, error) {
	if stmt.Error != nil {
		return t.transformEnsureErrorFallback(*stmt.Error)
	}
	return t.defaultAssertionErrorExpr(stmt), nil
}

// getAssertionStringForError returns a properly qualified assertion string for error messages
func (t *Transformer) getAssertionStringForError(assertion *ast.AssertionNode) string {
	// If BaseType is set, use it with the constraint name
	if assertion.BaseType != nil {
		return assertion.ToString(assertion.BaseType)
	}

	// Otherwise, try to get the inferred type from the typechecker
	if inferredType, err := t.TypeChecker.LookupInferredType(assertion, false); err == nil && len(inferredType) > 0 {
		// Use the most specific non-hash-based alias for error messages
		nonHash := t.TypeChecker.GetMostSpecificNonHashAlias(inferredType[0])
		return assertion.ToString(&nonHash.Ident)
	}

	// Fallback to the original string representation
	return assertion.String()
}

// findBestNamedTypeForReturnType tries to find a named type that matches the given hash-based type
func (t *Transformer) findBestNamedTypeForReturnType(hashType ast.TypeNode) string {
	// If it's already a named type (not hash-based), return it
	if !hashType.IsHashBased() {
		return string(hashType.Ident)
	}

	// Look through all named types to find one that's compatible
	for typeIdent, def := range t.TypeChecker.Defs {
		if _, ok := def.(ast.TypeDefNode); ok {
			// Skip hash-based types
			if string(typeIdent)[:2] == "T_" {
				continue
			}

			// Check if this named type is compatible with the hash-based type
			if t.TypeChecker.IsTypeCompatible(hashType, ast.TypeNode{Ident: typeIdent}) {
				return string(typeIdent)
			}
		}
	}

	return ""
}

// findBestNamedTypeForReturnStructLiteral finds the best named type for a struct literal in a return statement.
// It always attempts to find a structurally identical named type, even if the expected type is not hash-based.
func (t *Transformer) findBestNamedTypeForReturnStructLiteral(shapeNode ast.ShapeNode, expectedType *ast.TypeNode) *ast.TypeNode {
	// Always try to find a structurally identical named type for the shape
	if namedType := t.TypeChecker.FindAnyStructurallyIdenticalNamedType(shapeNode); namedType != "" {
		return &ast.TypeNode{Ident: namedType}
	}

	// If no structurally identical named type found, use the expected type
	if expectedType != nil {
		return expectedType
	}

	return nil
}

// getShapeNode extracts a *ast.ShapeNode from an ast.Node, handling both value and pointer types
func getShapeNode(value ast.Node) (*ast.ShapeNode, bool) {
	if sn, ok := value.(*ast.ShapeNode); ok {
		return sn, true
	}
	if snVal, ok := value.(ast.ShapeNode); ok {
		return &snVal, true
	}
	return nil, false
}
