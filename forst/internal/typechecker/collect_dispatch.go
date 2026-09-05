package typechecker

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) collectExplicitTypes(node ast.Node) error {
	tc.log.WithFields(logrus.Fields{
		"node":     node.String(),
		"function": "collectExplicitTypes",
	}).Trace("Collecting explicit types")

	switch node.(type) {
	case ast.CommentNode, ast.UseNode:
		_, err := tc.collectExplicitTypesLeaf(node)
		return err
	case ast.ImportNode, ast.ImportGroupNode:
		_, err := tc.collectExplicitTypesImport(node)
		return err
	case ast.AssignmentNode, ast.ConstGroupNode, ast.TypeDefNode, ast.FunctionNode, *ast.FunctionNode,
		ast.TypeGuardNode, *ast.TypeGuardNode:
		_, err := tc.collectExplicitTypesDecl(node)
		return err
	case ast.EnsureNode, ast.WithNode, ast.ElseIfNode, *ast.ElseIfNode, *ast.ElseBlockNode, ast.ElseBlockNode:
		_, err := tc.collectExplicitTypesScope(node)
		return err
	case ast.IfNode, *ast.IfNode, ast.ForNode, *ast.ForNode, ast.SwitchNode, *ast.SwitchNode,
		*ast.GotoNode, *ast.LabeledStmtNode:
		_, err := tc.collectExplicitTypesControl(node)
		return err
	}
	return nil
}

func (tc *TypeChecker) collectExplicitTypesLeaf(node ast.Node) (bool, error) {
	switch node.(type) {
	case ast.CommentNode, ast.UseNode:
		return true, nil
	}
	return false, nil
}

func (tc *TypeChecker) collectExplicitTypesImport(node ast.Node) (bool, error) {
	switch n := node.(type) {
	case ast.ImportNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Collecting import")
		return true, tc.collectImportNode(n)
	case ast.ImportGroupNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Collecting import group")
		for _, imp := range n.Imports {
			if err := tc.collectImportNode(imp); err != nil {
				return true, err
			}
		}
		return true, nil
	}
	return false, nil
}

func (tc *TypeChecker) collectExplicitTypesDecl(node ast.Node) (bool, error) {
	switch n := node.(type) {
	case ast.AssignmentNode:
		if !n.IsPackageLevel {
			return true, nil
		}
		return true, tc.collectPackageLevelVar(n)
	case ast.ConstGroupNode:
		return true, tc.collectConstGroup(n)
	case ast.TypeDefNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Collecting type definition")
		tc.registerType(n)
		return true, nil
	case ast.FunctionNode:
		return true, tc.collectFunctionNode(node, n)
	case *ast.FunctionNode:
		if n == nil {
			return true, nil
		}
		return true, tc.collectFunctionNode(node, *n)
	case ast.TypeGuardNode:
		return true, tc.collectTypeGuardNode(node, n)
	case *ast.TypeGuardNode:
		if n == nil {
			return true, nil
		}
		return true, tc.collectTypeGuardNode(node, *n)
	}
	return false, nil
}

func (tc *TypeChecker) collectExplicitTypesScope(node ast.Node) (bool, error) {
	switch n := node.(type) {
	case ast.EnsureNode:
		return true, tc.collectEnsureScope(node, n)
	case ast.WithNode:
		return true, tc.collectWithScope(node, n)
	case ast.ElseIfNode:
		return true, tc.collectExplicitTypes(&n)
	case *ast.ElseIfNode:
		return true, tc.collectElseIfScope(n)
	case *ast.ElseBlockNode:
		return true, tc.collectElseBlockScope(n)
	case ast.ElseBlockNode:
		eb := n
		return true, tc.collectExplicitTypes(&eb)
	}
	return false, nil
}

func (tc *TypeChecker) collectEnsureScope(scopeNode ast.Node, n ast.EnsureNode) error {
	tc.log.WithFields(logrus.Fields{
		"node":     n.String(),
		"function": "collectExplicitTypes",
	}).Debug("Storing scope for ensure")
	tc.pushScope(scopeNode)
	if n.Block != nil {
		tc.pushScope(n.Block)
		for _, child := range n.Block.Body {
			if err := tc.collectExplicitTypes(child); err != nil {
				return err
			}
		}
		tc.popScope()
	}
	tc.popScope()
	return nil
}

func (tc *TypeChecker) collectWithScope(scopeNode ast.Node, n ast.WithNode) error {
	tc.pushScope(scopeNode)
	for _, child := range n.Body {
		if err := tc.collectExplicitTypes(child); err != nil {
			return err
		}
	}
	tc.popScope()
	return nil
}

func (tc *TypeChecker) collectElseIfScope(n *ast.ElseIfNode) error {
	if n == nil {
		return nil
	}
	tc.log.WithFields(logrus.Fields{
		"node":     n.String(),
		"function": "collectExplicitTypes",
	}).Debug("Storing scope for else if")
	tc.pushScope(n)
	for _, child := range n.Body {
		if err := tc.collectExplicitTypes(child); err != nil {
			return err
		}
	}
	tc.popScope()
	return nil
}

func (tc *TypeChecker) collectElseBlockScope(n *ast.ElseBlockNode) error {
	if n == nil {
		return nil
	}
	tc.log.WithFields(logrus.Fields{
		"node":     n.String(),
		"function": "collectExplicitTypes",
	}).Debug("Storing scope for else block")
	tc.pushScope(n)
	for _, child := range n.Body {
		if err := tc.collectExplicitTypes(child); err != nil {
			return err
		}
	}
	tc.popScope()
	return nil
}

func (tc *TypeChecker) collectExplicitTypesControl(node ast.Node) (bool, error) {
	switch n := node.(type) {
	case ast.IfNode:
		return true, tc.collectIfNode(&n)
	case *ast.IfNode:
		return true, tc.collectIfNode(n)
	case ast.ForNode:
		return true, tc.collectForNode(&n)
	case *ast.ForNode:
		return true, tc.collectForNode(n)
	case ast.SwitchNode:
		return true, tc.collectSwitchNode(&n)
	case *ast.SwitchNode:
		return true, tc.collectSwitchNode(n)
	case *ast.GotoNode:
		return true, nil
	case *ast.LabeledStmtNode:
		if n != nil && n.Stmt != nil {
			return true, tc.collectExplicitTypes(n.Stmt)
		}
		return true, nil
	}
	return false, nil
}
