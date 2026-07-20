package lsp

import (
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestTransformDiagnostics_successReturnsEmpty(t *testing.T) {
	path, uri := importTestModuleFile(t, "ok.ft", `package main

func main() {
	println("ok")
}
`)
	s := NewLSPServer("8080", logrus.New())
	content := `package main

func main() {
	println("ok")
}
`
	s.documentMu.Lock()
	s.openDocuments[uri] = content
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.TC == nil {
		t.Fatal("analyzeForstDocument failed")
	}
	if ctx.CheckErr != nil {
		t.Fatalf("unexpected typecheck error: %v", ctx.CheckErr)
	}
	diags := s.transformDiagnostics(ctx, path)
	for _, d := range diags {
		if d.Source == "forst-transformer" {
			t.Fatalf("unexpected transform diagnostic: %+v", d)
		}
	}
}

func TestTransformDiagnostics_transformFailureReturnsDiagnostic(t *testing.T) {
	path, uri := importTestModuleFile(t, "pkg_var.ft", `package main

var Version = "1.0"

func main() {}
`)
	s := NewLSPServer("8080", logrus.New())
	content := `package main

var Version = "1.0"

func main() {}
`
	s.documentMu.Lock()
	s.openDocuments[uri] = content
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.TC == nil {
		t.Fatal("analyzeForstDocument failed")
	}
	if ctx.CheckErr != nil {
		t.Fatalf("unexpected typecheck error: %v", ctx.CheckErr)
	}
	ctx.Nodes = append(ctx.Nodes, ast.AssignmentNode{
		IsPackageLevel: true,
		LValues: []ast.ExpressionNode{
			ast.VariableNode{Ident: ast.Ident{ID: "A"}},
			ast.VariableNode{Ident: ast.Ident{ID: "B"}},
		},
		RValues: []ast.ExpressionNode{
			ast.IntLiteralNode{Value: 1},
			ast.IntLiteralNode{Value: 2},
		},
	})
	diags := s.transformDiagnostics(ctx, path)
	if len(diags) != 1 {
		t.Fatalf("diags = %+v", diags)
	}
	d := diags[0]
	if d.Source != "forst-transformer" || d.Code != ErrorCodeTransformationFailed {
		t.Fatalf("diagnostic = %+v", d)
	}
	if !strings.Contains(d.Message, "Transformation error") {
		t.Fatalf("message = %q", d.Message)
	}
}
