package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferNodeType(node ast.Node) ([]ast.TypeNode, error) {
	if tc.log.IsLevelEnabled(logrus.TraceLevel) {
		tc.log.WithFields(logrus.Fields{
			"node":     node.String(),
			"function": "inferNodeType",
		}).Trace("Inferring node type")
	}

	switch node.(type) {
	case ast.PackageNode, ast.TypeNode, ast.CommentNode, ast.ImportNode, ast.ImportGroupNode, ast.FallthroughNode:
		types, _, err := tc.inferNodeTypeLeaf(node)
		return types, err
	case ast.SimpleParamNode, ast.DestructuredParamNode:
		types, _, err := tc.inferNodeTypeParams(node)
		return types, err
	case ast.AssignmentNode, ast.ConstGroupNode, ast.TypeDefNode:
		types, _, err := tc.inferNodeTypeDecl(node)
		return types, err
	case ast.IfNode, *ast.IfNode, ast.ForNode, *ast.ForNode, ast.SwitchNode, *ast.SwitchNode,
		*ast.BreakNode, *ast.ContinueNode, *ast.GotoNode, *ast.LabeledStmtNode:
		types, _, err := tc.inferNodeTypeControl(node)
		return types, err
	case *ast.DeferNode, *ast.GoStmtNode:
		types, _, err := tc.inferNodeTypeDeferGo(node)
		return types, err
	case *ast.ElseBlockNode, ast.ElseBlockNode:
		types, _, err := tc.inferNodeTypeElseBlock(node)
		return types, err
	case ast.FunctionNode:
		types, _, err := tc.inferNodeTypeFunction(node)
		return types, err
	case ast.EnsureNode, ast.UseNode, ast.WithNode:
		types, _, err := tc.inferNodeTypeEnsure(node)
		return types, err
	case ast.ReturnNode:
		types, _, err := tc.inferNodeTypeReturn(node)
		return types, err
	case ast.TypeGuardNode, *ast.TypeGuardNode:
		types, _, err := tc.inferNodeTypeTypeGuard(node)
		return types, err
	case ast.ExpressionNode:
		types, _, err := tc.inferNodeTypeExpression(node)
		return types, err
	}
	return nil, fmt.Errorf("%s", typecheckErrorMessageWithNode(&node, fmt.Sprintf("unsupported node type %T", node)))
}
