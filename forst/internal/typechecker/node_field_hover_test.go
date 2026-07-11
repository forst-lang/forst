package typechecker

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/testutil"
)

func TestFieldHoverMarkdown_nodeInteropResultAfterOk(t *testing.T) {
	root := nodeInteropSyncExampleDir(t)
	srcBytes, err := os.ReadFile(filepath.Join(root, "main.ft"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(srcBytes)
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	tokens := lexer.New(srcBytes, "main.ft", testutil.TestLogger(t, nil)).Lex()

	for i := range tokens {
		tok := &tokens[i]
		if tok.Type != ast.TokenIdentifier || (tok.Value != "id" && tok.Value != "amount") {
			continue
		}
		if i < 2 || tokens[i-1].Type != ast.TokenDot || tokens[i-2].Value != "result" {
			continue
		}
		recvTok := &tokens[i-2]
		md, _, ok := tc.FieldHoverMarkdown(
			ast.Identifier("result"),
			ast.SpanFromToken(*recvTok),
			[]string{tok.Value},
			ast.SpanBetweenTokens(*recvTok, *tok),
		)
		if !ok || md == "" {
			t.Fatalf("no field hover for result.%s", tok.Value)
		}
		if !strings.Contains(md, tok.Value) {
			t.Fatalf("hover for result.%s missing field name: %q", tok.Value, md)
		}
	}
}

func nodeInteropSyncExampleDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "examples", "in", "rfc", "node-interop", "sync"))
}
