package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

func (tc *TypeChecker) inferTypeGuardNode(typeGuardNode ast.Node) ([]ast.TypeNode, error) {
	var guardNode ast.TypeGuardNode
	switch n := typeGuardNode.(type) {
	case *ast.TypeGuardNode:
		if n == nil {
			return nil, nil
		}
		guardNode = *n
	case ast.TypeGuardNode:
		guardNode = n
	default:
		return nil, fmt.Errorf("inferTypeGuardNode: unexpected node type %T", typeGuardNode)
	}
	tc.pushScope(typeGuardNode)

	for _, param := range guardNode.Parameters() {
		switch typedParam := param.(type) {
		case ast.SimpleParamNode:
			parameterTypes := []ast.TypeNode{typedParam.Type}
			tc.scopeStack.currentScope().RegisterSymbol(
				typedParam.Ident.ID,
				parameterTypes,
				SymbolVariable)
			tc.VariableTypes[typedParam.Ident.ID] = parameterTypes
		case ast.DestructuredParamNode:
			tc.registerDestructuredParamSymbols(typedParam.Fields, typedParam.Type, SymbolVariable)
		}
	}

	if err := tc.validateTypeGuardBody(guardNode.Body); err != nil {
		tc.popScope()
		return nil, err
	}

	tc.recordTypeGuardBodyIR(guardNode.Ident, guardNode.Body)

	for _, node := range guardNode.Body {
		switch stmt := node.(type) {
		case ast.CommentNode:
			continue
		case ast.IfNode:
			if _, err := tc.inferIfStatement(&stmt); err != nil {
				tc.popScope()
				return nil, err
			}
		case *ast.IfNode:
			if _, err := tc.inferIfStatement(stmt); err != nil {
				tc.popScope()
				return nil, err
			}
		case ast.EnsureNode:
			if _, err := tc.inferNodeType(stmt); err != nil {
				tc.popScope()
				return nil, err
			}
		case *ast.EnsureNode:
			if _, err := tc.inferNodeType(*stmt); err != nil {
				tc.popScope()
				return nil, err
			}
		}
	}

	tc.popScope()
	return nil, nil
}

// validateTypeGuardBody enforces the recursive guard statement whitelist (structure only).
func (tc *TypeChecker) validateTypeGuardBody(body []ast.Node) error {
	for _, node := range body {
		switch stmt := node.(type) {
		case ast.CommentNode:
			continue
		case ast.ReturnNode:
			return diagnosticf(ast.SourceSpan{}, "refinement-guard-forbidden-stmt",
				"type guards must not have return statements")
		case ast.AssignmentNode, *ast.AssignmentNode:
			return diagnosticf(ast.SourceSpan{}, "refinement-guard-forbidden-stmt",
				"type guards must not contain assignments")
		case ast.ForNode, *ast.ForNode:
			return diagnosticf(ast.SourceSpan{}, "refinement-guard-forbidden-stmt",
				"type guards must not contain loops")
		case ast.IfNode:
			if err := tc.validateTypeGuardIf(&stmt); err != nil {
				return err
			}
		case *ast.IfNode:
			if err := tc.validateTypeGuardIf(stmt); err != nil {
				return err
			}
		case ast.EnsureNode:
			if err := tc.validateTypeGuardEnsure(stmt); err != nil {
				return err
			}
		case *ast.EnsureNode:
			if err := tc.validateTypeGuardEnsure(*stmt); err != nil {
				return err
			}
		default:
			return diagnosticf(ast.SourceSpan{}, "refinement-guard-forbidden-stmt",
				"type guards may only contain if, else if, else, and ensure statements")
		}
	}
	return nil
}

func (tc *TypeChecker) validateTypeGuardIf(stmt *ast.IfNode) error {
	if binExpr, ok := stmt.Condition.(ast.BinaryExpressionNode); !ok || binExpr.Operator != ast.TokenIs {
		return diagnosticf(ast.SourceSpan{}, "refinement-guard-forbidden-stmt",
			"type guard conditions must use 'is' operator")
	}
	if err := tc.validateTypeGuardBody(stmt.Body); err != nil {
		return err
	}
	for i := range stmt.ElseIfs {
		ei := &stmt.ElseIfs[i]
		if binExpr, ok := ei.Condition.(ast.BinaryExpressionNode); !ok || binExpr.Operator != ast.TokenIs {
			return diagnosticf(ast.SourceSpan{}, "refinement-guard-forbidden-stmt",
				"type guard conditions must use 'is' operator")
		}
		if err := tc.validateTypeGuardBody(ei.Body); err != nil {
			return err
		}
	}
	if stmt.Else != nil {
		if err := tc.validateTypeGuardBody(stmt.Else.Body); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TypeChecker) validateTypeGuardEnsure(stmt ast.EnsureNode) error {
	if stmt.Error != nil {
		return diagnosticf(ast.SourceSpan{}, "refinement-else-in-guard",
			"typed `else` is not allowed inside type guards")
	}
	if stmt.Block != nil {
		return diagnosticf(ast.SourceSpan{}, "refinement-failure-block-in-guard",
			"failure blocks are not allowed inside type guards")
	}
	return nil
}
