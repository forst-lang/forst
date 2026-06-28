package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func TestStdLoggerSatisfiesLoggerContract(t *testing.T) {
	src := `package main

type Logger = {
	info(msg String)
	error(msg String)
}

type StdLogger = {
	level: String,
}

func (l StdLogger) info(msg String) {}

func (l StdLogger) error(msg String) {}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}

	methods, ok := tc.TypeMethods["StdLogger"]
	if !ok {
		t.Fatal("expected methods registered on StdLogger")
	}
	if _, ok := methods["info"]; !ok {
		t.Fatal("expected info method on StdLogger")
	}
	if _, ok := methods["error"]; !ok {
		t.Fatal("expected error method on StdLogger")
	}

	if !tc.IsTypeCompatible(ast.TypeNode{Ident: "StdLogger"}, ast.TypeNode{Ident: "Logger"}) {
		t.Fatal("StdLogger should structurally satisfy Logger contract")
	}
}

func TestGoQualifiedTypeAliasStableKey(t *testing.T) {
	src := `package main
import "io"

type W = io.Writer
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	key := tc.ProviderContractKey(ast.TypeNode{Ident: "W"})
	if key != "io.Writer" {
		t.Fatalf("ProviderContractKey(W): got %q want io.Writer", key)
	}
	keyDirect := tc.ProviderContractKey(ast.TypeNode{Ident: "io.Writer"})
	if keyDirect != "io.Writer" {
		t.Fatalf("ProviderContractKey(io.Writer): got %q", keyDirect)
	}
}

func TestUserTypeMethodCall(t *testing.T) {
	src := `package main

type Logger = {
	info(msg String)
}

type StdLogger = {}

func (l StdLogger) info(msg String) {}

func demo(l StdLogger) {
	l.info("hello")
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
}

func TestCheckContractShapeMethod_onContractType(t *testing.T) {
	src := `package main

type Clock = {
	now(): Int
}

func f() {
	use clock: Clock
	_ := clock.now()
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck contract method via use binding: %v", err)
	}
}

func TestCheckContractShapeMethod_wrongArity_errors(t *testing.T) {
	tc := New(nil, false)
	tc.Defs["Logger"] = ast.TypeDefNode{
		Ident: "Logger",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"info": {
						IsMethod: true,
						MethodParams: []ast.ParamNode{
							ast.SimpleParamNode{
								Ident: ast.Ident{ID: "msg"},
								Type:  ast.TypeNode{Ident: ast.TypeString},
							},
						},
					},
				},
			},
		},
	}
	_, err := tc.checkContractShapeMethod(ast.TypeNode{Ident: "Logger"}, "info", []ast.ExpressionNode{
		ast.StringLiteralNode{Value: "a"},
		ast.StringLiteralNode{Value: "b"},
	})
	if err == nil {
		t.Fatal("expected arity error for contract method call")
	}
}

func TestLookupFunctionReturnType_receiverMethod(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	tc := New(log, false)
	fn := &ast.FunctionNode{
		Ident: ast.Ident{ID: "info"},
		Receiver: &ast.SimpleParamNode{
			Type: ast.TypeNode{Ident: "NopLogger"},
		},
	}
	tc.TypeMethods = map[ast.TypeIdent]map[string]FunctionSignature{
		"NopLogger": {
			"info": {ReturnTypes: []ast.TypeNode{{Ident: ast.TypeVoid}}},
		},
	}
	got, err := tc.LookupFunctionReturnType(fn)
	if err != nil {
		t.Fatal(err)
	}
	if !IsVoidReturnTypes(got) {
		t.Fatalf("got %v, want void", formatTypeList(got))
	}
}

func TestCheckTypes_receiverMethod_trailingPrintln_voidContract_errors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module recvtest\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package main

import "fmt"

type Logger = { info(msg String) }
type NopLogger = {}

func (NopLogger) info(msg String) {
	fmt.Println(msg)
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected typecheck error for void contract with inferred non-void return")
	}
	if !strings.Contains(err.Error(), "contract requires void") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckTypes_receiverMethod_trailingPrintln_storesReturnInTypeMethods(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module recvtest\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package main

import "fmt"

type NopLogger = {}

func (NopLogger) log(msg String): Result(Int, Error) {
	fmt.Println(msg)
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	sig, ok := tc.TypeMethods["NopLogger"]["log"]
	if !ok {
		t.Fatal("missing log method on NopLogger")
	}
	if len(sig.ReturnTypes) != 1 || !sig.ReturnTypes[0].IsResultType() {
		t.Fatalf("log return types = %v, want Result(...)", formatTypeList(sig.ReturnTypes))
	}
	if _, ok := tc.Functions[ast.Identifier("log")]; ok {
		t.Fatal("receiver method log should not pollute Functions map")
	}
}
