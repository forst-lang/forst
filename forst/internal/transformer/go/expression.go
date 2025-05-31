package transformergo

import (
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
	"reflect"
	"strconv"
)

// negateCondition negates a condition
func negateCondition(condition goast.Expr) goast.Expr {
	return &goast.UnaryExpr{
		Op: token.NOT,
		X:  condition,
	}
}

// disjoin joins a list of conditions with OR ("any condition must match")
func disjoin(conditions []goast.Expr) goast.Expr {
	if len(conditions) == 0 {
		return &goast.Ident{Name: BoolConstantFalse}
	}
	combined := conditions[0]
	for i := 1; i < len(conditions); i++ {
		combined = &goast.BinaryExpr{
			X:  combined,
			Op: token.LOR,
			Y:  conditions[i],
		}
	}
	return combined
}

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
			Name: e.GetIdent(),
		}
	case ast.FunctionCallNode:
		args := make([]goast.Expr, len(e.Arguments))
		for i, arg := range e.Arguments {
			args[i] = transformExpression(arg)
		}
		return &goast.CallExpr{
			Fun:  goast.NewIdent(string(e.Function.ID)),
			Args: args,
		}
	}

	panic("Unsupported expression type: " + reflect.TypeOf(expr).String())
}
