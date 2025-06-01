package lexer

import "forst/internal/ast"

// Context for tracking file information
type Context struct {
	FilePath string
}

// LexerLocation tracks the current state of the lexer
type LexerLocation struct {
	LineNum int
	Column  int
	Path    string
}

// Lexer represents the lexer for the Forst language
type Lexer struct {
	input    []byte
	tokens   []ast.Token
	context  Context
	location LexerLocation

	// Block comment state
	accumulatingBlockComment bool
	blockCommentValue        string
	blockCommentStartLine    int
	blockCommentStartCol     int
	blockCommentNesting      int

	// Backtick string state
	accumulatingBacktickString bool
	backtickStringValue        string
	backtickStringStartLine    int
	backtickStringStartCol     int
}

// NewLexer creates a new lexer instance
func NewLexer(input []byte, filePath string) *Lexer {
	return &Lexer{
		input:    input,
		context:  Context{FilePath: filePath},
		location: LexerLocation{Path: filePath},
	}
}
