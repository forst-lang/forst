package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

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
	key := tc.UsableContractKey(ast.TypeNode{Ident: "W"})
	if key != "io.Writer" {
		t.Fatalf("UsableContractKey(W): got %q want io.Writer", key)
	}
	keyDirect := tc.UsableContractKey(ast.TypeNode{Ident: "io.Writer"})
	if keyDirect != "io.Writer" {
		t.Fatalf("UsableContractKey(io.Writer): got %q", keyDirect)
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
