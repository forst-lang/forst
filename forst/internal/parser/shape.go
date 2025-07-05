package parser

import (
	"fmt"
	"forst/internal/ast"
)

// parseShape parses a shape literal value
func (p *Parser) parseShape(baseType *ast.TypeIdent) ast.ShapeNode {
	p.expect(ast.TokenLBrace)

	fields := make(map[string]ast.ShapeFieldNode)
	// Parse fields until closing brace
	for p.current().Type != ast.TokenRBrace {
		// Parse field name
		name := p.expect(ast.TokenIdentifier).Value

		// If the next token is a colon, parse the value
		if p.current().Type == ast.TokenColon {
			p.advance() // Consume the colon

			// If the next token is a type, parse as a type annotation
			if isPossibleTypeIdentifier(p.current(), TypeIdentOpts{AllowLowercaseTypes: false}) || p.current().Type == ast.TokenStar {
				typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
				typeIdent := typ.Ident
				p.logParsedNodeWithMessage(typ, fmt.Sprintf("Parsed type for shape field %s and type ident %s (type: %+v)", name, typeIdent, typ))
				fields[name] = ast.ShapeFieldNode{
					Type: &typ,
				}
			} else {
				// Otherwise, parse as a value
				val := p.parseValue()
				var field ast.ShapeFieldNode
				switch v := val.(type) {
				case ast.ShapeNode:
					field = ast.ShapeFieldNode{Shape: &v}
				default:
					// For all other value types, wrap in an AssertionNode as a constraint argument
					field = ast.ShapeFieldNode{
						Assertion: &ast.AssertionNode{
							BaseType: nil,
							Constraints: []ast.ConstraintNode{{
								Name: "Value",
								Args: []ast.ConstraintArgumentNode{{
									Value: &val,
								}},
							}},
						},
					}
				}
				fields[name] = field
			}
		} else if p.current().Type == ast.TokenStar {
			// Handle pointer types (legacy, for robustness)
			fieldType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
			fields[name] = ast.ShapeFieldNode{
				Type: &fieldType,
			}
		} else {
			// If no colon, use the field name as both key and value (type assertion)
			typeIdent := ast.TypeIdent(name)
			fields[name] = ast.ShapeFieldNode{
				Type: &ast.TypeNode{
					Ident: typeIdent,
				},
			}
		}

		// Handle commas between fields
		if !isParenthesis(p.current()) && p.current().Type != ast.TokenRBrace {
			p.expect(ast.TokenComma)
		}
	}

	p.expect(ast.TokenRBrace)

	// Require at least one field in shape definitions
	if len(fields) == 0 {
		p.FailWithParseError(p.current(), "Shape type must have at least one field. Empty shapes are not allowed.")
	}

	return ast.ShapeNode{
		Fields:   fields,
		BaseType: baseType,
	}
}
