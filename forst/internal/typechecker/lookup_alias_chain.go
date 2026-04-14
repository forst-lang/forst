package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

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
			// Try inferred types if no explicit alias is found.
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

// GetMostSpecificNonHashAlias returns the first non-hash-based type in the alias chain,
// or the original type if all are hash-based.
func (tc *TypeChecker) GetMostSpecificNonHashAlias(typeNode ast.TypeNode) ast.TypeNode {
	chain := tc.GetTypeAliasChain(typeNode)
	for _, t := range chain {
		if !t.IsHashBased() {
			return t
		}
	}
	// If all are hash-based, return the first in the chain.
	return chain[0]
}
