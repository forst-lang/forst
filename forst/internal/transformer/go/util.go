package transformergo

import (
	"fmt"
	"forst/internal/ast"
)

// Helper to check if a type name is a Go built-in type
func isGoBuiltinType(typeName string) bool {
	switch typeName {
	case "string", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "complex64", "complex128", "bool", "byte", "rune", "error", "void":
		return true
	default:
		return false
	}
}

// getTypeAliasChain returns the chain of type aliases for a given type, ending with the base type.
func (t *Transformer) getTypeAliasChain(typeNode ast.TypeNode) []ast.TypeNode {
	chain := []ast.TypeNode{typeNode}
	visited := map[ast.TypeIdent]bool{typeNode.Ident: true}
	current := typeNode
	t.log.WithFields(map[string]interface{}{
		"typeNode": current.Ident,
		"function": "getTypeAliasChain",
	}).Debug("Starting type alias chain resolution")
	for {
		def, exists := t.TypeChecker.Defs[current.Ident]
		if !exists {
			// Try inferred types if no explicit alias is found
			hash, err := t.TypeChecker.Hasher.HashNode(current)
			if err == nil {
				if inferred, ok := t.TypeChecker.Types[hash]; ok && len(inferred) > 0 {
					inferredType := inferred[0]
					if !visited[inferredType.Ident] {
						t.log.WithFields(map[string]interface{}{
							"typeNode": inferredType.Ident,
							"function": "getTypeAliasChain",
						}).Debug("Following inferred type in alias chain")
						chain = append(chain, inferredType)
						visited[inferredType.Ident] = true
						current = inferredType
						continue
					}
				}
			}
			t.log.WithFields(map[string]interface{}{
				"typeNode": current.Ident,
				"function": "getTypeAliasChain",
			}).Debug("No definition or inferred type found for type")
			break
		}
		typeDef, ok := def.(ast.TypeDefNode)
		if !ok {
			t.log.WithFields(map[string]interface{}{
				"typeNode": current.Ident,
				"function": "getTypeAliasChain",
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
			t.log.WithFields(map[string]interface{}{
				"typeNode":  current.Ident,
				"exprType":  fmt.Sprintf("%T", typeDef.Expr),
				"exprValue": fmt.Sprintf("%#v", typeDef.Expr),
				"function":  "getTypeAliasChain",
			}).Debug("Definition is not a valid type alias")
			break
		}
		baseIdent := *assertionExpr.Assertion.BaseType
		if visited[baseIdent] {
			t.log.WithFields(map[string]interface{}{
				"typeNode": current.Ident,
				"function": "getTypeAliasChain",
			}).Debug("Cycle detected in type alias chain")
			break
		}
		baseType := ast.TypeNode{Ident: baseIdent}
		chain = append(chain, baseType)
		visited[baseIdent] = true
		current = baseType
		t.log.WithFields(map[string]interface{}{
			"typeNode": current.Ident,
			"function": "getTypeAliasChain",
		}).Debug("Added base type to alias chain")
	}
	return chain
}
