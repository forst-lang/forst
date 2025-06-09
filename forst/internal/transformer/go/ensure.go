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
	if symbol, exists := t.currentScope().LookupVariable(baseIdent, true); exists {
		if len(symbol.Types) > 0 {
			declaredType = symbol.Types[0]
		}
	}

	t.log.WithFields(logrus.Fields{
		"variable":     ensure.Variable.Ident.ID,
		"declaredType": declaredType.Ident,
		"inferredType": variableType.Ident,
		"assertion":    ensure.Assertion,
	}).Trace("transformEnsureCondition: TypeGuardLookup")

	// --- Type Guard Lookup: Prefer declared type (alias) over base type ---
	var typeGuardExprs []goast.Expr
	// 1. Try declared type (alias) first
	if _, exists := t.TypeChecker.Defs[declaredType.Ident]; exists {
		// Search for type guards defined for this alias
		for _, constraint := range ensure.Assertion.Constraints {
			t.log.Tracef("[transformEnsureCondition] [Alias] Checking constraint: %s, Args: %+v", constraint.Name, constraint.Args)
			for _, def := range t.TypeChecker.Defs {
				if tg, ok := def.(ast.TypeGuardNode); ok && tg.GetIdent() == constraint.Name {
					subjectType := tg.Subject.GetType()
					if t.TypeChecker.IsTypeCompatible(declaredType, subjectType) {
						t.log.Tracef("[transformEnsureCondition] [Alias] Type guard '%s' is compatible with declared variable type '%s' (subject type: '%s')", tg.Ident, declaredType.Ident, subjectType.Ident)
						typeGuardExpr, err := t.transformTypeGuardEnsure(ensure)
						if err != nil {
							return nil, fmt.Errorf("failed to transform type guard ensure: %w", err)
						}
						typeGuardExprs = append(typeGuardExprs, typeGuardExpr)
						break // Found a matching type guard for this constraint
					}
				}
			}
		}
	}
	if len(typeGuardExprs) > 0 {
		t.log.Tracef("[transformEnsureCondition] Returning conjoined type guard expressions (alias): %+v", typeGuardExprs)
		return conjoin(typeGuardExprs), nil
	}
	// 2. Fallback: Try base type if no alias type guard found
	baseType, err := t.getEnsureBaseType(ensure)
	if err == nil && baseType.Ident != declaredType.Ident {
		if _, exists := t.TypeChecker.Defs[baseType.Ident]; exists {
			for _, constraint := range ensure.Assertion.Constraints {
				t.log.Tracef("[transformEnsureCondition] [Base] Checking constraint: %s, Args: %+v", constraint.Name, constraint.Args)
				for _, def := range t.TypeChecker.Defs {
					if tg, ok := def.(ast.TypeGuardNode); ok && tg.GetIdent() == constraint.Name {
						subjectType := tg.Subject.GetType()
						if t.TypeChecker.IsTypeCompatible(baseType, subjectType) {
							t.log.Tracef("[transformEnsureCondition] [Base] Type guard '%s' is compatible with base variable type '%s' (subject type: '%s')", tg.Ident, baseType.Ident, subjectType.Ident)
							typeGuardExpr, err := t.transformTypeGuardEnsure(ensure)
							if err != nil {
								return nil, fmt.Errorf("failed to transform type guard ensure: %w", err)
							}
							typeGuardExprs = append(typeGuardExprs, typeGuardExpr)
							break // Found a matching type guard for this constraint
						}
					}
				}
			}
		}
	}
	if len(typeGuardExprs) > 0 {
		t.log.Tracef("[transformEnsureCondition] Returning conjoined type guard expressions (base): %+v", typeGuardExprs)
		return conjoin(typeGuardExprs), nil
	}

	t.log.Tracef("No type guard found, transforming ensure condition, var type %+v, assertion: %+v", variableType, ensure.Assertion)
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
