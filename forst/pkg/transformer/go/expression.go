package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"
	"reflect"
	"strconv"
)

func transformOperator(op ast.TokenIdent) token.Token {
	switch op {
	case ast.TokenPlus:
		return token.ADD
	case ast.TokenMinus:
		return token.SUB
	case ast.TokenMultiply:
		return token.MUL
	case ast.TokenDivide:
		return token.QUO
	case ast.TokenModulo:
		return token.REM
	case ast.TokenEquals:
		return token.EQL
	case ast.TokenNotEquals:
		return token.NEQ
	case ast.TokenGreater:
		return token.GTR
	case ast.TokenLess:
		return token.LSS
	case ast.TokenGreaterEqual:
		return token.GEQ
	case ast.TokenLessEqual:
		return token.LEQ
	case ast.TokenLogicalAnd:
		return token.LAND
	case ast.TokenLogicalOr:
		return token.LOR
	case ast.TokenLogicalNot:
		return token.NOT
	}

	panic("Unsupported operator: " + op)
}

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
			Value: strconv.Quote(e.Value),
		}
	case ast.BoolLiteralNode:
		if e.Value {
			return goast.NewIdent("true")
		}
		return goast.NewIdent("false")
	case ast.UnaryExpressionNode:
		return &goast.UnaryExpr{
			Op: transformOperator(e.Operator),
			X:  transformExpression(e.Operand),
		}
	case ast.BinaryExpressionNode:
		return &goast.BinaryExpr{
			X:  transformExpression(e.Left),
			Op: transformOperator(e.Operator),
			Y:  transformExpression(e.Right),
		}
	case ast.VariableNode:
		return &goast.Ident{
			Name: e.Id(),
		}
	case ast.FunctionCallNode:
		args := make([]goast.Expr, len(e.Arguments))
		for i, arg := range e.Arguments {
			args[i] = transformExpression(arg)
		}
		return &goast.CallExpr{
			Fun:  goast.NewIdent(string(e.Function.Id)),
			Args: args,
		}
	}

	panic("Unsupported expression type: " + reflect.TypeOf(expr).String())
}
