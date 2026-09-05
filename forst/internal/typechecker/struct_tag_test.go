package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestCheckTypes_jsonStructTag_setsGoExportAndWarning(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

type Config = {
  host: String ` + "`json:\"host\"`" + `
  plain: Int
}
`
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(ast.SetupTestLogger(nil), false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	def := tc.Defs["Config"].(ast.TypeDefNode)
	shape, _ := ast.PayloadShape(def.Expr)
	if !shape.Fields["host"].GoExport {
		t.Fatal("host should be marked GoExport")
	}
	if shape.Fields["plain"].GoExport {
		t.Fatal("plain should not be GoExport")
	}
	var found bool
	for _, w := range tc.Warnings {
		if w.Code == "struct-tag-json-unexported" && strings.Contains(w.Error(), "host") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected struct-tag-json-unexported warning, got %+v", tc.Warnings)
	}
}

func TestCheckTypes_jsonDashStructTag_noWarning(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

type Config = {
  secret: String ` + "`json:\"-\"`" + `
}
`
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(ast.SetupTestLogger(nil), false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	for _, w := range tc.Warnings {
		if w.Code == "struct-tag-json-unexported" {
			t.Fatalf("unexpected warning: %+v", w)
		}
	}
}
