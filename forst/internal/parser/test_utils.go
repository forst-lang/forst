package parser

import (
	"forst/internal/ast"
	"forst/internal/lexer"
	"testing"

	"github.com/sirupsen/logrus"
)

// setupParser creates a new parser with the given tokens
func setupParser(tokens []ast.Token) *Parser {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05.000",
	})
	return New(tokens, "", logger)
}

// assertNodeType checks if a node is of the expected type
func assertNodeType[T ast.Node](t *testing.T, node ast.Node, expectedType string) T {
	t.Helper()
	typedNode, ok := node.(T)
	if !ok {
		t.Fatalf("Expected %s, got %T", expectedType, node)
	}
	return typedNode
}

// NewTestParser creates a parser from a source string for testing
func NewTestParser(input string) *Parser {
	logger := logrus.New()
	lex := lexer.New([]byte(input), "<test>", logger)
	tokens := lex.Lex()
	return setupParser(tokens)
}
