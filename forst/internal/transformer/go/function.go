package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
)

func (t *Transformer) transformFunctionParams(params []ast.ParamNode) (*goast.FieldList, error) {
	t.log.Debugf("transformFunctionParams: processing %d parameters", len(params))

	fields := &goast.FieldList{
		List: []*goast.Field{},
	}

	for i, param := range params {
		var paramName string
		var paramType ast.TypeNode

		switch p := param.(type) {
		case ast.SimpleParamNode:
			paramName = string(p.Ident.ID)
			paramType = p.Type
		case ast.DestructuredParamNode:
			// Handle destructured params if needed
			continue
		}

		// Add debug output for parameter type
		t.log.Debugf("transformFunctionParams: param %d '%s' has type %q", i, param.GetIdent(), paramType.Ident)

		// Look up the inferred type from the type checker
		var inferredTypes []ast.TypeNode
		var err error

		if paramType.Assertion != nil {
			// For assertion types, check if we should preserve the original type name
			// If the assertion has a base type that's a user-defined type AND no constraints, use that
			if paramType.Assertion.BaseType != nil && len(paramType.Assertion.Constraints) == 0 {
				baseType := *paramType.Assertion.BaseType
				// Check if the base type is a user-defined type (not a hash-based type)
				baseTypeNode := ast.TypeNode{Ident: baseType}
				if !baseTypeNode.IsHashBased() {
					// Use the original type name instead of inferring a hash-based type
					name, err := t.TypeChecker.GetAliasedTypeName(baseTypeNode)
					if err != nil {
						return nil, fmt.Errorf("failed to get aliased type name for parameter %s: %w", paramName, err)
					}
					fields.List = append(fields.List, &goast.Field{
						Names: []*goast.Ident{goast.NewIdent(paramName)},
						Type:  goast.NewIdent(name),
					})
					continue
				}
			}

			// For other assertion types (with constraints), use the inferred type from the type checker
			inferredTypes, err = t.TypeChecker.InferAssertionType(paramType.Assertion, false, "", nil)
			if err != nil {
				return nil, fmt.Errorf("failed to infer assertion type for parameter %s: %w", paramName, err)
			}
		} else {
			// For non-assertion types, use the original type
			inferredTypes = []ast.TypeNode{paramType}
		}

		// Use the first inferred type (should be only one for parameters)
		if len(inferredTypes) == 0 {
			return nil, fmt.Errorf("no inferred type found for parameter %s", paramName)
		}

		actualType := inferredTypes[0]
		name, err := t.TypeChecker.GetAliasedTypeName(actualType)
		if err != nil {
			return nil, fmt.Errorf("failed to get aliased type name for parameter %s: %w", paramName, err)
		}

		fields.List = append(fields.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(paramName)},
			Type:  goast.NewIdent(name),
		})
	}

	return fields, nil
}

// transformFunction converts a Forst function node to a Go function declaration
func (t *Transformer) transformFunction(n ast.FunctionNode) (*goast.FuncDecl, error) {
	if err := t.restoreScope(n); err != nil {
		return nil, fmt.Errorf("failed to restore function scope: %s", err)
	}

	// Create function parameters
	params, err := t.transformFunctionParams(n.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to transform function parameters: %s", err)
	}

	// Create function return type
	returnType, err := t.TypeChecker.LookupFunctionReturnType(&n)
	if err != nil {
		return nil, err
	}
	var results *goast.FieldList
	isMainFunc := t.isMainPackage() && n.HasMainFunctionName()
	if !isMainFunc {
		results, err = t.transformTypes(returnType)
		if err != nil {
			return nil, fmt.Errorf("failed to transform types: %s", err)
		}
	}

	// Create function body statements
	stmts := []goast.Stmt{}

	for _, stmt := range n.Body {
		if err := t.restoreScope(n); err != nil {
			return nil, fmt.Errorf("failed to restore function scope in body: %s", err)
		}

		goStmt, err := t.transformStatement(stmt)
		if err != nil {
			return nil, fmt.Errorf("failed to transform statement: %s", err)
		}
		stmts = append(stmts, goStmt)
	}

	// Make sure that functions return nil if they return an error
	if !isMainFunc && len(returnType) > 0 {
		lastReturnType := returnType[len(returnType)-1]
		if lastReturnType.IsError() {
			var lastStmt ast.Node
			if len(n.Body) > 0 {
				lastStmt = n.Body[len(n.Body)-1]
			}
			if lastStmt == nil || lastStmt.Kind() != ast.NodeKindReturn {
				stmts = append(stmts, &goast.ReturnStmt{
					Results: []goast.Expr{
						goast.NewIdent("nil"),
					},
				})
			}
		}
	}

	// Create the function declaration
	return &goast.FuncDecl{
		Name: goast.NewIdent(n.Ident.String()),
		Type: &goast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: &goast.BlockStmt{
			List: stmts,
		},
	}, nil
}
