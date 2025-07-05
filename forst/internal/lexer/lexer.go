package lexer

import (
	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

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
	log      *logrus.Logger
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

// New creates a new lexer instance
func New(input []byte, filePath string, log *logrus.Logger) *Lexer {
	if log == nil {
		log = logrus.New()
		log.Warnf("No logger provided, using default logger")
	}
	return &Lexer{
		log:      log,
		input:    input,
		context:  Context{FilePath: filePath},
		location: LexerLocation{Path: filePath},
	}
}
