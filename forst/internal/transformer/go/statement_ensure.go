package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	logrus "github.com/sirupsen/logrus"
)

func (t *Transformer) transformEnsureStatement(ensureNode ast.EnsureNode, originalNode ast.Node) (goast.Stmt, error) {
	fnNode, err := t.closestFunction()
	if err != nil {
		return nil, fmt.Errorf("could not find enclosing function for EnsureNode: %w", err)
	}
	fn, ok := fnNode.(ast.FunctionNode)
	if !ok {
		return nil, fmt.Errorf("enclosing node is not a FunctionNode")
	}
	t.functionsWithEnsure[string(fn.Ident.ID)] = true
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function": "transformStatement",
			"action":   "tracking function with ensure",
			"fnName":   string(fn.Ident.ID),
		}).Debug("[PINPOINT] Function has ensure statement")
	}

	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function": "transformStatement",
			"stmtType": "EnsureNode",
			"stmt":     ensureNode.String(),
		}).Debug("[PINPOINT] Processing EnsureNode")
	}
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function":           "transformStatement",
			"scopeBeforeRestore": fmt.Sprintf("%v", t.currentScope()),
			"nodeType":           fmt.Sprintf("%T", originalNode),
		}).Debug("[DEBUG] Before restoreScope(EnsureNode)")
	}
	if err := t.restoreScope(originalNode); err != nil {
		return nil, fmt.Errorf("failed to restore ensure statement scope: %s", err)
	}

	stmts, err := t.transformEnsureCondition(&ensureNode)
	if err != nil {
		return nil, err
	}
	if len(stmts) == 0 {
		return nil, fmt.Errorf("transformEnsureCondition returned no statements")
	}

	exprStmt, ok := stmts[0].(*goast.ExprStmt)
	if !ok {
		return nil, fmt.Errorf("expected ExprStmt from transformEnsureCondition, got %T", stmts[0])
	}

	finalCondition := exprStmt.X
	shouldNegate := len(ensureNode.Assertion.Constraints) == 0
	for _, constraint := range ensureNode.Assertion.Constraints {
		if t.TypeChecker.IsTypeGuardConstraint(constraint.Name) {
			shouldNegate = true
			break
		}
		if (constraint.Name == "Ok" || constraint.Name == "Err") && !t.TypeChecker.IsTypeGuardConstraint(constraint.Name) {
			shouldNegate = true
			break
		}
	}
	if shouldNegate {
		finalCondition = &goast.UnaryExpr{Op: token.NOT, X: finalCondition}
	}

	finallyStmts := []goast.Stmt{}
	errorStmt := t.transformErrorStatement(fn, ensureNode)
	if ensureNode.Block != nil {
		if err := t.restoreScope(ensureNode.Block); err != nil {
			return nil, fmt.Errorf("failed to restore ensure block scope: %w", err)
		}
		for _, blockStatement := range ensureNode.Block.Body {
			if _, isReturn := blockStatement.(ast.ReturnNode); isReturn {
				continue
			}
			goStmt, err := t.transformStatement(blockStatement)
			if err != nil {
				return nil, err
			}
			finallyStmts = append(finallyStmts, goStmt)
		}
	}

	return &goast.IfStmt{
		Cond: finalCondition,
		Body: &goast.BlockStmt{List: append(finallyStmts, errorStmt)},
	}, nil
}
