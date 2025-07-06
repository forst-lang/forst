package parser

import (
	"fmt"
	"forst/internal/ast"
)

func (p *Parser) parseShapeTypeField(name string) ast.ShapeFieldNode {
	switch p.current().Type {
	case ast.TokenColon:
		p.advance() // Consume the colon

		// If the next token is a nested shape type
		if p.current().Type == ast.TokenLBrace {
			shape := p.parseShapeType()
			baseType := ast.TypeIdent(ast.TypeShape)
			return ast.ShapeFieldNode{
				Type: &ast.TypeNode{
					Ident: ast.TypeShape,
					Assertion: &ast.AssertionNode{
						BaseType: &baseType,
						Constraints: []ast.ConstraintNode{{
							Name: "Match",
							Args: []ast.ConstraintArgumentNode{{
								Shape: &shape,
							}},
						}},
					},
				},
			}
		}

		// Otherwise, parse as a type annotation
		if isPossibleTypeIdentifier(p.current(), TypeIdentOpts{AllowLowercaseTypes: false}) || p.current().Type == ast.TokenStar {
			typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
			typeIdent := typ.Ident
			p.logParsedNodeWithMessage(typ, fmt.Sprintf("Parsed type for shape field %s and type ident %s (type: %+v)", name, typeIdent, typ))
			return ast.ShapeFieldNode{
				Type: &typ,
			}
		}
	case ast.TokenStar:
		// Handle pointer types
		fieldType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
		return ast.ShapeFieldNode{
			Type: &fieldType,
		}
	case ast.TokenLBrace:
		shape := p.parseShapeType()
		return ast.ShapeFieldNode{
			Shape: &shape,
		}
	}
	// If no colon, use the field name as both key and value (type assertion)
	typeIdent := ast.TypeIdent(name)
	return ast.ShapeFieldNode{
		Type: &ast.TypeNode{
			Ident: typeIdent,
		},
	}
}

func (p *Parser) parseShapeType() ast.ShapeNode {
	p.log.WithField("token", p.current()).Trace("Entering parseShapeType")
	p.expect(ast.TokenLBrace)

	fields := make(map[string]ast.ShapeFieldNode)
	// Parse fields until closing brace
	for p.current().Type != ast.TokenRBrace {
		p.log.WithField("token", p.current()).Trace("parseShapeType: parsing field")
		// Parse field name
		name := p.expect(ast.TokenIdentifier).Value

		fields[name] = p.parseShapeTypeField(name)

		p.log.WithField("token", p.current()).Trace("parseShapeType: after field parse")
		// Handle commas between fields
		if p.current().Type == ast.TokenComma {
			p.advance() // Consume the comma
		}
	}

	p.expect(ast.TokenRBrace)

	// Require at least one field in shape definitions
	if len(fields) == 0 {
		p.FailWithParseError(p.current(), "Shape type must have at least one field. Empty shapes are not allowed.")
	}

	baseType := ast.TypeIdent(ast.TypeShape)
	return ast.ShapeNode{
		Fields:   fields,
		BaseType: &baseType,
	}
}

// parseShapeLiteral parses a shape literal value
func (p *Parser) parseShapeLiteral(baseType *ast.TypeIdent) ast.ShapeNode {
	p.log.WithField("token", p.current()).Trace("Entering parseShapeLiteral")
	p.expect(ast.TokenLBrace)

	fields := make(map[string]ast.ShapeFieldNode)
	// Parse fields until closing brace
	for p.current().Type != ast.TokenRBrace {
		p.log.WithField("token", p.current()).Trace("parseShapeLiteral: parsing field")
		// Parse field name
		name := p.expect(ast.TokenIdentifier).Value

		// If the next token is a colon, parse the value
		if p.current().Type == ast.TokenColon {
			p.advance() // Consume the colon

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
							Name: string(ast.ValueConstraint),
							Args: []ast.ConstraintArgumentNode{{
								Value: &val,
							}},
						}},
					},
				}
			}
			fields[name] = field
		} else {
			// If no colon, use the field name as both key and value (type assertion)
			typeIdent := ast.TypeIdent(name)
			fields[name] = ast.ShapeFieldNode{
				Type: &ast.TypeNode{
					Ident: typeIdent,
				},
			}
		}

		p.log.WithField("token", p.current()).Trace("parseShapeLiteral: after field parse")
		// Handle commas between fields
		if p.current().Type == ast.TokenComma {
			p.advance() // Consume the comma
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
