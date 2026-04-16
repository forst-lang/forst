package typechecker

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/parser"
)

// TestCheckTypes_examplesBundle loads several large example files to cover builtin dispatch,
// ensure, pointers, and go interop paths in isolation (typecheck-only, faster than full pipeline).
func TestCheckTypes_examplesBundle(t *testing.T) {
	t.Parallel()
	rel := []string{
		"basic.ft",
		"loop.ft",
		"echo.ft",
		"basic_function.ft",
		"result_if.ft",
		"result_ensure.ft",
		"generics.ft",
		"go_builtins.ft",
		"ensure.ft",
		"pointers.ft",
		"union_error_types.ft",
		"union_error_narrowing.ft",
		"nominal_error.ft",
	}
	root := filepath.Join("..", "..", "..", "examples", "in")
	for _, name := range rel {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(root, name)
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			log := setupTestLogger(nil)
			pr := parser.NewTestParser(string(src), log)
			nodes, err := pr.ParseFile()
			if err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
			chk := New(log, false)
			if err := chk.CheckTypes(nodes); err != nil {
				t.Fatalf("CheckTypes %s: %v", name, err)
			}
		})
	}
}
