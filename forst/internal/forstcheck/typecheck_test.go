package forstcheck_test

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstcheck"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestTypecheckFile_tempModule(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.ft")
	const src = `package main

func main() {
	println("ok")
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(nil)
	nodes := parseTestSource(t, path, src)
	tc, _, err := forstcheck.TypecheckFile(log, forstcheck.TypecheckFileOpts{
		FilePath: path,
		Nodes:    nodes,
	})
	if err != nil {
		t.Fatalf("TypecheckFile: %v", err)
	}
	if tc == nil {
		t.Fatal("expected typechecker")
	}
}

func parseTestSource(t *testing.T, path, src string) []ast.Node {
	t.Helper()
	lex := lexer.New([]byte(src), path, logrus.New())
	tokens := lex.Lex()
	psr := parser.New(tokens, path, logrus.New())
	nodes, err := psr.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return nodes
}
