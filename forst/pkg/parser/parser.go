package parser

import (
	"fmt"

	"forst/pkg/ast"
)

// Parser struct to keep track of tokens
type Parser struct {
	tokens       []ast.Token
	currentIndex int
}

// Create a new parser instance
func NewParser(tokens []ast.Token) *Parser {
	return &Parser{tokens: tokens, currentIndex: 0}
}

// Get the current token
func (p *Parser) current() ast.Token {
	if p.currentIndex < len(p.tokens) {
		return p.tokens[p.currentIndex]
	}
	return ast.Token{Type: ast.TokenEOF, Value: ""}
}

// Advance to the next token
func (p *Parser) advance() ast.Token {
	p.currentIndex++
	return p.current()
}

// Expect a token and advance
func (p *Parser) expect(tokenType ast.TokenType) ast.Token {
	token := p.current()
	if token.Type != tokenType {
		panic(fmt.Sprintf(`
Parse error in %s:%d:%d (line %d, column %d):
Expected token type '%s' but got '%s'
Token value: '%s'`,
			token.Path,
			token.Line,
			token.Column,
			token.Line,
			token.Column,
			tokenType,
			token.Type,
			token.Value,
		))
	}
	p.advance()
	return token
}

// Parse function parameters
func (p *Parser) parseFunctionSignature() []ast.ParamNode {
	p.expect(ast.TokenLParen)
	params := []ast.ParamNode{}

	// Handle empty parameter list
	if p.current().Type == ast.TokenRParen {
		p.advance()
		return params
	}

	// Parse parameters
	for {
		name := p.expect(ast.TokenIdentifier)
		p.expect(ast.TokenColon)
		paramType := p.expect(ast.TokenIdentifier)

		params = append(params, ast.ParamNode{
			Name: name.Value,
			Type: paramType.Value,
		})

		// Check if there are more parameters
		if p.current().Type == ast.TokenComma {
			p.advance()
		} else {
			break
		}
	}

	p.expect(ast.TokenRParen)
	return params
}

// Parse a function definition
func (p *Parser) parseFunctionDefinition() ast.FunctionNode {
	p.expect(ast.TokenFunction)           // Expect `fn`
	name := p.expect(ast.TokenIdentifier) // Function name

	params := p.parseFunctionSignature() // Parse function parameters

	p.expect(ast.TokenColon)                          // Expect `:` separating return type
	returnType := p.expect(ast.TokenIdentifier).Value // Return type

	p.expect(ast.TokenLBrace) // Expect `{`
	body := []ast.Node{}

	// Parse function body dynamically
	for p.current().Type != ast.TokenRBrace && p.current().Type != ast.TokenEOF {
		token := p.current()

		if token.Type == ast.TokenEnsure {
			p.advance() // Move past `ensure`
			condition := p.expect(ast.TokenIdentifier).Value
			p.expect(ast.TokenOr) // Expect `or`
			errorType := p.expect(ast.TokenIdentifier).Value
			body = append(body, ast.EnsureNode{Condition: condition, ErrorType: errorType})
		} else if token.Type == ast.TokenReturn {
			p.advance() // Move past `return`
			// TODO: Handle other return types
			value := p.expect(ast.TokenString).Value
			returnNode := ast.ReturnNode{Value: value, Type: ast.TypeNode{Name: "string"}}
			body = append(body, returnNode)
		} else {
			token := p.current()
			panic(fmt.Sprintf(
				"\nParse error in %s:%d:%d at line %d, column %d:\n"+
					"Unexpected token in function body: '%s'\n"+
					"Token value: '%s'",
				token.Path,
				token.Line,
				token.Column,
				token.Line,
				token.Column,
				token.Type,
				token.Value,
			))
		}
	}

	p.expect(ast.TokenRBrace) // Expect `}`

	returnTypeNode := ast.TypeNode{Name: returnType}
	return ast.FunctionNode{
		Name:       name.Value,
		ReturnType: returnTypeNode,
		Params:     params,
		Body:       body,
	}
}

// Parse the tokens into an AST
func (p *Parser) Parse() []ast.Node {
	nodes := []ast.Node{}

	for p.current().Type != ast.TokenEOF {
		if p.current().Type == ast.TokenPackage {
			p.advance() // Move past `package`
			packageName := p.expect(ast.TokenIdentifier).Value
			nodes = append(nodes, ast.PackageNode{Value: packageName})
		} else if p.current().Type == ast.TokenImport {
			p.advance() // Move past `import`

			// Check if this is a grouped import with parentheses
			if p.current().Type == ast.TokenLParen {
				p.advance() // Move past '('
				imports := []ast.ImportNode{}

				// Parse imports until we hit the closing paren
				for p.current().Type != ast.TokenRParen {
					importName := p.expect(ast.TokenIdentifier).Value
					imports = append(imports, ast.ImportNode{Path: importName})
				}

				p.expect(ast.TokenRParen)
				nodes = append(nodes, ast.ImportGroupNode{Imports: imports})
			} else {
				// Single import
				importName := p.expect(ast.TokenIdentifier).Value
				nodes = append(nodes, ast.ImportNode{Path: importName})
			}
		} else if p.current().Type == ast.TokenFunction {
			nodes = append(nodes, p.parseFunctionDefinition())
		} else {
			panic(fmt.Sprintf("Unexpected token in file: %s", p.current().Value))
		}
	}

	return nodes
}
