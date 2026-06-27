package typechecker

import (
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"
)

func TestCheckTypes_xpkgAlpha_logExpiry_allowedInLibraryPackage(t *testing.T) {
	src := `package alpha

type Logger = { info(msg String) }

func LogExpiry(id String) {
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
		t.Fatalf("library package export with Usables should typecheck: %v", err)
	}
	slots := tc.FunctionUsables["LogExpiry"]
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("LogExpiry usables = %v", slots)
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
		t.Fatalf("non-main public API with Usables should typecheck: %v", err)
	}
}
