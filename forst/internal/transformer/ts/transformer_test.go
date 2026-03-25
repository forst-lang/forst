package transformerts

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestTransformForstFileToTypeScript_echoShapeAndFunction(t *testing.T) {
	const src = `package main

type EchoRequest = {
	message: String
}

func Echo(input EchoRequest) {
	return {
		echo: input.message,
		timestamp: 1234567890,
	}
}
`
	logger := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	tc := typechecker.New(nil, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}

	tr := New(tc, logger)
	out, err := tr.TransformForstFileToTypeScript(nodes, "")
	if err != nil {
		t.Fatalf("transform: %v", err)
	}

	typesFile := out.GenerateTypesFile()
	for _, fragment := range []string{
		"export interface EchoRequest",
		"export function Echo(",
		"Promise<",
	} {
		if !strings.Contains(typesFile, fragment) {
			t.Fatalf("types file missing %q:\n%s", fragment, typesFile)
		}
	}

	clientFile := out.GenerateClientFile()
	for _, fragment := range []string{
		"invokeFunction",
		"'main'",
		"Echo",
	} {
		if !strings.Contains(clientFile, fragment) {
			t.Fatalf("client file missing %q:\n%s", fragment, clientFile)
		}
	}
}

func TestTransformForstFileToTypeScript_setsPackageName(t *testing.T) {
	const src = `package sidecartest

func Ping() {
	return 1
}
`
	logger := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	tc := typechecker.New(logger, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}

	tr := New(tc, logger)
	out, err := tr.TransformForstFileToTypeScript(nodes, "")
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if out.PackageName != "sidecartest" {
		t.Fatalf("PackageName: got %q, want sidecartest", out.PackageName)
	}
	if !strings.Contains(out.GenerateClientFile(), "'sidecartest'") {
		t.Fatalf("expected client to use package sidecartest:\n%s", out.GenerateClientFile())
	}
}

func TestTransformForstFileToTypeScript_sourceFileStemNamesExportNotPackage(t *testing.T) {
	const src = `package main

func F() { return 1 }
`
	logger := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := typechecker.New(logger, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	tr := New(tc, logger)
	out, err := tr.TransformForstFileToTypeScript(nodes, "api")
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	client := out.GenerateClientFile()
	if !strings.Contains(client, "export const api") {
		t.Fatalf("want export const api, got:\n%s", client)
	}
	if !strings.Contains(client, "'main'") {
		t.Fatalf("invokeFunction should still use Forst package main, got:\n%s", client)
	}
}

func TestNew_nilLoggerUsesDefault(t *testing.T) {
	tc := typechecker.New(nil, false)
	tr := New(tc, nil)
	if tr.log == nil {
		t.Fatal("expected non-nil logger")
	}
	// Exercise one call path that uses the logger without failing.
	tr.log.SetLevel(logrus.ErrorLevel)
}

func TestTransformForstFileToTypeScript_arrayMapAndPointerInShape(t *testing.T) {
	const src = `package main

type Row = {
	names: []String
	scores: map[String]Int
	next: *String
}

func Count() {
	return 0
}
`
	logger := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	tc := typechecker.New(nil, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}

	tr := New(tc, logger)
	out, err := tr.TransformForstFileToTypeScript(nodes, "")
	if err != nil {
		t.Fatalf("transform: %v", err)
	}

	typesFile := out.GenerateTypesFile()
	for _, frag := range []string{
		"string[]",
		"Record<string, number>",
		"(string) | null",
	} {
		if !strings.Contains(typesFile, frag) {
			t.Fatalf("types file missing %q:\n%s", frag, typesFile)
		}
	}
}
