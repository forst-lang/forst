package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestDeferGo_requireFunctionCall(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

func main() {
	defer 1
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected type error for defer non-call")
	}
	if !strings.Contains(err.Error(), "defer requires a function or method call") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeferGo_go_requires_call(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

func main() {
	go 2
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected type error for go non-call")
	}
	if !strings.Contains(err.Error(), "go requires a function or method call") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeferGo_rejects_len_builtin(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

func main() {
	defer len("a")
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected error for defer len")
	}
	if !strings.Contains(err.Error(), "built-in \"len\"") || !strings.Contains(err.Error(), "defer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeferGo_rejects_make_builtin(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

func main() {
	go make()
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected error for go make")
	}
	if !strings.Contains(err.Error(), "built-in \"make\"") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeferGo_println_allowed(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

func main() {
	defer println("ok")
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
