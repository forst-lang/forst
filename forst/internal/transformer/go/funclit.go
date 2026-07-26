package transformergo

import (
	"fmt"

	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
)

func (t *Transformer) transformFunctionLiteral(scopeNode ast.Node, lit ast.FunctionLiteralNode) (*goast.FuncLit, error) {
	if err := t.restoreScope(scopeNode); err != nil {
		return nil, fmt.Errorf("restore function literal scope: %w", err)
	}

	params, err := t.transformFunctionParams(ast.Identifier("_lit"), lit.Params)
	if err != nil {
		return nil, fmt.Errorf("transform function literal params: %w", err)
	}

	var results *goast.FieldList
	if len(lit.ReturnTypes) > 0 && !typechecker.IsVoidReturnTypes(lit.ReturnTypes) {
		results, err = t.transformTypes(lit.ReturnTypes)
		if err != nil {
			return nil, fmt.Errorf("transform function literal results: %w", err)
		}
	}

	stmts := []goast.Stmt{}
	for _, stmt := range lit.Body {
		goStmt, err := t.transformStatement(stmt)
		if err != nil {
			return nil, fmt.Errorf("transform function literal body: %w", err)
		}
		if goStmt != nil {
			stmts = append(stmts, goStmt)
		}
	}

	return &goast.FuncLit{
		Type: &goast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: &goast.BlockStmt{List: stmts},
	}, nil
}

func (t *Transformer) transformFunctionType(n ast.TypeNode) (*goast.FuncType, error) {
	params, err := t.transformFunctionParams(ast.Identifier("_fn"), n.FuncParams)
	if err != nil {
		return nil, err
	}
	var results *goast.FieldList
	if len(n.FuncReturns) > 0 && !typechecker.IsVoidReturnTypes(n.FuncReturns) {
		results, err = t.transformTypes(n.FuncReturns)
		if err != nil {
			return nil, err
		}
	}
	return &goast.FuncType{
		Params:  params,
		Results: results,
	}, nil
}
