package lexer

import (
	"forst/internal/ast"
	"testing"
)

const testFilePath = "test.ft"

func TestLexer_BasicTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ast.Token
	}{
		{
			name:  "identifiers",
			input: "hello world _123",
			expected: []ast.Token{
				{Type: ast.TokenIdentifier, Value: "hello", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "world", Path: testFilePath, Line: 1, Column: 7},
				{Type: ast.TokenIdentifier, Value: "_123", Path: testFilePath, Line: 1, Column: 13},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "numbers",
			input: "123 0x1F 0b1010 0o77 1.23 1e-10 1.23i",
			expected: []ast.Token{
				{Type: ast.TokenIntLiteral, Value: "123", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenIntLiteral, Value: "0x1F", Path: testFilePath, Line: 1, Column: 5},
				{Type: ast.TokenIntLiteral, Value: "0b1010", Path: testFilePath, Line: 1, Column: 10},
				{Type: ast.TokenIntLiteral, Value: "0o77", Path: testFilePath, Line: 1, Column: 17},
				{Type: ast.TokenFloatLiteral, Value: "1.23", Path: testFilePath, Line: 1, Column: 22},
				{Type: ast.TokenFloatLiteral, Value: "1e-10", Path: testFilePath, Line: 1, Column: 27},
				{Type: ast.TokenFloatLiteral, Value: "1.23i", Path: testFilePath, Line: 1, Column: 33},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "strings",
			input: `"hello" 'world' ` + "`raw\nstring`",
			expected: []ast.Token{
				{Type: ast.TokenStringLiteral, Value: `"hello"`, Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: "'world'", Path: testFilePath, Line: 1, Column: 9},
				{Type: ast.TokenStringLiteral, Value: "`raw\nstring`", Path: testFilePath, Line: 1, Column: 17},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "operators",
			input: "+ - * / % & |",
			expected: []ast.Token{
				{Type: ast.TokenPlus, Value: "+", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenMinus, Value: "-", Path: testFilePath, Line: 1, Column: 3},
				{Type: ast.TokenStar, Value: "*", Path: testFilePath, Line: 1, Column: 5},
				{Type: ast.TokenDivide, Value: "/", Path: testFilePath, Line: 1, Column: 7},
				{Type: ast.TokenModulo, Value: "%", Path: testFilePath, Line: 1, Column: 9},
				{Type: ast.TokenBitwiseAnd, Value: "&", Path: testFilePath, Line: 1, Column: 11},
				{Type: ast.TokenBitwiseOr, Value: "|", Path: testFilePath, Line: 1, Column: 13},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "comparison operators",
			input: "== != < <= > >=",
			expected: []ast.Token{
				{Type: ast.TokenEquals, Value: "==", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenNotEquals, Value: "!=", Path: testFilePath, Line: 1, Column: 3},
				{Type: ast.TokenLess, Value: "<", Path: testFilePath, Line: 1, Column: 5},
				{Type: ast.TokenLessEqual, Value: "<=", Path: testFilePath, Line: 1, Column: 7},
				{Type: ast.TokenGreater, Value: ">", Path: testFilePath, Line: 1, Column: 9},
				{Type: ast.TokenGreaterEqual, Value: ">=", Path: testFilePath, Line: 1, Column: 11},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "logical operators",
			input: "&& || !",
			expected: []ast.Token{
				{Type: ast.TokenLogicalAnd, Value: "&&", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenLogicalOr, Value: "||", Path: testFilePath, Line: 1, Column: 3},
				{Type: ast.TokenLogicalNot, Value: "!", Path: testFilePath, Line: 1, Column: 5},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "delimiters",
			input: "()[]{}.,;:",
			expected: []ast.Token{
				{Type: ast.TokenLParen, Value: "(", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenRParen, Value: ")", Path: testFilePath, Line: 1, Column: 2},
				{Type: ast.TokenLBracket, Value: "[", Path: testFilePath, Line: 1, Column: 3},
				{Type: ast.TokenRBracket, Value: "]", Path: testFilePath, Line: 1, Column: 4},
				{Type: ast.TokenLBrace, Value: "{", Path: testFilePath, Line: 1, Column: 5},
				{Type: ast.TokenRBrace, Value: "}", Path: testFilePath, Line: 1, Column: 6},
				{Type: ast.TokenDot, Value: ".", Path: testFilePath, Line: 1, Column: 7},
				{Type: ast.TokenComma, Value: ",", Path: testFilePath, Line: 1, Column: 8},
				{Type: ast.TokenSemicolon, Value: ";", Path: testFilePath, Line: 1, Column: 9},
				{Type: ast.TokenColon, Value: ":", Path: testFilePath, Line: 1, Column: 10},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "assignment and arrow operators",
			input: ":= ->",
			expected: []ast.Token{
				{Type: ast.TokenColonEquals, Value: ":=", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenArrow, Value: "->", Path: testFilePath, Line: 1, Column: 4},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "equals operators",
			input: "= ==",
			expected: []ast.Token{
				{Type: ast.TokenEquals, Value: "=", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenEquals, Value: "==", Path: testFilePath, Line: 1, Column: 3},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLexerTokens(t, tt)
		})
	}
}

func TestLexer_Keywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ast.Token
	}{
		{
			name:  "basic keywords",
			input: "package import type var const func",
			expected: []ast.Token{
				{Type: ast.TokenPackage, Value: "package", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenImport, Value: "import", Path: testFilePath, Line: 1, Column: 9},
				{Type: ast.TokenType, Value: "type", Path: testFilePath, Line: 1, Column: 15},
				{Type: ast.TokenVar, Value: "var", Path: testFilePath, Line: 1, Column: 19},
				{Type: ast.TokenConst, Value: "const", Path: testFilePath, Line: 1, Column: 23},
				{Type: ast.TokenFunc, Value: "func", Path: testFilePath, Line: 1, Column: 28},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "control flow keywords",
			input: "if else for range switch case default",
			expected: []ast.Token{
				{Type: ast.TokenIf, Value: "if", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenElse, Value: "else", Path: testFilePath, Line: 1, Column: 3},
				{Type: ast.TokenFor, Value: "for", Path: testFilePath, Line: 1, Column: 5},
				{Type: ast.TokenRange, Value: "range", Path: testFilePath, Line: 1, Column: 8},
				{Type: ast.TokenSwitch, Value: "switch", Path: testFilePath, Line: 1, Column: 13},
				{Type: ast.TokenCase, Value: "case", Path: testFilePath, Line: 1, Column: 19},
				{Type: ast.TokenDefault, Value: "default", Path: testFilePath, Line: 1, Column: 23},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "type keywords",
			input: "struct interface map chan",
			expected: []ast.Token{
				{Type: ast.TokenStruct, Value: "struct", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenInterface, Value: "interface", Path: testFilePath, Line: 1, Column: 7},
				{Type: ast.TokenMap, Value: "map", Path: testFilePath, Line: 1, Column: 15},
				{Type: ast.TokenChan, Value: "chan", Path: testFilePath, Line: 1, Column: 18},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "other keywords",
			input: "go defer return break continue fallthrough goto",
			expected: []ast.Token{
				{Type: ast.TokenGo, Value: "go", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenDefer, Value: "defer", Path: testFilePath, Line: 1, Column: 3},
				{Type: ast.TokenReturn, Value: "return", Path: testFilePath, Line: 1, Column: 8},
				{Type: ast.TokenBreak, Value: "break", Path: testFilePath, Line: 1, Column: 14},
				{Type: ast.TokenContinue, Value: "continue", Path: testFilePath, Line: 1, Column: 19},
				{Type: ast.TokenFallthrough, Value: "fallthrough", Path: testFilePath, Line: 1, Column: 27},
				{Type: ast.TokenGoto, Value: "goto", Path: testFilePath, Line: 1, Column: 37},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "built-in type keywords",
			input: "Int Float String Bool Array",
			expected: []ast.Token{
				{Type: ast.TokenInt, Value: "Int", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenFloat, Value: "Float", Path: testFilePath, Line: 1, Column: 5},
				{Type: ast.TokenString, Value: "String", Path: testFilePath, Line: 1, Column: 11},
				{Type: ast.TokenBool, Value: "Bool", Path: testFilePath, Line: 1, Column: 18},
				{Type: ast.TokenArray, Value: "Array", Path: testFilePath, Line: 1, Column: 23},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "forst-specific keywords",
			input: "ensure is or",
			expected: []ast.Token{
				{Type: ast.TokenEnsure, Value: "ensure", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenIs, Value: "is", Path: testFilePath, Line: 1, Column: 8},
				{Type: ast.TokenOr, Value: "or", Path: testFilePath, Line: 1, Column: 11},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "else if sequence",
			input: "else if",
			expected: []ast.Token{
				{Type: ast.TokenElse, Value: "else", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenIf, Value: "if", Path: testFilePath, Line: 1, Column: 6},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLexerTokens(t, tt)
		})
	}
}

func TestLexer_Comments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ast.Token
	}{
		{
			name:  "line comments",
			input: "// comment\nx // another comment",
			expected: []ast.Token{
				{Type: ast.TokenComment, Value: "// comment", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "x", Path: testFilePath, Line: 1, Column: 4},
				{Type: ast.TokenComment, Value: "// another comment", Path: testFilePath, Line: 2, Column: 1},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "block comments",
			input: "/* comment */ x /* multi\nline\ncomment */",
			expected: []ast.Token{
				{Type: ast.TokenComment, Value: "/* comment */", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "x", Path: testFilePath, Line: 2, Column: 1},
				{Type: ast.TokenComment, Value: "/* multi\nline\ncomment */", Path: testFilePath, Line: 4, Column: 1},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "nested block comments",
			input: "/* outer /* inner */ still outer */ x",
			expected: []ast.Token{
				{Type: ast.TokenComment, Value: "/* outer /* inner */ still outer */", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "x", Path: testFilePath, Line: 2, Column: 1},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLexerTokens(t, tt)
		})
	}
}

func TestLexer_StringLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ast.Token
	}{
		{
			name:  "double quoted strings",
			input: `"hello" "world\n" "with \"quote\""`,
			expected: []ast.Token{
				{Type: ast.TokenStringLiteral, Value: `"hello"`, Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: `"world\n"`, Path: testFilePath, Line: 1, Column: 9},
				{Type: ast.TokenStringLiteral, Value: `"with \"quote\""`, Path: testFilePath, Line: 1, Column: 19},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "raw strings",
			input: "`hello` `world\n` `with \"quote\"`",
			expected: []ast.Token{
				{Type: ast.TokenStringLiteral, Value: "`hello`", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: "`world\n`", Path: testFilePath, Line: 1, Column: 9},
				{Type: ast.TokenStringLiteral, Value: "`with \"quote\"`", Path: testFilePath, Line: 1, Column: 19},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "string with escapes",
			input: `"\\ \n \r \t \v \f \u1234 \U00001234"`,
			expected: []ast.Token{
				{Type: ast.TokenStringLiteral, Value: `"\\ \n \r \t \v \f \u1234 \U00001234"`, Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLexerTokens(t, tt)
		})
	}
}

func TestLexer_Whitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ast.Token
	}{
		{
			name:  "spaces and tabs",
			input: "x  y\tz",
			expected: []ast.Token{
				{Type: ast.TokenIdentifier, Value: "x", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "y", Path: testFilePath, Line: 1, Column: 3},
				{Type: ast.TokenIdentifier, Value: "z", Path: testFilePath, Line: 1, Column: 5},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "newlines",
			input: "x\ny\nz",
			expected: []ast.Token{
				{Type: ast.TokenIdentifier, Value: "x", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "y", Path: testFilePath, Line: 2, Column: 1},
				{Type: ast.TokenIdentifier, Value: "z", Path: testFilePath, Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "mixed whitespace",
			input: "x \t\ny \n\tz",
			expected: []ast.Token{
				{Type: ast.TokenIdentifier, Value: "x", Path: testFilePath, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "y", Path: testFilePath, Line: 2, Column: 3},
				{Type: ast.TokenIdentifier, Value: "z", Path: testFilePath, Line: 3, Column: 5},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLexerTokens(t, tt)
		})
	}
}

func TestLexer_ComplexProgram(t *testing.T) {
	tt := struct {
		name     string
		input    string
		expected []ast.Token
	}{
		name: "complex program",
		input: `
package main

import (
	"fmt"
	"math"
)

type Point struct {
	X, Y float64
}

func (p *Point) Distance(q *Point) float64 {
	return math.Sqrt(math.Pow(p.X-q.X, 2) + math.Pow(p.Y-q.Y, 2))
}

func main() {
	p1 := &Point{X: 1, Y: 2}
	p2 := &Point{X: 4, Y: 6}
	
	if dist := p1.Distance(p2); dist > 5 {
		fmt.Println("Points are far apart:", dist)
	} else {
		fmt.Println("Points are close:", dist)
	}
}
`,
		expected: []ast.Token{
			{Type: ast.TokenPackage, Value: "package", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "main", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenImport, Value: "import", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenStringLiteral, Value: "\"fmt\"", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenStringLiteral, Value: "\"math\"", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenType, Value: "type", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Point", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenStruct, Value: "struct", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "X", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Y", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "float64", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenFunc, Value: "func", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenStar, Value: "*", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Point", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Distance", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "q", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenStar, Value: "*", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Point", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "float64", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenReturn, Value: "return", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "math", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Sqrt", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "math", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Pow", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "X", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenMinus, Value: "-", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "q", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "X", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "2", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenPlus, Value: "+", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "math", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Pow", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Y", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenMinus, Value: "-", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "q", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Y", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "2", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenFunc, Value: "func", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "main", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p1", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenColonEquals, Value: ":=", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenBitwiseAnd, Value: "&", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Point", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "X", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenColon, Value: ":", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "1", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Y", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenColon, Value: ":", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "2", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p2", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenColonEquals, Value: ":=", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenBitwiseAnd, Value: "&", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Point", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "X", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenColon, Value: ":", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "4", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Y", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenColon, Value: ":", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "6", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIf, Value: "if", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "dist", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenColonEquals, Value: ":=", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p1", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Distance", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p2", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenSemicolon, Value: ";", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "dist", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenGreater, Value: ">", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "5", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "fmt", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Println", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenStringLiteral, Value: "\"Points are far apart:\"", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "dist", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenElse, Value: "else", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "fmt", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Println", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenStringLiteral, Value: "\"Points are close:\"", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "dist", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", Path: testFilePath, Line: 0, Column: 0},
			{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 0, Column: 0},
		},
	}
	testLexerTokens(t, tt)
}

func TestLexer_Simple(t *testing.T) {
	tt := struct {
		name     string
		input    string
		expected []ast.Token
	}{
		name:  "simple",
		input: "hello world",
		expected: []ast.Token{
			{Type: ast.TokenIdentifier, Value: "hello", Path: testFilePath, Line: 1, Column: 1},
			{Type: ast.TokenIdentifier, Value: "world", Path: testFilePath, Line: 1, Column: 7},
			{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
		},
	}
	testLexerTokens(t, tt)
}

func TestLexer_UnsupportedTokens(t *testing.T) {
	t.Skip("These tokens are not yet implemented in Forst")

	tests := []struct {
		name     string
		input    string
		expected []ast.Token
	}{
		{
			name:  "bitwise operators",
			input: "^ << >> &^",
			expected: []ast.Token{
				// {Type: ast.TokenXor, Value: "^", Path: "test.forst", Line: 1, Column: 1},
				// {Type: ast.TokenLShift, Value: "<<", Path: "test.forst", Line: 1, Column: 3},
				// {Type: ast.TokenRShift, Value: ">>", Path: "test.forst", Line: 1, Column: 6},
				// {Type: ast.TokenAndNot, Value: "&^", Path: "test.forst", Line: 1, Column: 9},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "compound assignment operators",
			input: "+= -= *= /= %= &= |= ^= <<= >>= &^=",
			expected: []ast.Token{
				// {Type: ast.TokenPlusEq, Value: "+=", Path: "test.forst", Line: 1, Column: 1},
				// {Type: ast.TokenMinusEq, Value: "-=", Path: "test.forst", Line: 1, Column: 4},
				// {Type: ast.TokenStarEq, Value: "*=", Path: "test.forst", Line: 1, Column: 7},
				// {Type: ast.TokenSlashEq, Value: "/=", Path: "test.forst", Line: 1, Column: 10},
				// {Type: ast.TokenPercentEq, Value: "%=", Path: "test.forst", Line: 1, Column: 13},
				// {Type: ast.TokenAndEq, Value: "&=", Path: "test.forst", Line: 1, Column: 16},
				// {Type: ast.TokenOrEq, Value: "|=", Path: "test.forst", Line: 1, Column: 19},
				// {Type: ast.TokenXorEq, Value: "^=", Path: "test.forst", Line: 1, Column: 22},
				// {Type: ast.TokenLShiftEq, Value: "<<=", Path: "test.forst", Line: 1, Column: 25},
				// {Type: ast.TokenRShiftEq, Value: ">>=", Path: "test.forst", Line: 1, Column: 29},
				// {Type: ast.TokenAndNotEq, Value: "&^=", Path: "test.forst", Line: 1, Column: 33},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
		{
			name:  "ellipsis",
			input: "...",
			expected: []ast.Token{
				// {Type: ast.TokenEllipsis, Value: "...", Path: "test.forst", Line: 1, Column: 1},
				{Type: ast.TokenEOF, Value: "", Path: testFilePath, Line: 2, Column: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLexerTokens(t, tt)
		})
	}
}

func testLexerTokens(t *testing.T, tt struct {
	name     string
	input    string
	expected []ast.Token
}) {
	log := setupTestLogger()
	l := New([]byte(tt.input), testFilePath, log)
	tokens := l.Lex()

	if len(tokens) != len(tt.expected) {
		if testing.Verbose() {
			t.Logf("Expected tokens:")
			for i, exp := range tt.expected {
				t.Logf("  [%d] Type: %q, Value: %q", i, exp.Type, exp.Value)
			}
			t.Logf("Actual tokens:")
			for i, tok := range tokens {
				t.Logf("  [%d] Type: %q, Value: %q", i, tok.Type, tok.Value)
			}
		}
		t.Errorf("expected %d tokens, got %d", len(tt.expected), len(tokens))
		return
	}

	for i, exp := range tt.expected {
		if tokens[i].Type != exp.Type {
			t.Errorf("token[%d] - type wrong. expected=%q, got=%q",
				i, exp.Type, tokens[i].Type)
		}
		if tokens[i].Value != exp.Value {
			t.Errorf("token[%d] - value wrong. expected=%q, got=%q",
				i, exp.Value, tokens[i].Value)
		}
	}
}
