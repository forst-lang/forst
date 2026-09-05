package typechecker

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

// thematicDirs lists typechecker refinement fixture topic roots.
var thematicDirs = []string{
	"guards",
	"assertion-ir",
	"identity",
	"transfer",
	"literal-union",
	"type-target",
	"boundary",
	"fact-deps",
	"clobber",
	"summary",
	"alias",
	"collection",
	"go-boundary",
	"escape",
	"hardening",
}

func TestRefinement_ThematicFixtures(t *testing.T) {
	for _, theme := range thematicDirs {
		theme := theme
		t.Run(theme, func(t *testing.T) {
			root := filepath.Join("testdata", "refinements", theme)
			fixtures, err := testutil.ListFixtureDirs(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(fixtures) == 0 {
				t.Fatalf("no fixtures under %s", theme)
			}
			for _, fx := range fixtures {
				fx := fx
				t.Run(testutil.SanitizeTestName(fx.Meta.ID), func(t *testing.T) {
					if err := checkTypecheckerFixture(fx); err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}

func checkTypecheckerFixture(fx testutil.Fixture) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%v", recovered)
		}
	}()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	var nodes []ast.Node
	var parseErr error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				parseErr = fmt.Errorf("%v", recovered)
			}
		}()
		nodes, parseErr = parser.NewTestParser(fx.Src, log).ParseFile()
	}()

	switch fx.Meta.Expect {
	case "typecheck-ok", "ir", "deps":
		if parseErr != nil {
			return fmt.Errorf("parse: %w", parseErr)
		}
		tc := New(log, false)
		if checkErr := tc.CheckTypes(nodes); checkErr != nil {
			return fmt.Errorf("%s: %w", fx.Meta.Expect, checkErr)
		}
		switch fx.Meta.Expect {
		case "ir":
			if fx.Meta.Matrix {
				return nil
			}
			return assertAssertionIR(fx, tc)
		case "deps":
			return assertFactDeps(fx, tc)
		}
		return nil
	case "typecheck-error":
		if parseErr != nil {
			return fmt.Errorf("typecheck-error fixture must parse; got parse error (move to parser): %w", parseErr)
		}
		checkErr := New(log, false).CheckTypes(nodes)
		if checkErr == nil {
			return fmt.Errorf("typecheck-error: expected error")
		}
		if fx.Meta.Code != "" && !strings.Contains(checkErr.Error(), fx.Meta.Code) {
			return fmt.Errorf("expected diagnostic code %q in %v", fx.Meta.Code, checkErr)
		}
		return nil
	default:
		if parseErr != nil {
			return parseErr
		}
		return New(log, false).CheckTypes(nodes)
	}
}
