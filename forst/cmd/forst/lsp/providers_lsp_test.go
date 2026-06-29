package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestWithScopeHoverMarkdown_nestedShadow(t *testing.T) {
	t.Parallel()
	const src = `package main

type Logger = { info(msg String) }
type NopLogger = { info(msg String) }

func main() {
	with {
		Logger: NopLogger{}
	} {
		with {
			Logger: NopLogger{}
		} {
		}
	}
}
`
	log := logrus.New()
	log.SetOutput(nil)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	ctx, ok := analyzeNodesForTest(t, nodes, toks, src)
	if !ok || ctx.TC == nil {
		t.Fatal("analyze failed")
	}
	// Second `with` keyword in nested fixture.
	var withTok *ast.Token
	withCount := 0
	for i := range toks {
		if toks[i].Type == ast.TokenWith {
			withCount++
			if withCount == 2 {
				withTok = &toks[i]
				break
			}
		}
	}
	if withTok == nil {
		t.Fatal("inner with token not found")
	}
	md := withScopeHoverMarkdown(ctx.TC, nodes, withTok)
	if md == "" {
		t.Fatal("expected hover markdown")
	}
	if !strings.Contains(md, "Effective scope:") {
		t.Fatalf("got %q", md)
	}
	if !strings.Contains(md, "shadows outer") {
		t.Fatalf("expected shadow marker, got %q", md)
	}
}

func TestCollectWithChainContainingPosition_nested(t *testing.T) {
	t.Parallel()
	const src = `package main

func main() {
	with {
	} {
		with {
		} {
		}
	}
}
`
	log := logrus.New()
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	chain := collectWithChainContainingPosition(nodes, 6, 3)
	if len(chain) != 2 {
		t.Fatalf("chain len = %d", len(chain))
	}
}

func analyzeNodesForTest(t *testing.T, _ []ast.Node, _ []ast.Token, src string) (*forstDocumentContext, bool) {
	t.Helper()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	path := filepath.Join(dir, "t.ft")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, path)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()
	return s.analyzeForstDocument(uri)
}
