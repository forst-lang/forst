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
		tbFailf(tb, "parse %s: %v", fileID, err)
		return nil
	}
	return nodes
}

// ParseSourceForBench is like ParseSource but for benchmarks (no testing.TB fatals on setup).
func ParseSourceForBench(tb testing.TB, src []byte, fileID string) []ast.Node {
	tb.Helper()
	if fileID == "" {
		fileID = "bench.ft"
	}
	log := logrus.New()
	log.SetOutput(nil)
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New(src, fileID, log).Lex()
	nodes, err := parser.New(toks, fileID, log).ParseFile()
	if err != nil {
		tbFailf(tb, "parse: %v", err)
		return nil
	}
	return nodes
}
