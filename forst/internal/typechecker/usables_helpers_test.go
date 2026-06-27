package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestUsableBindingName_rootAndQualified(t *testing.T) {
	if got := usableBindingName("Logger"); got != "logger" {
		t.Fatalf("Logger -> %q", got)
	}
	if got := usableBindingName("io.Writer"); got != "writer" {
		t.Fatalf("io.Writer -> %q", got)
	}
}

func TestShapeIsMethodOnlyContract(t *testing.T) {
	methodOnly := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"info": {IsMethod: true, MethodParams: []ast.ParamNode{ast.SimpleParamNode{}}},
		},
	}
	if !methodOnly.IsMethodOnlyContract() {
		t.Fatal("expected method-only contract")
	}

	mixed := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"level": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"info":  {IsMethod: true},
		},
	}
	if mixed.IsMethodOnlyContract() {
		t.Fatal("data + method shape should not be method-only contract")
	}

	empty := ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}
	if empty.IsMethodOnlyContract() {
		t.Fatal("empty shape should not be method-only contract")
	}
}

func TestObligationChain_formatsPath(t *testing.T) {
	chain := obligationChain("TestX", []UsableSlot{
		{RootIdent: "Logger"},
		{RootIdent: "UserRepo"},
	})
	want := "TestX → Logger → UserRepo"
	if chain != want {
		t.Fatalf("got %q want %q", chain, want)
	}
}

func TestCallerForwardsUsables_andWiringRootIdent(t *testing.T) {
	tc := New(nil, false)
	tc.FunctionUsables = map[ast.Identifier][]UsableSlot{
		"outer": {{RootIdent: "Logger", Key: "Logger"}, {RootIdent: "Clock", Key: "Clock"}},
		"inner": {{RootIdent: "Logger", Key: "Logger"}},
		"TestX": {{RootIdent: "Logger", Key: "Logger"}},
	}
	tc.Functions = map[ast.Identifier]FunctionSignature{
		"TestX": {
			Parameters: []ParameterSignature{
				{Type: ast.TypeNode{
					Ident:      ast.TypePointer,
					TypeParams: []ast.TypeNode{{Ident: "testing.T"}},
				}},
			},
		},
	}

	if !tc.callerForwardsUsables("outer", "inner") {
		t.Fatal("outer should forward Logger to inner")
	}
	if tc.callerForwardsUsables("inner", "outer") {
		t.Fatal("inner should not forward to outer")
	}
	if !tc.isWiringRootIdent("main") {
		t.Fatal("main is wiring root")
	}
	if !tc.isWiringRootIdent("TestX") {
		t.Fatal("Test* with *testing.T is wiring root")
	}
	if tc.isWiringRootIdent("outer") {
		t.Fatal("outer is not a wiring root")
	}
}

func TestIsExcludedUsableContract_testingT(t *testing.T) {
	tc := New(nil, false)
	ptr := ast.TypeNode{
		Ident:      ast.TypePointer,
		TypeParams: []ast.TypeNode{{Ident: "testing.T"}},
	}
	if !tc.isExcludedUsableContract(ptr) {
		t.Fatal("expected *testing.T to be excluded from Usables")
	}
	if tc.isExcludedUsableContract(ast.TypeNode{Ident: "Logger"}) {
		t.Fatal("Logger should not be excluded")
	}
}
