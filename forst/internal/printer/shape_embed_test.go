package printer

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestPrintShapeFieldEntry_embeddedOmitsColon(t *testing.T) {
	t.Parallel()
	p := &printer{cfg: Config{Indent: "  "}}
	inner := ast.TypeIdent("Inner")
	got, err := p.printShapeFieldEntry("Inner", ast.ShapeFieldNode{
		Type:     &ast.TypeNode{Ident: inner},
		Embedded: true,
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Inner" {
		t.Fatalf("embedded field = %q, want Inner", got)
	}
}

func TestPrintShapeOneLine_embeddedField(t *testing.T) {
	t.Parallel()
	p := &printer{cfg: Config{Indent: "  "}}
	inner := ast.TypeIdent("Inner")
	got, err := p.printShapeOneLine(ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"Inner": {
				Type:     &ast.TypeNode{Ident: inner},
				Embedded: true,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "{Inner}" {
		t.Fatalf("shape = %q, want {Inner}", got)
	}
	if strings.Contains(got, ":") {
		t.Fatalf("embedded shape must not contain a colon: %q", got)
	}
}

func TestFormatTypeDefNode_embeddedField(t *testing.T) {
	t.Parallel()
	inner := ast.TypeIdent("Inner")
	typeDef := ast.TypeDefNode{
		Ident: "Outer",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"Inner": {
						Type:     &ast.TypeNode{Ident: inner},
						Embedded: true,
					},
				},
			},
		},
	}
	out, err := FormatTypeDefNode(Config{Indent: "\t"}, typeDef)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "{Inner}") && !strings.Contains(out, "{\n\tInner,\n}") {
		t.Fatalf("typedef must keep embedded syntax, got:\n%s", out)
	}
	if strings.Contains(out, "Inner:") {
		t.Fatalf("fmt must not rewrite embedding to named field:\n%s", out)
	}
}

func TestFormatSource_structEmbedding_preservesTypeOnlyMember(t *testing.T) {
	t.Parallel()
	src := `package main

type Inner = {Value: Int}

type Outer = {Inner}

func main() {
	o := Outer{Inner: {Value: 10}}
	println(o.Value)
}
`
	out, err := FormatSource(src, "struct-embedding.ft", testLogger())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "type Outer = {Inner: Inner}") || strings.Contains(out, "Inner: Inner") {
		t.Fatalf("fmt rewrote embedding to named field:\n%s", out)
	}
	if !strings.Contains(out, "type Outer = {Inner}") && !strings.Contains(out, "type Outer = {\n\tInner,\n}") {
		t.Fatalf("fmt dropped embedded field:\n%s", out)
	}
}