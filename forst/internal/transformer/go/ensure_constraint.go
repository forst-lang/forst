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

// transformConstraintArg transforms a constraint argument node into a Go expression
func transformConstraintArg(arg ast.ConstraintArgumentNode) (goast.Expr, error) {
	if arg.Type != nil {
		return goast.NewIdent(string(arg.Type.Ident)), nil
	} else if arg.Shape != nil {
		return goast.NewIdent("struct{}"), nil
	} else if arg.Value != nil {
		switch v := (*arg.Value).(type) {
		case ast.IntLiteralNode:
			return &goast.BasicLit{Kind: gotoken.INT, Value: fmt.Sprintf("%d", v.Value)}, nil
		case ast.StringLiteralNode:
			return &goast.BasicLit{Kind: gotoken.STRING, Value: fmt.Sprintf("%q", v.Value)}, nil
		case ast.BoolLiteralNode:
			return goast.NewIdent(fmt.Sprintf("%v", v.Value)), nil
		case ast.FloatLiteralNode:
			return &goast.BasicLit{Kind: gotoken.FLOAT, Value: fmt.Sprintf("%f", v.Value)}, nil
		default:
			return nil, fmt.Errorf("unsupported value type in constraint argument: %T", v)
		}
	}
	return nil, fmt.Errorf("unsupported constraint argument type")
}

// validateConstraintArgs validates the number of arguments for a constraint
func (at *AssertionTransformer) validateConstraintArgs(constraint ast.ConstraintNode, expectedArgs int) error {
	if len(constraint.Args) != expectedArgs {
		return fmt.Errorf("%s constraint requires %d argument(s)", constraint.Name, expectedArgs)
	}
	return nil
}
