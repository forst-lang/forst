package transformergo

import (
	"fmt"

	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

// transformMapIndexResultCall lowers a map read m[k] typed as Result(V, Error) to an IIFE that uses
// Go's comma-ok internally and returns (V, error).
func (t *Transformer) transformMapIndexResultCall(e ast.IndexExpressionNode, resultT ast.TypeNode) (goast.Expr, error) {
	if !resultT.IsResultType() || len(resultT.TypeParams) < 1 {
		return nil, fmt.Errorf("transformMapIndexResultCall: expected Result")
	}
	succT := resultT.TypeParams[0]
	tgt, err := t.transformExpression(e.Target)
	if err != nil {
		return nil, err
	}
	idx, err := t.transformExpression(e.Index)
	if err != nil {
		return nil, err
	}
	velGo, err := t.transformType(succT)
	if err != nil {
		return nil, err
	}
	zero, err := t.goZeroValueGoAST(succT)
	if err != nil {
		return nil, err
	}
	t.Output.EnsureImport("errors")

	errorsNew := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   goast.NewIdent("errors"),
			Sel: goast.NewIdent("New"),
		},
		Args: []goast.Expr{&goast.BasicLit{Kind: token.STRING, Value: `"missing map key"`}},
	}

	// v, ok := m[k]
	assign := &goast.AssignStmt{
		Lhs: []goast.Expr{goast.NewIdent("v"), goast.NewIdent("ok")},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{&goast.IndexExpr{X: tgt, Index: idx}},
	}
	ifNotOk := &goast.IfStmt{
		Cond: &goast.UnaryExpr{Op: token.NOT, X: goast.NewIdent("ok")},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.ReturnStmt{Results: []goast.Expr{zero, errorsNew}},
		}},
	}
	retOk := &goast.ReturnStmt{Results: []goast.Expr{goast.NewIdent("v"), goast.NewIdent("nil")}}

	fn := &goast.FuncLit{
		Type: &goast.FuncType{
			Params: &goast.FieldList{},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: velGo},
					{Type: goast.NewIdent("error")},
				},
			},
		},
		Body: &goast.BlockStmt{List: []goast.Stmt{assign, ifNotOk, retOk}},
	}
	return &goast.CallExpr{Fun: fn}, nil
}

// goZeroValueGoAST returns a zero value expression for common Forst types used as map values.
func (t *Transformer) goZeroValueGoAST(vt ast.TypeNode) (goast.Expr, error) {
	switch vt.Ident {
	case ast.TypeInt:
		return &goast.BasicLit{Kind: token.INT, Value: "0"}, nil
	case ast.TypeFloat:
		return &goast.BasicLit{Kind: token.FLOAT, Value: "0"}, nil
	case ast.TypeString:
		return &goast.BasicLit{Kind: token.STRING, Value: `""`}, nil
	case ast.TypeBool:
		return goast.NewIdent("false"), nil
	default:
		gt, err := t.transformType(vt)
		if err != nil {
			return nil, err
		}
		return &goast.CompositeLit{Type: gt}, nil
	}
}
