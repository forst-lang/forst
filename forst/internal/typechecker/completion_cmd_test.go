package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestListMembersForExpression_cmdAfterExecCommand(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main

import "os/exec"

func main() {
	cmd := exec.Command("true")
	cmd.Run()
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	if !tc.GoImportPackageLoaded("exec") {
		t.Skip("os/exec not loaded")
	}
	if tc.GoTypeForVariable(ast.Identifier("cmd")) == nil {
		t.Fatal("expected Go type for cmd")
	}
	members := tc.ListMembersForExpression(ast.VariableNode{Ident: ast.Ident{ID: "cmd"}})
	if len(members) == 0 {
		t.Fatalf("expected Go members on cmd, got none; variableGoTypes=%v", tc.GoTypeForVariable("cmd"))
	}
	found := false
	for _, m := range members {
		if m == "Run" || m == "ProcessState" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Run or ProcessState in %v", members)
	}
}

func TestListMembersForExpression_cmdProcessStateField(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main

import "os/exec"

func main() {
	cmd := exec.Command("true")
	cmd.ProcessState.ExitCode()
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	if !tc.GoImportPackageLoaded("exec") {
		t.Skip("os/exec not loaded")
	}
	expr := ast.VariableNode{Ident: ast.Ident{ID: "cmd.ProcessState"}}
	members := tc.ListMembersForExpression(expr)
	if len(members) == 0 {
		t.Fatal("expected members on cmd.ProcessState")
	}
	found := false
	for _, m := range members {
		if m == "ExitCode" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ExitCode in %v", members)
	}
}
