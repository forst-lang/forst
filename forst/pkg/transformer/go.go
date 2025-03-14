package transformer

import (
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"
	"strconv"
)

// TransformForstToGo converts a Forst AST to a Go AST
func TransformForstFileToGo(nodes []ast.Node) *goast.File {
	var packageName string
	var decls []goast.Decl

	// Extract package name and declarations
	for _, node := range nodes {
		switch n := node.(type) {
		case ast.PackageNode:
			packageName = n.Value
		case ast.FunctionNode:
			decls = append(decls, transformFunction(n))
		}
	}

	if packageName == "" {
		packageName = "main"
	}

	// Create file node
	file := &goast.File{
		Name:  goast.NewIdent(packageName),
		Decls: decls,
	}

	return file
}

// transformFunction converts a Forst function node to a Go function declaration
func transformFunction(n ast.FunctionNode) *goast.FuncDecl {
	// Create function parameters
	params := &goast.FieldList{
		List: []*goast.Field{},
	}

	for _, param := range n.Params {
		params.List = append(params.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(param.Name)},
			Type:  goast.NewIdent(param.Type),
		})
	}

	// Create function return type
	var results *goast.FieldList
	if !n.ReturnType.IsImplicit() {
		results = &goast.FieldList{
			List: []*goast.Field{
				{
					Type: goast.NewIdent(n.ReturnType.Name),
				},
			},
		}
	}

	// Create function body statements
	stmts := []goast.Stmt{}

	for _, stmt := range n.Body {
		stmts = append(stmts, transformStatement(stmt))
	}

	// Create the function declaration
	return &goast.FuncDecl{
		Name: goast.NewIdent(n.Name),
		Type: &goast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: &goast.BlockStmt{
			List: stmts,
		},
	}
}

// transformStatement converts a Forst statement to a Go statement
func transformStatement(stmt ast.Node) goast.Stmt {
	switch s := stmt.(type) {
	case ast.EnsureNode:
		// Convert ensure to if statement with panic
		condition := parseExpr(s.Condition)
		notCondition := &goast.UnaryExpr{
			Op: token.NOT,
			X:  condition,
		}

		errorMsg := "assertion failed: " + s.Condition
		if s.ErrorType != "" {
			errorMsg = s.ErrorType
		}

		return &goast.IfStmt{
			Cond: notCondition,
			Body: &goast.BlockStmt{
				List: []goast.Stmt{
					&goast.ExprStmt{
						X: &goast.CallExpr{
							Fun: goast.NewIdent("panic"),
							Args: []goast.Expr{
								&goast.BasicLit{
									Kind:  token.STRING,
									Value: strconv.Quote(errorMsg),
								},
							},
						},
					},
				},
			},
		}
	case ast.ReturnNode:
		// Convert return statement
		return &goast.ReturnStmt{
			Results: []goast.Expr{
				parseExpr(s.Value),
			},
		}
	default:
		return &goast.EmptyStmt{}
	}
}

// Helper function to parse expressions from strings
// This is a simplified version - in a real implementation,
// you would need a proper expression parser
func parseExpr(expr string) goast.Expr {
	// For simplicity, just return an identifier
	// In a real implementation, you would parse the expression
	return goast.NewIdent(expr)
}