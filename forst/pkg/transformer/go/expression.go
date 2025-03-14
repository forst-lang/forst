package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"
	"reflect"
	"strconv"
)

func transformExpression(expr ast.ExpressionNode) goast.Expr {
	switch e := expr.(type) {
	case ast.IntLiteralNode:
		return &goast.BasicLit{
			Kind:  token.INT,
			Value: strconv.FormatInt(e.Value, 10),
		}
	case ast.FloatLiteralNode:
		return &goast.BasicLit{
			Kind:  token.FLOAT,
			Value: strconv.FormatFloat(e.Value, 'f', -1, 64),
		}
	case ast.StringLiteralNode:
		return &goast.BasicLit{
			Kind:  token.STRING,
			Value: e.Value,
		}
	case ast.BoolLiteralNode:
		if e.Value {
			return goast.NewIdent("true")
		}
		return goast.NewIdent("false")
	}

	panic("Unsupported expression type: " + reflect.TypeOf(expr).String())
}
