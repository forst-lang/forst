package parser

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
)

func TestParseFunctionCallSpans(t *testing.T) {
	src := "package p\nfunc main() {\n  fmt.Println(1, x)\n}\n"
	logger := ast.SetupTestLogger(nil)
	lex := lexer.New([]byte(src), "t.ft", logger)
	toks := lex.Lex()
	p := New(toks, "t.ft", logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var fn *ast.FunctionNode
	for _, n := range nodes {
		if f, ok := n.(ast.FunctionNode); ok && f.Ident.ID == "main" {
			fn = &f
			break
		}
	}
	if fn == nil {
		t.Fatal("main not found")
	}
	var call ast.FunctionCallNode
	for _, st := range fn.Body {
		if c, ok := st.(ast.FunctionCallNode); ok {
			call = c
			break
		}
	}
	if call.Function.ID != "fmt.Println" {
		t.Fatalf("call function = %q", call.Function.ID)
	}
	if !call.Function.Span.IsSet() {
		t.Fatal("Function.Span not set")
	}
	if call.Function.Span.StartLine != 3 || call.Function.Span.StartCol != 3 {
		t.Errorf("Function span start = %d:%d want 3:3", call.Function.Span.StartLine, call.Function.Span.StartCol)
	}
	if !call.CallSpan.IsSet() {
		t.Fatal("CallSpan not set")
	}
	if len(call.Arguments) != 2 || len(call.ArgSpans) != 2 {
		t.Fatalf("args/spans len got %d/%d", len(call.Arguments), len(call.ArgSpans))
	}
	if !call.ArgSpans[0].IsSet() || !call.ArgSpans[1].IsSet() {
		t.Fatal("ArgSpans not set")
	}
	// First arg is literal 1 on same line as call
	if call.ArgSpans[0].StartLine != 3 {
		t.Errorf("arg0 span line %d want 3", call.ArgSpans[0].StartLine)
	}
}
