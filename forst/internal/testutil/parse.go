package testutil

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

// ParseSource lexes and parses src into AST nodes.
func ParseSource(tb testing.TB, src, fileID string, log *logrus.Logger) []ast.Node {
	tb.Helper()
	if fileID == "" {
		fileID = "test.ft"
	}
	if log == nil {
		log = TestLogger(tb, nil)
	}
	toks := lexer.New([]byte(src), fileID, log).Lex()
	nodes, err := parser.New(toks, fileID, log).ParseFile()
	if err != nil {
		tb.Fatalf("parse %s: %v", fileID, err)
	}
	return nodes
}

// ParseSourceForBench is like ParseSource but for benchmarks (no testing.TB fatals on setup).
func ParseSourceForBench(b *testing.B, src []byte, fileID string) []ast.Node {
	b.Helper()
	if fileID == "" {
		fileID = "bench.ft"
	}
	log := logrus.New()
	log.SetOutput(nil)
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New(src, fileID, log).Lex()
	nodes, err := parser.New(toks, fileID, log).ParseFile()
	if err != nil {
		b.Fatalf("parse: %v", err)
	}
	return nodes
}
