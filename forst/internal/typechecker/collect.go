package typechecker

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// Traverses the AST to gather type definitions and function signatures
func (tc *TypeChecker) collectExplicitTypes(node ast.Node) error {
	tc.log.WithFields(logrus.Fields{
		"node":     node.String(),
		"function": "collectExplicitTypes",
	}).Trace("Collecting explicit types")

	switch n := node.(type) {
	case ast.CommentNode:
		return nil
	case ast.ImportNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Collecting import")
		return tc.collectImportNode(n)
	case ast.ImportGroupNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Collecting import group")
		for _, imp := range n.Imports {
			if err := tc.collectImportNode(imp); err != nil {
				return err
			}
		}
		return nil
	case ast.AssignmentNode:
		if !n.IsPackageLevel {
			return nil
		}
		return tc.collectPackageLevelVar(n)
	case ast.TypeDefNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Collecting type definition")
		tc.registerType(n)
	case ast.FunctionNode:
		return tc.collectFunctionNode(node, n)
	case *ast.FunctionNode:
		if n == nil {
			return nil
		}
		return tc.collectFunctionNode(node, *n)
	case ast.TypeGuardNode:
		return tc.collectTypeGuardNode(node, n)
	case *ast.TypeGuardNode:
		if n == nil {
			return nil
		}
		return tc.collectTypeGuardNode(node, *n)
	case ast.EnsureNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Storing scope for ensure")
		tc.pushScope(node)

		if n.Block != nil {
			tc.pushScope(n.Block)
			for _, node := range n.Block.Body {
				if err := tc.collectExplicitTypes(node); err != nil {
					return err
				}
			}
			tc.popScope()
		}
		tc.popScope()
		return nil
	case ast.UseNode:
		return nil
	case ast.WithNode:
		tc.pushScope(node)
		for _, node := range n.Body {
			if err := tc.collectExplicitTypes(node); err != nil {
				return err
			}
		}
		tc.popScope()
		return nil
	case ast.IfNode:
		if err := tc.collectIfNode(&n); err != nil {
			return err
		}
	case *ast.IfNode:
		if err := tc.collectIfNode(n); err != nil {
			return err
		}

	case ast.ForNode:
		if err := tc.collectForNode(&n); err != nil {
			return err
		}
	case *ast.ForNode:
		if err := tc.collectForNode(n); err != nil {
			return err
		}
	case ast.ElseIfNode:
		if err := tc.collectExplicitTypes(&n); err != nil {
			return err
		}
	case *ast.ElseIfNode:
		if n == nil {
			return nil
		}
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Storing scope for else if")
		tc.pushScope(n)

		for _, node := range n.Body {
			if err := tc.collectExplicitTypes(node); err != nil {
				return err
			}
		}

		tc.popScope()
	case *ast.ElseBlockNode:
		if n == nil {
			return nil
		}
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Storing scope for else block")
		tc.pushScope(n)

		for _, node := range n.Body {
			if err := tc.collectExplicitTypes(node); err != nil {
				return err
			}
		}

		tc.popScope()
	case ast.ElseBlockNode:
		eb := n
		return tc.collectExplicitTypes(&eb)
	}

	return nil
}

func (tc *TypeChecker) collectIfNode(n *ast.IfNode) error {
	tc.log.WithFields(logrus.Fields{
		"node":     n.String(),
		"function": "collectIfNode",
	}).Debug("Storing scope for if")
	tc.pushScope(n)

	for _, node := range n.Body {
		if err := tc.collectExplicitTypes(node); err != nil {
			return err
		}
	}

	for i := range n.ElseIfs {
		if err := tc.collectExplicitTypes(&n.ElseIfs[i]); err != nil {
			return err
		}
	}

	if n.Else != nil {
		if err := tc.collectExplicitTypes(n.Else); err != nil {
			return err
		}
	}

	tc.popScope()
	return nil
}

func (tc *TypeChecker) collectForNode(n *ast.ForNode) error {
	tc.log.WithFields(logrus.Fields{
		"function": "collectForNode",
	}).Debug("Storing scope for for-loop")
	tc.pushScope(n)

	if n.Init != nil {
		if err := tc.collectExplicitTypes(n.Init); err != nil {
			return err
		}
	}
	if n.Post != nil {
		if err := tc.collectExplicitTypes(n.Post); err != nil {
			return err
		}
	}

	for _, node := range n.Body {
		if err := tc.collectExplicitTypes(node); err != nil {
			return err
		}
	}

	tc.popScope()
	return nil
}

func (tc *TypeChecker) collectFunctionNode(scopeNode ast.Node, n ast.FunctionNode) error {
	tc.pushScope(scopeNode)
	tc.registerFunctionScope(n.Ident.ID, scopeNode)

	if n.Receiver != nil {
		if n.Receiver.Ident.ID != "" {
			tc.storeSymbol(n.Receiver.Ident.ID, []ast.TypeNode{n.Receiver.Type}, SymbolVariable)
		}
	}

	for _, param := range n.Params {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			tc.storeSymbol(p.Ident.ID, []ast.TypeNode{p.Type}, SymbolVariable)
		case ast.DestructuredParamNode:
			tc.registerDestructuredParamSymbols(p.Fields, p.Type, SymbolVariable)
		}
	}

	for _, bodyNode := range n.Body {
		if err := tc.collectExplicitTypes(bodyNode); err != nil {
			return err
		}
	}

	tc.registerFunction(n)
	tc.popScope()

	if n.Receiver == nil {
		tc.storeSymbol(n.Ident.ID, n.ReturnTypes, SymbolFunction)
	}
	return nil
}

func (tc *TypeChecker) collectTypeGuardNode(scopeNode ast.Node, n ast.TypeGuardNode) error {
	tc.pushScope(scopeNode)
	tc.registerTypeGuardScope(n.Ident, scopeNode)

	tc.globalScope().RegisterSymbol(n.Ident, []ast.TypeNode{{Ident: ast.TypeVoid}}, SymbolTypeGuard)

	for _, param := range n.Parameters() {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			tc.log.WithFields(logrus.Fields{
				"node":     p.String(),
				"function": "collectTypeGuardNode",
			}).Debug("Storing symbol for simple param of type guard")
			tc.storeSymbol(p.Ident.ID, []ast.TypeNode{p.Type}, SymbolParameter)
		case ast.DestructuredParamNode:
			tc.registerDestructuredParamSymbols(p.Fields, p.Type, SymbolParameter)
			tc.log.WithFields(logrus.Fields{
				"node":     p.String(),
				"function": "collectTypeGuardNode",
			}).Debug("Registered destructured type guard param fields")
		}
	}

	for _, bodyNode := range n.Body {
		if err := tc.collectExplicitTypes(bodyNode); err != nil {
			return err
		}
	}

	guard := new(ast.TypeGuardNode)
	*guard = n
	tc.registerTypeGuard(guard)

	tc.popScope()
	return nil
}

