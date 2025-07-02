package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

// LookupInferredType looks up the inferred type of a node in the current scope
func (tc *TypeChecker) LookupInferredType(node ast.Node, requireInferred bool) ([]ast.TypeNode, error) {
	hash, err := tc.Hasher.HashNode(node)
	if err != nil {
		return nil, fmt.Errorf("failed to hash node during LookupInferredType: %s", err)
	}
	if existingType, exists := tc.Types[hash]; exists {
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

// LookupVariableType finds a variable's type in the current scope chain
func (tc *TypeChecker) LookupVariableType(variable *ast.VariableNode, scope *Scope) (ast.TypeNode, error) {
	tc.log.Tracef("Looking up variable type for %s in scope %s", variable.Ident.ID, scope.String())

	parts := strings.Split(string(variable.Ident.ID), ".")
	baseIdent := ast.Identifier(parts[0])

	symbol, exists := scope.LookupVariable(baseIdent)
	if !exists {
		return ast.TypeNode{}, fmt.Errorf("undefined symbol: %s [scope: %s]", parts[0], scope.String())
	}

	if len(symbol.Types) != 1 {
		return ast.TypeNode{}, fmt.Errorf("expected single type for variable %s but got %d types", parts[0], len(symbol.Types))
	}

	if len(parts) == 1 {
		return symbol.Types[0], nil
	}

	// Use lookupFieldPath for multi-segment field access
	return tc.lookupFieldPath(symbol.Types[0], parts[1:])
}

// LookupEnsureBaseType looks up the base type of an ensure node in a given scope
func (tc *TypeChecker) LookupEnsureBaseType(ensure *ast.EnsureNode, scope *Scope) (*ast.TypeNode, error) {
	baseType, err := tc.LookupVariableType(&ensure.Variable, scope)
	if err != nil {
		return nil, err
	}
	return &baseType, nil
}

// GetTypeAliasChain returns the chain of type aliases for a given type, ending with the base type.
func (tc *TypeChecker) GetTypeAliasChain(typeNode ast.TypeNode) []ast.TypeNode {
	chain := []ast.TypeNode{typeNode}
	visited := map[ast.TypeIdent]bool{typeNode.Ident: true}
	current := typeNode
	tc.log.WithFields(map[string]interface{}{
		"typeNode": current.Ident,
		"function": "GetTypeAliasChain",
	}).Debug("Starting type alias chain resolution")
	for {
		def, exists := tc.Defs[current.Ident]
		if !exists {
			// Try inferred types if no explicit alias is found
			hash, err := tc.Hasher.HashNode(current)
			if err == nil {
				if inferred, ok := tc.Types[hash]; ok && len(inferred) > 0 {
					inferredType := inferred[0]
					if !visited[inferredType.Ident] {
						tc.log.WithFields(map[string]interface{}{
							"typeNode": inferredType.Ident,
							"function": "GetTypeAliasChain",
						}).Debug("Following inferred type in alias chain")
						chain = append(chain, inferredType)
						visited[inferredType.Ident] = true
						current = inferredType
						continue
					}
				}
			}
			tc.log.WithFields(map[string]interface{}{
				"typeNode": current.Ident,
				"function": "GetTypeAliasChain",
			}).Debug("No definition or inferred type found for type")
			break
		}
		typeDef, ok := def.(ast.TypeDefNode)
		if !ok {
			tc.log.WithFields(map[string]interface{}{
				"typeNode": current.Ident,
				"function": "GetTypeAliasChain",
			}).Debug("Definition is not a TypeDefNode")
			break
		}
		var assertionExpr *ast.TypeDefAssertionExpr
		switch expr := typeDef.Expr.(type) {
		case ast.TypeDefAssertionExpr:
			assertionExpr = &expr
		case *ast.TypeDefAssertionExpr:
			assertionExpr = expr
		}
		if assertionExpr == nil || assertionExpr.Assertion == nil || assertionExpr.Assertion.BaseType == nil {
			tc.log.WithFields(map[string]interface{}{
				"typeNode":  current.Ident,
				"exprType":  fmt.Sprintf("%T", typeDef.Expr),
				"exprValue": fmt.Sprintf("%#v", typeDef.Expr),
				"function":  "GetTypeAliasChain",
			}).Debug("Definition is not a valid type alias")
			break
		}
		baseIdent := *assertionExpr.Assertion.BaseType
		if visited[baseIdent] {
			tc.log.WithFields(map[string]interface{}{
				"typeNode": current.Ident,
				"function": "GetTypeAliasChain",
			}).Debug("Cycle detected in type alias chain")
			break
		}
		baseType := ast.TypeNode{Ident: baseIdent}
		chain = append(chain, baseType)
		visited[baseIdent] = true
		current = baseType
		tc.log.WithFields(map[string]interface{}{
			"typeNode": current.Ident,
			"function": "GetTypeAliasChain",
		}).Debug("Added base type to alias chain")
	}
	return chain
}
