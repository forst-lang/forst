package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
)

// transformEnsureCondition transforms an ensure node into Go statements
func (t *Transformer) transformEnsureCondition(ensure *ast.EnsureNode) ([]goast.Stmt, error) {
	if ensure.Assertion.BaseType == nil {
		t.log.Warnf("[transformEnsureCondition] Assertion.BaseType is nil for assertion: %v", ensure.Assertion)
	} else {
		t.log.Debugf("[transformEnsureCondition] Assertion.BaseType: %v", *ensure.Assertion.BaseType)
	}

	varType, err := t.TypeChecker.LookupVariableType(&ensure.Variable, t.currentScope())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup variable type: %w", err)
	}

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
