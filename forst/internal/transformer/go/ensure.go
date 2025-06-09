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

func (t *Transformer) getEnsureBaseType(ensure ast.EnsureNode) (ast.TypeNode, error) {
	if ensure.Assertion.BaseType != nil {
		return ast.TypeNode{Ident: *ensure.Assertion.BaseType}, nil
	}

	ensureBaseType, err := t.TypeChecker.LookupEnsureBaseType(&ensure, t.currentScope())
	if err != nil {
		return ast.TypeNode{}, err
	}
	return *ensureBaseType, nil
}

// getTypeAliasChain returns the chain of type aliases for a given type, ending with the base type.
func (t *Transformer) getTypeAliasChain(typeNode ast.TypeNode) []ast.TypeNode {
	chain := []ast.TypeNode{typeNode}
	visited := map[ast.TypeIdent]bool{typeNode.Ident: true}
	current := typeNode
	t.log.WithFields(map[string]interface{}{
		"typeNode": current.Ident,
		"function": "getTypeAliasChain",
	}).Debug("Starting type alias chain resolution")
	for {
		def, exists := t.TypeChecker.Defs[current.Ident]
		if !exists {
			// Try inferred types if no explicit alias is found
			hash, err := t.TypeChecker.Hasher.HashNode(current)
			if err == nil {
				if inferred, ok := t.TypeChecker.Types[hash]; ok && len(inferred) > 0 {
					inferredType := inferred[0]
					if !visited[inferredType.Ident] {
						t.log.WithFields(map[string]interface{}{
							"typeNode": inferredType.Ident,
							"function": "getTypeAliasChain",
						}).Debug("Following inferred type in alias chain")
						chain = append(chain, inferredType)
						visited[inferredType.Ident] = true
						current = inferredType
						continue
					}
				}
			}
			t.log.WithFields(map[string]interface{}{
				"typeNode": current.Ident,
				"function": "getTypeAliasChain",
			}).Debug("No definition or inferred type found for type")
			break
		}
		typeDef, ok := def.(ast.TypeDefNode)
		if !ok {
			t.log.WithFields(map[string]interface{}{
				"typeNode": current.Ident,
				"function": "getTypeAliasChain",
			}).Debug("Definition is not a TypeDefNode")
			break
		}
		// Accept both value and pointer types for TypeDefAssertionExpr
		var assertionExpr *ast.TypeDefAssertionExpr
		switch expr := typeDef.Expr.(type) {
		case ast.TypeDefAssertionExpr:
			assertionExpr = &expr
		case *ast.TypeDefAssertionExpr:
			assertionExpr = expr
		}
		if assertionExpr == nil || assertionExpr.Assertion == nil || assertionExpr.Assertion.BaseType == nil {
			t.log.WithFields(map[string]interface{}{
				"typeNode":  current.Ident,
				"exprType":  fmt.Sprintf("%T", typeDef.Expr),
				"exprValue": fmt.Sprintf("%#v", typeDef.Expr),
				"function":  "getTypeAliasChain",
			}).Debug("Definition is not a valid type alias")
			break
		}
		baseIdent := *assertionExpr.Assertion.BaseType
		if visited[baseIdent] {
			t.log.WithFields(map[string]interface{}{
				"typeNode": current.Ident,
				"function": "getTypeAliasChain",
			}).Debug("Cycle detected in type alias chain")
			break // prevent cycles
		}
		baseType := ast.TypeNode{Ident: baseIdent}
		chain = append(chain, baseType)
		visited[baseIdent] = true
		current = baseType
		t.log.WithFields(map[string]interface{}{
			"typeNode": current.Ident,
			"function": "getTypeAliasChain",
		}).Debug("Added base type to alias chain")
	}
	return chain
}

// transformEnsureConstraint transforms a single constraint in an ensure statement
func (t *Transformer) transformEnsureConstraint(ensure ast.EnsureNode, constraint ast.ConstraintNode, varType ast.TypeNode) (goast.Expr, error) {
	// First try built-in constraints
	transformed, err := t.assertionTransformer.TransformBuiltinConstraint(varType.Ident, ensure.Variable, constraint)
	if err == nil {
		return transformed, nil
	}

	// Then try type guards
	typeGuardDef, err := t.lookupTypeGuardNode(constraint.Name)
	if err == nil && typeGuardDef != nil {
		// Check if the type guard is compatible with the variable type
		if t.isTypeGuardCompatible(varType, typeGuardDef) {
			// Transform the type guard into a function call using the hash-based name
			hash, err := t.TypeChecker.Hasher.HashNode(*typeGuardDef)
			if err != nil {
				return nil, fmt.Errorf("failed to hash type guard node: %w", err)
			}
			guardFuncName := hash.ToGuardIdent()

			// Transform the variable expression
			expr, err := t.transformExpression(ensure.Variable)
			if err != nil {
				return nil, fmt.Errorf("failed to transform expression: %w", err)
			}

			// Build arguments list starting with the variable
			args := []goast.Expr{expr}
			for _, arg := range constraint.Args {
				var argExpr goast.Expr
				if arg.Type != nil {
					argExpr = goast.NewIdent(string(arg.Type.Ident))
				} else if arg.Shape != nil {
					argExpr = goast.NewIdent("struct{}")
				} else if arg.Value != nil {
					switch v := (*arg.Value).(type) {
					case ast.IntLiteralNode:
						argExpr = &goast.BasicLit{
							Kind:  gotoken.INT,
							Value: fmt.Sprintf("%d", v.Value),
						}
					case ast.StringLiteralNode:
						argExpr = &goast.BasicLit{
							Kind:  gotoken.STRING,
							Value: fmt.Sprintf("%q", v.Value),
						}
					case ast.BoolLiteralNode:
						argExpr = goast.NewIdent(fmt.Sprintf("%v", v.Value))
					case ast.FloatLiteralNode:
						argExpr = &goast.BasicLit{
							Kind:  gotoken.FLOAT,
							Value: fmt.Sprintf("%f", v.Value),
						}
					default:
						return nil, fmt.Errorf("unsupported value type in constraint argument: %T", v)
					}
				} else {
					return nil, fmt.Errorf("unsupported constraint argument type")
				}
				args = append(args, argExpr)
			}

			// Build the type guard function call
			callExpr := &goast.CallExpr{
				Fun:  goast.NewIdent(string(guardFuncName)),
				Args: args,
			}
			// Negate the call expression
			return &goast.UnaryExpr{
				Op: gotoken.NOT,
				X:  callExpr,
			}, nil
		}
	}

	// If not found, try all base types in the alias chain
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
			// Construct a call expression using the guard's function name and the ensure variable
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
			// Negate the call expression
			return &goast.UnaryExpr{
				Op: gotoken.NOT,
				X:  callExpr,
			}, nil
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

// transformEnsureCondition transforms an ensure node into Go statements
func (t *Transformer) transformEnsureCondition(ensure *ast.EnsureNode) ([]goast.Stmt, error) {
	// Get the variable type from the symbol table
	varType, err := t.TypeChecker.LookupVariableType(&ensure.Variable, t.currentScope())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup variable type: %w", err)
	}

	// Transform each constraint
	var transformedStmts []goast.Stmt
	for _, constraint := range ensure.Assertion.Constraints {
		transformed, err := t.transformEnsureConstraint(*ensure, constraint, varType)
		if err != nil {
			return nil, fmt.Errorf("failed to transform constraint: %w", err)
		}
		transformedStmts = append(transformedStmts, &goast.ExprStmt{X: transformed})
	}

	return transformedStmts, nil
}
