package parser

import (
	"fmt"
	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// parseShapeMemberName reads a shape field or method name. The `error` keyword is allowed
// when it starts a method signature (`error(msg String)`).
func (p *Parser) parseShapeMemberName() string {
	tok := p.current()
	switch tok.Type {
	case ast.TokenIdentifier:
		p.advance()
		return tok.Value
	case ast.TokenError:
		if p.peek().Type == ast.TokenLParen {
			p.advance()
			return tok.Value
		}
		p.FailWithParseError(tok, "expected shape member name")
	default:
		p.FailWithParseError(tok, "expected shape member name")
	}
	panic("unreachable")
}

func (p *Parser) parseShapeMethodSignature() ([]ast.ParamNode, []ast.TypeNode) {
	p.expect(ast.TokenLParen)
	params := []ast.ParamNode{}
	if p.current().Type != ast.TokenRParen {
		for {
			params = append(params, p.parseParameter())
			if p.current().Type == ast.TokenComma {
				p.advance()
			} else {
				break
			}
		}
	}
	p.expect(ast.TokenRParen)
	var returnTypes []ast.TypeNode
	if p.current().Type == ast.TokenColon {
		returnTypes = p.parseReturnType()
	}
	return params, returnTypes
}

func (p *Parser) parseShapeTypeField(name string) ast.ShapeFieldNode {
	if p.current().Type == ast.TokenLParen {
		params, returnTypes := p.parseShapeMethodSignature()
		return ast.ShapeFieldNode{
			IsMethod:          true,
			MethodParams:      params,
			MethodReturnTypes: returnTypes,
		}
	}
	switch p.current().Type {
	case ast.TokenColon:
		p.advance() // Consume the colon

		// If the next token is a nested shape type
		if p.current().Type == ast.TokenLBrace {
			shape := p.parseShapeType()
			// For shape types, nested shapes should be stored as Type with Assertion
			return ast.ShapeFieldNode{
				Type: &ast.TypeNode{
					Ident: ast.TypeShape,
					Assertion: &ast.AssertionNode{
						BaseType: nil,
						Constraints: []ast.ConstraintNode{{
							Name: "Shape",
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
		// For shape types, nested shapes should be stored as Type with Assertion
		return ast.ShapeFieldNode{
			Type: &ast.TypeNode{
				Ident: ast.TypeShape,
				Assertion: &ast.AssertionNode{
					BaseType: nil,
					Constraints: []ast.ConstraintNode{{
						Name: "Shape",
						Args: []ast.ConstraintArgumentNode{{
							Shape: &shape,
						}},
					}},
				},
			},
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
	return p.parseShapeTypeInternal(false)
}

func (p *Parser) parseShapeTypeAllowEmpty() ast.ShapeNode {
	return p.parseShapeTypeInternal(true)
}

func (p *Parser) parseShapeTypeInternal(allowEmpty bool) ast.ShapeNode {
	p.log.WithField("token", p.current()).Trace("Entering parseShapeType")
	p.expect(ast.TokenLBrace)

	fields := make(map[string]ast.ShapeFieldNode)
	var fieldOrder []string
	// Parse fields until closing brace
	for p.current().Type != ast.TokenRBrace {
		p.log.WithField("token", p.current()).Trace("parseShapeType: parsing field")
		// Parse field name
		name := p.parseShapeMemberName()

		fields[name] = p.parseShapeTypeField(name)
		fieldOrder = append(fieldOrder, name)

		p.log.WithField("token", p.current()).Trace("parseShapeType: after field parse")
		// Handle commas between fields
		if p.current().Type == ast.TokenComma {
			p.advance() // Consume the comma
		}
	}

	p.expect(ast.TokenRBrace)

	if len(fields) == 0 && !allowEmpty {
		p.FailWithParseError(p.current(), "Shape type must have at least one field. Empty shapes are not allowed.")
	}

	baseType := ast.TypeIdent(ast.TypeShape)
	return ast.ShapeNode{
		Fields:     fields,
		FieldOrder: fieldOrder,
		BaseType:   &baseType,
	}
}

// parseShapeTypeForError parses `{ ... }` for `error Name { ... }`. Empty payloads are allowed
// (e.g. `error RateLimited {}` for `RateLimited()` on `ensure … or`).
func (p *Parser) parseShapeTypeForError() ast.ShapeNode {
	p.expect(ast.TokenLBrace)
	fields := make(map[string]ast.ShapeFieldNode)
	var fieldOrder []string
	for p.current().Type != ast.TokenRBrace {
		name := p.parseShapeMemberName()
		fields[name] = p.parseShapeTypeField(name)
		fieldOrder = append(fieldOrder, name)
		if p.current().Type == ast.TokenComma {
			p.advance()
		}
	}
	p.expect(ast.TokenRBrace)
	baseType := ast.TypeIdent(ast.TypeShape)
	return ast.ShapeNode{
		Fields:     fields,
		FieldOrder: fieldOrder,
		BaseType:   &baseType,
	}
}

// parseShapeLiteral parses a shape literal value or type
// baseType is the optional base type that this shape extends
// parseAsTypes indicates whether to parse field values as type annotations (true) or as literal values (false).
// When parseAsTypes is true, field values are parsed as type declarations (e.g. String, Int, {name: String}).
// When parseAsTypes is false, field values are parsed as literal values (e.g. "hello", 42, {name: "Alice"}).
func (p *Parser) parseShapeLiteral(baseType *ast.TypeIdent, parseAsTypes bool) ast.ShapeNode {
	p.log.WithFields(logrus.Fields{
		"function":     "parseShapeLiteral",
		"baseType":     baseType,
		"parseAsTypes": parseAsTypes,
	}).Debug("Starting parseShapeLiteral")

	p.log.WithField("token", p.current()).Trace("Entering parseShapeLiteral")
	p.expect(ast.TokenLBrace)

	fields := make(map[string]ast.ShapeFieldNode)
	var fieldOrder []string
	// Parse fields until closing brace
	for p.current().Type != ast.TokenRBrace {
		p.log.WithField("token", p.current()).Trace("parseShapeLiteral: parsing field")
		// Parse field name
		name := p.parseShapeMemberName()

		// If the next token is a colon, parse the value or type
		if p.current().Type == ast.TokenColon {
			p.advance() // Consume the colon

			if parseAsTypes {
				// Parse as type annotation (like parseShapeTypeField)
				if p.current().Type == ast.TokenLBrace {
					// For nested shapes in type contexts, parse as shape literal with parseAsTypes=true
					shape := p.parseShapeLiteral(nil, true)
					fields[name] = ast.ShapeFieldNode{
						Shape: &shape,
					}
				} else if isPossibleTypeIdentifier(p.current(), TypeIdentOpts{AllowLowercaseTypes: false}) || p.current().Type == ast.TokenStar {
					typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
					typeIdent := typ.Ident
					p.logParsedNodeWithMessage(typ, fmt.Sprintf("Parsed type for shape field %s and type ident %s (type: %+v)", name, typeIdent, typ))
					fields[name] = ast.ShapeFieldNode{
						Type: &typ,
					}
				} else {
					p.FailWithParseError(p.current(), "Expected type annotation in shape type context")
				}
			} else {
				// Parse as value (literal context)
				val := p.parseValue()
				p.log.WithFields(logrus.Fields{
					"fieldName": name,
					"valType":   fmt.Sprintf("%T", val),
					"valValue":  fmt.Sprintf("%+v", val),
				}).Debug("Parsed value for shape field")

				valNode, ok := val.(ast.Node)
				if !ok {
					p.log.WithFields(logrus.Fields{
						"fieldName": name,
						"valType":   fmt.Sprintf("%T", val),
						"valValue":  fmt.Sprintf("%+v", val),
					}).Error("Value does not implement ast.Node")
					panic(fmt.Sprintf("parseShapeLiteral: value for field '%s' does not implement ast.Node: type=%T value=%+v", name, val, val))
				}
				var field ast.ShapeFieldNode
				switch v := val.(type) {
				case ast.ShapeNode:
					field = ast.ShapeFieldNode{Node: valNode, Shape: &v}
					p.log.WithFields(logrus.Fields{
						"fieldName": name,
						"nodeSet":   true,
					}).Debug("Created shape field with Node and Shape")
				default:
					field = ast.ShapeFieldNode{Node: valNode}
					p.log.WithFields(logrus.Fields{
						"fieldName": name,
						"nodeSet":   true,
					}).Debug("Created shape field with Node")
				}

				// For backward compatibility, also set the Type field for variable references
				if varNode, ok := val.(ast.VariableNode); ok {
					field.Type = &ast.TypeNode{
						Ident: ast.TypeIdent(string(varNode.Ident.ID)),
					}
					p.log.WithFields(logrus.Fields{
						"fieldName": name,
						"typeSet":   true,
						"typeIdent": string(varNode.Ident.ID),
					}).Debug("Set Type field for variable reference")
				}
				if _, isShape := val.(ast.ShapeNode); !isShape {
					field.Assertion = &ast.AssertionNode{
						BaseType: nil,
						Constraints: []ast.ConstraintNode{{
							Name: string(ast.ValueConstraint),
							Args: []ast.ConstraintArgumentNode{{
								Value: &val,
							}},
						}},
					}
				}
				fields[name] = field
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
		fieldOrder = append(fieldOrder, name)

		p.log.WithField("token", p.current()).Trace("parseShapeLiteral: after field parse")
		// Handle commas between fields
		if p.current().Type == ast.TokenComma {
			p.advance() // Consume the comma
		}
	}

	p.expect(ast.TokenRBrace)

	return ast.ShapeNode{
		Fields:     fields,
		FieldOrder: fieldOrder,
		BaseType:   baseType,
	}
}
