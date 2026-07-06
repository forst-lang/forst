package printer

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestPrintExpr_typedEmptySliceComposite(t *testing.T) {
	t.Parallel()
	var p printer
	p.cfg = DefaultConfig()
	got, err := p.printExpr(ast.ArrayLiteralNode{
		Type: ast.NewBuiltinType(ast.TypeString),
	})
	if err != nil {
		t.Fatalf("printExpr: %v", err)
	}
	if got != `[]String{}` {
		t.Fatalf("got %q want %q", got, `[]String{}`)
	}
}

func TestPrintExpr_typedSliceCompositeWithElements(t *testing.T) {
	t.Parallel()
	var p printer
	p.cfg = DefaultConfig()
	got, err := p.printExpr(ast.ArrayLiteralNode{
		Type:  ast.NewBuiltinType(ast.TypeString),
		Value: []ast.ExpressionNode{ast.StringLiteralNode{Value: "bun"}},
	})
	if err != nil {
		t.Fatalf("printExpr: %v", err)
	}
	if got != `[]String{"bun"}` {
		t.Fatalf("got %q want %q", got, `[]String{"bun"}`)
	}
}

func TestFormatSource_typedEmptySliceComposite_preservesElementType(t *testing.T) {
	t.Parallel()
	const src = `package main

type ParsedArgs = {
  forwarded: []String,
  invalid: Bool,
}

func ParseArgs(args []String): ParsedArgs {
  forwarded := []String{}
  forwarded = append(forwarded, args[0])
  return ParsedArgs { forwarded: []String{}, invalid: false }
}
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(src, "parseargs.ft", log)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(out, `[]String{}`) {
		t.Fatalf("expected []String{} in output, got:\n%s", out)
	}
	if strings.Contains(out, ":= []\n") || strings.Contains(out, "forwarded: [],") {
		t.Fatalf("typed empty slice must not lose element type, got:\n%s", out)
	}
	l := lexer.New([]byte(out), "parseargs.ft", log)
	p := parser.New(l.Lex(), "parseargs.ft", log)
	if _, err := p.ParseFile(); err != nil {
		t.Fatalf("re-parse: %v\n--- out ---\n%s", err, out)
	}
}

func TestFormatSource_returnTypedShapeLiteral_indentsFields(t *testing.T) {
	t.Parallel()
	const src = `package main

type ParsedArgs = {
  runner: String,
  projectRoot: String,
  dryRun: Bool,
  forwarded: []String,
  invalid: Bool,
}

func ParseArgs(args []String): ParsedArgs {
  if len(args) == 0 {
    return ParsedArgs { runner: "", projectRoot: "", dryRun: false, forwarded: []String{}, invalid: true }
  }
  return ParsedArgs { runner: "auto", projectRoot: ".", dryRun: false, forwarded: []String{}, invalid: false }
}
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(src, "parseargs.ft", log)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	// Multiline return shape fields must be indented inside the function body.
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, `runner: ""`) || strings.HasPrefix(trimmed, `runner: "auto"`) {
			if !strings.HasPrefix(line, "\t\t") {
				t.Fatalf("expected return shape field to be double-indented, got %q in:\n%s", line, out)
			}
		}
	}
	l := lexer.New([]byte(out), "parseargs.ft", log)
	p := parser.New(l.Lex(), "parseargs.ft", log)
	if _, err := p.ParseFile(); err != nil {
		t.Fatalf("re-parse: %v\n--- out ---\n%s", err, out)
	}
}
