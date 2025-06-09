package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	gotoken "go/token"
)

// transformConstraintArg transforms a constraint argument node into a Go expression
func transformConstraintArg(arg ast.ConstraintArgumentNode) (goast.Expr, error) {
	if arg.Type != nil {
		return goast.NewIdent(string(arg.Type.Ident)), nil
	} else if arg.Shape != nil {
		return goast.NewIdent("struct{}"), nil
	} else if arg.Value != nil {
		switch v := (*arg.Value).(type) {
		case ast.IntLiteralNode:
			return &goast.BasicLit{Kind: gotoken.INT, Value: fmt.Sprintf("%d", v.Value)}, nil
		case ast.StringLiteralNode:
			return &goast.BasicLit{Kind: gotoken.STRING, Value: fmt.Sprintf("%q", v.Value)}, nil
		case ast.BoolLiteralNode:
			return goast.NewIdent(fmt.Sprintf("%v", v.Value)), nil
		case ast.FloatLiteralNode:
			return &goast.BasicLit{Kind: gotoken.FLOAT, Value: fmt.Sprintf("%f", v.Value)}, nil
		default:
			return nil, fmt.Errorf("unsupported value type in constraint argument: %T", v)
		}
	}
	return nil, fmt.Errorf("unsupported constraint argument type")
}

// getEnsureBaseType returns the base type for an ensure node
func (t *Transformer) getEnsureBaseType(ensure ast.EnsureNode) (ast.TypeNode, error) {
	if ensure.Assertion.BaseType != nil {
		return ast.TypeNode{Ident: *ensure.Assertion.BaseType}, nil
	}
	ensureBaseType, err := t.TypeChecker.LookupEnsureBaseType(&ensure, t.currentScope())
	if err != nil {
		return ast.TypeNode{}, err
	}
	return *ensureBaseType, nil
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
