package transformergo_test

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/parser"
	"forst/internal/testutil"
	transformergo "forst/internal/transformer/go"
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
		t.Fatal("no transformer/go refinement fixtures")
	}
	for _, fx := range fixtures {
		fx := fx
		t.Run(testutil.SanitizeTestName(fx.Meta.ID), func(t *testing.T) {
			if err := checkGoFixture(fx); err != nil {
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

func checkGoFixture(fx testutil.Fixture) (err error) {
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
	tr := transformergo.New(tc, log, false)
	goFile, err := tr.TransformForstFileToGo(nodes)
	if err != nil {
		return fmt.Errorf("transform: %w", err)
	}
	if fx.Meta.ID == "literal-union/ts-emit-literal-union" {
		var buf bytes.Buffer
		if err := format.Node(&buf, token.NewFileSet(), goFile); err != nil {
			return fmt.Errorf("format: %w", err)
		}
		out := buf.String()
		for _, needle := range []string{
			"type TaskStatus string",
			"TaskStatus_todo",
			"TaskStatus_done",
			"func isTaskStatus",
		} {
			if !strings.Contains(out, needle) {
				return fmt.Errorf("Go literal-union emit missing %q in:\n%s", needle, out)
			}
		}
	}
	switch fx.Meta.Expect {
	case "emit-ok", "runtime-false":
		return nil
	default:
		return nil
	}
}
