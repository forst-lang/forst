package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestForstReceiverMethodHoverMarkdown_contractMethod(t *testing.T) {
	t.Parallel()
	src := `package main

type Messenger = {
  send(message String): Error,
}

func f() {
  use messenger: Messenger
  messenger.send("hello")
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatal(err)
	}
	md, ok := tc.ForstReceiverMethodHoverMarkdown(ast.TypeNode{Ident: ast.TypeIdent("Messenger")}, "send")
	if !ok || md == "" {
		t.Fatal("expected contract method hover")
	}
	if !strings.Contains(md, "send") || !strings.Contains(md, "message") || !strings.Contains(md, "Messenger") {
		t.Fatalf("unexpected hover: %q", md)
	}
}

func TestForstReceiverMethodHoverMarkdown_registeredReceiverMethod(t *testing.T) {
	t.Parallel()
	src := `package main

type Logger = { info(msg String) }

type StdLogger = {}

func (StdLogger) info(msg String) {
  println(msg)
}

func f() {
  use logger: Logger
  logger.info("hi")
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatal(err)
	}
	md, ok := tc.ForstReceiverMethodHoverMarkdown(ast.TypeNode{Ident: ast.TypeIdent("Logger")}, "info")
	if !ok || md == "" {
		t.Fatal("expected contract method hover for Logger.info")
	}
	if !strings.Contains(md, "info") {
		t.Fatalf("unexpected hover: %q", md)
	}
}
