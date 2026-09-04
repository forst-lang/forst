package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"forst/internal/ast"
)

// Local fixture loader — parser tests cannot import testutil (import cycle via parse.go).

type fixtureMeta struct {
	ID     string `json:"id"`
	Expect string `json:"expect"`
	Code   string `json:"code"`
}

type fixture struct {
	Dir  string
	Meta fixtureMeta
	Src  string
}

func loadFixtureDir(dir string) (fixture, error) {
	metaBytes, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return fixture{}, err
	}
	var m fixtureMeta
	if err := json.Unmarshal(metaBytes, &m); err != nil {
		return fixture{}, fmt.Errorf("%s/meta.json: %w", dir, err)
	}
	src, err := os.ReadFile(filepath.Join(dir, "input.ft"))
	if err != nil {
		return fixture{}, err
	}
	if m.ID == "" {
		m.ID = filepath.Base(dir)
	}
	return fixture{Dir: dir, Meta: m, Src: string(src)}, nil
}

func listFixtureDirs(root string) ([]fixture, error) {
	var out []fixture
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Name() != "meta.json" {
			return nil
		}
		fx, err := loadFixtureDir(filepath.Dir(path))
		if err != nil {
			return err
		}
		out = append(out, fx)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Meta.ID < out[j].Meta.ID })
	return out, nil
}

func sanitizeFixtureName(id string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-', r == '/':
			return r
		default:
			return '_'
		}
	}, id)
}

// TestEnsure_RefinementFixtures drives parser/testdata/refinements/{ensure,or,guards,failure}.
func TestEnsure_RefinementFixtures(t *testing.T) {
	root := filepath.Join("testdata", "refinements")
	fixtures, err := listFixtureDirs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no parser refinement fixtures")
	}
	for _, fx := range fixtures {
		fx := fx
		t.Run(sanitizeFixtureName(fx.Meta.ID), func(t *testing.T) {
			if err := checkParserFixture(t, fx); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func checkParserFixture(t *testing.T, fx fixture) (err error) {
	t.Helper()
	var nodes []ast.Node
	var parseErr error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				parseErr = fmt.Errorf("%v", recovered)
			}
		}()
		log := ast.SetupTestLoggerFor(t, nil)
		nodes, parseErr = NewTestParser(fx.Src, log).ParseFile()
	}()

	switch fx.Meta.Expect {
	case "parse-ok":
		if parseErr != nil {
			return fmt.Errorf("parse-ok: %w", parseErr)
		}
		if len(nodes) == 0 {
			return fmt.Errorf("parse-ok: empty AST")
		}
		return nil
	case "parse-error":
		if parseErr != nil {
			return nil
		}
		return fmt.Errorf("parse-error: expected parse failure")
	default:
		return parseErr
	}
}
