package typechecker

import (
	"fmt"

	"forst/pkg/ast"

	log "github.com/sirupsen/logrus"
)

func findAlreadyInferredType(tc *TypeChecker, node ast.Node) ([]ast.TypeNode, error) {
	hash := tc.Hasher.HashNode(node)
	if existingType, exists := tc.Types[hash]; exists {
		// Ignore types that are still marked as implicit, as they are not yet inferred
		if len(existingType) > 0 {
			return nil, nil
		}
		return existingType, nil
	}
	return nil, nil
}

// lookupVariableType finds a variable's type in the current scope chain
func (tc *TypeChecker) LookupVariableType(variable *ast.VariableNode, currentScope *Scope) (ast.TypeNode, error) {
	symbol, err := tc.lookupSymbol(variable.Ident.Id, currentScope)
	if err != nil {
		return ast.TypeNode{}, err
	}
	if len(symbol.Types) != 1 {
		err := fmt.Errorf("expected single type for variable %s but got %d types", variable.Ident.String(), len(symbol.Types))
		log.WithError(err).Error("lookup variable type failed")
		return ast.TypeNode{}, err
	}
	return symbol.Types[0], nil
}

func (tc *TypeChecker) lookupSymbol(ident ast.Identifier, currentScope *Scope) (*Symbol, error) {
	for scope := currentScope; scope != nil; scope = scope.Parent {
		if sym, exists := scope.Symbols[ident]; exists {
			return &sym, nil
		}
	}
	err := fmt.Errorf("undefined symbol: %s", ident)
	log.WithError(err).Error("lookup symbol failed")
	return nil, err
}

func (tc *TypeChecker) LookupFunctionReturnType(function *ast.FunctionNode, currentScope *Scope) ([]ast.TypeNode, error) {
	sig, exists := tc.Functions[function.Id()]
	if !exists {
		err := fmt.Errorf("undefined function: %s", function.Ident)
		log.WithError(err).Error("lookup function return type failed")
		return nil, err
	}
	return sig.ReturnTypes, nil
}

func (tc *TypeChecker) LookupEnsureBaseType(ensure *ast.EnsureNode, currentScope *Scope) (*ast.TypeNode, error) {
	baseType, err := tc.LookupVariableType(&ensure.Variable, currentScope)
	if err != nil {
		return nil, err
	}
	return &baseType, nil
}

func (tc *TypeChecker) LookupAssertionType(assertion *ast.AssertionNode, currentScope *Scope) (*ast.TypeNode, error) {
	hash := tc.Hasher.HashNode(assertion)
	if existingType, exists := tc.Types[hash]; exists {
		if len(existingType) != 1 {
			err := fmt.Errorf("expected single type for assertion %s but got %d types", hash.ToTypeIdent(), len(existingType))
			log.WithError(err).Error("lookup assertion type failed")
			return nil, err
		}
		log.Trace(fmt.Sprintf("existingType: %s", existingType))
		return &existingType[0], nil
	}
	typeNode := &ast.TypeNode{
		Ident:     hash.ToTypeIdent(),
		Assertion: assertion,
	}
	log.Trace(fmt.Sprintf("Storing new looked up assertion type: %s", typeNode.Ident))
	tc.storeInferredType(assertion, []ast.TypeNode{*typeNode})
	return typeNode, nil
}
