package lexer

import (
	"forst/internal/ast"
	"testing"
)

const testFileID = "f1"

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
				{Type: ast.TokenIdentifier, Value: "hello", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "world", FileID: testFileID, Line: 1, Column: 7},
				{Type: ast.TokenIdentifier, Value: "_123", FileID: testFileID, Line: 1, Column: 13},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "numbers",
			input: "123 0x1F 0b1010 0o77 1.23 1e-10 1.23i",
			expected: []ast.Token{
				{Type: ast.TokenIntLiteral, Value: "123", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenIntLiteral, Value: "0x1F", FileID: testFileID, Line: 1, Column: 5},
				{Type: ast.TokenIntLiteral, Value: "0b1010", FileID: testFileID, Line: 1, Column: 10},
				{Type: ast.TokenIntLiteral, Value: "0o77", FileID: testFileID, Line: 1, Column: 17},
				{Type: ast.TokenFloatLiteral, Value: "1.23", FileID: testFileID, Line: 1, Column: 22},
				{Type: ast.TokenFloatLiteral, Value: "1e-10", FileID: testFileID, Line: 1, Column: 27},
				{Type: ast.TokenFloatLiteral, Value: "1.23i", FileID: testFileID, Line: 1, Column: 33},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "strings",
			input: `"hello" 'world' ` + "`raw\nstring`",
			expected: []ast.Token{
				{Type: ast.TokenStringLiteral, Value: `"hello"`, FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: "'world'", FileID: testFileID, Line: 1, Column: 9},
				{Type: ast.TokenStringLiteral, Value: "`raw\nstring`", FileID: testFileID, Line: 1, Column: 17},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "operators",
			input: "+ - * / % & |",
			expected: []ast.Token{
				{Type: ast.TokenPlus, Value: "+", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenMinus, Value: "-", FileID: testFileID, Line: 1, Column: 3},
				{Type: ast.TokenStar, Value: "*", FileID: testFileID, Line: 1, Column: 5},
				{Type: ast.TokenDivide, Value: "/", FileID: testFileID, Line: 1, Column: 7},
				{Type: ast.TokenModulo, Value: "%", FileID: testFileID, Line: 1, Column: 9},
				{Type: ast.TokenBitwiseAnd, Value: "&", FileID: testFileID, Line: 1, Column: 11},
				{Type: ast.TokenBitwiseOr, Value: "|", FileID: testFileID, Line: 1, Column: 13},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "comparison operators",
			input: "== != < <= > >=",
			expected: []ast.Token{
				{Type: ast.TokenEquals, Value: "==", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenNotEquals, Value: "!=", FileID: testFileID, Line: 1, Column: 3},
				{Type: ast.TokenLess, Value: "<", FileID: testFileID, Line: 1, Column: 5},
				{Type: ast.TokenLessEqual, Value: "<=", FileID: testFileID, Line: 1, Column: 7},
				{Type: ast.TokenGreater, Value: ">", FileID: testFileID, Line: 1, Column: 9},
				{Type: ast.TokenGreaterEqual, Value: ">=", FileID: testFileID, Line: 1, Column: 11},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "logical operators",
			input: "&& || !",
			expected: []ast.Token{
				{Type: ast.TokenLogicalAnd, Value: "&&", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenLogicalOr, Value: "||", FileID: testFileID, Line: 1, Column: 3},
				{Type: ast.TokenLogicalNot, Value: "!", FileID: testFileID, Line: 1, Column: 5},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "delimiters",
			input: "()[]{}.,;:",
			expected: []ast.Token{
				{Type: ast.TokenLParen, Value: "(", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenRParen, Value: ")", FileID: testFileID, Line: 1, Column: 2},
				{Type: ast.TokenLBracket, Value: "[", FileID: testFileID, Line: 1, Column: 3},
				{Type: ast.TokenRBracket, Value: "]", FileID: testFileID, Line: 1, Column: 4},
				{Type: ast.TokenLBrace, Value: "{", FileID: testFileID, Line: 1, Column: 5},
				{Type: ast.TokenRBrace, Value: "}", FileID: testFileID, Line: 1, Column: 6},
				{Type: ast.TokenDot, Value: ".", FileID: testFileID, Line: 1, Column: 7},
				{Type: ast.TokenComma, Value: ",", FileID: testFileID, Line: 1, Column: 8},
				{Type: ast.TokenSemicolon, Value: ";", FileID: testFileID, Line: 1, Column: 9},
				{Type: ast.TokenColon, Value: ":", FileID: testFileID, Line: 1, Column: 10},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "assignment and arrow operators",
			input: ":= ->",
			expected: []ast.Token{
				{Type: ast.TokenColonEquals, Value: ":=", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenArrow, Value: "->", FileID: testFileID, Line: 1, Column: 4},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "equals operators",
			input: "= ==",
			expected: []ast.Token{
				{Type: ast.TokenEquals, Value: "=", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenEquals, Value: "==", FileID: testFileID, Line: 1, Column: 3},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
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
				{Type: ast.TokenPackage, Value: "package", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenImport, Value: "import", FileID: testFileID, Line: 1, Column: 9},
				{Type: ast.TokenType, Value: "type", FileID: testFileID, Line: 1, Column: 15},
				{Type: ast.TokenVar, Value: "var", FileID: testFileID, Line: 1, Column: 19},
				{Type: ast.TokenConst, Value: "const", FileID: testFileID, Line: 1, Column: 23},
				{Type: ast.TokenFunc, Value: "func", FileID: testFileID, Line: 1, Column: 28},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "control flow keywords",
			input: "if else for range switch case default",
			expected: []ast.Token{
				{Type: ast.TokenIf, Value: "if", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenElse, Value: "else", FileID: testFileID, Line: 1, Column: 3},
				{Type: ast.TokenFor, Value: "for", FileID: testFileID, Line: 1, Column: 5},
				{Type: ast.TokenRange, Value: "range", FileID: testFileID, Line: 1, Column: 8},
				{Type: ast.TokenSwitch, Value: "switch", FileID: testFileID, Line: 1, Column: 13},
				{Type: ast.TokenCase, Value: "case", FileID: testFileID, Line: 1, Column: 19},
				{Type: ast.TokenDefault, Value: "default", FileID: testFileID, Line: 1, Column: 23},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "type keywords",
			input: "struct interface map chan",
			expected: []ast.Token{
				{Type: ast.TokenStruct, Value: "struct", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenInterface, Value: "interface", FileID: testFileID, Line: 1, Column: 7},
				{Type: ast.TokenMap, Value: "map", FileID: testFileID, Line: 1, Column: 15},
				{Type: ast.TokenChan, Value: "chan", FileID: testFileID, Line: 1, Column: 18},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "other keywords",
			input: "go defer return break continue fallthrough goto",
			expected: []ast.Token{
				{Type: ast.TokenGo, Value: "go", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenDefer, Value: "defer", FileID: testFileID, Line: 1, Column: 3},
				{Type: ast.TokenReturn, Value: "return", FileID: testFileID, Line: 1, Column: 8},
				{Type: ast.TokenBreak, Value: "break", FileID: testFileID, Line: 1, Column: 14},
				{Type: ast.TokenContinue, Value: "continue", FileID: testFileID, Line: 1, Column: 19},
				{Type: ast.TokenFallthrough, Value: "fallthrough", FileID: testFileID, Line: 1, Column: 27},
				{Type: ast.TokenGoto, Value: "goto", FileID: testFileID, Line: 1, Column: 37},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "built-in type keywords",
			input: "Int Float String Bool Array",
			expected: []ast.Token{
				{Type: ast.TokenInt, Value: "Int", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenFloat, Value: "Float", FileID: testFileID, Line: 1, Column: 5},
				{Type: ast.TokenString, Value: "String", FileID: testFileID, Line: 1, Column: 11},
				{Type: ast.TokenBool, Value: "Bool", FileID: testFileID, Line: 1, Column: 18},
				{Type: ast.TokenArray, Value: "Array", FileID: testFileID, Line: 1, Column: 23},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "forst-specific keywords",
			input: "ensure is or",
			expected: []ast.Token{
				{Type: ast.TokenEnsure, Value: "ensure", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenIs, Value: "is", FileID: testFileID, Line: 1, Column: 8},
				{Type: ast.TokenOr, Value: "or", FileID: testFileID, Line: 1, Column: 11},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "else if sequence",
			input: "else if",
			expected: []ast.Token{
				{Type: ast.TokenElse, Value: "else", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenIf, Value: "if", FileID: testFileID, Line: 1, Column: 6},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
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
				{Type: ast.TokenComment, Value: "// comment", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "x", FileID: testFileID, Line: 1, Column: 4},
				{Type: ast.TokenComment, Value: "// another comment", FileID: testFileID, Line: 2, Column: 1},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "block comments",
			input: "/* comment */ x /* multi\nline\ncomment */",
			expected: []ast.Token{
				{Type: ast.TokenComment, Value: "/* comment */", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "x", FileID: testFileID, Line: 2, Column: 1},
				{Type: ast.TokenComment, Value: "/* multi\nline\ncomment */", FileID: testFileID, Line: 4, Column: 1},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "nested block comments",
			input: "/* outer /* inner */ still outer */ x",
			expected: []ast.Token{
				{Type: ast.TokenComment, Value: "/* outer /* inner */ still outer */", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "x", FileID: testFileID, Line: 2, Column: 1},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
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
				{Type: ast.TokenStringLiteral, Value: `"hello"`, FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: `"world\n"`, FileID: testFileID, Line: 1, Column: 9},
				{Type: ast.TokenStringLiteral, Value: `"with \"quote\""`, FileID: testFileID, Line: 1, Column: 19},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "raw strings",
			input: "`hello` `world\n` `with \"quote\"`",
			expected: []ast.Token{
				{Type: ast.TokenStringLiteral, Value: "`hello`", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: "`world\n`", FileID: testFileID, Line: 1, Column: 9},
				{Type: ast.TokenStringLiteral, Value: "`with \"quote\"`", FileID: testFileID, Line: 1, Column: 19},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "string with escapes",
			input: `"\\ \n \r \t \v \f \u1234 \U00001234"`,
			expected: []ast.Token{
				{Type: ast.TokenStringLiteral, Value: `"\\ \n \r \t \v \f \u1234 \U00001234"`, FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
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
				{Type: ast.TokenIdentifier, Value: "x", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "y", FileID: testFileID, Line: 1, Column: 3},
				{Type: ast.TokenIdentifier, Value: "z", FileID: testFileID, Line: 1, Column: 5},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "newlines",
			input: "x\ny\nz",
			expected: []ast.Token{
				{Type: ast.TokenIdentifier, Value: "x", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "y", FileID: testFileID, Line: 2, Column: 1},
				{Type: ast.TokenIdentifier, Value: "z", FileID: testFileID, Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "mixed whitespace",
			input: "x \t\ny \n\tz",
			expected: []ast.Token{
				{Type: ast.TokenIdentifier, Value: "x", FileID: testFileID, Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "y", FileID: testFileID, Line: 2, Column: 3},
				{Type: ast.TokenIdentifier, Value: "z", FileID: testFileID, Line: 3, Column: 5},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
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
			{Type: ast.TokenPackage, Value: "package", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "main", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenImport, Value: "import", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenStringLiteral, Value: "\"fmt\"", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenStringLiteral, Value: "\"math\"", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenType, Value: "type", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Point", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenStruct, Value: "struct", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "X", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Y", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "float64", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenFunc, Value: "func", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenStar, Value: "*", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Point", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Distance", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "q", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenStar, Value: "*", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Point", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "float64", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenReturn, Value: "return", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "math", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Sqrt", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "math", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Pow", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "X", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenMinus, Value: "-", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "q", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "X", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "2", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenPlus, Value: "+", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "math", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Pow", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Y", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenMinus, Value: "-", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "q", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Y", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "2", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenFunc, Value: "func", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "main", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p1", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenColonEquals, Value: ":=", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenBitwiseAnd, Value: "&", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Point", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "X", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenColon, Value: ":", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "1", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Y", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenColon, Value: ":", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "2", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p2", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenColonEquals, Value: ":=", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenBitwiseAnd, Value: "&", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Point", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "X", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenColon, Value: ":", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "4", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Y", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenColon, Value: ":", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "6", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIf, Value: "if", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "dist", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenColonEquals, Value: ":=", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p1", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Distance", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "p2", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenSemicolon, Value: ";", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "dist", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenGreater, Value: ">", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIntLiteral, Value: "5", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "fmt", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Println", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenStringLiteral, Value: "\"Points are far apart:\"", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "dist", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenElse, Value: "else", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLBrace, Value: "{", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "fmt", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenDot, Value: ".", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "Println", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenLParen, Value: "(", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenStringLiteral, Value: "\"Points are close:\"", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenComma, Value: ",", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenIdentifier, Value: "dist", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRParen, Value: ")", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenRBrace, Value: "}", FileID: testFileID, Line: 0, Column: 0},
			{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 0, Column: 0},
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
			{Type: ast.TokenIdentifier, Value: "hello", FileID: testFileID, Line: 1, Column: 1},
			{Type: ast.TokenIdentifier, Value: "world", FileID: testFileID, Line: 1, Column: 7},
			{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
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
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
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
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
			},
		},
		{
			name:  "ellipsis",
			input: "...",
			expected: []ast.Token{
				// {Type: ast.TokenEllipsis, Value: "...", Path: "test.forst", Line: 1, Column: 1},
				{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
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
	log := setupTestLogger(nil)
	l := New([]byte(tt.input), testFileID, log)
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
