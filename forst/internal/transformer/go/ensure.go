package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
)

// transformEnsureCondition transforms an ensure node into Go statements
func (t *Transformer) transformEnsureCondition(ensure *ast.EnsureNode) ([]goast.Stmt, error) {
	if ensure.Assertion.BaseType == nil {
		t.log.Debugf("[transformEnsureCondition] Assertion.BaseType is nil for assertion: %v", ensure.Assertion)
	} else {
		t.log.Debugf("[transformEnsureCondition] Assertion.BaseType: %v", *ensure.Assertion.BaseType)
	}

	varType, err := t.TypeChecker.LookupVariableType(&ensure.Variable, t.currentScope())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup variable type: %w", err)
	}

	var transformedStmts []goast.Stmt

	// Handle case where assertion has a BaseType but no constraints (type guard call)
	if ensure.Assertion.BaseType != nil && len(ensure.Assertion.Constraints) == 0 {
		t.log.Debugf("[transformEnsureCondition] Assertion has BaseType but no constraints - treating as type guard call")
		// Treat the BaseType as a type guard name
		typeGuardName := string(*ensure.Assertion.BaseType)
		t.log.Debugf("[transformEnsureCondition] Looking up type guard: %s", typeGuardName)
		typeGuardDef, err := t.lookupTypeGuardNode(typeGuardName)
		if err != nil {
			t.log.Debugf("[transformEnsureCondition] Type guard lookup failed: %v", err)
		} else if typeGuardDef == nil {
			t.log.Debugf("[transformEnsureCondition] Type guard not found: %s", typeGuardName)
		} else {
			t.log.Debugf("[transformEnsureCondition] Type guard found: %s", typeGuardName)
			t.log.Debugf("[transformEnsureCondition] Variable type: %v", varType)
			t.log.Debugf("[transformEnsureCondition] Type guard subject type: %v", typeGuardDef.Subject.GetType())
			compatible := t.isTypeGuardCompatible(varType, typeGuardDef)
			t.log.Debugf("[transformEnsureCondition] Type guard compatibility check: %v", compatible)
			if compatible {
				t.log.Debugf("[transformEnsureCondition] Type guard is compatible")
				hash, err := t.TypeChecker.Hasher.HashNode(*typeGuardDef)
				if err != nil {
					return nil, fmt.Errorf("failed to hash type guard node: %w", err)
				}
				guardFuncName := hash.ToGuardIdent()
				t.log.Debugf("[transformEnsureCondition] Guard function name: %s", guardFuncName)
				expr, err := t.transformExpression(ensure.Variable)
				if err != nil {
					return nil, fmt.Errorf("failed to transform expression: %w", err)
				}
				callExpr := &goast.CallExpr{
					Fun:  goast.NewIdent(string(guardFuncName)),
					Args: []goast.Expr{expr},
				}
				transformedStmts = append(transformedStmts, &goast.ExprStmt{X: callExpr})
				t.log.Debugf("[transformEnsureCondition] Returning %d statements", len(transformedStmts))
				return transformedStmts, nil
			} else {
				t.log.Debugf("[transformEnsureCondition] Type guard is not compatible")
			}
		}
	}

	// Handle constraints (existing logic)
	for _, constraint := range ensure.Assertion.Constraints {
		transformed, err := t.transformEnsureConstraint(*ensure, constraint, varType)
		if err != nil {
			return nil, fmt.Errorf("failed to transform constraint: %w", err)
		}
		transformedStmts = append(transformedStmts, &goast.ExprStmt{X: transformed})
	}
	t.log.Debugf("[transformEnsureCondition] Returning %d statements from constraints", len(transformedStmts))
	return transformedStmts, nil
}
