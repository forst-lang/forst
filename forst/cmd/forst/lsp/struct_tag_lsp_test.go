package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

func TestCompileForstFile_jsonStructTagWarning(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("struct_tag_lsp")), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(dir, "tags.ft")
	src := `package main

type Config = {
  host: String ` + "`json:\"host\"`" + `
}

func main() {
  c := Config{ host: "x" }
  println(c.host)
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewLSPServer("8080", logrus.New())
	diags := s.compileForstFile(ft, src, nil)
	var found bool
	for _, d := range diags {
		if d.Code == "struct-tag-json-unexported" && d.Severity == LSPDiagnosticSeverityWarning {
			found = true
			if !strings.Contains(d.Message, "host") {
				t.Fatalf("message = %q", d.Message)
			}
		}
	}
	if !found {
		t.Fatalf("expected json struct tag warning, got %+v", diags)
	}
}
