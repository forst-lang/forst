package transformer_go

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
)

type Transformer struct {
	TypeChecker  *typechecker.TypeChecker
	currentScope *typechecker.Scope

	Output *TransformerOutput
}

func New(tc *typechecker.TypeChecker) *Transformer {
	return &Transformer{
		TypeChecker:  tc,
		currentScope: tc.GlobalScope(),
		Output:       &TransformerOutput{},
	}
}

// TransformForstFileToGo converts a Forst AST to a Go AST
// The nodes should already have their types inferred/checked
func (t *Transformer) TransformForstFileToGo(nodes []ast.Node) (*goast.File, error) {
	// First, collect and register shape types from type definitions
	if err := t.defineShapeTypes(); err != nil {
		return nil, err
	}

	for _, def := range t.TypeChecker.Defs {
		switch def := def.(type) {
		case ast.TypeDefNode:
			decl, err := t.transformTypeDef(def)
			if err != nil {
				return nil, fmt.Errorf("failed to transform type def %s: %w", def.Ident, err)
			}
			t.Output.AddType(decl)
		case *ast.TypeGuardNode:
			decl, err := t.transformTypeGuard(*def)
			if err != nil {
				return nil, fmt.Errorf("failed to transform type guard %s: %w", def.Ident, err)
			}
			t.Output.AddFunction(decl)
		}
	}

	for _, node := range nodes {
		switch n := node.(type) {
		case ast.PackageNode:
			t.Output.SetPackageName(string(n.Ident.Id))
		case ast.ImportNode:
			decl := t.transformImport(n)
			t.Output.AddImport(decl)
		case ast.ImportGroupNode:
			decl := t.transformImportGroup(n)
			t.Output.AddImportGroup(decl)
		case ast.FunctionNode:
			decl, err := t.transformFunction(n)
			if err != nil {
				return nil, fmt.Errorf("failed to transform function %s: %w", n.Ident.Id, err)
			}
			t.Output.AddFunction(decl)
		}
	}

	return t.Output.GenerateFile()
}

func (t *Transformer) currentFunction() (ast.FunctionNode, error) {
	scope := t.currentScope
	for scope != nil && !scope.IsFunction() && scope.Parent != nil {
		scope = scope.Parent
	}
	if scope.Node == nil {
		return ast.FunctionNode{}, fmt.Errorf("no function found")
	}
	return scope.Node.(ast.FunctionNode), nil
}

func (t *Transformer) isMainPackage() bool {
	return t.Output.PackageName() == "main"
}

func (t *Transformer) isMainFunction() bool {
	if !t.isMainPackage() {
		return false
	}

	currentFunction, err := t.currentFunction()
	if err != nil {
		return false
	}
	return currentFunction.HasMainFunctionName()
}

func (t *Transformer) transformTypeGuard(n ast.TypeGuardNode) (*goast.FuncDecl, error) {
	// Create function parameters
	params := &goast.FieldList{
		List: []*goast.Field{},
	}

	for _, param := range n.Parameters {
		var paramName string
		var paramType ast.TypeNode

		switch p := param.(type) {
		case ast.SimpleParamNode:
			paramName = string(p.Ident.Id)
			paramType = p.Type
		case ast.DestructuredParamNode:
			// Handle destructured params if needed
			continue
		}

		ident, err := t.transformType(paramType)
		if err != nil {
			return nil, fmt.Errorf("failed to transform type: %s", err)
		}
		params.List = append(params.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(paramName)},
			Type:  ident,
		})
	}

	// Create function body statements
	stmts := []goast.Stmt{}

	for _, stmt := range n.Body {
		goStmt := t.transformStatement(stmt)
		stmts = append(stmts, goStmt)
	}

	// Create the function declaration
	return &goast.FuncDecl{
		Recv: nil,
		Name: goast.NewIdent(string(n.Ident)),
		Type: &goast.FuncType{
			Params:  params,
			Results: &goast.FieldList{List: []*goast.Field{{Type: goast.NewIdent("bool")}}},
		},
		Body: &goast.BlockStmt{
			List: stmts,
		},
	}, nil
}
