package typechecker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/goload"
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
		"generic_function.ft",
		"generic_first.ft",
		"generic_pick.ft",
		"generic_eq.ft",
		"generic_map.ft",
		"generic_pointer.ft",
		"generic_variadic.ft",
		"go_builtins.ft",
		"slices.ft",
		"go_interop/cli.ft",
		"go_interop/tuple.ft",
		"go_interop/http_handle.ft",
		"go_interop/maps.ft",
		"ensure.ft",
		"pointers.ft",
		"union_error_types.ft",
		"union_error_narrowing.ft",
		"nominal_error.ft",
		"map_catalog.ft",
		"tictactoe/main/engine.ft",
		"rfc/guard/basic_guard.ft",
		"rfc/guard/shape_guard.ft",
		"rfc/providers/providers_demo/providers.ft",
	}
	for _, name := range rel {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := testutil.ExamplePath(t, name)
			srcBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			opts := testutil.TypecheckOpts{FileID: name}
			if strings.HasPrefix(name, "go_interop/") {
				dir := filepath.Dir(path)
				opts.GoWorkspaceDir = goload.FindModuleRoot(dir)
				opts.SamePackageGoImport = "go_interop"
				switch name {
				case "go_interop/cli.ft":
					opts.SkipUnlessGoImport = "exec"
				case "go_interop/tuple.ft":
					opts.SkipUnlessGoImport = "strconv"
				case "go_interop/http_handle.ft":
					opts.SkipUnlessGoImport = "http"
				case "go_interop/maps.ft":
					// same-package Go only; no import local required
				}
			}
			MustTypecheck(t, string(srcBytes), opts)
		})
	}
}
