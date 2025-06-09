package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	gotoken "go/token"
)

// transformEnsureConstraints transforms an ensure node based on its type
func (at *AssertionTransformer) transformEnsureConstraints(ensure ast.EnsureNode) (goast.Expr, error) {
	baseType, err := at.transformer.getEnsureBaseType(ensure)
	if err != nil {
		return nil, err
	}
	transformed, err := at.TransformBuiltinConstraint(baseType.Ident, ensure.Variable, ensure.Assertion.Constraints[0])
	if err != nil {
		return nil, fmt.Errorf("failed to transform ensure conditions constraints: %w", err)
	}
	return transformed, nil
}

// transformEnsureConstraint transforms a single constraint in an ensure statement
func (t *Transformer) transformEnsureConstraint(ensure ast.EnsureNode, constraint ast.ConstraintNode, varType ast.TypeNode) (goast.Expr, error) {
	// Try built-in constraints first
	if transformed, err := t.assertionTransformer.TransformBuiltinConstraint(varType.Ident, ensure.Variable, constraint); err == nil {
		return transformed, nil
	}

	// Try type guards
	typeGuardDef, err := t.lookupTypeGuardNode(constraint.Name)
	if err == nil && typeGuardDef != nil && t.isTypeGuardCompatible(varType, typeGuardDef) {
		hash, err := t.TypeChecker.Hasher.HashNode(*typeGuardDef)
		if err != nil {
			return nil, fmt.Errorf("failed to hash type guard node: %w", err)
		}
		guardFuncName := hash.ToGuardIdent()
		expr, err := t.transformExpression(ensure.Variable)
		if err != nil {
			return nil, fmt.Errorf("failed to transform expression: %w", err)
		}
		args := []goast.Expr{expr}
		for _, arg := range constraint.Args {
			argExpr, err := transformConstraintArg(arg)
			if err != nil {
				return nil, err
			}
			args = append(args, argExpr)
		}
		callExpr := &goast.CallExpr{
			Fun:  goast.NewIdent(string(guardFuncName)),
			Args: args,
		}
		return &goast.UnaryExpr{Op: gotoken.NOT, X: callExpr}, nil
	}

	// Try all base types in the alias chain
	aliasChain := t.getTypeAliasChain(varType)
	t.log.WithFields(map[string]interface{}{
		"aliasChain": aliasChain,
		"function":   "transformEnsureConstraint",
	}).Debug("Alias chain for type alias")
	for _, baseType := range aliasChain[1:] {
		t.log.WithFields(map[string]interface{}{
			"baseType":   baseType.Ident,
			"constraint": constraint.Name,
			"function":   "transformEnsureConstraint",
		}).Debug("Trying type guard lookup for base type in alias chain")
		guard, err := t.lookupTypeGuardNode(constraint.Name)
		if err == nil && guard != nil && t.isTypeGuardCompatible(baseType, guard) {
			expr, err := t.transformExpression(ensure.Variable)
			if err != nil {
				return nil, fmt.Errorf("failed to transform expression: %w", err)
			}
			hash, err := t.TypeChecker.Hasher.HashNode(*guard)
			if err != nil {
				return nil, fmt.Errorf("failed to hash type guard node: %w", err)
			}
			guardFuncName := hash.ToGuardIdent()
			callExpr := &goast.CallExpr{
				Fun:  goast.NewIdent(string(guardFuncName)),
				Args: []goast.Expr{expr},
			}
			return &goast.UnaryExpr{Op: gotoken.NOT, X: callExpr}, nil
		}
		// Retry built-in constraint for base type
		t.log.WithFields(map[string]interface{}{
			"baseType":   baseType.Ident,
			"constraint": constraint.Name,
			"function":   "transformEnsureConstraint",
		}).Debug("Retrying built-in constraint for base type in alias chain")
		typeIdent := ast.TypeIdent(baseType.Ident)
		if result, err := t.assertionTransformer.TransformBuiltinConstraint(typeIdent, ensure.Variable, constraint); err == nil {
			return result, nil
		}
	}

	return nil, fmt.Errorf("no valid transformation found for constraint: %s", constraint.Name)
}
