package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestListMembersForExpression_goQualifiedCall(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	log := setupTestLogger(nil)
	src := `package main

import "time"

func main() {
	now := time.Now()
	println(now)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	if !tc.GoImportPackageLoaded("time") {
		t.Skip("time package not loaded")
	}
	var mainFn ast.FunctionNode
	for _, n := range nodes {
		if fn, ok := n.(ast.FunctionNode); ok {
			mainFn = fn
			break
		}
	}
	var call ast.FunctionCallNode
	for _, st := range mainFn.Body {
		asg, ok := st.(ast.AssignmentNode)
		if !ok || len(asg.RValues) != 1 {
			continue
		}
		if c, ok := asg.RValues[0].(ast.FunctionCallNode); ok {
			call = c
			break
		}
	}
	members := tc.ListMembersForExpression(call)
	if len(members) == 0 {
		t.Fatal("expected Go methods on time.Now()")
	}
	found := false
	for _, m := range members {
		if m == "Format" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Format in %v", members)
	}
}

func TestTopLevelPackageVariables(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

var Version = "1.0"

func main() {}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	vars := tc.TopLevelPackageVariables()
	if len(vars) != 1 || string(vars[0]) != "Version" {
		t.Fatalf("TopLevelPackageVariables = %v", vars)
	}
}
