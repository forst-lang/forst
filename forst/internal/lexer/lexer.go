package lexer

import (
	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// Context for tracking file information
type Context struct {
	FileID string
}

// Location tracks the current state of the lexer.
type Location struct {
	LineNum int
	Column  int
	FileID  string
}

// Lexer represents the lexer for the Forst language
type Lexer struct {
	log      *logrus.Logger
	input    []byte
	tokens   []ast.Token
	context  Context
	location Location

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
func New(input []byte, fileID string, log *logrus.Logger) *Lexer {
	if log == nil {
		log = logrus.New()
		log.Warnf("No logger provided, using default logger")
	}
	return &Lexer{
		log:      log,
		input:    input,
		context:  Context{FileID: fileID},
		location: Location{FileID: fileID},
	}
}
