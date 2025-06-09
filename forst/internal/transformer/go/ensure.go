package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"strings"

	"github.com/sirupsen/logrus"
)

// transformEnsureConstraints transforms an ensure node based on its type
func (at *AssertionTransformer) transformEnsureConstraints(ensure ast.EnsureNode) (goast.Expr, error) {
	baseType, err := at.transformer.getEnsureBaseType(ensure)
	if err != nil {
		return nil, err
	}

	switch baseType.Ident {
	case ast.TypeString:
		return at.transformStringEnsure(ensure)
	case ast.TypeInt:
		return at.transformIntEnsure(ensure)
	case ast.TypeFloat:
		return at.transformFloatEnsure(ensure)
	case ast.TypeBool:
		return at.transformBoolEnsure(ensure)
	case ast.TypeError:
		return at.transformErrorEnsure(ensure)
	default:
		ident, err := at.transformer.TypeChecker.LookupAssertionType(&ensure.Assertion)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup assertion type: %s", err)
		}
		at.transformer.log.Errorf("No type guard found, using assertion identifier: %s", string(ident.Ident))
		return goast.NewIdent(string(ident.Ident)), nil
	}
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
	for {
		def, exists := t.TypeChecker.Defs[current.Ident]
		if !exists {
			break
		}
		typeDef, ok := def.(ast.TypeDefNode)
		if !ok {
			break
		}
		assertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr)
		if !ok || assertionExpr.Assertion == nil || assertionExpr.Assertion.BaseType == nil {
			break
		}
		baseIdent := *assertionExpr.Assertion.BaseType
		if visited[baseIdent] {
			break // prevent cycles
		}
		baseType := ast.TypeNode{Ident: baseIdent}
		chain = append(chain, baseType)
		visited[baseIdent] = true
		current = baseType
	}
	return chain
}

func (t *Transformer) transformEnsureCondition(ensure ast.EnsureNode) (goast.Expr, error) {
	// Look up the variable type first (inferred)
	variableType, err := t.TypeChecker.LookupVariableType(&ensure.Variable, t.currentScope())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup ensure variable type: %w, scope: %s", err, t.currentScope().String())
	}

	// Look up the declared type from the symbol table
	declaredType := variableType
	parts := strings.Split(string(ensure.Variable.Ident.ID), ".")
	baseIdent := ast.Identifier(parts[0])
	var symbolTypes []ast.TypeNode
	if symbol, exists := t.currentScope().LookupVariable(baseIdent, true); exists {
		if len(symbol.Types) > 0 {
			declaredType = symbol.Types[0]
			symbolTypes = symbol.Types
		}
	}

	aliasChain := t.getTypeAliasChain(declaredType)
	aliasChainStrs := make([]string, len(aliasChain))
	for i, tn := range aliasChain {
		aliasChainStrs[i] = string(tn.Ident)
	}

	t.log.WithFields(logrus.Fields{
		"variable":     ensure.Variable.Ident.ID,
		"declaredType": declaredType.Ident,
		"inferredType": variableType.Ident,
		"symbolTypes":  symbolTypes,
		"aliasChain":   aliasChainStrs,
		"assertion":    ensure.Assertion,
	}).Debug("transformEnsureCondition: TypeGuardLookup (diagnostic)")

	// --- Type Guard Lookup: Try all aliases and base type ---
	var typeGuardExprs []goast.Expr
	aliasChain = t.getTypeAliasChain(declaredType)
	tried := map[ast.TypeIdent]bool{}
	for _, typeToCheck := range aliasChain {
		tried[typeToCheck.Ident] = true
		for _, constraint := range ensure.Assertion.Constraints {
			t.log.WithFields(logrus.Fields{
				"constraint": constraint.Name,
				"args":       constraint.Args,
				"type":       typeToCheck.Ident,
				"function":   "transformEnsureCondition",
			}).Tracef("[AliasChain] Checking constraint")
			for _, def := range t.TypeChecker.Defs {
				if tg, ok := def.(ast.TypeGuardNode); ok && tg.GetIdent() == constraint.Name {
					subjectType := tg.Subject.GetType()
					if t.TypeChecker.IsTypeCompatible(typeToCheck, subjectType) {
						t.log.WithFields(logrus.Fields{
							"typeGuard": tg.Ident,
							"type":      typeToCheck.Ident,
							"subject":   subjectType.Ident,
							"function":  "transformEnsureCondition",
						}).Tracef("[AliasChain] Type guard is compatible")
						typeGuardExpr, err := t.transformTypeGuardEnsure(ensure)
						if err != nil {
							t.log.Errorf("Failed to transform type guard ensure: %v", err)
							return nil, err
						}
						typeGuardExprs = append(typeGuardExprs, typeGuardExpr)
						break // Found a matching type guard for this constraint
					}
				}
			}
		}
	}
	// Also try the inferred type if not already tried
	if !tried[variableType.Ident] {
		for _, constraint := range ensure.Assertion.Constraints {
			t.log.WithFields(logrus.Fields{
				"constraint": constraint.Name,
				"args":       constraint.Args,
				"type":       variableType.Ident,
				"function":   "transformEnsureCondition",
			}).Tracef("[Inferred] Checking constraint")
			for _, def := range t.TypeChecker.Defs {
				if tg, ok := def.(ast.TypeGuardNode); ok && tg.GetIdent() == constraint.Name {
					subjectType := tg.Subject.GetType()
					if t.TypeChecker.IsTypeCompatible(variableType, subjectType) {
						t.log.WithFields(logrus.Fields{
							"typeGuard": tg.Ident,
							"type":      variableType.Ident,
							"subject":   subjectType.Ident,
							"function":  "transformEnsureCondition",
						}).Tracef("[Inferred] Type guard is compatible")
						typeGuardExpr, err := t.transformTypeGuardEnsure(ensure)
						if err != nil {
							t.log.WithFields(logrus.Fields{
								"error":    err,
								"function": "transformEnsureCondition",
							}).Errorf("Failed to transform type guard ensure")
							return nil, err
						}
						typeGuardExprs = append(typeGuardExprs, typeGuardExpr)
						break // Found a matching type guard for this constraint
					}
				}
			}
		}
	}
	if len(typeGuardExprs) > 0 {
		t.log.WithFields(logrus.Fields{
			"typeGuardExprs": typeGuardExprs,
			"function":       "transformEnsureCondition",
		}).Tracef("Returning conjoined type guard expressions (alias chain)")
		return conjoin(typeGuardExprs), nil
	}

	t.log.WithFields(logrus.Fields{
		"variableType": variableType,
		"assertion":    ensure.Assertion,
		"function":     "transformEnsureCondition",
	}).Tracef("No type guard found, transforming ensure condition")
	expr, err := t.assertionTransformer.transformEnsureConstraints(ensure)
	if err != nil {
		return nil, fmt.Errorf("failed to transform ensure conditions constraints: %w", err)
	}
	return expr, nil
}

