package transformergo

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
)

func (t *Transformer) transformFunctionParamField(paramName string, paramType ast.TypeNode) (*goast.Field, error) {
	var inferredTypes []ast.TypeNode
	var err error

	if paramType.Assertion != nil {
		if paramType.Assertion.BaseType != nil && len(paramType.Assertion.Constraints) == 0 {
			baseType := *paramType.Assertion.BaseType
			baseTypeNode := ast.TypeNode{Ident: baseType}
			if !baseTypeNode.IsHashBased() {
				name, err := t.TypeChecker.GetAliasedTypeName(baseTypeNode, typechecker.GetAliasedTypeNameOptions{AllowStructuralAlias: true})
				if err != nil {
					return nil, fmt.Errorf("failed to get aliased type name for parameter %s: %w", paramName, err)
				}
				return &goast.Field{
					Names: []*goast.Ident{goast.NewIdent(paramName)},
					Type:  goast.NewIdent(name),
				}, nil
			}
		}
		inferredTypes, err = t.TypeChecker.InferAssertionType(paramType.Assertion, false, "", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to infer assertion type for parameter %s: %w", paramName, err)
		}
	} else {
		inferredTypes = []ast.TypeNode{paramType}
	}

	if len(inferredTypes) == 0 {
		return nil, fmt.Errorf("no inferred type found for parameter %s", paramName)
	}

	typeExpr, err := t.transformType(inferredTypes[0])
	if err != nil {
		return nil, fmt.Errorf("failed to transform type for parameter %s: %w", paramName, err)
	}

	return &goast.Field{
		Names: []*goast.Ident{goast.NewIdent(paramName)},
		Type:  typeExpr,
	}, nil
}

func (t *Transformer) transformFunctionParams(params []ast.ParamNode) (*goast.FieldList, error) {
	t.log.Debugf("transformFunctionParams: processing %d parameters", len(params))

	fields := &goast.FieldList{
		List: []*goast.Field{},
	}

	for i, param := range params {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			t.log.Debugf("transformFunctionParams: param %d '%s' has type %q", i, p.Ident.ID, p.Type.Ident)
			field, err := t.transformFunctionParamField(string(p.Ident.ID), p.Type)
			if err != nil {
				return nil, err
			}
			fields.List = append(fields.List, field)
		case ast.DestructuredParamNode:
			shapeFields, ok := t.TypeChecker.ShapeFieldsFromParamType(p.Type)
			if !ok {
				return nil, fmt.Errorf("destructured parameter has no shape fields in type %s", p.Type.Ident)
			}
			for _, fieldName := range p.Fields {
				sf, ok := shapeFields[fieldName]
				if !ok {
					return nil, fmt.Errorf("destructured field %s not found in parameter type", fieldName)
				}
				fieldType, ok := typechecker.ShapeFieldTypeNode(sf)
				if !ok {
					return nil, fmt.Errorf("destructured field %s has no type", fieldName)
				}
				t.log.Debugf("transformFunctionParams: destructured field %s has type %q", fieldName, fieldType.Ident)
				field, err := t.transformFunctionParamField(fieldName, fieldType)
				if err != nil {
					return nil, err
				}
				fields.List = append(fields.List, field)
			}
		default:
			return nil, fmt.Errorf("unsupported parameter type %T", param)
		}
	}

	return fields, nil
}

