package typechecker

import "forst/internal/ast"

// Traverses the AST to gather type definitions and function signatures
func (tc *TypeChecker) collectExplicitTypes(node ast.Node) error {
	tc.log.Tracef("[collectExplicitTypes] Collecting explicit types for type %s", node.String())
	switch n := node.(type) {
	case ast.ImportNode:
		tc.log.Debugf("[collectExplicitTypes] Collecting import: %v", n)
		tc.imports = append(tc.imports, n)
	case ast.ImportGroupNode:
		tc.log.Debugf("[collectExplicitTypes] Collecting import group: %v", n)
		tc.imports = append(tc.imports, n.Imports...)
	case ast.TypeDefNode:
		tc.log.Debugf("[collectExplicitTypes] Collecting type definition: %v", n)
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
	case *ast.TypeGuardNode:
		tc.pushScope(n)

		tc.registerTypeGuard(n)

		tc.popScope()

		// Store type guard symbol in global scope
		tc.storeSymbol(n.Ident, []ast.TypeNode{{Ident: ast.TypeVoid}}, SymbolTypeGuard)
	case ast.EnsureNode:
		tc.log.Debugf("[collectExplicitTypes] Storing scope for ensure: %v", n)
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
	case ast.IfNode:
		tc.log.Debugf("[collectExplicitTypes] Storing scope for if: %v", n)
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
		tc.log.Debugf("[collectExplicitTypes] Storing scope for else if: %v", n)
		tc.pushScope(n)

		for _, node := range n.Body {
			if err := tc.collectExplicitTypes(node); err != nil {
				return err
			}
		}

		tc.popScope()
	case ast.ElseBlockNode:
		tc.log.Debugf("[collectExplicitTypes] Storing scope for else block: %v", n)
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
