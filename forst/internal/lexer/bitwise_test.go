package lexer

import (
	"testing"

	"forst/internal/ast"
)

func TestLexer_bitwiseOperators(t *testing.T) {
	t.Parallel()
	testLexerTokens(t, struct {
		name     string
		input    string
		expected []ast.Token
	}{
		name:  "bitwise operators",
		input: "^ << >> &^",
		expected: []ast.Token{
			{Type: ast.TokenXor, Value: "^", FileID: testFileID, Line: 1, Column: 1},
			{Type: ast.TokenLShift, Value: "<<", FileID: testFileID, Line: 1, Column: 3},
			{Type: ast.TokenRShift, Value: ">>", FileID: testFileID, Line: 1, Column: 6},
			{Type: ast.TokenAndNot, Value: "&^", FileID: testFileID, Line: 1, Column: 9},
			{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
		},
	})
}

func TestLexer_bitwiseCompoundAssignment(t *testing.T) {
	t.Parallel()
	testLexerTokens(t, struct {
		name     string
		input    string
		expected []ast.Token
	}{
		name:  "bitwise compound assignment",
		input: "^= <<= >>= &^=",
		expected: []ast.Token{
			{Type: ast.TokenXorEq, Value: "^=", FileID: testFileID, Line: 1, Column: 1},
			{Type: ast.TokenLShiftEq, Value: "<<=", FileID: testFileID, Line: 1, Column: 4},
			{Type: ast.TokenRShiftEq, Value: ">>=", FileID: testFileID, Line: 1, Column: 8},
			{Type: ast.TokenAndNotEq, Value: "&^=", FileID: testFileID, Line: 1, Column: 12},
			{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
		},
	})
}
