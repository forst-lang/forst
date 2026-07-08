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

type CommandOptions = {
  extraArgs: []String,
  invalid: Bool,
}

func parseOptions(args []String): CommandOptions {
  extraArgs := []String{}
  extraArgs = append(extraArgs, args[0])
  return CommandOptions { extraArgs: []String{}, invalid: false }
}
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(src, "parseoptions.ft", log)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(out, `[]String{}`) {
		t.Fatalf("expected []String{} in output, got:\n%s", out)
	}
	if strings.Contains(out, ":= []\n") || strings.Contains(out, "extraArgs: [],") {
		t.Fatalf("typed empty slice must not lose element type, got:\n%s", out)
	}
	l := lexer.New([]byte(out), "parseoptions.ft", log)
	p := parser.New(l.Lex(), "parseoptions.ft", log)
	if _, err := p.ParseFile(); err != nil {
		t.Fatalf("re-parse: %v\n--- out ---\n%s", err, out)
	}
}

func TestFormatSource_returnTypedShapeLiteral_indentsFields(t *testing.T) {
	t.Parallel()
	const src = `package main

type CommandOptions = {
  name: String,
  workDir: String,
  dryRun: Bool,
  extraArgs: []String,
  invalid: Bool,
}

func parseOptions(args []String): CommandOptions {
  if len(args) == 0 {
    return CommandOptions { name: "", workDir: "", dryRun: false, extraArgs: []String{}, invalid: true }
  }
  return CommandOptions { name: "auto", workDir: ".", dryRun: false, extraArgs: []String{}, invalid: false }
}
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(src, "parseoptions.ft", log)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, `name: ""`) || strings.HasPrefix(trimmed, `name: "auto"`) {
			if !strings.HasPrefix(line, "\t\t") {
				t.Fatalf("expected return shape field to be double-indented, got %q in:\n%s", line, out)
			}
		}
	}
	l := lexer.New([]byte(out), "parseoptions.ft", log)
	p := parser.New(l.Lex(), "parseoptions.ft", log)
	if _, err := p.ParseFile(); err != nil {
		t.Fatalf("re-parse: %v\n--- out ---\n%s", err, out)
	}
}