// transformFunction converts a Forst function node to a Go function declaration
func (t *Transformer) transformFunction(scopeNode ast.Node, n ast.FunctionNode) (*goast.FuncDecl, error) {
	if err := t.restoreScope(scopeNode); err != nil {
		return nil, fmt.Errorf("failed to restore function scope: %s", err)
	}

	prevResultSplit := t.resultLocalSplit
	t.resultLocalSplit = make(map[string]resultLocalSplit)
	defer func() { t.resultLocalSplit = prevResultSplit }()

	prevMapIndexCache := t.mapIndexFuncLitCache
	t.mapIndexFuncLitCache = make(map[string]*goast.FuncLit)
	defer func() { t.mapIndexFuncLitCache = prevMapIndexCache }()

	// Create function parameters
	params, err := t.transformFunctionParams(n.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to transform function parameters: %s", err)
	}
	var providersParamName string
	params, providersParamName, err = t.prependProvidersParam(params, n)
	if err != nil {
		return nil, fmt.Errorf("failed to prepend providers parameter: %w", err)
	}

	prevProvidersName := t.currentFnProvidersName
	prevProvidersSlots := t.currentFnProvidersSlots
	if t.functionNeedsProvidersParam(n) {
		t.currentFnProvidersName = providersParamName
		t.currentFnProvidersSlots = t.TypeChecker.FunctionProviders[n.Ident.ID]
	} else {
		t.currentFnProvidersName = ""
		t.currentFnProvidersSlots = nil
	}
	defer func() {
		t.currentFnProvidersName = prevProvidersName
		t.currentFnProvidersSlots = prevProvidersSlots
	}()

	// Create function return type
	returnType, err := t.TypeChecker.LookupFunctionReturnType(&n)
	if err != nil {
		return nil, err
	}
	var results *goast.FieldList
	isMainFunc := t.isMainPackage() && n.HasMainFunctionName()
	if !isMainFunc && !typechecker.IsVoidReturnTypes(returnType) {
		results, err = t.transformTypes(returnType)
		if err != nil {
			return nil, fmt.Errorf("failed to transform types: %s", err)
		}
	}

	implicitIdx, hasTrailingExpr := lastImplicitReturnIndex(n.Body)
	emitImplicitReturn := hasTrailingExpr && !isMainFunc && !typechecker.IsVoidReturnTypes(returnType)

	// Create function body statements
	stmts := []goast.Stmt{}

	for i, stmt := range n.Body {
		if emitImplicitReturn && i == implicitIdx {
			continue
		}
		if err := t.restoreScope(scopeNode); err != nil {
			return nil, fmt.Errorf("failed to restore function scope in body: %s", err)
		}

		switch s := stmt.(type) {
		case ast.UseNode:
			goStmt, err := t.transformUseStatement(s)
			if err != nil {
				return nil, fmt.Errorf("failed to transform use statement: %w", err)
			}
			if _, ok := goStmt.(*goast.EmptyStmt); ok && s.Ident == nil {
				continue
			}
			stmts = append(stmts, goStmt)
		case ast.WithNode:
			withStmts, err := t.transformWithStatements(stmt, s)
			if err != nil {
				return nil, fmt.Errorf("failed to transform with block: %w", err)
			}
			stmts = append(stmts, withStmts...)
		default:
			goStmt, err := t.transformStatement(stmt)
			if err != nil {
				return nil, fmt.Errorf("failed to transform statement: %s", err)
			}
			stmts = append(stmts, goStmt)
		}
	}

	if emitImplicitReturn {
		implicitExpr, ok := n.Body[implicitIdx].(ast.ExpressionNode)
		if !ok {
			return nil, fmt.Errorf("internal error: implicit return index is not an expression")
		}
		retStmt, err := t.transformStatement(ast.ReturnNode{
			Values: []ast.ExpressionNode{implicitExpr},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to transform implicit return: %w", err)
		}
		stmts = append(stmts, retStmt)
	}

	var recv *goast.FieldList
	if n.Receiver != nil {
		recvType, err := t.transformType(n.Receiver.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to transform receiver type: %w", err)
		}
		var names []*goast.Ident
		if n.Receiver.Ident.ID != "" {
			names = []*goast.Ident{goast.NewIdent(string(n.Receiver.Ident.ID))}
		}
		recv = &goast.FieldList{
			List: []*goast.Field{{Names: names, Type: recvType}},
		}
	}

	// Make sure that functions return nil if they return an error
	if !isMainFunc && !typechecker.IsVoidReturnTypes(returnType) && len(returnType) > 0 {
		lastReturnType := returnType[len(returnType)-1]
		if lastReturnType.IsError() {
			var lastStmt ast.Node
			for i := len(n.Body) - 1; i >= 0; i-- {
				if _, ok := n.Body[i].(ast.CommentNode); ok {
					continue
				}
				lastStmt = n.Body[i]
				break
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
		Recv: recv,
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
