package compiler_test

import (
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/compiler"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestTypecheckForCompileEntry_crossPkgHandle(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	entry := filepath.Join(root, "api", "handle.ft")
	logger := logrus.New()
	logger.SetOutput(nil)
	logger.SetLevel(logrus.PanicLevel)

	c := compiler.New(compiler.Args{
		Command:  "run",
		FilePath: entry,
		LogLevel: "error",
	}, logger)
	tc, modResult, err := c.TypecheckForCompileEntry()
	if err != nil {
		t.Fatalf("TypecheckForCompileEntry: %v", err)
	}
	if modResult == nil {
		t.Fatal("expected module result")
	}
	apiTC := modResult.ForstPackageTypeChecker("api")
	if apiTC == nil {
		t.Fatal("missing api typechecker")
	}
	slots := apiTC.FunctionProviders[ast.Identifier("HandleRequest")]
	roots := typechecker.ProviderRootIdentsFromSlots(slots)
	if len(roots) != 1 || roots[0] != "Logger" {
		t.Fatalf("HandleRequest roots = %v", roots)
	}
	if tc != apiTC {
		direct := typechecker.ProviderRootIdentsFromSlots(tc.FunctionProviders[ast.Identifier("HandleRequest")])
		if len(direct) != 1 || direct[0] != "Logger" {
			t.Fatalf("returned tc HandleRequest roots = %v", direct)
		}
	}
}
