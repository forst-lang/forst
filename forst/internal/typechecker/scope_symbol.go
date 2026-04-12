// Scope stack operations and symbol registration during collect/infer passes.
package typechecker

import (
	"fmt"
	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// DebugPrintCurrentScope prints details about symbols defined in the current scope
func (tc *TypeChecker) DebugPrintCurrentScope() {
	currentScope := tc.scopeStack.currentScope()
	if currentScope == nil {
		tc.log.Debug("Current scope is nil")
		return
	}

	if currentScope.Node == nil {
		tc.log.Debug("Current scope node is nil")
	} else {
		tc.log.WithFields(logrus.Fields{
			"scope": currentScope.String(),
			"addr":  fmt.Sprintf("%p", currentScope),
		}).Debug("Current scope")
	}

	tc.log.WithFields(logrus.Fields{
		"total": len(currentScope.Symbols),
	}).Debug("Defined symbols")

	for _, symbol := range currentScope.Symbols {
		tc.log.Debugf("    %s: %s\n", symbol.Identifier, symbol.Types)
	}
}

// globalScope returns the root scope
func (tc *TypeChecker) globalScope() *Scope {
	return tc.scopeStack.globalScope()
}

// pushScope creates a new scope for the given node
// Intended for use in the collection pass of the typechecker, not the transformer
func (tc *TypeChecker) pushScope(node ast.Node) *Scope {
	scope := tc.scopeStack.pushScope(node)

	tc.log.WithFields(logrus.Fields{
		"scope":    scope.String(),
		"addr":     fmt.Sprintf("%p", scope),
		"function": "pushScope",
	}).Debug("Pushed scope")
	return scope
}

// popScope removes the current scope and returns to the parent scope
// Intended for use in the collection pass of the typechecker, not the transformer
func (tc *TypeChecker) popScope() {
	currentScope := tc.CurrentScope()
	tc.scopeStack.popScope()

	tc.log.WithFields(logrus.Fields{
		"scope":    currentScope.String(),
		"addr":     fmt.Sprintf("%p", currentScope),
		"function": "popScope",
	}).Debug("Popped scope")
}

// RestoreScope restores the scope for a given node
// Intended for use after the collection pass of the typechecker has completed
func (tc *TypeChecker) RestoreScope(node ast.Node) error {
	return tc.scopeStack.restoreScope(node)
}

// storeSymbol stores a symbol definition in the current scope
func (tc *TypeChecker) storeSymbol(ident ast.Identifier, types []ast.TypeNode, kind SymbolKind) {
	// Ensure types have correct TypeKind
	processedTypes := make([]ast.TypeNode, len(types))
	for i, typ := range types {
		// For user-defined types, ensure they're marked as user-defined
		if typ.TypeKind != ast.TypeKindHashBased && !tc.isBuiltinType(typ.Ident) {
			processedTypes[i] = ensureUserDefinedType(typ)
		} else {
			processedTypes[i] = typ
		}
	}

	currentScope := tc.CurrentScope()
	currentScope.Symbols[ident] = Symbol{
		Identifier: ident,
		Types:      processedTypes,
		Kind:       kind,
		Scope:      currentScope,
		Position:   tc.path,
	}

	tc.log.WithFields(logrus.Fields{
		"ident":    ident,
		"types":    processedTypes,
		"kind":     kind,
		"function": "storeSymbol",
	}).Trace("Stored symbol")
}

// CurrentScope returns the current scope
func (tc *TypeChecker) CurrentScope() *Scope {
	return tc.scopeStack.currentScope()
}
