package parser

import "forst/pkg/ast"

func (p *Parser) parseExpression() ast.ExpressionNode {
	return p.parseExpressionLevel(0)
}

func (p *Parser) parseExpressionLevel(level int) ast.ExpressionNode {
	if level > 20 {
		panic("Expression level too deep - maximum nesting depth is 20")
	}

	var expr ast.ExpressionNode

	// Handle unary not operator
	if p.current().Type == ast.TokenLogicalNot {
		p.advance() // Consume the not operator
		operand := p.parseExpressionLevel(level + 1)
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
		expr = p.parseExpressionLevel(level + 1)
		p.expect(ast.TokenRParen) // Consume the right parenthesis
	} else {
		expr = p.parseLiteral() // parseLiteral should advance the token internally
	}

	// Handle binary operators
	for p.current().Type == ast.TokenPlus || p.current().Type == ast.TokenMinus ||
		p.current().Type == ast.TokenMultiply || p.current().Type == ast.TokenDivide ||
		p.current().Type == ast.TokenModulo || p.current().Type == ast.TokenEquals ||
		p.current().Type == ast.TokenNotEquals || p.current().Type == ast.TokenGreater ||
		p.current().Type == ast.TokenLess || p.current().Type == ast.TokenGreaterEqual ||
		p.current().Type == ast.TokenLessEqual || p.current().Type == ast.TokenLogicalAnd ||
		p.current().Type == ast.TokenLogicalOr {

		operator := p.current().Type
		p.advance() // Consume the operator

		right := p.parseExpressionLevel(level + 1)
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
		switch operator {
		case ast.TokenPlus, ast.TokenMinus, ast.TokenMultiply, ast.TokenDivide, ast.TokenModulo:
			if leftType.Name != rightType.Name {
				panic("Type mismatch in arithmetic expression")
			}
			resultType = leftType
		case ast.TokenEquals, ast.TokenNotEquals, ast.TokenGreater, ast.TokenLess, ast.TokenGreaterEqual, ast.TokenLessEqual:
			if leftType.Name != rightType.Name {
				panic("Type mismatch in comparison expression")
			}
			resultType = ast.TypeNode{Name: ast.TypeBool}
		case ast.TokenLogicalAnd, ast.TokenLogicalOr:
			if leftType.Name != ast.TypeBool || rightType.Name != ast.TypeBool {
				panic("Logical operators require boolean operands")
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
