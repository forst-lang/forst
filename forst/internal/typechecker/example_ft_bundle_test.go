package typechecker

import (
	"os"
	"testing"

	"forst/internal/testutil"
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
		"map_catalog.ft",
		"tictactoe/engine.ft",
		"rfc/guard/basic_guard.ft",
		"rfc/guard/shape_guard.ft",
		"rfc/providers/providers.ft",
	}
	for _, name := range rel {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := testutil.ExamplePath(t, name)
			srcBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			MustTypecheck(t, string(srcBytes), testutil.TypecheckOpts{
				FileID: name,
			})
		})
	}
}
