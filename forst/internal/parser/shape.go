package parser

import (
	"fmt"
	"forst/internal/ast"

	log "github.com/sirupsen/logrus"
)

func (p *Parser) parseShape() ast.ShapeNode {
	p.expect(ast.TokenLBrace)

	fields := make(map[string]ast.ShapeFieldNode)
	// Parse fields until closing brace
	for p.current().Type != ast.TokenRBrace {
		// Parse field name
		name := p.expect(ast.TokenIdentifier).Value

		// If the next token is a colon, parse the value
		if p.current().Type == ast.TokenColon {
			p.advance() // Consume the colon

			// Parse field value (can be another shape, a type assertion, or a variable reference)
			var value ast.ShapeFieldNode
			if p.current().Type == ast.TokenLBrace {
				shape := p.parseShape()
				value = ast.ShapeFieldNode{
					Shape: &shape,
				}
			} else if p.current().Type == ast.TokenStar {
				// Handle pointer type
				p.advance() // consume *
				baseType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
				pointerType := ast.TypeIdent(ast.TypePointer)
				value = ast.ShapeFieldNode{
					Assertion: &ast.AssertionNode{
						BaseType: &pointerType,
						Constraints: []ast.ConstraintNode{{
							Name: "Pointer",
							Args: []ast.ConstraintArgumentNode{{
								Shape: &ast.ShapeNode{
									Fields: map[string]ast.ShapeFieldNode{
										"type": {
											Assertion: &ast.AssertionNode{
												BaseType: &baseType.Ident,
											},
										},
									},
								},
							}},
						}},
					},
				}
			} else if p.current().Type == ast.TokenIdentifier {
				// Handle variable reference
				ident := p.expect(ast.TokenIdentifier)
				typeIdent := ast.TypeIdent(ident.Value)
				value = ast.ShapeFieldNode{
					Assertion: &ast.AssertionNode{
						BaseType: &typeIdent,
					},
				}
			} else {
				assertion := p.parseAssertionChain(true)
				log.Trace(fmt.Sprintf("Parsed assertion chain: %s", assertion))
				value = ast.ShapeFieldNode{
					Assertion: &assertion,
				}
			}
			fields[name] = value
		} else {
			// If no colon, use the field name as both key and value
			typeIdent := ast.TypeIdent(name)
			fields[name] = ast.ShapeFieldNode{
				Assertion: &ast.AssertionNode{
					BaseType: &typeIdent,
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
		panic(parseErrorMessage(p.current(), "Shape type must have at least one field. Empty shapes are not allowed."))
	}

	return ast.ShapeNode{Fields: fields}
}
