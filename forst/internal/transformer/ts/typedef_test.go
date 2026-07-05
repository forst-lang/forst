package transformerts

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestTransformTypeDef_errorNominalEmitsInterface(t *testing.T) {
	tr := New(typechecker.New(logrus.New(), false), nil)
	out, err := tr.transformTypeDef(ast.TypeDefNode{
		Ident: "NotPositive",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"field": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "export interface NotPositive") {
		t.Fatalf("expected interface, got:\n%s", out)
	}
	if !strings.Contains(out, "field") || !strings.Contains(out, "string") {
		t.Fatalf("expected payload field in TS output:\n%s", out)
	}
}

func TestTransformTypeDef_assertionAlias_emitsExtendsBase(t *testing.T) {
	tr := New(typechecker.New(logrus.New(), false), nil)
	base := ast.TypeString
	out, err := tr.transformTypeDef(ast.TypeDefNode{
		Ident: "Label",
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &base,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "export interface Label extends string") {
		t.Fatalf("expected assertion typedef to extend base TS type, got:\n%s", out)
	}
}

func TestTransformShapeToTypeScript_nilShape_errors(t *testing.T) {
	tr := New(typechecker.New(logrus.New(), false), nil)
	_, err := tr.transformShapeToTypeScript(nil, "T")
	if err == nil || !strings.Contains(err.Error(), "shape is nil") {
		t.Fatalf("expected nil shape error, got %v", err)
	}
}

func TestTransformAssertionToTypeScript_nilAssertion_errors(t *testing.T) {
	tr := New(typechecker.New(logrus.New(), false), nil)
	_, err := tr.transformAssertionToTypeScript(nil, "T")
	if err == nil || !strings.Contains(err.Error(), "assertion is nil") {
		t.Fatalf("expected nil assertion error, got %v", err)
	}
}

func TestTransformTypeDef_unionBinaryExpr_emitsExportType(t *testing.T) {
	const src = `package main

error ParseError { code: Int }
error IoError { path: String }
type ErrKind = ParseError | IoError
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

	var errKindDef ast.TypeDefNode
	var found bool
	for _, n := range nodes {
		td, ok := n.(ast.TypeDefNode)
		if !ok || td.Ident != "ErrKind" {
			continue
		}
		errKindDef = td
		found = true
		break
	}
	if !found {
		t.Fatal("ErrKind typedef not found")
	}

	out, err := tr.transformTypeDef(errKindDef)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "export type ErrKind") || !strings.Contains(out, "|") {
		t.Fatalf("expected union export type, got:\n%s", out)
	}
}

func TestTransformTypeDef_unsupportedExpr_errors(t *testing.T) {
	tr := New(typechecker.New(logrus.New(), false), nil)
	_, err := tr.transformTypeDef(ast.TypeDefNode{
		Ident: "Bad",
		Expr:  nil,
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported type definition expression") {
		t.Fatalf("expected unsupported expr error, got %v", err)
	}
}

func TestTransformAssertionToTypeScript_defaultsToAnyWithoutBaseType(t *testing.T) {
	tr := New(typechecker.New(logrus.New(), false), nil)
	out, err := tr.transformAssertionToTypeScript(&ast.AssertionNode{}, "NoBase")
	if err != nil {
		t.Fatal(err)
	}
	if out != "export interface NoBase extends any {}" {
		t.Fatalf("got %q", out)
	}
}

func TestTransformAssertionToTypeScript_wrapsBaseTypeMappingErrors(t *testing.T) {
	tr := New(typechecker.New(logrus.New(), false), nil)
	tr.typeMapping = &alwaysErrMapper{err: errForcedMapping}
	base := ast.TypeString
	_, err := tr.transformAssertionToTypeScript(&ast.AssertionNode{BaseType: &base}, "Bad")
	if err == nil || !strings.Contains(err.Error(), "failed to get TypeScript type for assertion base type") {
		t.Fatalf("expected wrapped base type error, got %v", err)
	}
}

func TestTransformTypeDef_binaryExpr_wrapsCanonicalizationErrors(t *testing.T) {
	tr := New(typechecker.New(logrus.New(), false), nil)
	_, err := tr.transformTypeDef(ast.TypeDefNode{
		Ident: "BadUnion",
		Expr: ast.TypeDefBinaryExpr{
			Left:  nil,
			Op:    ast.TokenBitwiseOr,
			Right: nil,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "binary typedef BadUnion") {
		t.Fatalf("expected wrapped binary typedef error, got %v", err)
	}
}
