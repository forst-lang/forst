package parser

import (
	"fmt"
	"forst/pkg/ast"
)

const MAX_EXPRESSION_DEPTH = 20

func (p *Parser) parseExpression(context *Context) ast.ExpressionNode {
	return p.parseExpressionLevel(0, context)
}

func (p *Parser) parseExpressionLevel(level int, context *Context) ast.ExpressionNode {
	if level > MAX_EXPRESSION_DEPTH {
		panic(parseErrorWithValue(
			p.current(),
			fmt.Sprintf("Expression level too deep - maximum nesting depth is %d", MAX_EXPRESSION_DEPTH),
		))
	}

	var expr ast.ExpressionNode

	// Handle unary not operator
	if p.current().Type == ast.TokenLogicalNot {
		p.advance() // Consume the not operator
		operand := p.parseExpressionLevel(level+1, context)
		expr = ast.UnaryExpressionNode{
			Operator: ast.TokenLogicalNot,
			Operand:  operand,
			Type:     ast.TypeNode{Name: ast.TypeBool},
		}
		return expr
	}

	// Handle parentheses
	if p.current().Type == ast.TokenLParen {
		p.advance() // Consume the left parenthesis
		expr = p.parseExpressionLevel(level+1, context)
		p.expect(ast.TokenRParen) // Consume the right parenthesis
	} else {
		expr = p.parseValue(context) // parseValue should advance the token internally
	}

	// Handle binary operators
	for p.current().Type.IsBinaryOperator() {
		operator := p.current().Type
		p.advance() // Consume the operator

		right := p.parseExpressionLevel(level+1, context)
		// Get the types of the left and right expressions
		var leftType ast.TypeNode
		var rightType ast.TypeNode

		switch l := expr.(type) {
		case ast.BinaryExpressionNode:
			leftType = l.Type
		case ast.UnaryExpressionNode:
			leftType = l.Type
		case ast.IntLiteralNode:
			leftType = ast.TypeNode{Name: ast.TypeInt}
		case ast.FloatLiteralNode:
			leftType = ast.TypeNode{Name: ast.TypeFloat}
		case ast.StringLiteralNode:
			leftType = ast.TypeNode{Name: ast.TypeString}
		case ast.BoolLiteralNode:
			leftType = ast.TypeNode{Name: ast.TypeBool}
		}

		switch r := right.(type) {
		case ast.BinaryExpressionNode:
			rightType = r.Type
		case ast.UnaryExpressionNode:
			rightType = r.Type
		case ast.IntLiteralNode:
			rightType = ast.TypeNode{Name: ast.TypeInt}
		case ast.FloatLiteralNode:
			rightType = ast.TypeNode{Name: ast.TypeFloat}
		case ast.StringLiteralNode:
			rightType = ast.TypeNode{Name: ast.TypeString}
		case ast.BoolLiteralNode:
			rightType = ast.TypeNode{Name: ast.TypeBool}
		}

		// Check type compatibility and set result type
		var resultType ast.TypeNode
		if operator.IsArithmeticBinaryOperator() {
			if leftType.Name != rightType.Name {
				panic(parseErrorWithValue(
					p.current(),
					"Type mismatch in arithmetic expression",
				))
			}
			resultType = leftType
		} else if operator.IsComparisonBinaryOperator() {
			if leftType.Name != rightType.Name {
				panic(parseErrorWithValue(
					p.current(),
					"Type mismatch in comparison expression",
				))
			}
			resultType = ast.TypeNode{Name: ast.TypeBool}
		} else if operator.IsLogicalBinaryOperator() {
			if leftType.Name != rightType.Name {
				panic(parseErrorWithValue(
					p.current(),
					"Type mismatch in logical expression",
				))
			}
			resultType = ast.TypeNode{Name: ast.TypeBool}
		}

		expr = ast.BinaryExpressionNode{
			Left:     expr,
			Operator: operator,
			Right:    right,
			Type:     resultType,
		}
	}

	return expr
}
