package lsp

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"

	"github.com/sirupsen/logrus"
)

func TestReceiverExpressionSourceBeforeDot_qualifiedCall(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "time", Line: 1, Column: 1},
		{Type: ast.TokenDot, Value: ".", Line: 1, Column: 5},
		{Type: ast.TokenIdentifier, Value: "Now", Line: 1, Column: 6},
		{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
		{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
		{Type: ast.TokenDot, Value: ".", Line: 1, Column: 11},
		{Type: ast.TokenIdentifier, Value: "Format", Line: 1, Column: 12},
	}
	src, ok := receiverExpressionSourceBeforeDot(tokens, 5)
	if !ok {
		t.Fatal("expected ok")
	}
	if src != "time.Now()" {
		t.Fatalf("got %q", src)
	}
}

func TestReceiverExpressionSourceBeforeDot_simpleIdent(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 1},
		{Type: ast.TokenDot, Value: ".", Line: 1, Column: 2},
		{Type: ast.TokenIdentifier, Value: "y", Line: 1, Column: 3},
	}
	src, ok := receiverExpressionSourceBeforeDot(tokens, 1)
	if !ok || src != "x" {
		t.Fatalf("got %q ok=%v", src, ok)
	}
}

func TestReceiverExpressionSourceBeforeDot_cmdProcessState(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "cmd", Line: 1, Column: 1},
		{Type: ast.TokenDot, Value: ".", Line: 1, Column: 4},
		{Type: ast.TokenIdentifier, Value: "ProcessState", Line: 1, Column: 5},
		{Type: ast.TokenDot, Value: ".", Line: 1, Column: 17},
		{Type: ast.TokenIdentifier, Value: "ExitCode", Line: 1, Column: 18},
	}
	src, ok := receiverExpressionSourceBeforeDot(tokens, 3)
	if !ok {
		t.Fatal("expected ok")
	}
	if src != "cmd.ProcessState" {
		t.Fatalf("got %q", src)
	}
}

func TestLspPositionAfterDot_cmdProcessStateExitCode(t *testing.T) {
	const src = "package main\n\nimport \"os/exec\"\n\nfunc main() {\n\tcmd := exec.Command(\"true\")\n\tcmd.ProcessState.ExitCode()\n}\n"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	var dotTok *ast.Token
	for i := range toks {
		if toks[i].Type == ast.TokenIdentifier && toks[i].Value == "ExitCode" && i > 0 && toks[i-1].Type == ast.TokenDot {
			dotTok = &toks[i-1]
			break
		}
	}
	if dotTok == nil {
		t.Fatal("dot before ExitCode not found")
	}
	pos := lspPositionAfterDot(src, *dotTok)
	dotIdx := memberAccessDotIndex(toks, pos)
	if dotIdx < 0 {
		t.Fatalf("memberAccessDotIndex failed at %+v", pos)
	}
	recv, ok := parseReceiverExpressionBeforeDot(toks, dotIdx, "t.ft")
	if !ok {
		t.Fatal("parseReceiver failed")
	}
	if vn, ok := recv.(ast.VariableNode); !ok || string(vn.Ident.ID) != "cmd.ProcessState" {
		t.Fatalf("receiver = %#v", recv)
	}
}

func TestLspPositionAfterDot_execCommandCallResult(t *testing.T) {
	const src = "package main\n\nimport \"os/exec\"\n\nfunc main() {\n\texec.Command(\"true\").Run()\n}\n"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	var dotTok *ast.Token
	for i := range toks {
		if toks[i].Type == ast.TokenIdentifier && toks[i].Value == "Run" && i > 0 && toks[i-1].Type == ast.TokenDot {
			dotTok = &toks[i-1]
			break
		}
	}
	if dotTok == nil {
		t.Fatal("dot before Run not found")
	}
	pos := lspPositionAfterDot(src, *dotTok)
	dotIdx := memberAccessDotIndex(toks, pos)
	if dotIdx < 0 {
		t.Fatalf("memberAccessDotIndex failed at %+v", pos)
	}
	recv, ok := parseReceiverExpressionBeforeDot(toks, dotIdx, "t.ft")
	if !ok {
		t.Fatal("parseReceiver failed")
	}
	if _, ok := recv.(ast.FunctionCallNode); !ok {
		t.Fatalf("receiver = %#v", recv)
	}
}

func TestReceiverExpressionSourceBeforeDot_argvSubslice(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "argv", Line: 1, Column: 1},
		{Type: ast.TokenLBracket, Value: "[", Line: 1, Column: 5},
		{Type: ast.TokenIntLiteral, Value: "1", Line: 1, Column: 6},
		{Type: ast.TokenColon, Value: ":", Line: 1, Column: 7},
		{Type: ast.TokenRBracket, Value: "]", Line: 1, Column: 8},
		{Type: ast.TokenDot, Value: ".", Line: 1, Column: 9},
		{Type: ast.TokenIdentifier, Value: "Len", Line: 1, Column: 10},
	}
	src, ok := receiverExpressionSourceBeforeDot(tokens, 5)
	if !ok {
		t.Fatal("expected ok")
	}
	if src != "argv[1:]" {
		t.Fatalf("got %q", src)
	}
}
