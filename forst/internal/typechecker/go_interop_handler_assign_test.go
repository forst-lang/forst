package typechecker

import (
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestFieldAssign_serveMuxAssignableToServerHandler(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main

import "net/http"

func main() {
	mux := http.NewServeMux()
	srv := &http.Server{}
	srv.Handler = mux
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
}
