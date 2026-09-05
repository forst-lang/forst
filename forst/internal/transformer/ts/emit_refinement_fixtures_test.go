package transformerts_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/parser"
	"forst/internal/testutil"
	transformerts "forst/internal/transformer/ts"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestEmit_RefinementFixtures(t *testing.T) {
	root := testdataRefinements(t)
	fixtures, err := testutil.ListFixtureDirs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no transformer/ts refinement fixtures")
	}
	for _, fx := range fixtures {
		fx := fx
		t.Run(testutil.SanitizeTestName(fx.Meta.ID), func(t *testing.T) {
			if err := checkTSFixture(fx); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func testdataRefinements(tb testing.TB) string {
	tb.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "refinements")
}

func checkTSFixture(fx testutil.Fixture) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%v", recovered)
		}
	}()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	nodes, err := parser.NewTestParser(fx.Src, log).ParseFile()
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		return fmt.Errorf("typecheck: %w", err)
	}
	tr := transformerts.New(tc, log)
	tsOut, err := tr.TransformForstFileToTypeScript(nodes, "")
	if err != nil {
		return fmt.Errorf("transform: %w", err)
	}
	out := tsOut.GenerateTypesFile()
	if fx.Meta.ID == "literal-union/literal-union-ts" {
		if !strings.Contains(out, `"todo"`) || !strings.Contains(out, `"done"`) || !strings.Contains(out, "|") {
			return fmt.Errorf("TS literal-union emit missing members in:\n%s", out)
		}
		if !strings.Contains(out, "export type") {
			return fmt.Errorf("TS literal-union emit missing export type in:\n%s", out)
		}
	}
	return nil
}
