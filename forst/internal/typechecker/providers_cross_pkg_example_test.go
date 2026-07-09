package typechecker

import (
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"
)

func TestCheckTypes_crossPkgAuth_logEvent_allowedInLibraryPackage(t *testing.T) {
	src := `package auth

type Logger = { info(msg String) }

func LogEvent(id String) {
	use logger: Logger
	logger.info("expire " + id)
}
`
	log := testLogger(t)
	toks := lexer.New([]byte(src), "log.ft", log).Lex()
	nodes, err := parser.New(toks, "log.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("library package export with Providers should typecheck: %v", err)
	}
	slots := tc.FunctionProviders["LogEvent"]
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("LogEvent providers = %v", slots)
	}
}

func TestCheckTypes_sidecarExport_onlyMainPackage(t *testing.T) {
	src := `package lib

type Logger = { info(msg String) }

func PublicApi() {
	use logger: Logger
}
`
	log := testLogger(t)
	toks := lexer.New([]byte(src), "lib.ft", log).Lex()
	nodes, err := parser.New(toks, "lib.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("non-main public API with Providers should typecheck: %v", err)
	}
}
