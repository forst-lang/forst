package typechecker

import (
	"fmt"

	"forst/internal/ast"

	log "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) LookupInferredType(node ast.Node, requireInferred bool) ([]ast.TypeNode, error) {
	hash := tc.Hasher.HashNode(node)
	if existingType, exists := tc.Types[hash]; exists {
		// Ignore types that are still marked as implicit, as they are not yet inferred
		if len(existingType) == 0 {
			if requireInferred {
				return nil, fmt.Errorf("expected type of node to have been inferred, found: implicit type")
			}
			return nil, nil
		}
		return existingType, nil
	}
	if requireInferred {
		return nil, fmt.Errorf("expected type of node to have been inferred, found: no registered type")
	}
	return nil, nil
}

// lookupVariableType finds a variable's type in the current scope chain
func (tc *TypeChecker) LookupVariableType(variable *ast.VariableNode, currentScope *Scope) (ast.TypeNode, error) {
	log.Tracef("Looking up variable type for %s in scope %s", variable.Ident.ID, currentScope.Node)
	symbol, exists := currentScope.LookupVariable(variable.Ident.ID)
	if !exists {
		err := fmt.Errorf("undefined symbol: %s", variable.Ident.ID)
		log.WithError(err).Error("lookup symbol failed")
		return ast.TypeNode{}, err
	}
	if len(symbol.Types) != 1 {
		err := fmt.Errorf("expected single type for variable %s but got %d types", variable.Ident.ID, len(symbol.Types))
		log.WithError(err).Error("lookup symbol failed")
		return ast.TypeNode{}, err
	}
	return symbol.Types[0], nil
}

func (tc *TypeChecker) LookupFunctionReturnType(function *ast.FunctionNode, currentScope *Scope) ([]ast.TypeNode, error) {
	sig, exists := tc.Functions[function.Ident.ID]
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