func (t *AssertionTransformer) transformEnsureCondition(ensure *ast.EnsureNode) ([]goast.Stmt, error) {
	// Get the variable type from the symbol table
	varType, err := t.transformer.TypeChecker.LookupVariableType(&ensure.Variable, t.transformer.currentScope())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup variable type: %w", err)
	}

	// Log variable type information
	variableHash, err := t.transformer.TypeChecker.Hasher.HashNode(ensure.Variable)
	if err != nil {
		return nil, fmt.Errorf("failed to hash variable: %w", err)
	}
	t.transformer.log.WithFields(logrus.Fields{
		"variable":     ensure.Variable.GetIdent(),
		"declaredType": varType.Ident,
		"inferredType": t.transformer.TypeChecker.InferredTypes[variableHash][0].Ident,
	}).Trace("[transformEnsureCondition] Variable type information")

	// Transform each constraint
	var transformedStmts []goast.Stmt
	for _, constraint := range ensure.Assertion.Constraints {
		// Log constraint details
		t.transformer.log.WithFields(logrus.Fields{
			"constraint":     constraint.String(),
			"constraintType": fmt.Sprintf("%T", constraint),
		}).Trace("[transformEnsureCondition] Processing constraint")

		// Check if this constraint name matches a registered type guard
		typeGuardDef, err := t.transformer.lookupTypeGuardNode(constraint.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup type guard node: %w", err)
		}
		if typeGuardDef != nil {
			t.transformer.log.WithFields(logrus.Fields{
				"typeGuard":     constraint.Name,
				"typeGuardDef":  typeGuardDef,
				"typeGuardType": fmt.Sprintf("%T", typeGuardDef),
			}).Trace("[transformEnsureCondition] Found type guard constraint by name")

			// Check if the type guard is compatible with the variable type
			if t.isTypeGuardCompatible(varType, typeGuardDef) {
				t.transformer.log.WithFields(logrus.Fields{
					"typeGuard": constraint.Name,
					"varType":   varType.Ident,
				}).Trace("[transformEnsureCondition] Type guard is compatible")

				// Transform the type guard into a function call
				transformed, err := t.transformer.transformTypeGuardEnsure(ast.EnsureNode{
					Variable:  ensure.Variable,
					Assertion: ast.AssertionNode{Constraints: []ast.ConstraintNode{constraint}},
				})
				if err != nil {
					return nil, fmt.Errorf("failed to transform type guard ensure: %w", err)
				}
				transformedStmts = append(transformedStmts, &goast.ExprStmt{X: transformed})
				continue
			}

			t.transformer.log.WithFields(logrus.Fields{
				"typeGuard": constraint.Name,
				"varType":   varType.Ident,
			}).Trace("[transformEnsureCondition] Type guard is not compatible")
		}

		// If not a type guard or not compatible, try to transform as a regular constraint
		t.transformer.log.WithFields(logrus.Fields{
			"constraint": constraint.String(),
		}).Trace("[transformEnsureCondition] Attempting to transform as regular constraint")

		transformed, err := t.transformer.assertionTransformer.transformEnsureConstraints(ast.EnsureNode{
			Variable:  ensure.Variable,
			Assertion: ast.AssertionNode{Constraints: []ast.ConstraintNode{constraint}},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to transform ensure conditions constraints: %w", err)
		}
		transformedStmts = append(transformedStmts, &goast.ExprStmt{X: transformed})
	}

	return transformedStmts, nil
}
