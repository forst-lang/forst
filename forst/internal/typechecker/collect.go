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
	case ast.ImportNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Collecting import")
		tc.imports = append(tc.imports, n)
	case ast.ImportGroupNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Collecting import group")
		tc.imports = append(tc.imports, n.Imports...)
	case ast.TypeDefNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Collecting type definition")
		tc.registerType(n)
	case ast.FunctionNode:
		tc.pushScope(n)

		for _, param := range n.Params {
			switch p := param.(type) {
			case ast.SimpleParamNode:
				tc.storeSymbol(p.Ident.ID, []ast.TypeNode{p.Type}, SymbolVariable)
			case ast.DestructuredParamNode:
				// TODO: Handle destructured params
				continue
			}
		}

		for _, node := range n.Body {
			if err := tc.collectExplicitTypes(node); err != nil {
				return err
			}
		}

		tc.registerFunction(n)
		tc.popScope()

		// Store function symbol
		tc.storeSymbol(n.Ident.ID, n.ReturnTypes, SymbolFunction)
	case *ast.FunctionNode:
		return tc.collectExplicitTypes(*n)
	case ast.TypeGuardNode:
		tc.pushScope(n)

		// Register type guard in Defs
		tc.Defs[ast.TypeIdent(n.Ident)] = n

		// Register type guard symbol in global scope
		tc.globalScope().RegisterSymbol(n.Ident, []ast.TypeNode{{Ident: ast.TypeVoid}}, SymbolTypeGuard)

		// Register parameters in the current scope
		for _, param := range n.Parameters() {
			switch p := param.(type) {
			case ast.SimpleParamNode:
				tc.log.WithFields(logrus.Fields{
					"node":     p.String(),
					"function": "collectExplicitTypes",
				}).Debug("Storing symbol for simple param of type guard")
				tc.storeSymbol(p.Ident.ID, []ast.TypeNode{p.Type}, SymbolParameter)
			case ast.DestructuredParamNode:
				// Handle destructured params if needed
				tc.log.WithFields(logrus.Fields{
					"node":     p.String(),
					"function": "collectExplicitTypes",
				}).Warn("Destructured params are not supported for type guards yet")
				continue
			}
		}

		// Recursively collect explicit types for each node in the body
		for _, node := range n.Body {
			if err := tc.collectExplicitTypes(node); err != nil {
				return err
			}
		}

		// Register the type guard in the type checker
		tc.registerTypeGuard(&n)

		tc.popScope()
	case *ast.TypeGuardNode:
		tc.collectExplicitTypes(*n)
	case ast.EnsureNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Storing scope for ensure")
		tc.pushScope(n)

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
	case ast.IfNode:
		tc.log.WithFields(logrus.Fields{
			"node":     n.String(),
			"function": "collectExplicitTypes",
		}).Debug("Storing scope for if")
		tc.pushScope(n)

		for _, node := range n.Body {
			if err := tc.collectExplicitTypes(node); err != nil {
				return err
			}
		}

		for _, node := range n.ElseIfs {
			if err := tc.collectExplicitTypes(node); err != nil {
				return err
			}
		}

		if n.Else != nil {
			if err := tc.collectExplicitTypes(n.Else); err != nil {
				return err
			}
		}

		tc.popScope()
	case ast.ElseIfNode:
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
	case ast.ElseBlockNode:
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
	}

	return nil
}
