package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func typecheckSwitchSource(t *testing.T, src string) error {
	t.Helper()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	return tc.CheckTypes(nodes)
}

func TestInferSwitch_tagAndBooleanForms(t *testing.T) {
	t.Parallel()
	if err := typecheckSwitchSource(t, `package main

func classify(n: Int): String {
	switch n {
	case 1, 2:
		return "small"
	case 3:
		return "three"
	default:
		return "other"
	}
}

func pick(flag: Bool): String {
	switch {
	case flag:
		return "yes"
	default:
		return "no"
	}
}
`); err != nil {
		t.Fatal(err)
	}
}

func TestInferSwitch_duplicateCaseLiteralRejected(t *testing.T) {
	t.Parallel()
	err := typecheckSwitchSource(t, `package main

func main() {
	switch 1 {
	case 1:
		println("a")
	case 1:
		println("b")
	}
}
`)
	if err == nil {
		t.Fatal("expected duplicate case error")
	}
	if !strings.Contains(err.Error(), "duplicate case") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInferSwitch_fallthroughOutsideSwitchRejected(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "main"},
		Body: []ast.Node{
			ast.FallthroughNode{},
		},
	}
	if err := tc.CheckTypes([]ast.Node{fn}); err == nil {
		t.Fatal("expected fallthrough outside switch error")
	}
}

func TestInferSwitch_incompatibleCaseTypeRejected(t *testing.T) {
	t.Parallel()
	err := typecheckSwitchSource(t, `package main

func main() {
	switch 1 {
	case "x":
		println("bad")
	}
}
`)
	if err == nil {
		t.Fatal("expected incompatible case type error")
	}
	if !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("unexpected error: %v", err)
	}
}
